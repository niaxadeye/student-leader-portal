#!/usr/bin/env bash
# Проверка, что публичный origin отдаёт JSON API, а не HTML-challenge (KillBot).
# Использование: ./infra/ops/check-public-api.sh https://eazytech.ru
set -euo pipefail

BASE="${1:-https://www.eazytech.ru}"
BASE="${BASE%/}"

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

check_json() {
  local path="$1" needle="$2"
  local hdr body ctype
  hdr="$(mktemp)"
  body="$(curl -fsS -D "$hdr" -m 20 "${BASE}${path}" || true)"
  ctype="$(grep -i '^content-type:' "$hdr" | tr -d '\r' || true)"
  rm -f "$hdr"
  printf '%s\n' "$ctype" | grep -qi 'application/json' || fail "${path}: Content-Type не JSON (${ctype:-empty})"
  printf '%s' "$body" | grep -q '<html' && fail "${path}: тело HTML, ожидался JSON"
  printf '%s' "$body" | grep -q "$needle" || fail "${path}: нет маркера ${needle}"
  printf 'OK %s\n' "${path}"
}

check_json '/health/ready' '"status":"ready"'
check_json '/api/v1/config' '"features"'
