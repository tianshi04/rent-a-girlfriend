#!/bin/bash
set -euo pipefail

deploy_interaction() {
  echo "Deploying interaction-service..."
  helm upgrade --install interaction-service services/interaction-service/deployments/helm \
    -f services/interaction-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/interaction-values.yaml \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set secrets.DATABASE_URL="postgres://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db?sslmode=disable" \
    --create-namespace \
    -n interaction-service
}

deploy_profile() {
  echo "Deploying profile-service..."
  helm upgrade --install profile-service services/profile-service/deployments/helm \
    -f services/profile-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/profile-values.yaml \
    --set secrets.DATABASE_URL="postgresql+asyncpg://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db" \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set config.S3_ENDPOINT_URL="http://minio.minio.svc.cluster.local:9000" \
    --set secrets.S3_ACCESS_KEY_ID="minio_admin" \
    --set secrets.S3_SECRET_ACCESS_KEY="minio_secure_password" \
    --create-namespace \
    -n profile-service
}

deploy_booking() {
  echo "Deploying booking-service..."
  helm upgrade --install booking-service services/booking-service/deployments/helm \
    -f services/booking-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/booking-values.yaml \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set secrets.DATABASE_URL="postgres://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db?sslmode=disable" \
    --create-namespace \
    -n booking-service
}

deploy_identity() {
  echo "Deploying identity-service..."
  helm upgrade --install identity-service services/identity-service/deployments/helm \
    -f services/identity-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/identity-values.yaml \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set secrets.DB_URL="postgres://postgres:mysecretpassword@postgres-postgresql.postgres.svc.cluster.local:5432/shared_dev_db?sslmode=disable" \
    --set secrets.REDIS_URL="redis://redis-master.redis.svc.cluster.local:6379" \
    --create-namespace \
    -n identity-service
}

deploy_dispute() {
  echo "Deploying dispute-service..."
  helm upgrade --install dispute-service services/dispute-service/deployments/helm \
    -f services/dispute-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/dispute-values.yaml \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set config.DB_HOST="postgres-postgresql.postgres.svc.cluster.local" \
    --set-string config.DB_PORT="5432" \
    --set config.DB_NAME="shared_dev_db" \
    --set secrets.DB_USER="postgres" \
    --set secrets.DB_PASSWORD="mysecretpassword" \
    --set-string config.USE_MOCKS="false" \
    --create-namespace \
    -n dispute-service
}

deploy_finance() {
  echo "Deploying finance-service..."
  helm upgrade --install finance-service services/finance-service/deployments/helm \
    -f services/finance-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/finance-values.yaml \
    --set config.KAFKA_BROKERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set config.DB_HOST="postgres-postgresql.postgres.svc.cluster.local" \
    --set-string config.DB_PORT="5432" \
    --set config.DB_NAME="shared_dev_db" \
    --set secrets.DB_USER="postgres" \
    --set secrets.DB_PASSWORD="mysecretpassword" \
    --create-namespace \
    -n finance-service
}

deploy_notification() {
  echo "Deploying notification-service..."
  helm upgrade --install notification-service services/notification-service/deployments/helm \
    -f services/notification-service/deployments/helm/values.dev.yaml \
    -f infra/k8s/base/helm-values/notification-values.yaml \
    --set config.KAFKA_BOOTSTRAP_SERVERS="kafka-headless.kafka.svc.cluster.local:9092" \
    --set config.DB_HOST="postgres-postgresql.postgres.svc.cluster.local" \
    --set-string config.DB_PORT="5432" \
    --set config.DB_NAME="shared_dev_db" \
    --set config.DB_USERNAME="postgres" \
    --set secrets.DB_PASSWORD="mysecretpassword" \
    --set config.REDIS_HOST="redis-master.redis.svc.cluster.local" \
    --set-string config.REDIS_PORT="6379" \
    --create-namespace \
    -n notification-service
}

SERVICE=${1:-all}

case "$SERVICE" in
  interaction-service|interaction)
    deploy_interaction
    ;;
  profile-service|profile)
    deploy_profile
    ;;
  booking-service|booking)
    deploy_booking
    ;;
  identity-service|identity)
    deploy_identity
    ;;
  dispute-service|dispute)
    deploy_dispute
    ;;
  finance-service|finance)
    deploy_finance
    ;;
  notification-service|notification)
    deploy_notification
    ;;
  all)
    echo "Deploying all microservices..."
    deploy_interaction
    deploy_profile
    deploy_booking
    deploy_identity
    deploy_dispute
    deploy_finance
    deploy_notification
    ;;
  *)
    echo "Error: Unknown service '$SERVICE'"
    echo "Usage: $0 [service-name|all]"
    exit 1
    ;;
esac

echo "Services deployed successfully!"
