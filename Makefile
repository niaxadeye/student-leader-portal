
.PHONY: help up down logs deps-build api-build worker-build build \
        api-run worker-run frontend-build frontend-dev lint fmt migrate-up migrate-down seed

# Загружаем .env в окружение целей (если файл существует).
ifneq (,$(wildcard .env))
include .env
export
endif

help: ## Список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n",$$1,$$2}'

up: ## Поднять инфраструктуру (postgres, redis, minio)
	docker compose up -d

down: ## Остановить инфраструктуру
	docker compose down

logs: ## Логи инфраструктуры
	docker compose logs -f

api-build: ## Собрать бинарь api
	cd backend && go build -o bin/api ./cmd/api

worker-build: ## Собрать бинарь worker
	cd backend && go build -o bin/worker ./cmd/worker

build: api-build worker-build frontend-build ## Собрать всё

api-run: ## Запустить api локально
	cd backend && go run ./cmd/api

worker-run: ## Запустить worker локально
	cd backend && go run ./cmd/worker

frontend-build: ## Собрать фронтенд
	cd frontend && npm run build

frontend-dev: ## Dev-сервер фронтенда
	cd frontend && npm run dev

lint: ## Линтеры (go vet + eslint)
	cd backend && go vet ./...
	cd frontend && npm run lint

fmt: ## Форматирование
	cd backend && gofmt -w .
	cd frontend && npm run format

migrate-up: ## Применить миграции (добавится на Этапе 1)
	@echo "migrations добавятся на Этапе 1"

seed: ## Сиды (добавится на Этапе 1)
	@echo "seed добавится на Этапе 1"
