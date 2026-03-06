# Go Gadget API

Production-minded backend API for modern gadget e-commerce, built with Go and designed to showcase real-world backend engineering depth: modular architecture, robust transaction handling, payment workflow integration, and async event-driven processing.

## Why This Project Stands Out

This repository is not a CRUD demo. It demonstrates practical backend concerns that matter in production:

- Security-first API design: JWT auth, role-based access control, and verified webhook signature handling.
- Reliability patterns: idempotent checkout, outbox pattern, async worker-consumer pipeline.
- Abuse protection: endpoint-specific rate limiting by IP and by authenticated user.
- Clean modular boundaries: domain-driven package structure with service and repository isolation.
- Real operational setup: Dockerized infra (PostgreSQL, Redis, Kafka), migrations, seeding, and tests.

## Core Stack

- Language: Go 1.25
- Web framework: Gin
- Database: PostgreSQL + `sqlc` generated queries
- Cache / coordination: Redis (`go-redis/v9`)
- Messaging: Kafka (`segmentio/kafka-go`)
- Payment gateway: Midtrans Snap API
- Media storage: Cloudinary
- Email service: Resend
- Auth: JWT (`golang-jwt/jwt/v5`)
- Logging: Zap structured logger
- Testing: Testify, GoMock, SQLMock

## Architecture

The codebase follows layered modular architecture per domain:

1. `routes`: endpoint registration + middleware composition
2. `handler`: HTTP binding, response envelope, transport concerns
3. `service`: business rules, validation, transaction orchestration
4. `repo`: data access with `sqlc` generated queries
5. `dto/errors`: API contracts and domain-specific error mapping

Dependency wiring is centralized in `internal/app/registry.go`.

## Domain Modules

Main modules implemented in `internal/`:

- `auth`
- `product`
- `category`
- `brand`
- `review`
- `cart`
- `order`
- `address`
- `customer`
- `wishlist`
- `dashboard`

Supporting modules:

- `middleware` (auth, rate limit, idempotency, request context)
- `outbox` + `messaging/kafka` (event publishing and consumption)
- `midtrans`, `cloudinary`, `email`
- `shared/database` (migrations, sqlc queries, generated code, seed)

## Advanced API Features

### 1) Adaptive Rate Limiting (IP + User)
Implemented via token bucket (`golang.org/x/time/rate`) with endpoint-specific policy.

- Public endpoints: rate limit by IP
- Authenticated endpoints: rate limit by `user_id`
- Sensitive routes are stricter:
  - Register/login
  - Checkout
  - Review create/update
  - Admin mutation endpoints

This protects from brute force, spam, accidental double-submit, and abusive scraping.

### 2) Idempotent Checkout with Redis Lock + Response Cache
`POST /api/v1/orders/checkout` uses `Idempotency-Key` middleware:

- Build key by route + user + idempotency key
- Return cached success response if same request already completed
- Use Redis `SETNX` lock to block concurrent duplicates
- Return conflict when duplicate request is still processing

This prevents duplicate order creation in retry/double-click scenarios.

### 3) Transactional Checkout + Outbox Pattern
Checkout flow is wrapped in DB transaction:

- Create order
- Create order items
- Insert outbox event (`DELETE_CART`)
- Commit once all successful

A dedicated worker polls pending outbox events and publishes to Kafka (`order.events`), then marks them sent. This ensures reliable event publishing without dual-write inconsistency.

### 4) Async Worker + Consumer Pipeline
Separate executables:

- `cmd/worker`: publish outbox events to Kafka
- `cmd/consumer`: consume `order.events` and apply side effects (cart cleanup)

This separation demonstrates scalable asynchronous architecture beyond synchronous request/response.

### 5) Midtrans Payment Integration with Webhook Verification
Order service supports Midtrans Snap token creation and webhook handling:

- Signature validation (`SHA-512`) against `MIDTRANS_SERVER_KEY`
- Gross amount validation to detect payload mismatch
- Payment status transition handling (`UNPAID`, `PAID`, `REFUNDED`)
- Support continue-payment with token refresh on expiry

