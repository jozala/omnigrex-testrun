package main

import "fmt"

func main() {
	fmt.Print("Enter your name: ")
	var name string
	_, err := fmt.Scanln(&name)
	if err != nil {
		fmt.Println("Hello, World!")
		return
	}
	fmt.Printf("Hello, %s!\n", name)
}
