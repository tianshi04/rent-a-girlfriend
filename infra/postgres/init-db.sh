#!/bin/bash
set -e

# Array of databases to create
databases=(
    "booking_db"
    "dispute_service"
    "finance_service"
    "identity_service"
    "interaction_service"
    "notification_db"
    "profile_service"
)

echo "Initializing databases..."

for db in "${databases[@]}"; do
    echo "Checking/creating database: $db"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        SELECT 'CREATE DATABASE $db OWNER postgres'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db')\gexec
EOSQL
done

echo "Databases initialized successfully!"
