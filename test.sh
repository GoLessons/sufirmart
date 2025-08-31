#!/bin/bash
set -e

# Для упрощения прогона тестов во время разработки

DIR=${1:-cmd/gophermart}
BIN="$DIR/sufirmart"

(cd "$DIR" && go build -buildvcs=false -o sufirmart)

./tools/gophermarttest \
  -test.v -test.run=^TestGophermart$ \
  -gophermart-binary-path="$BIN" \
  -gophermart-host=localhost \
  -gophermart-port=8080 \
  -gophermart-database-uri="postgresql://sufirmart:sufirmart@localhost:15432/sufirmart?sslmode=disable" \
  -accrual-binary-path=cmd/accrual/accrual_linux_amd64 \
  -accrual-host=localhost \
  -accrual-port=8081 \
  -accrual-database-uri="postgresql://sufirmart:sufirmart@localhost:15432/sufirmart?sslmode=disable"
