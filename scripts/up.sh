#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="dev.env"
SPIN='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'

green='\033[0;32m'
red='\033[0;31m'
reset='\033[0m'

spinner() {
  local pid=$1
  local label=$2
  local i=0
  while kill -0 "$pid" 2>/dev/null; do
    local c="${SPIN:$((i % ${#SPIN})):1}"
    printf "\r  %s  %s" "$c" "$label"
    sleep 0.1
    ((i++)) || true
  done
}

step() {
  local label=$1
  shift
  "$@" >"$LOGFILE" 2>&1 &
  local pid=$!
  spinner "$pid" "$label"
  if wait "$pid"; then
    printf "\r  ${green}✓${reset}  %s\n" "$label"
  else
    printf "\r  ${red}✗${reset}  %s (see $LOGFILE)\n" "$label"
    exit 1
  fi
}

LOGFILE=$(mktemp /tmp/generic-microservice-up.XXXXXX)
trap 'rm -f "$LOGFILE"' EXIT

source_env() {
  set -a
  # shellcheck disable=SC1090
  source <(grep -v '^#' "$ENV_FILE")
  set +a
}

source_env

echo ""
echo "  generic-microservice"
echo ""

step "Starting services" \
  docker-compose --env-file "$ENV_FILE" up -d postgres localstack kafka kafka-connect kafka-ui filestash

step "Waiting for LocalStack" \
  bash -c 'until curl -s http://localhost:4566/_localstack/health 2>/dev/null | grep -qE "\"s3\": \"(running|available)\""; do sleep 2; done'

step "Provisioning infrastructure" \
  bash -c "env $(grep -v '^#' "$ENV_FILE" | xargs) terraform -chdir=infra init -upgrade -no-color && \
           env $(grep -v '^#' "$ENV_FILE" | xargs) terraform -chdir=infra apply -auto-approve -no-color"

step "Waiting for Kafka Connect" \
  bash -c 'until curl -s http://localhost:8083/connectors > /dev/null 2>&1; do sleep 2; done'

step "Applying migrations" \
  migrate -path migrations -database "$DB_CONN_STRING" up

step "Registering Debezium connector" \
  bash -c "env $(grep -v '^#' "$ENV_FILE" | xargs) envsubst '\${POSTGRES_HOST} \${POSTGRES_PORT} \${POSTGRES_USER} \${POSTGRES_PASSWORD} \${POSTGRES_DB}' < debezium/outbox-connector.json | \
    curl -sf -X PUT http://localhost:8083/connectors/outbox-connector/config \
    -H 'Content-Type: application/json' -d @-"

echo ""
echo "  Starting application..."
echo ""

trap 'kill 0' INT TERM
env $(grep -v '^#' "$ENV_FILE" | xargs) go run ./cmd/entry-create/ &
env $(grep -v '^#' "$ENV_FILE" | xargs) go run ./cmd/entry-processor/ &
env $(grep -v '^#' "$ENV_FILE" | xargs) go run ./cmd/entry-publisher/ &
wait
