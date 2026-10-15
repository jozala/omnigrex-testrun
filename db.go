package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ensureGreetingTable creates the greeting_counts table when it does not exist.
func ensureGreetingTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS greeting_counts (
    name TEXT PRIMARY KEY,
    count BIGINT NOT NULL DEFAULT 0
)`)
	return err
}

// incrementGreetingCount atomically increments the greeting counter for name
// and returns the updated value. Names are trimmed and treated case-sensitively.
func incrementGreetingCount(ctx context.Context, conn *pgx.Conn, name string) (int64, error) {
	key := strings.TrimSpace(name)
	var count int64
	err := conn.QueryRow(ctx, `INSERT INTO greeting_counts (name, count)
VALUES ($1, 1)
ON CONFLICT (name)
DO UPDATE SET count = greeting_counts.count + 1
RETURNING count`, key).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
