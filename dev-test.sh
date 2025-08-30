#!/bin/bash
set -e

cd "$(dirname "$0")"

DB_NAME="sufirmart_test"
CLEANUP=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    -db) DB_NAME="$2"; shift 2 ;;
    -c|--cleanup) CLEANUP=true; shift ;;
    *) echo "Usage: $0 [-db <dbname>] [-c|--cleanup]"; exit 1 ;;
  esac
done

psql_db() {
  docker compose exec -T db psql -U sufirmart -d sufirmart -tAc "$1"
}

docker compose up -d db

until docker compose exec -T db pg_isready -U sufirmart >/dev/null 2>&1; do sleep 1; done

if $CLEANUP; then
  psql_db "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE)"
  psql_db "CREATE DATABASE $DB_NAME"
else
  if ! psql_db "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1; then
    psql_db "CREATE DATABASE $DB_NAME"
  fi
fi

docker compose exec -T db migrate -path /migrations \
  -database "postgresql://sufirmart:sufirmart@localhost:5432/$DB_NAME?sslmode=disable&search_path=public" up

if [ -z "$TEST_DATABASE_URI" ]; then
  TEST_DATABASE_URI="postgresql://sufirmart:sufirmart@localhost:15432/$DB_NAME?sslmode=disable"
fi

if [ -z "$MIGRATIONS_DIR" ]; then
  MIGRATIONS_DIR="$(pwd)/migrations"
fi

echo "Using TEST_DATABASE_URI=$TEST_DATABASE_URI"
echo "Using MIGRATIONS_DIR=$MIGRATIONS_DIR"

TEST_DATABASE_URI="$TEST_DATABASE_URI" MIGRATIONS_DIR="$MIGRATIONS_DIR" go test ./...
