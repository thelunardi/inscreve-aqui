-include .env
export

.PHONY: help db-up db-down db-reset migrate-up migrate-status migrate-down migration build vet test check

help: ## Lista os comandos disponíveis
	@grep -hE '^[a-z-]+:.*## ' Makefile | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

db-up: ## Sobe o PostgreSQL local e espera ficar pronto
	docker compose up -d --wait postgres

db-down: ## Para o PostgreSQL local (mantém os dados)
	docker compose down

db-reset: ## Apaga os dados do PostgreSQL local e sobe de novo
	docker compose down -v
	docker compose up -d --wait postgres

migrate-up: ## Aplica as migrations pendentes
	go run ./cmd/migrate up

migrate-status: ## Mostra as migrations aplicadas e pendentes
	go run ./cmd/migrate status

migrate-down: ## Desfaz a última migration (somente APP_ENV=development)
	go run ./cmd/migrate down

migration: ## Cria uma migration vazia: make migration name=create_x
	@test -n "$(name)" || (echo "uso: make migration name=create_x" && exit 1)
	@n=$$(ls migrations/*.sql 2>/dev/null | sed -E 's|.*/0*([0-9]+)_.*|\1|' | sort -n | tail -1); \
	f=$$(printf "migrations/%05d_%s.sql" $$(( $${n:-0} + 1 )) "$(name)"); \
	printf -- "-- +goose Up\n\n-- +goose Down\n" > $$f; \
	echo "criado $$f"

build: ## Compila todos os pacotes
	go build ./...

vet: ## Roda o go vet
	go vet ./...

test: ## Roda os testes (precisa do banco: make db-up)
	go test ./...

check: build vet test ## Build, vet e testes
