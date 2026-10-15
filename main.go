package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
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

// run executes the greeting program and returns the process exit code.
// It is separated from main so it can be tested without spawning a process.
func run(in io.Reader, stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("helloworld", flag.ContinueOnError)
	fs.SetOutput(stderr)
	nameFlag := fs.String("name", "", "name to greet without prompting")
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
		reader := bufio.NewReader(in)
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimSpace(line)
		if err != nil && len(line) == 0 {
			fallback = true
		} else if trimmed == "" {
			fallback = true
		} else {
			name = trimmed
		}
	}

	if fallback {
		fmt.Fprintln(stdout, "Hello, World!")
	} else {
		fmt.Fprintf(stdout, "Hello, %s!\n", name)
	}

	selectedQuote := quotes[rand.Intn(len(quotes))]
	fmt.Fprintln(stdout, "Quote of the day:")
	fmt.Fprintf(stdout, "“%s”\n", selectedQuote.text)
	fmt.Fprintf(stdout, "— %s\n", selectedQuote.author)
	return 0
}
