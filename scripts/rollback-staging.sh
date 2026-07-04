#!/usr/bin/env bash
set -euo pipefail

TARGET_REF="${1:-HEAD~1}"
ENV_FILE="${ENV_FILE:-.env.staging}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8081/health}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE." >&2
  exit 1
fi

echo "Rolling back to $TARGET_REF..."
git fetch --all --prune
git checkout "$TARGET_REF"

echo "Rebuilding and restarting services..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

echo "Checking health..."
curl -fsS "$HEALTH_URL" >/dev/null
echo "Rollback complete."
