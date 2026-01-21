# =========================
# ENV
# =========================
include .env
export

APP_NAME=gadget-api
MIGRATE=migrate
MIGRATIONS_PATH=db/migrations
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
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

.PHONY: migrate-force
migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(version)

.PHONY: migrate-status
migrate-status:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version

# shortcut untuk dirty migration
.PHONY: migrate-fix
migrate-fix:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(version)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

# =========================
# DATABASE (DEV ONLY)
# =========================
.PHONY: reset-dev
reset-dev:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" drop -f
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up
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
	$(GO) run ./cmd/api/main.go
