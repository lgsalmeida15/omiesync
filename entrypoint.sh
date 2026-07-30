#!/bin/sh
set -e

# Garante sslmode=disable para o migrate CLI (pgx da app já trata internamente)
case "$DATABASE_URL" in
  *sslmode=*) MIGRATE_URL="$DATABASE_URL" ;;
  *\?*)       MIGRATE_URL="${DATABASE_URL}&sslmode=disable" ;;
  *)          MIGRATE_URL="${DATABASE_URL}?sslmode=disable" ;;
esac

echo "Rodando migrations..."
migrate -path /app/db/migrations -database "$MIGRATE_URL" up
echo "Migrations aplicadas."

exec ./omie-sync-api
