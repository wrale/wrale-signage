#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE wrale_test;
    GRANT ALL PRIVILEGES ON DATABASE wrale_test TO postgres;
EOSQL
