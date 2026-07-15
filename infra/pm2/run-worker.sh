#!/usr/bin/env bash
# Обёртка для pm2: подхватывает .env и запускает worker (см. ../../ecosystem.config.js).
set -Eeuo pipefail
ROOT="/var/www/student-leader-portal"
set -a
. "$ROOT/.env"
set +a
exec "$ROOT/backend/bin/worker"
