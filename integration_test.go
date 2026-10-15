//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

var uniqueCounter atomic.Int64

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		t.Fatal("DATABASE_URL must be set to run integration tests (see scripts/test-integration.sh)")
	}
	return url
}

func integrationConnect(t *testing.T, ctx context.Context) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, integrationDatabaseURL(t))
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})
	return conn
}

func integrationContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func uniqueUser(t *testing.T, prefix string) string {
	t.Helper()
	n := uniqueCounter.Add(1)
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	return fmt.Sprintf("%s_%s_%d_%d", prefix, safe, time.Now().UnixNano(), n)
}

func storedCount(t *testing.T, ctx context.Context, conn *pgx.Conn, name string) (int64, bool) {
	t.Helper()
	var count int64
	err := conn.QueryRow(ctx, `SELECT count FROM greeting_counts WHERE name = $1`, name).Scan(&count)
	if err != nil {
		return 0, false
	}
	return count, true
}

func mustStoredCount(t *testing.T, ctx context.Context, conn *pgx.Conn, name string) int64 {
	t.Helper()
	count, ok := storedCount(t, ctx, conn, name)
	if !ok {
		t.Fatalf("expected a stored count for %q, found none", name)
	}
	return count
}

func TestIntegrationFirstGreeting(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	name := uniqueUser(t, "first")
	count, err := incrementGreetingCount(ctx, conn, name)
	if err != nil {
		t.Fatalf("incrementGreetingCount failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected first count to be 1, got %d", count)
	}
	if got := mustStoredCount(t, ctx, conn, name); got != 1 {
		t.Fatalf("expected stored count 1, got %d", got)
	}
}

func TestIntegrationRepeatedGreeting(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	name := uniqueUser(t, "repeat")
	first, err := incrementGreetingCount(ctx, conn, name)
	if err != nil {
		t.Fatalf("first increment failed: %v", err)
	}
	if first != 1 {
		t.Fatalf("expected first count 1, got %d", first)
	}
	second, err := incrementGreetingCount(ctx, conn, name)
	if err != nil {
		t.Fatalf("second increment failed: %v", err)
	}
	if second != 2 {
		t.Fatalf("expected second count 2, got %d", second)
	}
	if got := mustStoredCount(t, ctx, conn, name); got != 2 {
		t.Fatalf("expected stored count 2, got %d", got)
	}
}

func TestIntegrationIndependentUsers(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	alice := uniqueUser(t, "alice")
	bob := uniqueUser(t, "bob")

	if c, err := incrementGreetingCount(ctx, conn, alice); err != nil || c != 1 {
		t.Fatalf("expected Alice first count 1, got %d, err %v", c, err)
	}
	if c, err := incrementGreetingCount(ctx, conn, bob); err != nil || c != 1 {
		t.Fatalf("expected Bob first count 1, got %d, err %v", c, err)
	}
	if c, err := incrementGreetingCount(ctx, conn, alice); err != nil || c != 2 {
		t.Fatalf("expected Alice second count 2, got %d, err %v", c, err)
	}

	if got := mustStoredCount(t, ctx, conn, alice); got != 2 {
		t.Fatalf("expected stored Alice count 2, got %d", got)
	}
	if got := mustStoredCount(t, ctx, conn, bob); got != 1 {
		t.Fatalf("expected stored Bob count 1, got %d, err", got)
	}
}

func TestIntegrationPersistenceAcrossConnectionsAndExecutions(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()

	name := uniqueUser(t, "persist")

	// Direct persistence: close and reopen the connection.
	conn1 := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn1); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}
	if c, err := incrementGreetingCount(ctx, conn1, name); err != nil || c != 1 {
		t.Fatalf("expected count 1, got %d, err %v", c, err)
	}
	_ = conn1.Close(ctx)

	conn2 := integrationConnect(t, ctx)
	if got := mustStoredCount(t, ctx, conn2, name); got != 1 {
		t.Fatalf("expected persisted count 1 after reconnect, got %d", got)
	}
	if c, err := incrementGreetingCount(ctx, conn2, name); err != nil || c != 2 {
		t.Fatalf("expected count 2 after reconnect, got %d, err %v", c, err)
	}
	_ = conn2.Close(ctx)

	// Persistence across separate application executions via run().
	exit1, stdout1, _ := runApp(t, "", "--name", name, "--track")
	if exit1 != 0 {
		t.Fatalf("first run with --track failed: exit %d, stdout:\n%s", exit1, stdout1)
	}
	if !strings.Contains(stdout1, "You've been greeted 3 times.") {
		t.Errorf("expected third greeting to report 3 times, got:\n%s", stdout1)
	}
	if countQuotes(stdout1) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout1)
	}

	exit2, stdout2, _ := runApp(t, "", "--name", name, "--track")
	if exit2 != 0 {
		t.Fatalf("second run with --track failed: exit %d, stdout:\n%s", exit2, stdout2)
	}
	if !strings.Contains(stdout2, "You've been greeted 4 times.") {
		t.Errorf("expected fourth greeting to report 4 times, got:\n%s", stdout2)
	}

	conn3 := integrationConnect(t, ctx)
	if got := mustStoredCount(t, ctx, conn3, name); got != 4 {
		t.Fatalf("expected persisted count 4 after two app executions, got %d", got)
	}
}

