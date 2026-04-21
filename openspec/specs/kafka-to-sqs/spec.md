### Requirement: Entry Processor binary exists
The project SHALL include a `cmd/entry-processor/main.go` binary that runs as a long-lived process consuming from the `generic-microservice` Kafka topic.

#### Scenario: Binary compiles
- **WHEN** `go build ./cmd/entry-processor/` is run
- **THEN** the binary SHALL compile without errors

### Requirement: Publisher filters for entry.created events only
The Entry Processor SHALL process only Kafka messages where `event_data.event_type = "entry.created"`. All other event types SHALL be silently ignored and their Kafka offsets committed.

#### Scenario: entry.created event is processed
- **WHEN** a Kafka message with `event_type = "entry.created"` is consumed
- **THEN** the publisher SHALL send an SQS message and update the entry status

#### Scenario: Other event types are ignored
- **WHEN** a Kafka message with any other `event_type` is consumed
- **THEN** no SQS message SHALL be sent and the Kafka offset SHALL be committed

### Requirement: SQS message is sent to input-generic-microservice
For each `entry.created` event, the publisher SHALL send a message to the SQS queue `input-generic-microservice`. The message body SHALL be a JSON object with `business_key` (entry UUID) and `file` (object with `name` and `path` pointing to the new **processing** snapshot).

#### Scenario: SQS message format
- **WHEN** an `entry.created` event with `business_key = X` is processed
- **THEN** the SQS message body SHALL deserialize to:
  ```json
  { "business_key": "X", "file": { "name": "<processingSnapshotID>.json", "path": "s3://<bucket>/entries/<processingSnapshotID>.json" } }
  ```
  Where `<processingSnapshotID>` is a new random UUID generated for the processing snapshot.

### Requirement: Entry status is updated to processing after SQS send
After a successful SQS send, the publisher SHALL:
1. Parse the created snapshot ID from `event.file.name`
2. Read the `entry.created` snapshot from S3 at `entries/<createdSnapshotID>.json`
3. Set `status = "processing"` and `updated_at = now`
4. Generate a new random `processingSnapshotID`
5. Write a new independent snapshot to S3 at `entries/<processingSnapshotID>.json`
6. Update the entry status to `processing` in a transaction that also inserts an outbox record with `event_type = "entry.processing"` in the full `event_data` format

The S3 write SHALL occur before the database transaction. If the S3 write fails, the database transaction SHALL NOT be executed. The SQS message `file` fields SHALL point to the new processing snapshot.

#### Scenario: Status updated to processing with independent S3 snapshot
- **WHEN** the SQS message is sent successfully
- **THEN** a new file `entries/<processingSnapshotID>.json` SHALL exist in S3 with `status = "processing"`
- **AND** the original created snapshot SHALL remain unchanged
- **AND** the entry status in the database SHALL be `processing`
- **AND** a new outbox row with `event_type = "entry.processing"` SHALL be inserted with the full `event_data` format

#### Scenario: S3 write failure aborts DB transaction
- **WHEN** the S3 write of the processing snapshot fails
- **THEN** the database status update and outbox insert SHALL NOT occur

### Requirement: Duplicate processing is rejected with a business error
If the entry status is not `created` when the publisher attempts to process it, the publisher SHALL return a business error `GMS-002` (ErrEntryAlreadyProcessing). The Kafka offset SHALL be committed (the event was already handled).

#### Scenario: Entry already processing
- **WHEN** an `entry.created` Kafka event is consumed but the entry status is already `processing`
- **THEN** the publisher SHALL return business error `GMS-002` and commit the Kafka offset without sending a new SQS message

### Requirement: Kafka offset committed only after full success
The Kafka offset SHALL be committed only after both the SQS send and the DB transaction succeed. If either fails, the offset SHALL NOT be committed and the message will be redelivered.

#### Scenario: Offset not committed on SQS failure
- **WHEN** the SQS send returns an error
- **THEN** the Kafka offset SHALL NOT be committed

#### Scenario: Offset not committed on DB failure
- **WHEN** the DB transaction fails after a successful SQS send
- **THEN** the Kafka offset SHALL NOT be committed

### Requirement: EventPublisher port exists
The project SHALL define an `EventPublisher` interface in `internal/port/event_publisher.go` with a single method `Publish(ctx context.Context, body string) error`.

#### Scenario: SQS adapter implements EventPublisher
- **WHEN** the SQS adapter is instantiated
- **THEN** it SHALL satisfy the `EventPublisher` interface at compile time

### Requirement: Publisher is configurable via environment variables
The Entry Processor binary SHALL read configuration from environment variables: `KAFKA_BROKERS` (required), `KAFKA_TOPIC` (default `generic-microservice`), `KAFKA_GROUP_ID` (default `entry-processor`), `SQS_INPUT_QUEUE_URL` (required), `S3_BUCKET` (required), `AWS_REGION` (optional), `S3_ENDPOINT` (optional).

#### Scenario: Missing required config causes startup failure
- **WHEN** `SQS_INPUT_QUEUE_URL`, `KAFKA_BROKERS`, or `S3_BUCKET` is empty
- **THEN** the binary SHALL exit with a non-zero status and a descriptive error message

### Requirement: input-generic-microservice SQS queue is provisioned
The Terraform configuration SHALL provision an `aws_sqs_queue` resource named `input-generic-microservice` in addition to the existing `generic-microservice` queue.

#### Scenario: Queue created on apply
- **WHEN** `terraform apply` is run against LocalStack
- **THEN** the SQS queue `input-generic-microservice` SHALL exist
