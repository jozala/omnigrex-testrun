# Hello World in Go

A simple Go console app that greets you by name and prints a randomly selected quote of the day.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.21 or later

## How to run

```sh
go run main.go
```

Enter your name when prompted. If no name can be read, the app prints `Hello, World!` instead. In either case, it then prints one quote and its author, selected from the collection each time the program starts. For example:

```
Enter your name: Ada
Hello, Ada!
Quote of the day:
“The best way out is always through.”
— Robert Frost
```

## How to build

```sh
go build -o helloworld .
./helloworld
```
