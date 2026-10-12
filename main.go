package main

import (
	"fmt"
	"math/rand"
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
	fmt.Print("Enter your name: ")
	var name string
	_, err := fmt.Scanln(&name)
	if err != nil {
		fmt.Println("Hello, World!")
	} else {
		fmt.Printf("Hello, %s!\n", name)
	}

	selectedQuote := quotes[rand.Intn(len(quotes))]
	fmt.Println("Quote of the day:")
	fmt.Printf("“%s”\n", selectedQuote.text)
	fmt.Printf("— %s\n", selectedQuote.author)
}
