package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// RunMigrations executes every *.sql file inside migrationsDir in
// lexicographic order. This is intentionally simple: each file should be
// idempotent (use IF NOT EXISTS, IF EXISTS, etc.) so re-running on startup
// is safe.
func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir %q: %w", migrationsDir, err)
	}

	// Collect and sort .sql files so they always run in numbered order.
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading migration %q: %w", f, err)
		}

		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("executing migration %q: %w", f, err)
		}

		fmt.Printf("applied migration: %s\n", filepath.Base(f))
	}

	return nil
}
