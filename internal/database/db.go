package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver registered as a side effect
)

// Connect opens a connection pool to PostgreSQL and verifies connectivity.
// It retries up to maxRetries times with a short delay to handle the case
// where the database container is still starting up.
func Connect(databaseURL string) (*sqlx.DB, error) {
	const maxRetries = 10
	const retryDelay = 2 * time.Second

	var db *sqlx.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", databaseURL)
		if err == nil {
			break
		}
		fmt.Printf("database not ready, retrying in %s... (%d/%d)\n", retryDelay, i+1, maxRetries)
		time.Sleep(retryDelay)
	}

	if err != nil {
		return nil, fmt.Errorf("could not connect to database after %d attempts: %w", maxRetries, err)
	}

	// Tune the connection pool for a small background-worker workload.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
