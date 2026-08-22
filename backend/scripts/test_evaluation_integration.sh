#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER="slc-evaluation-test-$$"
DB_NAME="evaluation_test"
DB_USER="evaluation_test"
DB_PASSWORD="evaluation_test_password"

cleanup() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

docker run -d --rm \
  --name "$CONTAINER" \
  -e POSTGRES_DB="$DB_NAME" \
  -e POSTGRES_USER="$DB_USER" \
  -e POSTGRES_PASSWORD="$DB_PASSWORD" \
  -p 127.0.0.1::5432 \
  postgres:16-alpine >/dev/null

for _ in $(seq 1 30); do
  if docker exec "$CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null

BINDING="$(docker port "$CONTAINER" 5432/tcp)"
DB_PORT="${BINDING##*:}"
export POSTGRES_HOST="127.0.0.1"
export POSTGRES_PORT="$DB_PORT"
export POSTGRES_DB="$DB_NAME"
export POSTGRES_USER="$DB_USER"
export POSTGRES_PASSWORD="$DB_PASSWORD"
export POSTGRES_SSLMODE="disable"
export JWT_ACCESS_SECRET="integration-access-secret-at-least-32-bytes"
export JWT_REFRESH_SECRET="integration-refresh-secret-at-least-32-bytes"
export PARTICIPANT_QR_SECRET="integration-participant-secret-at-least-32-bytes"
export TEST_DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@127.0.0.1:$DB_PORT/$DB_NAME?sslmode=disable"

cd "$ROOT"
env GOCACHE=/tmp/student-leader-go-cache go run ./cmd/admin migrate >/dev/null
env GOCACHE=/tmp/student-leader-go-cache go test -tags=integration -count=1 \
	./internal/modules/evaluation ./internal/modules/auth ./internal/modules/submissions \
	-run '^(TestApplyScoreMutationIntegration|TestRotateRefreshAtomicallyIntegration|TestSubmissionFileBindingIntegration)$'
