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

echo "Using database configuration:"
echo "Host: $DB_HOST"
echo "Port: $DB_PORT"
echo "User: $DB_USER"
echo "Database: $DB_NAME"

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

# Create test database with debug output
echo "Creating test database..."
export PGPASSWORD=$DB_PASSWORD
psql -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<-EOSQL
    SELECT current_user, current_database();
    DROP DATABASE IF EXISTS wrale_test;
    CREATE DATABASE wrale_test;
    GRANT ALL PRIVILEGES ON DATABASE wrale_test TO $DB_USER;
EOSQL

# Test connection to new database
echo "Testing connection to wrale_test database..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d wrale_test -c "SELECT current_user, current_database();"

echo "Test database created successfully"