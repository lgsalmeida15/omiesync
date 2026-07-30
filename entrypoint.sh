#!/bin/sh
set -e

echo "Rodando migrations..."
migrate -path /app/db/migrations -database "$DATABASE_URL" up
echo "Migrations aplicadas."

exec ./omie-sync-api
