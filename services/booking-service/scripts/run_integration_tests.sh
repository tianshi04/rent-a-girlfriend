#!/usr/bin/env bash
# run_integration_tests.sh — Run Integration Tests on Unix
# Starts isolated Docker dependencies, runs tests, and tears them down.

set -e # Exit immediately on error

scriptDir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
serviceRoot="$(cd "$scriptDir/.." && pwd)"
dockerFile="$serviceRoot/docker-compose.test.yml"

# Always execute cleanup on script exit
cleanup() {
  echo "==> [test-runner] Tearing down Docker dependencies and cleaning up volumes..."
  docker compose -f "$dockerFile" down -v
}
trap cleanup EXIT

echo "==> [test-runner] Starting Docker dependencies..."
docker compose -f "$dockerFile" up -d --wait

echo "==> [test-runner] Pre-creating Kafka topics..."
docker exec booking-kafka-test /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic booking-events-test --partitions 1 --replication-factor 1
docker exec booking-kafka-test /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic finance-events-test --partitions 1 --replication-factor 1
docker exec booking-kafka-test /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic interaction-events-test --partitions 1 --replication-factor 1
docker exec booking-kafka-test /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic dispute-events-test --partitions 1 --replication-factor 1

# Inject Environment Variables pointing to isolated test ports
export GIN_MODE="test"
export DATABASE_URL="postgres://postgres:postgres@localhost:5433/booking_test_db?sslmode=disable"
export DB_HOST="localhost"
export DB_PORT="5433"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="booking_test_db"
export DB_SSLMODE="disable"
export KAFKA_BROKERS="localhost:9094"
export KAFKA_TOPIC_BOOKING="booking-events-test"
export KAFKA_TOPIC_FINANCE="finance-events-test"
export KAFKA_TOPIC_INTERACTION="interaction-events-test"
export KAFKA_TOPIC_DISPUTE="dispute-events-test"
export PROFILE_SERVICE_ADDR="localhost:59991"
export FINANCE_SERVICE_ADDR="localhost:59992"

echo "==> [test-runner] Executing Integration tests..."
go test -v ./tests/integration/... -count=1

echo "==> [test-runner] All tests passed!"
