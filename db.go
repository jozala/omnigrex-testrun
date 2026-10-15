package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// connectDB opens a PostgreSQL connection using DATABASE_URL.
// It returns a clear error when the configuration is missing or the
// database cannot be reached. The caller owns the connection and must close it.
func connectDB(ctx context.Context) (*pgx.Conn, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set: set DATABASE_URL to a PostgreSQL connection string to use --track")
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return conn, nil
}

// ensureSchema creates the greeting_counts table when it does not exist.
func ensureSchema(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS greeting_counts (
    name TEXT PRIMARY KEY,
    count BIGINT NOT NULL DEFAULT 0
)`)
	if err != nil {
		return fmt.Errorf("failed to initialize greeting_counts table: %w", err)
	}
	return nil
}

// incrementGreetingCount atomically increments the greeting counter for name
// and returns the updated value. Names are trimmed and treated as
// case-sensitive identifiers.
func incrementGreetingCount(ctx context.Context, conn *pgx.Conn, name string) (int64, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, fmt.Errorf("name must not be empty")
	}
	var count int64
	err := conn.QueryRow(ctx, `INSERT INTO greeting_counts (name, count)
VALUES ($1, 1)
ON CONFLICT (name)
DO UPDATE SET count = greeting_counts.count + 1
RETURNING count`, trimmed).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to increment greeting count: %w", err)
	}
	return count, nil
}
