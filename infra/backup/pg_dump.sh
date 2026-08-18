#!/usr/bin/env bash
# Ежедневный pg_dump контейнера slc-postgres. Не печатает секреты.
# Cron/timer не ставит — только ручной запуск или внешнее расписание.
set -euo pipefail

ROOT="${ROOT:-/var/www/student-leader-portal}"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"
[ -f "$ENV_FILE" ] || { echo "нет $ENV_FILE" >&2; exit 1; }

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

OUTDIR="${BACKUP_DIR:-/var/backups/student-leader-portal}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DB="${POSTGRES_DB:-student_leader}"
USER="${POSTGRES_USER:-student_leader}"
CONTAINER="${POSTGRES_CONTAINER:-slc-postgres}"

mkdir -p "$OUTDIR"
chmod 700 "$OUTDIR"
FILE="${OUTDIR}/pg_${DB}_${STAMP}.sql.gz"

docker exec -e PGPASSWORD="${POSTGRES_PASSWORD}" "$CONTAINER" \
  pg_dump -U "$USER" -d "$DB" --no-owner --format=plain \
  | gzip -9 > "$FILE"
chmod 600 "$FILE"

if [ -n "${BACKUP_GPG_RECIPIENT:-}" ]; then
  gpg --batch --yes --encrypt --recipient "$BACKUP_GPG_RECIPIENT" --output "${FILE}.gpg" "$FILE"
  shred -u "$FILE" 2>/dev/null || rm -f "$FILE"
  FILE="${FILE}.gpg"
  chmod 600 "$FILE"
fi

find "$OUTDIR" -type f \( -name 'pg_*.sql.gz' -o -name 'pg_*.sql.gz.gpg' \) -mtime "+${KEEP_DAYS}" -delete

echo "backup written: $(basename "$FILE") bytes=$(wc -c < "$FILE")"
