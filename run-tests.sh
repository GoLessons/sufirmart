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

docker compose up -d db
until docker compose exec -T db pg_isready -U sufirmart >/dev/null 2>&1; do sleep 1; done

if $CLEANUP; then
  psql_db "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE); CREATE DATABASE $DB_NAME"
else
  psql_db "DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME') THEN
      EXECUTE 'CREATE DATABASE $DB_NAME';
    END IF;
  END \$\$;"
fi

docker compose exec -T db migrate -path /migrations -database "postgresql://sufirmart:sufirmart@localhost:5432/$DB_NAME?sslmode=disable" up

if [ -z "$TEST_DATABASE_URI" ]; then
  TEST_DATABASE_URI="postgresql://sufirmart:sufirmart@localhost:15432/$DB_NAME?sslmode=disable"
fi

go test ./...
