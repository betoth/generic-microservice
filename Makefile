DB_URL ?= $(shell grep ^DB_CONN_STRING dev.env | cut -d '=' -f2-)

.PHONY: up down run build tidy migrate migrate-down connector-register connector-status infra-up infra-destroy dev-up s3-ls s3-cat entry-processor

up:
	@bash scripts/up.sh

down:
	docker-compose --env-file dev.env down -v

run:
	env $$(cat dev.env | grep -v '^#' | xargs) go run ./cmd/entry-create/

build:
	go build -o bin/entry-create ./cmd/entry-create/
	go build -o bin/entry-processor ./cmd/entry-processor/
	go build -o bin/entry-publisher ./cmd/entry-publisher/

migrate:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

tidy:
	go mod tidy

connector-register:
	env $$(cat dev.env | grep -v '^#' | xargs) envsubst '$${POSTGRES_HOST} $${POSTGRES_PORT} $${POSTGRES_USER} $${POSTGRES_PASSWORD} $${POSTGRES_DB}' < debezium/outbox-connector.json | \
		curl -X PUT http://localhost:8083/connectors/outbox-connector/config \
		-H "Content-Type: application/json" \
		-d @-

connector-status:
	curl -s http://localhost:8083/connectors/outbox-connector/status | python3 -m json.tool

entry-processor:
	env $$(cat dev.env | grep -v '^#' | xargs) go run ./cmd/entry-processor/

s3-ls:
	AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
		aws --endpoint-url=http://localhost:4566 s3 ls s3://generic-microservice/entries/

s3-cat:
	AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
		aws --endpoint-url=http://localhost:4566 s3 cp s3://generic-microservice/entries/$(ID).json -

infra-up:
	env $$(cat dev.env | grep -v '^#' | xargs) terraform -chdir=infra init -upgrade && \
	env $$(cat dev.env | grep -v '^#' | xargs) terraform -chdir=infra apply -auto-approve

infra-destroy:
	env $$(cat dev.env | grep -v '^#' | xargs) terraform -chdir=infra destroy -auto-approve

dev-up:
	docker-compose --env-file dev.env up -d postgres localstack kafka kafka-connect kafka-ui filestash
	@echo "Waiting for LocalStack to be ready..."
	@until curl -s http://localhost:4566/_localstack/health 2>/dev/null | grep -qE '"s3": "(running|available)"'; do sleep 2; done
	@echo "Provisioning infrastructure..."
	@$(MAKE) infra-up
	@echo "Waiting for Kafka Connect to be ready..."
	@until curl -s http://localhost:8083/connectors > /dev/null 2>&1; do sleep 2; done
	@echo "Registering Debezium connector..."
	@$(MAKE) connector-register
