//go:build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

var nameCounter atomic.Int64

func requireDatabaseURL(t *testing.T) string {
	t.Helper()
	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		t.Fatal("DATABASE_URL must be set to run integration tests (see scripts/test-integration.sh)")
	}
	return url
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func openTestConn(t *testing.T, ctx context.Context, databaseURL string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL at %s: %v", redactURL(databaseURL), err)
	}
	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})
	return conn
}

func redactURL(url string) string {
	// Avoid leaking credentials in failure messages.
	if i := strings.LastIndex(url, "@"); i >= 0 {
		return "postgres://***@" + url[i+1:]
	}
	return "postgres://***"
}

func uniqueName(t *testing.T, prefix string) string {
	t.Helper()
	n := nameCounter.Add(1)
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, t.Name())
	return fmt.Sprintf("%s_%s_%d_%d", prefix, safe, time.Now().UnixNano(), n)
}

func ensureTable(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	if err := ensureGreetingTable(ctx, conn); err != nil {
		t.Fatalf("ensureGreetingTable failed: %v", err)
	}
}

func storedCount(t *testing.T, ctx context.Context, conn *pgx.Conn, name string) (int64, bool) {
	t.Helper()
	var count int64
	err := conn.QueryRow(ctx, `SELECT count FROM greeting_counts WHERE name = $1`, strings.TrimSpace(name)).Scan(&count)
	if err != nil {
		return 0, false
	}
	return count, true
}

func TestIntegrationFirstGreeting(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	ensureTable(t, ctx, conn)

	name := uniqueName(t, "first")
	count, err := incrementGreetingCount(ctx, conn, name)
	if err != nil {
		t.Fatalf("incrementGreetingCount failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected first greeting count 1, got %d", count)
	}
	if got, ok := storedCount(t, ctx, conn, name); !ok || got != 1 {
		t.Fatalf("expected stored count 1, got %d (found=%v)", got, ok)
	}
}

func TestIntegrationRepeatedGreeting(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	ensureTable(t, ctx, conn)

	name := uniqueName(t, "repeat")
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
	if got, ok := storedCount(t, ctx, conn, name); !ok || got != 2 {
		t.Fatalf("expected stored count 2, got %d (found=%v)", got, ok)
	}
}

func TestIntegrationIndependentUsers(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	ensureTable(t, ctx, conn)

	alice := uniqueName(t, "alice")
	bob := uniqueName(t, "bob")

	if _, err := incrementGreetingCount(ctx, conn, alice); err != nil {
		t.Fatalf("alice first increment failed: %v", err)
	}
	if _, err := incrementGreetingCount(ctx, conn, alice); err != nil {
		t.Fatalf("alice second increment failed: %v", err)
	}
	if _, err := incrementGreetingCount(ctx, conn, bob); err != nil {
		t.Fatalf("bob increment failed: %v", err)
	}

	aliceCount, ok := storedCount(t, ctx, conn, alice)
	if !ok || aliceCount != 2 {
		t.Fatalf("expected alice count 2, got %d (found=%v)", aliceCount, ok)
	}
	bobCount, ok := storedCount(t, ctx, conn, bob)
	if !ok || bobCount != 1 {
		t.Fatalf("expected bob count 1, got %d (found=%v)", bobCount, ok)
	}
}

func TestIntegrationPersistenceAcrossConnections(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	name := uniqueName(t, "persist")

	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	ensureTable(t, ctx, conn)
	if _, err := incrementGreetingCount(ctx, conn, name); err != nil {
		t.Fatalf("first increment failed: %v", err)
	}
	if _, err := incrementGreetingCount(ctx, conn, name); err != nil {
		t.Fatalf("second increment failed: %v", err)
	}
	// Close and reopen the connection.
	_ = conn.Close(context.Background())

	ctx2, cancel2 := testContext()
	defer cancel2()
	conn2 := openTestConn(t, ctx2, databaseURL)
	got, ok := storedCount(t, ctx2, conn2, name)
	if !ok || got != 2 {
		t.Fatalf("expected persisted count 2 after reconnect, got %d (found=%v)", got, ok)
	}
	third, err := incrementGreetingCount(ctx2, conn2, name)
	if err != nil {
		t.Fatalf("third increment after reconnect failed: %v", err)
	}
	if third != 3 {
		t.Fatalf("expected count 3 after reconnect, got %d", third)
	}
}

func TestIntegrationPersistenceAcrossAppExecutions(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	t.Setenv("DATABASE_URL", databaseURL)
	name := uniqueName(t, "appexec")

	var out1, err1 bytes.Buffer
	exit1 := run(strings.NewReader(""), &out1, &err1, []string{"--track", "--name", name})
	if exit1 != 0 {
		t.Fatalf("first app execution failed: exit=%d stderr=%s", exit1, err1.String())
	}
	if !strings.Contains(out1.String(), "You've been greeted 1 time.") {
		t.Errorf("expected first execution to report 1 time, got:\n%s", out1.String())
	}

	var out2, err2 bytes.Buffer
	exit2 := run(strings.NewReader(""), &out2, &err2, []string{"--track", "--name", name})
	if exit2 != 0 {
		t.Fatalf("second app execution failed: exit=%d stderr=%s", exit2, err2.String())
	}
	if !strings.Contains(out2.String(), "You've been greeted 2 times.") {
		t.Errorf("expected second execution to report 2 times, got:\n%s", out2.String())
	}

	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	got, ok := storedCount(t, ctx, conn, name)
	if !ok || got != 2 {
		t.Fatalf("expected stored count 2 after two app executions, got %d (found=%v)", got, ok)
	}
}

