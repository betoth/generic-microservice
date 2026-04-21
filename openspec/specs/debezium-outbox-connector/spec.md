### Requirement: Connector configuration file exists
The project SHALL include `debezium/outbox-connector.json` with a valid Debezium Postgres connector configuration targeting the `outbox` table using the Outbox Event Router SMT.

#### Scenario: Connector config is present
- **WHEN** the project is checked out
- **THEN** `debezium/outbox-connector.json` SHALL exist with connector class `io.debezium.connector.postgresql.PostgresConnector` and transform `io.debezium.transforms.outbox.EventRouter`

### Requirement: Connector captures only the outbox table
The connector SHALL be scoped exclusively to `public.outbox` — no other tables SHALL be captured.

#### Scenario: Table include list is set
- **WHEN** the connector is registered
- **THEN** `table.include.list` SHALL equal `public.outbox`

### Requirement: Events are routed to topics by the topic column
The Outbox Event Router SMT SHALL route each outbox insert to the Kafka topic named by the `topic` column value. All current events SHALL be routed to `generic-microservice`.

#### Scenario: entry.created event routing
- **WHEN** an outbox row with `topic = generic-microservice` is inserted
- **THEN** Debezium SHALL publish the event to the Kafka topic `generic-microservice`

### Requirement: Kafka message value is event_data content
The connector SHALL set `table.field.event.payload = event_data` so that Debezium publishes only the content of the `event_data` column as the Kafka message value — the summarized partial event. Consumers that need the full entry fetch from S3 using `file.path`.

#### Scenario: Kafka message contains summarized event
- **WHEN** Debezium captures an outbox insert
- **THEN** the Kafka message value SHALL be the content of the `event_data` column (not the full row)

### Requirement: Message key is the entry ID
The Kafka message key SHALL be the `business_key` column value (entry UUID), ensuring all events for the same entry land on the same partition.

#### Scenario: Message key set to business_key
- **WHEN** Debezium publishes an outbox event
- **THEN** the Kafka message key SHALL equal the `business_key` value of that outbox row

### Requirement: Connector can be registered via make target
The Makefile SHALL expose a `connector-register` target that registers (or updates) the connector via the Kafka Connect REST API using `PUT /connectors/<name>/config`. The operation SHALL be idempotent.

#### Scenario: Successful registration
- **WHEN** `make connector-register` is run and Kafka Connect is available at port 8083
- **THEN** the connector SHALL be registered and respond with HTTP 200 or 201

#### Scenario: Idempotent re-registration
- **WHEN** `make connector-register` is run a second time
- **THEN** the connector config SHALL be updated without error

### Requirement: Connector status can be checked via make target
The Makefile SHALL expose a `connector-status` target that queries the Kafka Connect REST API and prints the connector status.

#### Scenario: Status check
- **WHEN** `make connector-status` is run
- **THEN** the connector state SHALL be printed (e.g. `RUNNING`, `FAILED`, `PAUSED`)
