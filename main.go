package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

type quote struct {
	text   string
	author string
}

var quotes = [...]quote{
	{text: "The best way out is always through.", author: "Robert Frost"},
	{text: "Well done is better than well said.", author: "Benjamin Franklin"},
	{text: "The only way to do great work is to love what you do.", author: "Steve Jobs"},
	{text: "If there is no struggle, there is no progress.", author: "Frederick Douglass"},
	{text: "Life shrinks or expands in proportion to one's courage.", author: "Anaïs Nin"},
	{text: "Not everything that is faced can be changed, but nothing can be changed until it is faced.", author: "James Baldwin"},
	{text: "To live is the rarest thing in the world. Most people exist, that is all.", author: "Oscar Wilde"},
	{text: "There is no charm equal to tenderness of heart.", author: "Jane Austen"},
	{text: "One can never consent to creep when one feels an impulse to soar.", author: "Helen Keller"},
	{text: "It is not enough to be busy; so are the ants. The question is: What are we busy about?", author: "Henry David Thoreau"},
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

// recordGreeting persists one greeting for name and returns the updated count.
// It opens its own connection so counts persist across separate executions
// and concurrent runs remain atomic via a PostgreSQL UPSERT.
func recordGreeting(name string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := connectDB(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close(ctx)
	if err := ensureSchema(ctx, conn); err != nil {
		return 0, err
	}
	return incrementGreetingCount(ctx, conn, name)
}

// run executes the greeting program and returns the process exit code.
// It is separated from main so it can be tested without spawning a process.
func run(in io.Reader, stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("helloworld", flag.ContinueOnError)
	fs.SetOutput(stderr)
	nameFlag := fs.String("name", "", "name to greet without prompting")
	trackFlag := fs.Bool("track", false, "track greeting counts in PostgreSQL")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	nameSupplied := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "name" {
			nameSupplied = true
		}
	})

	var name string
	fallback := false

	if nameSupplied {
		trimmed := strings.TrimSpace(*nameFlag)
		if trimmed == "" {
			fmt.Fprintln(stderr, "error: --name must not be empty or whitespace-only")
			return 1
		}
		name = trimmed
	} else {
		fmt.Fprint(stdout, "Enter your name: ")
		var interactiveName string
		if _, err := fmt.Fscanln(in, &interactiveName); err != nil {
			fallback = true
		} else {
			name = interactiveName
		}
	}

	if fallback {
		fmt.Fprintln(stdout, "Hello, World!")
	} else {
		fmt.Fprintf(stdout, "Hello, %s!\n", name)
	}

	if *trackFlag && !fallback {
		count, err := recordGreeting(name)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		suffix := "s"
		if count == 1 {
			suffix = ""
		}
		fmt.Fprintf(stdout, "You've been greeted %d time%s.\n", count, suffix)
		fmt.Fprintln(stdout)
	}

	selectedQuote := quotes[rand.Intn(len(quotes))]
	fmt.Fprintln(stdout, "Quote of the day:")
	fmt.Fprintf(stdout, "“%s”\n", selectedQuote.text)
	fmt.Fprintf(stdout, "— %s\n", selectedQuote.author)
	return 0
}
