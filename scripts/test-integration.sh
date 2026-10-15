#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="compose.integration.yml"
export DATABASE_URL="${DATABASE_URL:-postgres://greet:greet@localhost:5432/greetings_test?sslmode=disable}"

cleanup() {
  docker compose -f "$COMPOSE_FILE" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -f "$COMPOSE_FILE" up -d --wait

go test -tags integration ./...
