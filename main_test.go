package main

import (
	"bytes"
	"strings"
	"testing"
)

func runApp(t *testing.T, stdin string, args ...string) (exitCode int, stdout, stderr string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	exitCode = run(strings.NewReader(stdin), &outBuf, &errBuf, args)
	return exitCode, outBuf.String(), errBuf.String()
}

func countQuotes(stdout string) int {
	return strings.Count(stdout, "Quote of the day:")
}

func stdoutContainsQuote(stdout string) bool {
	for _, q := range quotes {
		if strings.Contains(stdout, q.text) && strings.Contains(stdout, q.author) {
			return true
		}
	}
	return false
}

func TestNonInteractiveSpaceSeparatedValue(t *testing.T) {
	exit, stdout, _ := runApp(t, "", "--name", "Ada Lovelace")
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "Hello, Ada Lovelace!\n") {
		t.Errorf("expected greeting for Ada Lovelace, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "Enter your name:") {
		t.Errorf("expected no interactive prompt when --name is supplied, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
	if !stdoutContainsQuote(stdout) {
		t.Errorf("expected one known quote, got:\n%s", stdout)
	}
}

func TestNonInteractiveEqualsForm(t *testing.T) {
	exit, stdout, _ := runApp(t, "", "--name=Ada Lovelace")
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "Hello, Ada Lovelace!\n") {
		t.Errorf("expected greeting for Ada Lovelace, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
}

func TestInteractiveName(t *testing.T) {
	exit, stdout, _ := runApp(t, "Ada\n")
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "Enter your name:") {
		t.Errorf("expected interactive prompt, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Hello, Ada!\n") {
		t.Errorf("expected greeting for Ada, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
	if !stdoutContainsQuote(stdout) {
		t.Errorf("expected one known quote, got:\n%s", stdout)
	}
	// Greeting must come before the quote.
	greetingIdx := strings.Index(stdout, "Hello, Ada!")
	quoteIdx := strings.Index(stdout, "Quote of the day:")
	if greetingIdx < 0 || quoteIdx < 0 || greetingIdx > quoteIdx {
		t.Errorf("expected greeting before quote, got:\n%s", stdout)
	}
}

func TestInteractiveMultiTokenFallsBackToWorld(t *testing.T) {
	for _, input := range []string{"Ada Lovelace\n", "Ada\tLovelace\n"} {
		exit, stdout, _ := runApp(t, input)
		if exit != 0 {
			t.Fatalf("input %q: expected exit 0, got %d", input, exit)
		}
		if !strings.Contains(stdout, "Hello, World!\n") {
			t.Errorf("input %q: expected Hello, World! fallback preserving Scanln semantics, got:\n%s", input, stdout)
		}
		if countQuotes(stdout) != 1 {
			t.Errorf("input %q: expected exactly one quote, got:\n%s", input, stdout)
		}
	}
}

func TestFallbackOnEOF(t *testing.T) {
	exit, stdout, _ := runApp(t, "")
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "Hello, World!\n") {
		t.Errorf("expected Hello, World! fallback, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
	if !stdoutContainsQuote(stdout) {
		t.Errorf("expected one known quote, got:\n%s", stdout)
	}
}

func TestFallbackOnEmptyLine(t *testing.T) {
	exit, stdout, _ := runApp(t, "\n")
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "Hello, World!\n") {
		t.Errorf("expected Hello, World! fallback, got:\n%s", stdout)
	}
	if countQuotes(stdout) != 1 {
		t.Errorf("expected exactly one quote, got:\n%s", stdout)
	}
}

func TestEmptyNameRejected(t *testing.T) {
	for _, args := range [][]string{{"--name", ""}, {"--name="}} {
		exit, stdout, stderr := runApp(t, "", args...)
		if exit == 0 {
			t.Errorf("args %q: expected non-zero exit, got 0", args)
		}
		if !strings.Contains(stderr, "--name must not be empty") {
			t.Errorf("args %q: expected clear error on stderr, got stdout:\n%s\nstderr:\n%s", args, stdout, stderr)
		}
		if strings.Contains(stdout, "Hello,") {
			t.Errorf("args %q: expected no greeting on success path, got:\n%s", args, stdout)
		}
	}
}

func TestWhitespaceOnlyNameRejected(t *testing.T) {
	for _, args := range [][]string{{"--name", "   "}, {"--name=   "}} {
		exit, _, stderr := runApp(t, "", args...)
		if exit == 0 {
			t.Errorf("args %q: expected non-zero exit, got 0", args)
		}
		if !strings.Contains(stderr, "--name must not be empty") {
			t.Errorf("args %q: expected clear error on stderr, got:\n%s", args, stderr)
		}
	}
}
