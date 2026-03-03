# Subscription Service

REST-сервис для агрегации данных об онлайн-подписках пользователей.  
Тестовое задание — Effective Mobile, Junior Golang Developer.

## Стек

- **Go 1.24**
- **Gin** — HTTP-роутер
- **PostgreSQL 15** — база данных
- **zerolog** — структурированное логирование
- **golang-migrate** — миграции БД
- **swaggo/swag** — Swagger-документация
- **Docker Compose** — запуск окружения

## Структура проекта

```
.
├── cmd/server/
│   └── main.go                  # Точка входа
├── internal/
│   ├── config/                  # Загрузка конфигурации из .env
│   ├── handler/                 # HTTP-хендлеры + интерфейс сервиса
│   ├── service/                 # Бизнес-логика + интерфейс репозитория
│   ├── repository/              # Работа с PostgreSQL
│   └── model/                   # Модели данных и DTO
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── docs/                        # Сгенерированная Swagger-документация
├── .env.example                 # Пример переменных окружения
├── .gitignore
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## Быстрый старт

### 1. Клонировать репозиторий

```bash
git clone <repo-url>
cd <repo>
```

### 2. Создать .env

```bash
cp .env.example .env
```

### 3. Сгенерировать Swagger-документацию

```bash
swag init -g cmd/server/main.go -o docs/
```

> Для установки swag: `go install github.com/swaggo/swag/cmd/swag@latest`

### 4. Запустить через Docker Compose

```bash
docker compose up --build -d
```

Сервис поднимется на `http://localhost:8080`.  
Миграции применяются **автоматически** при старте приложения.

### 5. Проверить что сервис запустился

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## API

### Эндпоинты

| Метод    | Путь                            | Описание                                |
|----------|---------------------------------|-----------------------------------------|
| `POST`   | `/api/v1/subscriptions/`        | Создать подписку                        |
| `GET`    | `/api/v1/subscriptions/`        | Список подписок (фильтр по `user_id`)   |
| `GET`    | `/api/v1/subscriptions/:id`     | Получить подписку по ID                 |
| `PUT`    | `/api/v1/subscriptions/:id`     | Обновить подписку                       |
| `DELETE` | `/api/v1/subscriptions/:id`     | Удалить подписку                        |
| `GET`    | `/api/v1/subscriptions/total`   | Подсчёт суммарной стоимости за период   |
| `GET`    | `/health`                       | Health check                            |

### Swagger

После запуска документация доступна по адресу:

```
http://localhost:8080/swagger/index.html
```

### Примеры запросов

**Создать подписку**
```bash
curl -X POST http://localhost:8080/api/v1/subscriptions/ \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'
# {"id": 1}
```

**Создать подписку с датой окончания**
```bash
curl -X POST http://localhost:8080/api/v1/subscriptions/ \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Netflix",
    "price": 799,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "01-2025",
    "end_date": "06-2025"
  }'
```

**Получить список подписок пользователя**
```bash
curl "http://localhost:8080/api/v1/subscriptions/?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba"
```

**Обновить подписку**
```bash
curl -X PUT http://localhost:8080/api/v1/subscriptions/1 \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 500,
    "start_date": "07-2025"
  }'
```

**Подсчитать стоимость за период**
```bash
curl "http://localhost:8080/api/v1/subscriptions/total?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&from=01-2025&to=12-2025"
# {"total_cost": 9988}
```

**Подсчитать стоимость по конкретному сервису**
```bash
curl "http://localhost:8080/api/v1/subscriptions/total?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Netflix&from=01-2025&to=12-2025"
```

**Удалить подписку**
```bash
curl -X DELETE http://localhost:8080/api/v1/subscriptions/1
```

## Переменные окружения

| Переменная          | По умолчанию | Описание                         |
|---------------------|--------------|----------------------------------|
| `PORT`              | `8080`       | Порт сервиса                     |
| `DB_URL`            | —            | Строка подключения к PostgreSQL  |
| `LOG_LEVEL`         | `info`       | Уровень логов: `debug/info/warn` |
| `DB_MAX_IDLE_CONNS` | `5`          | Макс. кол-во idle-соединений     |
| `DB_MAX_OPEN_CONNS` | `10`         | Макс. кол-во открытых соединений |

Формат `DB_URL`:
```
postgres://user:password@localhost:5432/subs_db?sslmode=disable
```

## Makefile

```bash
make up            # Собрать и запустить через docker compose
make down          # Остановить контейнеры
make logs          # Логи приложения
make swag          # Сгенерировать swagger-документацию
make test          # Запустить тесты
make test-cover    # Тесты с отчётом покрытия (coverage.html)
make lint          # Запустить линтер
make migrate-up    # Применить миграции вручную (нужен DB_URL в окружении)
make migrate-down  # Откатить миграции
make clean         # Удалить бинарник и volumes
```

## Тесты

```bash
make test
# или
go test ./... -v
```

Покрыты все три слоя приложения:

- **handler** — HTTP-статусы, валидация входных данных (`httptest` + mock-сервис)
- **service** — бизнес-логика, расчёт суммы, валидация дат (mock-репозиторий)
- **repository** — SQL-запросы, обработка `ErrNotFound` (`sqlmock`)

## Остановка

```bash
docker compose down       # остановить контейнеры
docker compose down -v    # остановить и удалить volumes (данные БД)
```