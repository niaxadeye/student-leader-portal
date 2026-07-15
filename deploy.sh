#!/usr/bin/env bash
# Деплой портала из git-репозитория (workflow: локальная разработка → push → deploy на сервере).
# Запуск на сервере из корня проекта:  ./deploy.sh
# Идемпотентно; при ошибке любого шага прерывается (set -e).
set -Eeuo pipefail

ROOT="/var/www/student-leader-portal"
BRANCH="${DEPLOY_BRANCH:-master}"
cd "$ROOT"

log()  { printf '\n\033[36m==> %s\033[0m\n' "$*"; }
fail() { printf '\n\033[31m!!! %s\033[0m\n' "$*" >&2; exit 1; }

[ -f .env ] || fail ".env не найден в $ROOT (секреты не в git — создать вручную из .env.example)"

# 1. Обновление кода
log "git fetch + pull ($BRANCH)"
git fetch --prune origin
git checkout "$BRANCH"
BEFORE=$(git rev-parse HEAD)
git pull --ff-only origin "$BRANCH"
AFTER=$(git rev-parse HEAD)
if [ "$BEFORE" = "$AFTER" ]; then
  log "изменений нет ($AFTER) — пересобираю на всякий случай"
else
  log "обновлено: $BEFORE → $AFTER"
  git --no-pager log --oneline "$BEFORE..$AFTER" | sed 's/^/    /'
fi

# 2. Сборка бэкенда (api, worker, admin)
log "сборка backend"
cd "$ROOT/backend"
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
go build -o bin/admin ./cmd/admin

# 3. Миграции БД (forward-only, идемпотентны)
log "применение миграций"
set -a; . "$ROOT/.env"; set +a
./bin/admin migrate

# 4. Сборка фронтенда
log "сборка frontend"
cd "$ROOT/frontend"
if [ package-lock.json -nt node_modules/.package-lock.json ] 2>/dev/null || [ ! -d node_modules ]; then
  npm ci
fi
npm run build

# 5. Рестарт сервисов
log "рестарт сервисов"
sudo systemctl restart eazytech-api eazytech-worker
sleep 2
systemctl is-active --quiet eazytech-api    || fail "eazytech-api не поднялся (journalctl -u eazytech-api)"
systemctl is-active --quiet eazytech-worker || fail "eazytech-worker не поднялся (journalctl -u eazytech-worker)"

# 6. Health-check (роуты вне версионированного префикса, см. router.go)
log "health-check"
PORT="${HTTP_PORT:-8080}"
for i in $(seq 1 10); do
  if curl -fsS "http://127.0.0.1:${PORT}/health/ready" >/dev/null 2>&1; then
    log "OK: API готов (/health/ready)"
    break
  fi
  [ "$i" = 10 ] && fail "API не ответил на /health/ready за 10 попыток (journalctl -u eazytech-api)"
  sleep 1
done

log "деплой завершён успешно ✔  ($AFTER)"
