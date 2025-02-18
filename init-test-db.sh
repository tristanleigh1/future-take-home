#!/bin/bash
set -e

echo "Creating test database..."
psql -v ON_ERROR_STOP=1 --dbname future <<-EOSQL
    CREATE DATABASE future_test;
EOSQL
echo "Test database created." 
