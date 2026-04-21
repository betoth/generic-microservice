# generic-microservice

A simple Go microservice built as a target application for automation testing, covering HTTP, Postgres, SQS, S3, Kafka, and the outbox pattern.

## Why the Outbox Pattern?

When a request arrives at `POST /entries`, two things must happen reliably: persist the entry **and** dispatch an event. Calling SQS directly inside the HTTP handler creates a risk — if the message send fails after the database commit, the event is lost silently.

The outbox pattern solves this by writing the event as a row in the `outbox` table **inside the same transaction** as the entry. Debezium (CDC) then captures every insert via Postgres WAL and publishes to Kafka, guaranteeing that no event is lost even if the service crashes mid-request.

Every status change in `entries` inserts a new outbox record — records are never updated. All records for the same entry share the same `business_key` (the entry ID), making it possible to reconstruct the full event history for any given entry. **Debezium is the only component that reads from the outbox table** — no Go code polls or reads the outbox directly.

## Architecture

```
POST /entries
    → entries (status: created) + outbox record (event_type: entry.created)
    → S3: entries/<snapshotID1>.json (created snapshot)

Debezium (CDC)  ← reads Postgres WAL, no polling, no outbox modification
    → publishes event_data to Kafka topic generic-microservice

Entry Processor  ← reads Kafka
    → event_type = entry.created
    → reads S3 entries/<snapshotID1>.json
    → sends SQS message to input-generic-microservice (file: <snapshotID2>.json)
    → writes S3 entries/<snapshotID2>.json (processing snapshot)
    → updates entry to processing + inserts outbox (event_type: entry.processing)

Entry Publisher  ← reads SQS input-generic-microservice
    → reads S3 entries/<snapshotID2>.json
    → writes S3 entries/<snapshotID3>.json (published snapshot)
    → updates entry to published + inserts outbox (event_type: entry.published)
    → any error → DLQ (dlq-generic-microservice) → delete from input queue
```

Each component has a single responsibility:

| Component        | Role                                                              |
|------------------|-------------------------------------------------------------------|
| Postgres         | Persistent state (entries + outbox)                               |
| Debezium         | CDC — reads WAL, publishes outbox events to Kafka                 |
| Entry Processor  | Reads Kafka, forwards `created` events to SQS                    |
| Entry Publisher  | Reads SQS, marks entries as `published`                          |
| Kafka            | Event bus — central source of truth for consumers                 |
| S3               | Immutable snapshot per event (`entries/<snapshotID>.json`)        |
| SQS              | Three queues: events, input, DLQ                                  |

## S3 Snapshot Model

Each event generates an independent, immutable snapshot in S3 at `entries/<snapshotID>.json`. The snapshot ID is a new random UUID generated at write time — independent of the entry ID and all other snapshots. Files are flat under `entries/` (no subdirectories).

The snapshot ID is propagated forward via `file.name` in the event payload, so each step knows what to read:

| Step             | Reads                         | Writes                        |
|------------------|-------------------------------|-------------------------------|
| entry-create     | —                             | `entries/<uuid1>.json`        |
| entry-processor  | `entries/<uuid1>.json`        | `entries/<uuid2>.json`        |
| entry-publisher  | `entries/<uuid2>.json`        | `entries/<uuid3>.json`        |

## Event Flow

### 1. `POST /entries`
- Persists entry to `entries` table with `status = created`
- Generates `snapshotID1 = uuid.New()`, writes `entries/<snapshotID1>.json` to S3
- Inserts outbox record (`event_type = entry.created`) with `file.name = "<snapshotID1>.json"` — transactional

### 2. Debezium (CDC)
- Reads Postgres WAL — no polling, no modification to the outbox table
- Captures every insert on the `outbox` table
- Publishes `event_data` JSON to Kafka topic `generic-microservice`

### 3. Entry Processor (`cmd/entry-processor`)
- Reads Kafka events with `event_type = entry.created`
- Parses `snapshotID1` from `event_data.file.name`
- Reads `entries/<snapshotID1>.json` from S3
- Generates `snapshotID2 = uuid.New()`, writes `entries/<snapshotID2>.json` to S3
- Sends SQS message to `input-generic-microservice` with `file.name = "<snapshotID2>.json"`
- Updates entry `status = processing` + inserts outbox record (`event_type = entry.processing`)

### 4. Entry Publisher (`cmd/entry-publisher`)
- Reads SQS messages from `input-generic-microservice`
- Parses `snapshotID2` from `message.file.name`
- Reads `entries/<snapshotID2>.json` from S3
- Generates `snapshotID3 = uuid.New()`, writes `entries/<snapshotID3>.json` to S3
- Updates entry `status = published` + inserts outbox record (`event_type = entry.published`) — transactional
- Deletes message from SQS input queue
- On any error (technical or business): sends to DLQ, then deletes from input queue

### 5. Kafka event_data format

All events share the same `event_data` structure:

```json
{
  "id": "<event-uuid>",
  "business_key": "<entry-uuid>",
  "event_type": "entry.created | entry.processing | entry.published",
  "topic": "generic-microservice",
  "entry_data": {
    "subject": "...",
    "status": "created | processing | published",
    "created_at": "...",
    "updated_at": "..."
  },
  "file": {
    "name": "<snapshotID>.json",
    "path": "s3://<bucket>/entries/<snapshotID>.json"
  }
}
```