func TestIntegrationConcurrentUpdates(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	ctx, cancel := testContext()
	defer cancel()
	conn := openTestConn(t, ctx, databaseURL)
	ensureTable(t, ctx, conn)

	name := uniqueName(t, "concurrent")
	const workers = 20

	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cctx, ccancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer ccancel()
			cconn, err := pgx.Connect(cctx, databaseURL)
			if err != nil {
				errCh <- err
				return
			}
			defer cconn.Close(context.Background())
			if _, err := incrementGreetingCount(cctx, cconn, name); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent increment failed: %v", err)
	}

	got, ok := storedCount(t, ctx, conn, name)
	if !ok || got != workers {
		t.Fatalf("expected final count %d after %d concurrent greetings, got %d (found=%v)", workers, workers, got, ok)
	}
}

func replaceDatabaseName(t *testing.T, rawURL, dbName string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse DATABASE_URL: %v", err)
	}
	u.Path = "/" + dbName
	return u.String()
}

// provisionScratchDatabase creates a fresh, empty database on the same
// PostgreSQL server and returns its connection URL. The caller owns cleanup
// via the returned function. The schema test runs exclusively inside this
// database so it never drops greeting_counts in the shared test database,
// let alone in an external database.
func provisionScratchDatabase(t *testing.T, ctx context.Context, adminConn *pgx.Conn, baseURL string) (string, func()) {
	t.Helper()
	scratchName := fmt.Sprintf("stest_%d_%d", time.Now().UnixNano(), nameCounter.Add(1))
	if _, err := adminConn.Exec(ctx, `CREATE DATABASE "`+scratchName+`"`); err != nil {
		t.Fatalf("failed to create scratch database: %v", err)
	}
	scratchURL := replaceDatabaseName(t, baseURL, scratchName)
	cleanup := func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer ccancel()
		// FORCE terminates any lingering scratch connections so the drop succeeds.
		_, _ = adminConn.Exec(cctx, `DROP DATABASE IF EXISTS "`+scratchName+`" WITH (FORCE)`)
	}
	return scratchURL, cleanup
}

func TestIntegrationSchemaInitialization(t *testing.T) {
	databaseURL := requireDatabaseURL(t)
	ctx, cancel := testContext()
	defer cancel()
	adminConn := openTestConn(t, ctx, databaseURL)

	scratchURL, cleanupScratch := provisionScratchDatabase(t, ctx, adminConn, databaseURL)
	defer cleanupScratch()

	scratchConn, err := pgx.Connect(ctx, scratchURL)
	if err != nil {
		t.Fatalf("failed to connect to scratch database: %v", err)
	}
	defer scratchConn.Close(context.Background())

	// A freshly provisioned database must not contain the table yet.
	var exists bool
	err = scratchConn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'greeting_counts')`).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if exists {
		t.Fatal("expected greeting_counts table to be missing in a fresh database")
	}

	if err := ensureGreetingTable(ctx, scratchConn); err != nil {
		t.Fatalf("ensureGreetingTable failed: %v", err)
	}

	err = scratchConn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'greeting_counts')`).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to re-check table existence: %v", err)
	}
	if !exists {
		t.Fatal("expected greeting_counts table to exist after ensureGreetingTable")
	}

	// Verify the created table is usable.
	name := uniqueName(t, "schema")
	count, err := incrementGreetingCount(ctx, scratchConn, name)
	if err != nil {
		t.Fatalf("increment after schema init failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1 after schema init, got %d", count)
	}
}

func TestIntegrationDatabaseFailure(t *testing.T) {
	// Unreachable database: reserved TEST-NET-1 address plus closed port.
	t.Setenv("DATABASE_URL", "postgres://greetings_test:greetings_test@192.0.2.1:5432/greetings_test?connect_timeout=2&sslmode=disable")

	var stdout, stderr bytes.Buffer
	exit := run(strings.NewReader(""), &stdout, &stderr, []string{"--track", "--name", uniqueName(t, "unreachable")})
	if exit == 0 {
		t.Fatal("expected non-zero exit when PostgreSQL cannot be reached")
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Error("expected an error message on stderr when the database cannot be reached")
	} else if !strings.Contains(strings.ToLower(stderr.String()), "database") && !strings.Contains(strings.ToLower(stderr.String()), "connect") {
		t.Errorf("expected database-related error on stderr, got:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "Hello,") {
		t.Errorf("expected no greeting on database failure, got:\n%s", stdout.String())
	}
}

func TestIntegrationMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	var stdout, stderr bytes.Buffer
	exit := run(strings.NewReader(""), &stdout, &stderr, []string{"--track", "--name", uniqueName(t, "nodatabaseurl")})
	if exit == 0 {
		t.Fatal("expected non-zero exit when DATABASE_URL is missing")
	}
	if !strings.Contains(stderr.String(), "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL error on stderr, got:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "Hello,") {
		t.Errorf("expected no greeting when DATABASE_URL is missing, got:\n%s", stdout.String())
	}
}