func TestIntegrationConcurrentUpdates(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	setup := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, setup); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	name := uniqueUser(t, "concurrent")
	const workers = 20

	var wg sync.WaitGroup
	errs := make([]error, workers)
	counts := make([]int64, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer ccancel()
			conn, err := pgx.Connect(cctx, integrationDatabaseURL(t))
			if err != nil {
				errs[idx] = err
				return
			}
			defer conn.Close(cctx)
			c, err := incrementGreetingCount(cctx, conn, name)
			if err != nil {
				errs[idx] = err
				return
			}
			counts[idx] = c
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d failed: %v", i, err)
		}
	}

	seen := make(map[int64]bool, workers)
	for _, c := range counts {
		if c < 1 || c > workers {
			t.Fatalf("unexpected returned count %d, want 1..%d", c, workers)
		}
		if seen[c] {
			t.Fatalf("duplicate returned count %d, increments were lost or duplicated", c)
		}
		seen[c] = true
	}

	final := mustStoredCount(t, ctx, setup, name)
	if final != workers {
		t.Fatalf("expected final count %d after %d concurrent greetings, got %d", workers, workers, final)
	}
}

func TestIntegrationSchemaInitialization(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)

	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS greeting_counts`); err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}

	// Confirm the table is missing before initialization.
	var missing bool
	err := conn.QueryRow(ctx, `SELECT count FROM greeting_counts LIMIT 1`).Scan(new(int64))
	if err == nil {
		t.Fatal("expected greeting_counts to be missing after DROP, but query succeeded")
	}
	missing = true
	if !missing {
		t.Fatal("expected table to be missing")
	}

	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	name := uniqueUser(t, "schema")
	if c, err := incrementGreetingCount(ctx, conn, name); err != nil || c != 1 {
		t.Fatalf("expected increment to work after schema init, got %d, err %v", c, err)
	}
	if got := mustStoredCount(t, ctx, conn, name); got != 1 {
		t.Fatalf("expected stored count 1 after schema init, got %d", got)
	}
}

func TestIntegrationDatabaseFailure(t *testing.T) {
	oldURL, hadOld := os.LookupEnv("DATABASE_URL")
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv("DATABASE_URL", oldURL)
		} else {
			_ = os.Unsetenv("DATABASE_URL")
		}
	})

	// Unroutable port with a short connect timeout so the failure is fast.
	_ = os.Setenv("DATABASE_URL", "postgres://greet:greet@127.0.0.1:1/greetings_test?sslmode=disable&connect_timeout=2")

	name := uniqueUser(t, "unreachable")
	exit, stdout, stderr := runApp(t, "", "--name", name, "--track")
	if exit == 0 {
		t.Fatalf("expected non-zero exit when PostgreSQL is unreachable, got 0 (stdout:\n%s)", stdout)
	}
	combined := strings.ToLower(stdout + "\n" + stderr)
	if !strings.Contains(combined, "database") && !strings.Contains(combined, "connect") && !strings.Contains(combined, "database_url") {
		t.Fatalf("expected a clear database error on stderr, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if strings.Contains(stdout, "Quote of the day:") {
		t.Fatalf("expected no quote when tracking fails, got stdout:\n%s", stdout)
	}

	// Missing configuration must also fail clearly.
	_ = os.Unsetenv("DATABASE_URL")
	exit, _, stderr = runApp(t, "", "--name", name, "--track")
	if exit == 0 {
		t.Fatal("expected non-zero exit when DATABASE_URL is missing")
	}
	if !strings.Contains(strings.ToLower(stderr), "database_url") {
		t.Fatalf("expected DATABASE_URL error on stderr, got:\n%s", stderr)
	}
}

func TestIntegrationNamesTrimmedAndCaseSensitive(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	base := uniqueUser(t, "names")
	padded := "  " + base + "  "
	c1, err := incrementGreetingCount(ctx, conn, padded)
	if err != nil {
		t.Fatalf("increment with padding failed: %v", err)
	}
	if c1 != 1 {
		t.Fatalf("expected 1 for padded name, got %d", c1)
	}
	c2, err := incrementGreetingCount(ctx, conn, base)
	if err != nil {
		t.Fatalf("increment with trimmed name failed: %v", err)
	}
	if c2 != 2 {
		t.Fatalf("expected padded and trimmed names to share a counter (got %d, want 2)", c2)
	}

	lower := strings.ToLower(base) + "_case"
	upper := strings.ToUpper(lower)
	if lower == upper {
		t.Skip("cannot test case sensitivity with case-invariant name")
	}
	if c, err := incrementGreetingCount(ctx, conn, lower); err != nil || c != 1 {
		t.Fatalf("expected lower-case first count 1, got %d, err %v", c, err)
	}
	if c, err := incrementGreetingCount(ctx, conn, upper); err != nil || c != 1 {
		t.Fatalf("expected upper-case first count 1 (independent counter), got %d, err %v", c, err)
	}
}

func TestIntegrationFallbackDoesNotRecord(t *testing.T) {
	ctx, cancel := integrationContext(t)
	defer cancel()
	conn := integrationConnect(t, ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}

	exit, stdout, _ := runApp(t, "", "--track")
	if exit != 0 {
		t.Fatalf("expected fallback with --track to succeed, got exit %d:\n%s", exit, stdout)
	}
	if !strings.Contains(stdout, "Hello, World!") {
		t.Fatalf("expected Hello, World! fallback, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "You've been greeted") {
		t.Fatalf("expected no greeting count for fallback, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Fatalf("expected exactly one quote for fallback, got:\n%s", stdout)
	}
	var worldCount int64
	err := conn.QueryRow(ctx, `SELECT count FROM greeting_counts WHERE name = 'World'`).Scan(&worldCount)
	if err == nil {
		t.Fatalf("fallback must not record a greeting, but found World count %d", worldCount)
	}
}
