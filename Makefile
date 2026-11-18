.PHONY: db-check db-create api worker web test test-backend test-frontend build

db-check:
	pg_isready -h localhost -p 5432

db-create:
	createdb ledgerops

api:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; go -C backend run ./cmd/api

worker:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; go -C backend run ./cmd/worker

web:
	npm --prefix frontend run dev

test: test-backend test-frontend

test-backend:
	go -C backend test ./...

test-frontend:
	npm --prefix frontend test

build:
	go -C backend build ./cmd/api ./cmd/worker
	npm --prefix frontend run build
