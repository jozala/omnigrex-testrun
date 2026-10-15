# Hello World in Go

A simple Go console app that greets you by name and prints a randomly selected quote of the day.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.21 or later
- PostgreSQL (only required for greeting tracking with `--track`)
- [Docker Compose](https://docs.docker.com/compose/) (only required for integration tests)

## How to run

```sh
go run .
```

Enter your name when prompted. If no name can be read, the app prints `Hello, World!` instead. In either case, it then prints one quote and its author, selected from the collection each time the program starts. For example:

```
Enter your name: Ada
Hello, Ada!
Quote of the day:
“The best way out is always through.”
— Robert Frost
```

### Non-interactive greeting

Pass `--name` to skip the prompt:

```sh
go run . --name "Ada Lovelace"
go run . --name="Ada Lovelace"
```

This prints `Hello, Ada Lovelace!` followed by one quote, without prompting. An explicitly empty or whitespace-only `--name` value is rejected with a clear error and a non-zero exit code.

### Greeting tracking with PostgreSQL

Pass `--track` to persist how many times each user has been greeted:

```sh
export DATABASE_URL="postgres://greet:greet@localhost:5432/greetings_test?sslmode=disable"
go run . --track
```

The connection string is read from the `DATABASE_URL` environment variable. A database connection is only established when `--track` is enabled; running without `--track` preserves the existing behavior and requires no database.

Names are trimmed and treated as case-sensitive identifiers. If no name is provided and the `Hello, World!` fallback is used, no greeting is recorded.

First execution:

```
Enter your name: Alice
Hello, Alice!
You've been greeted 1 time.

Quote of the day:
“The best way out is always through.”
— Robert Frost
```

Second execution with the same name:

```
Enter your name: Alice
Hello, Alice!
You've been greeted 2 times.

Quote of the day:
“Well done is better than well said.”
— Benjamin Franklin
```

Tracking also works non-interactively:

```sh
go run . --name Alice --track
```

When tracking is requested but `DATABASE_URL` is missing or PostgreSQL cannot be reached, the app prints a clear error to stderr and exits with a non-zero status. Database connections are closed after each execution.

## How to build

```sh
go build -o helloworld .
./helloworld
```

## How to test

Run unit tests (no Docker or PostgreSQL required):

```sh
go test ./...
```

Run the full integration suite against a real PostgreSQL container:

```sh
./scripts/test-integration.sh
```

The script starts PostgreSQL with `compose.integration.yml`, waits until it is healthy, sets `DATABASE_URL`, runs `go test -tags integration ./...`, and then stops and removes containers and volumes even when tests fail. Test failures propagate through the script exit code.

To run integration steps manually:

```sh
docker compose -f compose.integration.yml up -d --wait
export DATABASE_URL="postgres://greet:greet@localhost:5432/greetings_test?sslmode=disable"
go test -tags integration ./...
docker compose -f compose.integration.yml down -v
```

Check for vet issues:

```sh
go vet ./...
```
