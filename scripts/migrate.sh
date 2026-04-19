#!/bin/sh
# ============================================================
# Script untuk menjalankan database migration
# Requirement: golang-migrate CLI sudah terinstall
# Install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
# ============================================================

set -e

# Load .env jika ada
if [ -f ".env" ]; then
  export $(grep -v '^#' .env | xargs)
fi

DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

case "$1" in
  up)
    echo "Running migrations UP..."
    migrate -path ./migrations -database "$DB_URL" up
    ;;
  down)
    echo "Rolling back 1 migration..."
    migrate -path ./migrations -database "$DB_URL" down 1
    ;;
  version)
    migrate -path ./migrations -database "$DB_URL" version
    ;;
  force)
    echo "Forcing version $2..."
    migrate -path ./migrations -database "$DB_URL" force "$2"
    ;;
  *)
    echo "Usage: $0 {up|down|version|force <version>}"
    exit 1
    ;;
esac
