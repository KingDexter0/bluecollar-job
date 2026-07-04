#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${ENV_FILE:-.env.staging}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
SMOKE_BASE_URL="${SMOKE_BASE_URL:-http://localhost:8081}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE. Copy .env.staging.example and fill real staging values." >&2
  exit 1
fi

echo "Pulling latest code..."
git pull --ff-only

echo "Building staging containers..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build

echo "Running migrations..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" run --rm api /app/migrate up

echo "Starting services..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

echo "Waiting for API readiness..."
for attempt in {1..30}; do
  if curl -fsS "$SMOKE_BASE_URL/ready" >/dev/null; then
    break
  fi
  sleep 2
done

echo "Running smoke test..."
./scripts/smoke-test.sh "$SMOKE_BASE_URL"

echo "Staging deployment complete."
