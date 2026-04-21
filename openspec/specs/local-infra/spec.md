### Requirement: Terraform configuration exists
The project SHALL include Terraform files in `infra/` that provision local infrastructure against LocalStack using the `hashicorp/aws` provider.

#### Scenario: Terraform files present
- **WHEN** the project is checked out
- **THEN** `infra/main.tf`, `infra/variables.tf`, and `infra/outputs.tf` SHALL exist

### Requirement: S3 bucket is provisioned
Terraform SHALL declare an `aws_s3_bucket` resource named `generic-microservice` with `force_destroy = true`.

#### Scenario: Bucket created on apply
- **WHEN** `terraform apply` is run against LocalStack
- **THEN** the S3 bucket `generic-microservice` SHALL exist and be accessible at the LocalStack endpoint

### Requirement: Three SQS queues are provisioned
Terraform SHALL declare three `aws_sqs_queue` resources:
- `generic-microservice` — receives Debezium/Kafka events (legacy integration)
- `input-generic-microservice` — receives processing requests from the Entry Processor
- `dlq-generic-microservice` — receives failed messages from the Entry Publisher (DLQ)

#### Scenario: Queues created on apply
- **WHEN** `terraform apply` is run against LocalStack
- **THEN** all three SQS queues SHALL exist and be accessible at the LocalStack endpoint

### Requirement: LocalStack endpoint is configurable
The Terraform AWS provider SHALL target LocalStack via a configurable `localstack_endpoint` variable (default `http://localhost:4566`).

#### Scenario: Provider uses LocalStack endpoint
- **WHEN** `terraform apply` is run with the default variable value
- **THEN** all resources SHALL be created at `http://localhost:4566`

### Requirement: State and provider cache are gitignored
The `infra/terraform.tfstate*` files and `infra/.terraform/` directory SHALL be listed in `.gitignore`. The `infra/.terraform.lock.hcl` file SHALL NOT be gitignored — it pins provider versions and must be committed.

#### Scenario: State files not tracked
- **WHEN** `git status` is run after `terraform apply`
- **THEN** `terraform.tfstate*` and `.terraform/` SHALL not appear as untracked or modified, but `.terraform.lock.hcl` SHALL be tracked

### Requirement: Full local environment can be started via make targets
The Makefile SHALL expose two targets for local development:
- `dev-up`: starts all containers, provisions infrastructure (via `make infra-up`), and registers the Debezium connector — without starting any Go processes
- `up`: delegates to `dev-up` then starts all three Go processes in parallel (`cmd/entry-create`, `cmd/entry-processor`, `cmd/entry-publisher`)

#### Scenario: Infrastructure-only setup
- **WHEN** a developer runs `make dev-up` on a clean machine
- **THEN** all containers SHALL start, the S3 bucket and SQS queues SHALL be created, and the Debezium connector SHALL be registered — with no manual steps required

#### Scenario: Full stack startup
- **WHEN** a developer runs `make up`
- **THEN** all infrastructure SHALL be provisioned (via `dev-up`) and all three Go processes SHALL start concurrently

### Requirement: Infrastructure can be provisioned via make target
The Makefile SHALL expose an `infra-up` target that initializes and applies the Terraform configuration against LocalStack. The operation SHALL be idempotent.

#### Scenario: Successful provisioning
- **WHEN** LocalStack is running and `make infra-up` is run
- **THEN** the S3 bucket and SQS queues SHALL be created (or confirmed as up-to-date) with exit code 0

#### Scenario: Idempotent re-apply
- **WHEN** `make infra-up` is run a second time
- **THEN** no resources SHALL be re-created and the command SHALL succeed
