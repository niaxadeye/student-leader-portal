#!/usr/bin/env bash
# Восстановление pg_dump в контейнер slc-postgres.
# ОПАСНО: перезаписывает живую БД. Сначала остановите API/worker.
# Использование: ./infra/backup/pg_restore.sh /var/backups/student-leader-portal/pg_....sql.gz
set -euo pipefail

DUMP="${1:-}"
[ -n "$DUMP" ] && [ -f "$DUMP" ] || { echo "usage: $0 <dump.sql.gz|.gpg>" >&2; exit 1; }

ROOT="${ROOT:-/var/www/student-leader-portal}"
set -a
# shellcheck disable=SC1091
. "$ROOT/.env"
set +a

DB="${POSTGRES_DB:-student_leader}"
USER="${POSTGRES_USER:-student_leader}"
CONTAINER="${POSTGRES_CONTAINER:-slc-postgres}"

echo "This will DROP and recreate database ${DB} in ${CONTAINER}." >&2
echo "API must be stopped. Type YES to continue." >&2
read -r confirm
[ "$confirm" = "YES" ] || { echo "aborted"; exit 1; }

work="$DUMP"
cleanup() { [ -n "${tmp:-}" ] && rm -f "$tmp"; }
trap cleanup EXIT

if [[ "$DUMP" == *.gpg ]]; then
  tmp="$(mktemp)"
  gpg --batch --decrypt --output "$tmp" "$DUMP"
  work="$tmp"
fi

docker exec -e PGPASSWORD="${POSTGRES_PASSWORD}" "$CONTAINER" \
  psql -U "$USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${DB}' AND pid <> pg_backend_pid();" \
  -c "DROP DATABASE IF EXISTS ${DB};" \
  -c "CREATE DATABASE ${DB} OWNER ${USER};"

if [[ "$work" == *.gz ]]; then
  gzip -dc "$work" | docker exec -i -e PGPASSWORD="${POSTGRES_PASSWORD}" "$CONTAINER" \
    psql -U "$USER" -d "$DB" -v ON_ERROR_STOP=1
else
  docker exec -i -e PGPASSWORD="${POSTGRES_PASSWORD}" "$CONTAINER" \
    psql -U "$USER" -d "$DB" -v ON_ERROR_STOP=1 < "$work"
fi

echo "restore finished"
