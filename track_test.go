package main

import (
	"os"
	"strings"
	"testing"
)

func withEnv(t *testing.T, key, value string, present bool) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if present {
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(key, old)
			} else {
				_ = os.Unsetenv(key)
			}
		})
		_ = os.Setenv(key, value)
	} else {
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(key, old)
			}
		})
		_ = os.Unsetenv(key)
	}
}

func TestTrackRequiresDatabaseConfiguration(t *testing.T) {
	withEnv(t, "DATABASE_URL", "", false)
	exit, stdout, stderr := runApp(t, "", "--name", "Alice", "--track")
	if exit == 0 {
		t.Fatalf("expected non-zero exit when DATABASE_URL is missing, got 0 (stdout:\n%s)", stdout)
	}
	if !strings.Contains(stderr, "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL error on stderr, got:\n%s", stderr)
	}
	if strings.Contains(stdout, "Quote of the day:") {
		t.Errorf("expected no quote when tracking fails, got:\n%s", stdout)
	}
}

func TestTrackFallbackDoesNotRequireDatabase(t *testing.T) {
	withEnv(t, "DATABASE_URL", "", false)
	exit, stdout, _ := runApp(t, "", "--track")
	if exit != 0 {
		t.Fatalf("expected fallback with --track to succeed without DATABASE_URL, got exit %d:\n%s", exit, stdout)
	}
	if !strings.Contains(stdout, "Hello, World!") {
		t.Errorf("expected Hello, World! fallback, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "You've been greeted") {
		t.Errorf("expected no count for fallback, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
}

func TestWithoutTrackRequiresNoDatabase(t *testing.T) {
	withEnv(t, "DATABASE_URL", "", false)
	exit, stdout, _ := runApp(t, "", "--name", "Alice")
	if exit != 0 {
		t.Fatalf("expected exit 0 without --track, got %d", exit)
	}
	if !strings.Contains(stdout, "Hello, Alice!") {
		t.Errorf("expected greeting, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "You've been greeted") {
		t.Errorf("expected no count without --track, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
}
