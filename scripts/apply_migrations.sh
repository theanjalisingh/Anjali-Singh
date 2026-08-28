#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${ROOT_DIR}/migrations"

DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/future_environs?sslmode=disable}"

echo "Applying migrations to ${DATABASE_URL}"

for file in \
  "${MIGRATIONS_DIR}/001_identity_schema.sql" \
  "${MIGRATIONS_DIR}/002_identity_tables.sql" \
  "${MIGRATIONS_DIR}/003_identity_procedures.sql" \
  "${MIGRATIONS_DIR}/004_identity_seed.sql"
do
  echo "-> $(basename "${file}")"
  psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${file}"
done

echo "Migrations applied successfully."
