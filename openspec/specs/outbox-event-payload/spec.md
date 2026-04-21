### Requirement: Outbox carries topic and event_data
Every outbox record SHALL contain `topic` (Kafka topic name) and `event_data` (partial Kafka payload). The `status` and `file_content` columns SHALL NOT exist on the outbox table.

#### Scenario: Outbox record created with entry
- **WHEN** `CreateEntry` is called successfully
- **THEN** the inserted outbox row SHALL have `topic` and `event_data` populated and non-empty

### Requirement: event_data is a structured partial payload
`event_data` SHALL be a JSON object with: `id` (a new UUID generated for the event, not the outbox row ID), `business_key` (entry UUID), `event_type`, `topic`, a nested `entry_data` object (with `subject`, `status`, `created_at`, `updated_at`), and a nested `file` object pointing to the independent S3 snapshot generated for that specific event.

This format is **mandatory for all event types**: `entry.created`, `entry.processing`, and `entry.published`. No event type SHALL omit `entry_data` or `file`.

The `entry_data.status` SHALL reflect the status at the moment the event was generated. The `file` fields SHALL point to the snapshot written to S3 as part of that same event processing. The snapshot ID is a random UUID generated at write time — not derived from the entry ID.

#### Scenario: event_data structure on entry creation
- **WHEN** an entry is created with id `X`
- **THEN** `event_data` SHALL deserialize to:
  ```json
  {
    "id": "<new-uuid>",
    "business_key": "X",
    "event_type": "entry.created",
    "topic": "generic-microservice",
    "entry_data": {
      "subject": "<entry subject>",
      "status": "created",
      "created_at": "<timestamp>",
      "updated_at": "<timestamp>"
    },
    "file": {
      "name": "<snapshotID>.json",
      "path": "s3://<bucket>/entries/<snapshotID>.json"
    }
  }
  ```
  Where `<snapshotID>` is a random UUID independent of the entry ID.

#### Scenario: event_data structure on entry processing
- **WHEN** an entry is transitioned to `processing`
- **THEN** `event_data` SHALL deserialize to the same structure with `event_type = "entry.processing"`, `entry_data.status = "processing"`, and a new independent `<snapshotID>` in `file`.

#### Scenario: event_data structure on entry published
- **WHEN** an entry is transitioned to `published`
- **THEN** `event_data` SHALL deserialize to the same structure with `event_type = "entry.published"`, `entry_data.status = "published"`, and a new independent `<snapshotID>` in `file`.

### Requirement: event_data.id is not the outbox row ID
The `id` field inside `event_data` SHALL be a freshly generated UUID at serialization time, independent of the outbox table's primary key.

#### Scenario: event_data.id differs from outbox id
- **WHEN** an outbox row is inserted
- **THEN** `event_data.id` SHALL be a different UUID from the outbox row `id`

### Requirement: Outbox insert is atomic with entry insert
The outbox record SHALL be inserted in the same database transaction as the `entries` row.

#### Scenario: Transaction rollback on outbox failure
- **WHEN** the outbox insert fails
- **THEN** the entry insert SHALL also be rolled back and no partial state SHALL be persisted
