# =========================
# ENV
# =========================
include .env
export

APP_NAME=gadget-api
MIGRATE=migrate
MIGRATIONS_PATH=internal/shared/database/migrations
SQLC=sqlc
GO=go

# =========================
# HELP
# =========================
.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Migration:"
	@echo "  make migrate-create name=create_users_table"
	@echo "  make migrate-up"
	@echo "  make migrate-down"
	@echo "  make migrate-force version=1"
	@echo "  make migrate-status"
	@echo "  make migrate-fix version=1"
	@echo ""
	@echo "Database:"
	@echo "  make reset-dev"
	@echo ""
	@echo "sqlc:"
	@echo "  make sqlc"
	@echo ""
	@echo "Test:"
	@echo "  make test"
	@echo ""
	@echo "Run:"
	@echo "  make run"

# =========================
# MIGRATION
# =========================
.PHONY: migrate-create
migrate-create:
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

.PHONY: migrate-up
migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' up
	$(MAKE) sqlc

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' down 1

.PHONY: migrate-force
migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' force $(version)

.PHONY: migrate-status
migrate-status:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' version

# shortcut untuk dirty migration
.PHONY: migrate-fix
migrate-fix:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' force $(version)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' up

# =========================
# DOCKER
# =========================
.PHONY: docker-up
docker-up:
	docker-compose up -d --build

.PHONY: docker-down
docker-down:
	docker-compose down

.PHONY: docker-infra
docker-infra:
	docker-compose up -d postgres redis kafka kafka-ui worker consumer

.PHONY: docker-infra-stop
docker-infra-stop:
	docker-compose stop postgres redis kafka kafka-ui worker consumer

.PHONY: docker-logs
docker-logs:
	docker-compose logs -f

# Menjalankan migrasi di dalam docker
.PHONY: docker-migrate
docker-migrate:
	docker-compose run --rm migrator

# =========================
# MONITORING
# =========================

# Cek status container proyek ini saja
.PHONY: ps
ps:
	docker compose ps

# Cek semua container yang ada di komputer dengan format tabel
.PHONY: docker-ls
docker-ls:
	docker ps -a

# =========================
# DATABASE (DEV ONLY)
# =========================
.PHONY: reset-dev
reset-dev:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' drop -f
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database '$(DB_MIGRATION_URL)' up
	$(SQLC) generate

# =========================
# SQLC
# =========================
.PHONY: sqlc
sqlc:
	$(SQLC) generate

# =========================
# TEST
# =========================
.PHONY: test
test:
	$(GO) test ./... -v

# =========================
# RUN
# =========================
.PHONY: run
run:
	$(GO) run ./cmd/api

# =========================
# SEEDER / LOAD TEST
# =========================
.PHONY: seed
seed:
	@echo "Menjalankan seeder..."
	$(GO) run ./cmd/seed/main.go

.PHONY: load-test
load-test:
	@echo "=== Testing Performa Dashboard ==="
	@echo "Request 1 (Ambil dari Database):"
	time curl -s http://localhost:3000/api/v1/dashboard/products > /dev/null
	@echo "\nRequest 2 (Ambil dari Redis Cache):"
	time curl -s http://localhost:3000/api/v1/dashboard/products > /dev/null

.PHONY: reset-seed
reset-seed:
	@echo "Membersihkan data lama dan mengisi ulang..."
	postgres -u$(DB_USER) -p$(DB_PASSWORD) -e "DELETE FROM products; DELETE FROM categories;" $(DB_NAME)
	$(MAKE) seed

# =========================
# SWAGGER
# =========================
.PHONY: swag
swag:
	swag init -g cmd/api/main.go	