### 6) Auth + Authorization + Context-Aware Logging

- JWT access/refresh flow
- Role middleware for admin-only routes (`ADMIN`, `SUPERADMIN`)
- Optional auth middleware for hybrid guest/auth endpoints
- Request context propagation with `request_id` and `user_id` into structured logs

### 7) Verified Purchase Review Integrity
Review creation requires:

- user is authenticated
- product exists
- user has completed purchase of the product
- user has not reviewed the same product already

This keeps review quality trustworthy.

### 8) Soft Delete + Restore + Media Cleanup Safety
For resources like products/categories/brands:

- soft-delete style lifecycle with restore endpoints
- Cloudinary upload/update flow with rollback/cleanup safeguards when transaction fails

## API Base URL

- Base path: `/api/v1`

## Main Endpoint Surface

- `auth`: register, login, refresh, me, logout, password reset, email confirmation
- `products`: public listing/detail, admin management, review eligibility
- `categories` / `brands`: public catalog + admin CRUD/restore
- `reviews`: create/list/update/delete with eligibility enforcement
- `carts`: item operations, count/detail, clear cart
- `orders`: checkout, list/detail, cancel/complete, continue payment, admin status update
- `midtrans`: payment notification webhook
- `addresses`: customer address management
- `customers`: profile update + admin customer management
- `wishlists`: add/remove/list items
- `admin/dashboard`: aggregated business metrics

Postman collection is available at:

- `postman_collection.json`

## API Response Envelope

The API uses a consistent envelope:

- success: `ok: true`, `data`, optional `meta`
- failure: `ok: false`, structured `error { code, message, details }`

This keeps frontend integration predictable.

## Local Setup

### 1) Prerequisites

- Go 1.25+
- PostgreSQL
- Redis
- Kafka
- `migrate` CLI
- `sqlc` CLI

### 2) Configure Environment

Copy `.env.example` to `.env`, then fill required values:

```bash
cp .env.example .env
```

Important env vars:

- `PORT`
- `DB_URL`
- `JWT_SECRET`
- `REDIS_ADDR`
- `KAFKA_BROKER`
- `MIDTRANS_*`
- `CLOUDINARY_*`
- `RESEND_API_KEY`, `RESEND_FROM_EMAIL`
- `WEBSTORE_URL`

### 3) Start Infrastructure

```bash
make docker-infra
```

This starts: PostgreSQL, Redis, Kafka, Kafka UI, worker, and consumer.

### 4) Run Migrations + SQLC Generate

```bash
make migrate-up
```

### 5) Run API

```bash
make run
```

API will run on `http://localhost:3000` (default).

## Helpful Commands

- `make test` - run tests (all modules or specific module)
- `make seed` - seed sample data
- `make docker-up` - build and run full stack
- `make docker-down` - stop stack
- `make ps` - show service status

## Testing

The project includes unit tests across handler and service layers, with mocks for repository/service boundaries and SQL-level scenarios.

Run all tests:

```bash
go test ./... -v
```

## Project Structure

```text
cmd/api                     # API entrypoint
cmd/worker                  # outbox -> kafka publisher worker
cmd/consumer                # kafka consumer worker
cmd/seed                    # database seeder
internal/app                # bootstrap and dependency registry
internal/<domain>           # domain modules (handler/service/repo/routes)
internal/middleware         # auth, rate limit, idempotency, logging context
internal/messaging/kafka    # producer/consumer implementation
internal/outbox             # outbox persistence and processing
internal/shared/database    # migrations, sql queries, sqlc generated code, seed
reference/                  # technical notes and references
```

## Engineering Notes for Recruiters

This codebase intentionally demonstrates backend capabilities expected from a production-ready Go engineer:

- Designing secure and resilient API flows, not only happy-path CRUD.
- Applying distributed-system safety patterns (idempotency + outbox + async workers).
- Building with operational clarity (structured logs, retries, graceful shutdown, clear env-driven configuration).
- Maintaining testability and modularity through explicit interfaces and dependency injection.

If you are evaluating backend engineering maturity, this repo highlights practical implementation tradeoffs and production-oriented decision making.
