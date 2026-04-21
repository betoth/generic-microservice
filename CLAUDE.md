# CLAUDE.md

Instructions for working on this project.

## Language

Go 1.25+. Follow [Effective Go](https://go.dev/doc/effective_go) conventions.

## Architecture

This project uses **Hexagonal Architecture** (Ports & Adapters) combined with **Clean Architecture** principles:

- Business logic lives in the domain — it must not depend on frameworks, databases, or external services
- Dependencies point inward: adapters depend on ports, never the other way around
- Each layer communicates through interfaces (ports)
- Every status change in `entries` must insert a new outbox record — the outbox is append-only
- Debezium (CDC) is the only mechanism that reads the outbox table — it captures WAL inserts and publishes to Kafka + S3. No Go code should poll or read the outbox table directly

### Layer structure

```
cmd/
├── api/            # HTTP server
├── entry-processor/  # (future) Reads Kafka (created) → SQS
└── sqs-consumer/   # (future) Reads SQS → marks entry as published
internal/
├── domain/        # Entities and business rules — no external dependencies
├── port/          # Interfaces (input ports, output ports)
├── service/       # Use cases — orchestrates domain logic via ports
├── adapter/
│   ├── in/
│   │   ├── http/      # HTTP handlers (input adapter) — package handler
│   │   └── sqs/       # (future) SQS consumer (input adapter)
│   └── out/
│       ├── repository/# Postgres repositories (output adapter)
│       ├── s3/        # (future) S3 client (output adapter)
│       ├── sqs/       # (future) SQS publisher (output adapter)
│       └── kafka/     # (future) Kafka producer (output adapter)
└── config/        # Configuration loading
migrations/        # SQL migration files
infra/             # Terraform files
```

## Dependencies

Do not add new libraries without explicit approval. If a use case seems to require a new dependency, ask first.

| Library | Purpose |
|---|---|
| `uptrace/bun` | Query builder / ORM for Postgres |
| `uptrace/bun/dialect/pgdialect` | Postgres dialect for bun |
| `uptrace/bun/driver/pgdriver` | Pure Go Postgres driver — used by both bun and golang-migrate (no external driver needed) |
| `golang-migrate/migrate/v4` | Database migrations |
| `go-chi/chi/v5` | HTTP router |
| `google/uuid` | UUID generation |
| `aws/aws-sdk-go-v2` | S3 and SQS clients (future phases) |
| `segmentio/kafka-go` | Kafka consumer — Entry Processor reads from Kafka (future phases) |

## Tests

Unit tests only using Go's standard `testing` package. No integration tests, no external dependencies in tests.

## Logging

Not used for now.

## Idempotency

Kafka does not guarantee exactly-once delivery. All consumers (Entry Processor, SQS Consumer) must handle duplicate events idempotently — processing the same event twice must produce the same result without side effects.

## Error handling

Two error categories:

- **Business errors** — domain rule violations. Defined in `internal/domain/errors.go` as instances of `*domain.BusinessError`. Code follows `GMS-XXX` pattern (e.g. `GMS-001`). HTTP 422.
- **Technical errors** — contract violations, infrastructure failures. Handled directly in the HTTP adapter. Code is the HTTP status as string (e.g. `"400"`, `"500"`). Description is the standard HTTP message (`Bad Request`, `Internal Server Error`).

The HTTP adapter uses a single `errors.As(err, &bizErr)` check — it never references specific error codes. Adding a new business error only requires a new variable in `domain/errors.go`.

All errors return a consistent JSON structure:
```json
{ "id": "uuid", "code": "...", "description": "..." }
```

Full business error code registry is defined in README.md.

## Environment variables

One file per environment at the project root. All `*.env` files are gitignored — never commit them.

- `dev.env` — local development
- `hml.env` — staging
- `prd.env` — production

In production, secrets should come from a secret manager (e.g. AWS Secrets Manager), not from files.

## Infrastructure as Code

Use **Terraform** to manage infrastructure (LocalStack for local development). Keep it simple — no modules, no remote state for local setup.

Terraform files live in `infra/`.

## Atomic transactions

When a use case requires atomicity across multiple repositories, use the `TransactionManager` port — never hold `*bun.DB` or `bun.IDB` in the service layer.

The transaction is carried through `context.Context`. Repositories extract it internally via `txFromContext(ctx)` defined in `adapter/out/repository/tx.go`. Nothing outside `adapter/out/repository` should reference bun transaction types.

```
port.TransactionManager.RunInTx(ctx, func(ctx context.Context) error {
    entryRepo.Insert(ctx, entry)   // tx comes from ctx
    outboxRepo.Insert(ctx, outbox) // tx comes from ctx
})
```

ORM models (`entryModel`, `outboxModel`) live in `adapter/repository/models.go` and are never exposed to the service or domain layers. Domain entities are mapped to ORM models inside the repository.

## Go conventions

- Interfaces are defined where they are **used** (port layer), not where they are implemented
- Keep interfaces small — prefer single-method interfaces when possible. When methods are always used together by the same consumers (positive trade-off), a single multi-method interface is acceptable
- Return errors, do not panic in business logic
- Use `context.Context` as the first parameter in all functions that perform I/O
- No `init()` functions — use explicit initialization in `main.go`
- Avoid global state
- Name receivers consistently within a type