#!/bin/bash

# run_e2e.sh
# Script to run End-to-End tests for Identity Service

set -e

echo "====================================="
echo "   Identity Service E2E Test Suite   "
echo "====================================="

# Navigate to the service directory
cd "$(dirname "$0")/.."

# Export necessary environment variables for testing
export APP_ENV=test
export ENABLE_TEST_ROUTES=true
export GIN_MODE=release
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=identity_db
export DB_SSLMODE=disable
export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
export REDIS_URL="redis://localhost:6379"
export KAFKA_BROKERS="localhost:9092,localhost:9093,localhost:9094"
export OUTBOX_POLLING_INTERVAL_MS=500
export SERVER_PORT=8081
export GRPC_PORT=50051

echo "[1/4] Starting dependencies via docker-compose..."
# Start infrastructure (Postgres, Redis, Kafka)
docker-compose -f ../../infrastructure/kafka/docker-compose.yml up -d
# If the service itself has a docker-compose for db/redis, we can also use it
# Wait for dependencies
echo "Waiting for PostgreSQL..."
while ! nc -z localhost 5432; do   
  sleep 1 
done

echo "Waiting for Redis..."
while ! nc -z localhost 6379; do   
  sleep 1 
done

echo "Waiting for Kafka..."
while ! nc -z localhost 9092; do   
  sleep 1 
done

echo "[2/4] Starting Identity Service in background..."
go run cmd/server/main.go > e2e_server.log 2>&1 &
SERVER_PID=$!

echo "Waiting for Identity Service HTTP to be ready..."
until $(curl --output /dev/null --silent --head --fail http://localhost:${SERVER_PORT}/health); do
    printf '.'
    sleep 1
done
echo "Service is up!"

echo "[3/4] Running E2E Tests..."
# We run the tests in e2e folder
go test -v ./tests/e2e/...

# Cleanup
echo "[4/4] Tearing down environment..."
kill $SERVER_PID
echo "Identity Service stopped."
# Optionally, stop docker-compose if we want fully clean runs
# docker-compose -f ../../infrastructure/kafka/docker-compose.yml down

echo "E2E Test Suite Completed Successfully!"
