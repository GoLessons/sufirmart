# Выпускной проект

Накопительная система лояльности

## Тестирование

### Приёмочные тесты

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в репозиторий.

Запуск выполняется командой:
```bash
gophermarttest \
    -test.v -test.run=^TestGophermart$ \
    -gophermart-binary-path=cmd/gophermart/gophermart \
    -gophermart-host=localhost \
    -gophermart-port=8080 \
    -gophermart-database-uri="postgresql://postgres:postgres@postgres/praktikum?sslmode=disable" \
    -accrual-binary-path=cmd/accrual/accrual_linux_amd64 \
    -accrual-host=localhost \
    -accrual-port=$(random unused-port) \
    -accrual-database-uri="postgresql://postgres:postgres@postgres/praktikum?sslmode=disable"
```

### Тестирование эндпоинтов

Для тестирования эндпоинтов на соотвестве спецификации используется [httptest-openapi](https://gitlab.com/jamietanna/httptest-openapi/)
