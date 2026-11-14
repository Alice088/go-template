# go-template

Личный шаблон Go-бэкенда: HTTP API на chi, PostgreSQL через pgx, JWT-аутентификация,
конфигурация из окружения, структурированные логи, типизированные ошибки и swagger.

`go 1.26` · chi v5 · pgx v5 · zerolog · swaggo

## Возможности

- **Слоистая структура** `cmd` → `router` → `handler` → `service` → `repository` → `entity`.
- **Конфигурация** из переменных окружения с валидацией: приложение не стартует с
  плейсхолдерным JWT-секретом, кривым портом или неизвестным уровнем логов.
- **Типизированные ошибки** (`internal/apierr`): у каждой ошибки есть внутреннее
  сообщение для логов и безопасное публичное сообщение для клиента.
- **Единый JSON-формат ошибок** для всех ответов, включая 404/405 и паники.
- **Middleware**: request ID (в логах и в заголовке `X-Request-Id`), логирование,
  recovery, CORS, лимит тела запроса, таймаут запроса через `r.Context()`.
- **Health-check**: `/v1/health` (liveness) и `/v1/ready` (readiness с проверкой БД).
- **Graceful shutdown** по SIGINT/SIGTERM с настраиваемым таймаутом.
- **Swagger** на `/swagger/index.html`, спека генерируется из аннотаций.
- **Тесты** на конфиг, ошибки, JWT, middleware, хендлеры и роутер.
- Docker-образ без root, с `HEALTHCHECK`, и `.dockerignore`.
- CI: линт, тесты, сборка образа.

## Быстрый старт

```bash
git clone git@github.com:Alice088/go-template.git my-service
cd my-service

cp .env.example .env
# JWT-секрет обязателен: приложение не запустится с плейсхолдером
sed -i "s/change-me-in-production/$(openssl rand -hex 32)/" .env

# вариант 1: всё в докере (поднимает postgres и api)
make docker-up

# вариант 2: postgres в докере, api локально
docker compose up -d db
make migrate-up
make run
```

Проверка:

```bash
curl -s localhost:8080/v1/health   # {"status":"ok",...}
curl -s localhost:8080/v1/ready    # {"status":"ok","checks":{"database":"up"}}
open localhost:8080/swagger/index.html
```

## Make-таргеты

```
make help          # список всех таргетов
make run           # запустить локально
make build         # собрать bin/api (версия из git в -ldflags)
make test          # тесты с -race
make test-cover    # тесты + bin/coverage.out
make check         # fmt + vet + lint + test
make lint          # golangci-lint run
make fmt           # golangci-lint fmt (gofumpt + goimports)
make tidy          # go mod tidy
make swagger       # перегенерировать docs/
make migrate-up    # применить миграции
make migrate-down  # откатить последнюю миграцию
make docker-build  # собрать образ
make docker-up     # поднять стек
```

`swag` и `golang-migrate` не нужно устанавливать: таргеты запускают их через `go run`.
`DB_URL` для миграций собирается из переменных в `.env`.

## Конфигурация

Всё через переменные окружения, список с дефолтами — в `.env.example`.
Ключевое:

| Переменная | Назначение | Дефолт |
|---|---|---|
| `SERVER_HOST` / `SERVER_PORT` | адрес HTTP-сервера | `0.0.0.0` / `8080` |
| `SERVER_READ_HEADER_TIMEOUT` | защита от slowloris | `5s` |
| `SERVER_REQUEST_TIMEOUT` | таймаут обработки запроса | `30s` |
| `SERVER_SHUTDOWN_TIMEOUT` | таймаут graceful shutdown | `10s` |
| `SERVER_MAX_BODY_BYTES` | лимит тела запроса | `1048576` |
| `SERVER_ALLOWED_ORIGINS` | CORS, через запятую | `*` |
| `DB_HOST` … `DB_SSLMODE` | подключение к PostgreSQL | localhost, `gotemplate` |
| `DB_MAX_CONNS` / `DB_MIN_CONNS` | размер пула | `25` / `5` |
| `JWT_SECRET_KEY` | ключ подписи, **минимум 32 символа** | — |
| `JWT_ACCESS_EXPIRY` / `JWT_REFRESH_EXPIRY` | время жизни токенов | `15m` / `168h` |
| `LOG_LEVEL` / `LOG_FORMAT` | `debug\|info\|warn\|error`, `json\|pretty` | `info` / `json` |

`docker compose` читает тот же `.env`: значения `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`
используются и приложением, и сервисом postgres, поэтому расхождений быть не может.

## Ошибки

`internal/apierr` — одна точка правды для ошибок.

- `apierr.Error` — базовая ошибка: внутренний текст (в логи) и публичный текст (в ответ).
- `apierr.ValidationError` — ошибка валидации с разбивкой по полям.
- `apierr.PublicError` — интерфейс «эту ошибку безопасно отдать клиенту»:
  `HTTPStatus()`, `Code()`, `PublicMessage()`, `Details()`.

```go
entity, err := h.repo.Get(ctx, id)
if errors.Is(err, repository.ErrNotFound) {
    httpx.Error(w, r, apierr.NotFound("user not found").WithCause(err))
    return
}
if err != nil {
    httpx.Error(w, r, apierr.Internal("load user").WithCause(err)) // клиент увидит 500 + generic
    return
}
```

Ответ всегда один и тот же конверт:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "validation failed",
    "details": {"email": "is required"}
  }
}
```

`httpx.Error` сам решает: известная ошибка — отдаёт её статус/код/публичный текст,
неизвестная — логирует причину и отдаёт `500 internal_error`, не раскрывая детали.
`details` попадают в ответ только для ошибок, которые их задают явно.

## Структура

```
cmd/api/            точка входа, сборка зависимостей, graceful shutdown
internal/
  apierr/           типизированные ошибки и интерфейс PublicError
  auth/             JWT и bcrypt
  config/           загрузка и валидация конфигурации
  database/         пул pgx
  dto/              структуры запросов/ответов
  entity/           доменные модели
  handler/          HTTP-хендлеры
  httpx/            JSON-ответы и рендер ошибок
  logger/           настройка zerolog
  middleware/       request id, логи, recovery, CORS, JWT, роли
  repository/       доступ к данным
  router/v1/        маршруты и middleware-цепочка
  service/          бизнес-логика
docs/               сгенерированная OpenAPI-спека
migrations/         SQL-миграции golang-migrate
```

## Как добавить эндпоинт

1. `internal/entity` — модель, `internal/repository` — доступ к данным.
2. `internal/service` — бизнес-логика, ошибки возвращаются через `apierr`.
3. `internal/handler` — разбор запроса, `httpx.JSON` / `httpx.Error`, swagger-аннотации.
4. `internal/router/v1/router.go` — зарегистрировать маршрут, `make swagger`.
5. Тест на хендлер: `httptest.NewRecorder()` + фейковый сервис.

Про таймауты и лимиты: `middleware.Timeout` только ограничивает `r.Context()`, ответ
пишет сам хендлер. Поэтому передавайте контекст в БД и внешние вызовы, а ошибку
дедлайна возвращайте как `apierr.Unavailable(...)`. Аналогично `SERVER_MAX_BODY_BYTES`
ограничивает чтение тела: ошибку `*http.MaxBytesError` из `json.Decoder` нужно
превратить в `apierr.BadRequest("request body too large")`.

## Лицензия

MIT, см. [LICENSE](LICENSE).
