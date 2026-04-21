### Requirement: Each binary has its own config struct
Each Go binary (`cmd/entry-create`, `cmd/entry-processor`, `cmd/entry-publisher`) has a dedicated config struct in `internal/config/`. Configs are loaded from environment variables at startup. Missing required variables cause immediate exit with a descriptive error.

### Requirement: entry-create config carries S3 and DB fields
The `Config` struct in `internal/config/config.go` SHALL include the following fields loaded from environment variables:

| Field          | Env var          | Required | Description                               |
|----------------|------------------|----------|-------------------------------------------|
| `DBConnString` | `DB_CONN_STRING` | yes      | Postgres connection string                |
| `S3Bucket`     | `S3_BUCKET`      | yes      | Name of the S3 bucket for entry snapshots |
| `S3Endpoint`   | `S3_ENDPOINT`    | no       | Override endpoint URL (LocalStack only)   |
| `AWSRegion`    | `AWS_REGION`     | no       | AWS region                                |

#### Scenario: All S3 env vars present
- **WHEN** `S3_BUCKET`, `S3_ENDPOINT`, and `AWS_REGION` are set in the environment
- **THEN** the loaded `Config` SHALL have all three fields populated with those values

#### Scenario: S3Endpoint absent (production)
- **WHEN** `S3_ENDPOINT` is not set
- **THEN** `Config.S3Endpoint` SHALL be an empty string and the S3 client SHALL use the default AWS resolver

### Requirement: entry-processor config carries Kafka, SQS, and S3 fields
The `PublisherConfig` struct in `internal/config/publisher_config.go` SHALL include:

| Field              | Env var               | Required | Description                       |
|--------------------|-----------------------|----------|-----------------------------------|
| `KafkaBrokers`     | `KAFKA_BROKERS`       | yes      | Comma-separated broker addresses  |
| `KafkaTopic`       | `KAFKA_TOPIC`         | no       | Default `generic-microservice`    |
| `KafkaGroupID`     | `KAFKA_GROUP_ID`      | no       | Default `entry-processor`         |
| `SQSInputQueueURL` | `SQS_INPUT_QUEUE_URL` | yes      | SQS input queue URL               |
| `S3Bucket`         | `S3_BUCKET`           | yes      | S3 bucket for snapshots           |
| `AWSRegion`        | `AWS_REGION`          | no       | AWS region                        |
| `S3Endpoint`       | `S3_ENDPOINT`         | no       | LocalStack override               |

### Requirement: entry-publisher config carries SQS, DLQ, DB, and S3 fields
The `EntryPublisherConfig` struct in `internal/config/entry_publisher_config.go` SHALL include:

| Field              | Env var               | Required | Description                       |
|--------------------|-----------------------|----------|-----------------------------------|
| `DBConnString`     | `DB_CONN_STRING`      | yes      | Postgres connection string        |
| `SQSInputQueueURL` | `SQS_INPUT_QUEUE_URL` | yes      | SQS input queue URL               |
| `SQSDLQURL`        | `SQS_DLQ_URL`         | yes      | SQS DLQ URL                       |
| `S3Bucket`         | `S3_BUCKET`           | yes      | S3 bucket for snapshots           |
| `AWSRegion`        | `AWS_REGION`          | no       | Default `us-east-1`               |
| `SQSEndpoint`      | `SQS_ENDPOINT`        | no       | LocalStack override for SQS       |
| `S3Endpoint`       | `S3_ENDPOINT`         | no       | LocalStack override for S3        |
