#!/bin/sh
set -e

echo "Installing postgresql-client..."
apk add --no-cache postgresql-client

# Function to poll HTTP endpoint
wait_for_http() {
  local url=$1
  local name=$2
  echo "Waiting for $name to be healthy at $url..."
  until wget -qO- "$url" > /dev/null 2>&1; do
    sleep 2
  done
  echo "✅ $name is ready!"
}

# Function to poll TCP port
wait_for_port() {
  local host=$1
  local port=$2
  local name=$3
  echo "Waiting for $name port at $host:$port..."
  until nc -z -w 1 "$host" "$port" > /dev/null 2>&1; do
    sleep 2
  done
  echo "✅ $name port is open!"
}

# 1. Wait for infrastructure
wait_for_port postgres 5432 "PostgreSQL"
wait_for_port redis 6379 "Redis"
wait_for_port kafka 9092 "Kafka"

# 2. Wait for services to finish migration and boot up
wait_for_http "http://identity-service:8080/health/ready" "Identity Service"
wait_for_http "http://profile-service:8080/health" "Profile Service"
wait_for_http "http://booking-service:8080/health/ready" "Booking Service"
wait_for_port "interaction-service" 8080 "Interaction Service"
wait_for_http "http://notification-service:8080/actuator/health" "Notification Service"
wait_for_http "http://finance-service:8080/health" "Finance Service"
wait_for_http "http://dispute-service:8082/api/v1/health" "Dispute Service"

echo "All services are up. Seeding database tables..."

# 3. Seed SQL data
echo "Seeding identity_service..."
PGPASSWORD=postgres psql -h postgres -U postgres -d identity_service -f /scripts/seeds/identity_service.sql

echo "Seeding profile_service..."
PGPASSWORD=postgres psql -h postgres -U postgres -d profile_service -f /scripts/seeds/profile_service.sql

echo "Seeding booking_db..."
PGPASSWORD=postgres psql -h postgres -U postgres -d booking_db -f /scripts/seeds/booking_db.sql

echo "Seeding finance_service..."
PGPASSWORD=postgres psql -h postgres -U postgres -d finance_service -f /scripts/seeds/finance_service.sql

echo "Seeding interaction_service..."
PGPASSWORD=postgres psql -h postgres -U postgres -d interaction_service -f /scripts/seeds/interaction_service.sql

echo "Seeding notification_db..."
PGPASSWORD=postgres psql -h postgres -U postgres -d notification_db -f /scripts/seeds/notification_db.sql

echo "Seeding dispute_service..."
PGPASSWORD=postgres psql -h postgres -U postgres -d dispute_service -f /scripts/seeds/dispute_service.sql

echo "✅ All databases seeded successfully!"
