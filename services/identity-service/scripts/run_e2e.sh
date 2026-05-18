#!/bin/bash

# run_e2e.sh
# Script to run End-to-End tests for Identity Service using isolated Docker Compose

set -e

echo "====================================="
echo "   Identity Service E2E Test Suite   "
echo "====================================="

# Navigate to the service directory
cd "$(dirname "$0")/.."

echo "[1/3] Building Test Environment..."
docker compose -f docker-compose.test.yml build

echo "[2/3] Starting Infrastructure & Services..."
docker compose -f docker-compose.test.yml up -d --wait

echo "[3/3] Running E2E Tests..."
set +e
docker compose -f docker-compose.test.yml run --rm e2e-runner
EXIT_CODE=$?
set -e

echo "[Cleanup] Tearing down environment..."
docker compose -f docker-compose.test.yml down -v

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "\n✅ E2E Test Suite Completed Successfully!"
else
    echo -e "\n❌ E2E Test Suite Failed!"
    exit $EXIT_CODE
fi
