BINARY=main
CMD=./cmd/app/main.go
MIGRATE=github.com/golang-migrate/migrate/v4/cmd/migrate

.PHONY: all build run stop logs swag migrate-up migrate-down lint test clean

all: swag build

build:
	go build -o $(BINARY) $(CMD)

run:
	go run $(CMD)

up:
	docker compose up --build -d

down:
	docker compose down

stop:
	docker compose stop

logs:
	docker compose logs -f app

swag:
	swag init -g $(CMD) -o docs/

migrate-up:
	go run -tags 'postgres' $(MIGRATE) -path migrations -database "$(DB_URL)" up

migrate-down:
	go run -tags 'postgres' $(MIGRATE) -path migrations -database "$(DB_URL)" down

lint:
	golangci-lint run ./...

test:
	go test ./... -v -race

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

tidy:
	go mod tidy

clean:
	-del /f main 2>nul
	-del /f main.exe 2>nul
	docker compose down -v

help:
	@echo "Available commands:"
	@echo "  make up           - build and start via docker compose"
	@echo "  make down         - stop and remove containers"
	@echo "  make logs         - show app logs"
	@echo "  make swag         - generate swagger documentation"
	@echo "  make migrate-up   - apply migrations (requires DB_URL)"
	@echo "  make migrate-down - rollback migrations (requires DB_URL)"
	@echo "  make lint         - run linter"
	@echo "  make test         - run tests"
	@echo "  make test-cover   - run tests with coverage report"
	@echo "  make tidy         - go mod tidy"
	@echo "  make clean        - remove binary and volumes"