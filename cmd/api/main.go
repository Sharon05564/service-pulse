package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/sharonowusu/servicepulse/internal/config"
	"github.com/sharonowusu/servicepulse/internal/database"
	"github.com/sharonowusu/servicepulse/internal/handlers"
	"github.com/sharonowusu/servicepulse/internal/repository"
	"github.com/sharonowusu/servicepulse/internal/worker"
)

func main() {
	// Structured JSON logging — easy to parse in production.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// Load configuration from environment / .env file.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Connect to PostgreSQL.
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer db.Close()

	// Run SQL migrations from the migrations/ directory.
	// __FILE__ trick: resolve the path relative to the binary's source file so
	// it works both with `go run` and a compiled binary placed next to the project.
	migrationsDir := migrationsPath()
	if err := database.RunMigrations(db, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Build repositories.
	serviceRepo := repository.NewServiceRepository(db)
	checkRepo := repository.NewCheckRepository(db)
	incidentRepo := repository.NewIncidentRepository(db)

	// Build handlers.
	routerDeps := handlers.RouterDeps{
		Service:  handlers.NewServiceHandler(serviceRepo),
		Check:    handlers.NewCheckHandler(checkRepo),
		Incident: handlers.NewIncidentHandler(incidentRepo),
		Metrics:  handlers.NewMetricsHandler(checkRepo, incidentRepo),
	}
	router := handlers.NewRouter(routerDeps)

	// Start the background health checker in its own goroutine.
	checker := worker.NewHealthChecker(serviceRepo, checkRepo, incidentRepo, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go checker.Run(ctx)

	// Configure and start the HTTP server.
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run the server in a goroutine so we can listen for OS signals.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server started", "port", cfg.ServerPort, "env", cfg.Env)
		serverErr <- srv.ListenAndServe()
	}()

	// Block until we receive SIGINT or SIGTERM, then shut down gracefully.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig)
	}

	// Cancel the worker context first, then give in-flight HTTP requests 10 s.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped gracefully")
	return nil
}

// migrationsPath returns the path to the migrations directory.
// It walks up from the source file's directory so `go run ./cmd/api` works
// without needing to hard-code an absolute path.
func migrationsPath() string {
	// runtime.Caller(0) gives the path of this source file at compile time.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "migrations"
	}
	// cmd/api/main.go → ../../migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}
