#!/usr/bin/env bash
set -euo pipefail

# Runs PostgreSQL-backed integration tests against a real database container.
# Usage: ./scripts/test-integration.sh
#
# Lifecycle:
#   1. Starts PostgreSQL via Docker Compose.
#   2. Waits until PostgreSQL is healthy.
#   3. Sets DATABASE_URL for the Go integration tests.
#   4. Runs the Go integration tests.
#   5. Stops PostgreSQL and removes test resources, even on failure.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/compose.integration.yml"

export DATABASE_URL="${DATABASE_URL:-postgres://greetings_test:greetings_test@localhost:5433/greetings_test?sslmode=disable}"

cleanup() {
  echo "Stopping integration PostgreSQL..." >&2
  docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "Starting integration PostgreSQL..." >&2
docker compose -f "${COMPOSE_FILE}" up -d --wait

echo "Running integration tests..." >&2
cd "${REPO_ROOT}"
go test -tags integration -count=1 ./...
