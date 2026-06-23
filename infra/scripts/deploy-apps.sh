#!/bin/bash
set -euo pipefail

echo "Deploying application microservices..."

# Deploy Interaction Service
echo "Deploying interaction-service..."
helm upgrade --install interaction-service services/interaction-service/deployments/helm \
  -f services/interaction-service/deployments/helm/values.dev.yaml \
  -f infra/k8s/envs/dev/interaction-values.dev.yaml \
  --set config.KAFKA_BROKERS="kafka-broker-headless.kafka.svc.cluster.local:9092" \
  --set secrets.DATABASE_URL="postgres://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db?sslmode=disable" \
  -n interaction-service

# Deploy Profile Service
echo "Deploying profile-service..."
helm upgrade --install profile-service services/profile-service/deployments/helm \
  -f services/profile-service/deployments/helm/values.dev.yaml \
  -f infra/k8s/envs/dev/profile-values.dev.yaml \
  --set secrets.DB_HOST="postgres-postgresql.postgres.svc.cluster.local" \
  --set secrets.DB_PORT="5432" \
  --set secrets.DB_USER="postgres" \
  --set secrets.DB_PASSWORD="mysecretpassword" \
  --set secrets.DB_NAME="shared_dev_db" \
  --set config.KAFKA_BROKERS="kafka-broker-headless.kafka.svc.cluster.local:9092" \
  --set config.S3_ENDPOINT_URL="http://minio.minio.svc.cluster.local:9000" \
  --set secrets.S3_ACCESS_KEY_ID="minio_admin" \
  --set secrets.S3_SECRET_ACCESS_KEY="minio_secure_password" \
  -n profile-service

echo "Services deployed successfully!"
