#!/bin/bash
set -e

# Default values
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${POSTGRES_USER:-postgres}
DB_PASSWORD=${POSTGRES_PASSWORD:-postgres}
DB_NAME=${POSTGRES_DB:-postgres}
MAX_RETRIES=${MAX_RETRIES:-5}
RETRY_INTERVAL=${RETRY_INTERVAL:-5}

# Function to check database connection
check_connection() {
    PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" >/dev/null 2>&1
}

# Wait for database to be ready
echo "Waiting for PostgreSQL to be ready..."
retries=0
until check_connection || [ $retries -eq $MAX_RETRIES ]; do
    echo "Waiting for PostgreSQL to become available... ($((retries+1))/$MAX_RETRIES)"
    sleep $RETRY_INTERVAL
    retries=$((retries+1))
done

if [ $retries -eq $MAX_RETRIES ]; then
    echo "Error: Could not connect to PostgreSQL after $MAX_RETRIES attempts"
    exit 1
fi

echo "PostgreSQL is ready"

# Create test database
echo "Creating test database..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<-EOSQL
    DROP DATABASE IF EXISTS wrale_test;
    CREATE DATABASE wrale_test;
    GRANT ALL PRIVILEGES ON DATABASE wrale_test TO $DB_USER;
EOSQL

echo "Test database created successfully"