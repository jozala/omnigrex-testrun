# Hello World in Go

A simple Go console app that greets you by name and prints a randomly selected quote of the day.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.25 or later
- [Docker](https://docs.docker.com/get-docker/) with Docker Compose (only required for PostgreSQL-backed tracking and integration tests)

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

### Tracked greetings with PostgreSQL

Pass `--track` to persist a per-user greeting counter in PostgreSQL:

```sh
export DATABASE_URL="postgres://greetings_test:greetings_test@localhost:5433/greetings_test?sslmode=disable"
go run . --track
go run . --track --name "Alice"
```

PostgreSQL prerequisites for tracking:

- A reachable PostgreSQL instance.
- The connection string in the `DATABASE_URL` environment variable.
- No manual schema setup: the app creates the `greeting_counts(name TEXT PRIMARY KEY, count BIGINT)` table automatically when `--track` is used.

Example with persistent counts:

```
Enter your name: Alice
Hello, Alice!
You've been greeted 1 time.

Quote of the day:
“The best way out is always through.”
— Robert Frost
```

Running the same command again for `Alice` prints `You've been greeted 2 times.` Counts are independent per user (names are case-sensitive, surrounding whitespace is trimmed), survive separate executions, and are updated atomically. If no name is provided and the `Hello, World!` fallback is used, no greeting is recorded. Running without `--track` never touches the database.

When `--track` is set but `DATABASE_URL` is missing or PostgreSQL cannot be reached, the app prints an error to stderr and exits with a non-zero status.

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

Run checks:

```sh
go vet ./...
```

### Integration tests with Docker

Integration tests use a real PostgreSQL container (no mocks or SQLite). Start it with:

```sh
docker compose -f compose.integration.yml up -d --wait
export DATABASE_URL="postgres://greetings_test:greetings_test@localhost:5433/greetings_test?sslmode=disable"
go test -tags integration -count=1 ./...
docker compose -f compose.integration.yml down -v
```

Or run the whole lifecycle (start, wait, test, cleanup) with:

```sh
./scripts/test-integration.sh
```

The script propagates test failures through its exit code and always stops containers and removes volumes, even when tests fail.