| Event               | Trigger                                     |
|---------------------|---------------------------------------------|
| `entry.created`     | Entry saved to Postgres                     |
| `entry.processing`  | Entry Processor forwarded the event to SQS  |
| `entry.published`   | Entry Publisher finished processing         |

## Database Schema

### `entries`

| Column     | Type      |
|------------|-----------|
| id         | UUID (PK) |
| date       | timestamptz |
| subject    | text      |
| content    | text      |
| status     | enum      |
| created_at | timestamp |
| updated_at | timestamp |

### `outbox`

| Column       | Type      | Description                               |
|--------------|-----------|-------------------------------------------|
| id           | UUID (PK) | Unique outbox row ID                      |
| business_key | UUID      | Entry ID — links all events for one entry |
| event_type   | text      | e.g. `entry.created`                      |
| topic        | text      | Kafka topic (e.g. `generic-microservice`) |
| event_data   | text      | JSON payload published to Kafka           |
| created_at   | timestamp |                                           |

## Infrastructure

Services managed via `deploy/docker-compose.yml`:

| Service        | Port  | Image                                          |
|----------------|-------|------------------------------------------------|
| Postgres       | 5432  | postgres:15                                    |
| LocalStack     | 4566  | localstack/localstack:4.14.0                   |
| Kafka          | 9092  | apache/kafka:3.7.2                             |
| Kafka Connect  | 8083  | debezium/connect (with Postgres connector)     |
| Kafka UI       | 8090  | provectuslabs/kafka-ui                         |
| Filestash      | 8334  | machines/filestash                             |

LocalStack exposes **S3** and **SQS** (three queues: `generic-microservice`, `input-generic-microservice`, `dlq-generic-microservice`). Postgres must have `wal_level = logical` enabled for Debezium CDC.

## Error Responses

All errors return a consistent JSON structure:

```json
{
  "id": "uuid",
  "code": "...",
  "description": "..."
}
```

### Technical errors

Code is the HTTP status as a string. Description is the standard HTTP message.

```json
{ "id": "uuid", "code": "400", "description": "Bad Request" }
{ "id": "uuid", "code": "500", "description": "Internal Server Error" }
```

### Business errors

Code follows the pattern `GMS-XXX`.

| Code    | Description                                           | HTTP |
|---------|-------------------------------------------------------|------|
| GMS-001 | date must be today                                    | 422  |
| GMS-002 | entry is not in created status and cannot be processed | 422  |
| GMS-003 | entry already published                               | 422  |

## DLQ Message Format

When the Entry Publisher routes a message to the DLQ:

```json
{
  "error": {
    "id": "<uuid>",
    "code": "GMS-003 or other",
    "details": "error description"
  },
  "message": {
    "business_key": "<entry-uuid>",
    "file": {
      "name": "<snapshotID>.json",
      "path": "s3://<bucket>/entries/<snapshotID>.json"
    }
  }
}
```

`message` preserves the original SQS message payload as-is.

## Usage

Copy `.env.example` to `dev.env` and fill in the values before running:

```bash
cp .env.example dev.env
```

Then start everything:

```bash
make up
```

This starts all Docker services, provisions Terraform infrastructure (S3 + SQS queues), registers the Debezium connector, and starts all three Go processes concurrently.

### Create an entry

```bash
curl -X POST http://localhost:8080/entries \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2026-04-10T12:00:00.000Z",
    "subject": "example subject",
    "content": "entry content goes here"
  }'
```

Expected response (`201 Created`):

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-04-10T12:00:00.000Z"
}
```

### Health check

```bash
curl http://localhost:8080/health
```

### Inspect S3 snapshots

```bash
# List all snapshots
make s3-ls

# Read a specific snapshot
make s3-cat ID=<snapshotID>
```

## Scope

### Phase 1 — API + Outbox
- [x] `POST /entries` endpoint
- [x] `entries` and `outbox` tables with migrations
- [x] Outbox record inserted transactionally on entry creation

### Phase 2 — Infrastructure
- [x] Terraform infra (LocalStack S3 + SQS, Postgres via docker-compose, Kafka)
- [x] Debezium connector configured for `outbox` table (Postgres WAL)
- [x] Debezium publishes `event_data` to Kafka

### Phase 3 — S3 Snapshots + SQS Flow
- [x] S3 adapter with `WriteSnapshot` / `ReadSnapshot` (flat `entries/<uuid>.json` format)
- [x] `cmd/entry-processor`: reads Kafka (`entry.created`) → writes processing snapshot → sends SQS → inserts outbox (`entry.processing`)
- [x] `cmd/entry-publisher`: reads SQS → writes published snapshot → inserts outbox (`entry.published`) → DLQ on error

### Phase 4 — MCP Server
- [ ] MCP server exposing project documentation as resources
- [ ] Resources: architecture overview, entry flow, event payload spec, error codes
- [ ] Compatible with OpenCode and other MCP clients

### Phase 5 — Idempotency
- [ ] All Kafka consumers handle duplicate events idempotently
