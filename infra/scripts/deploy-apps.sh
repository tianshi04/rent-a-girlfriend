#!/bin/bash
set -euo pipefail

echo "Deploying application microservices..."

# Deploy Interaction Service
echo "Deploying interaction-service..."
helm upgrade --install interaction-service services/interaction-service/deployments/helm \
  -f services/interaction-service/deployments/helm/values.dev.yaml \
  --set secrets.DATABASE_URL="postgres://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db?sslmode=disable" \
  -n interaction-service

echo "Services deployed successfully!"
