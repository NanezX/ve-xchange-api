#!/bin/bash
# Creates the vexchange_db database if it does not already exist.
# Runs as the postgres superuser during container first-start.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE vexchange_db'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'vexchange_db')\gexec

    GRANT ALL PRIVILEGES ON DATABASE vexchange_db TO "$POSTGRES_USER";
EOSQL
