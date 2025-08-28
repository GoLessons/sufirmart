#!/bin/bash
set -e
cd "$(dirname "$0")"

DB_NAME="sufirmart_test"
CLEANUP=false

# разбор аргументов
while [[ $# -gt 0 ]]; do
  case "$1" in
    -db) DB_NAME="$2"; shift 2 ;;
    -c|--cleanup) CLEANUP=true; shift ;;
    *) echo "Usage: $0 [-db <dbname>] [-c|--cleanup]"; exit 1 ;;
  esac
done

# обёртка для psql
psql_db() {
  docker compose exec -T db psql -U sufirmart -d sufirmart -tAc "$1"
}

docker compose up -d db

# ждём готовности базы
until docker compose exec -T db pg_isready -U sufirmart >/dev/null 2>&1; do sleep 1; done

if $CLEANUP; then
  psql_db "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE)"
  psql_db "CREATE DATABASE $DB_NAME"
else
  # создаём базу только если её нет
  if ! psql_db "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1; then
    psql_db "CREATE DATABASE $DB_NAME"
  fi
fi

# применяем миграции
docker compose exec -T db migrate -path /migrations \
  -database "postgresql://sufirmart:sufirmart@localhost:5432/$DB_NAME?sslmode=disable" up

# устанавливаем TEST_DATABASE_URI только если она не задана
if [ -z "$TEST_DATABASE_URI" ]; then
  TEST_DATABASE_URI="postgresql://sufirmart:sufirmart@localhost:15432/$DB_NAME?sslmode=disable"
fi

echo "Using TEST_DATABASE_URI=$TEST_DATABASE_URI"

# запускаем тесты
go test ./...
