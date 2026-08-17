#!/usr/bin/env bash
# End-to-end regression for EventParticipant CRUD, status transitions and login.
set -u

BASE="${SMOKE_BASE:-https://eazytech.ru}"
ORIGIN="${SMOKE_ORIGIN:-https://eazytech.ru}"
CONTEST_ID="${SMOKE_EVENT_ID:-35868516-d200-40c8-b3d6-9bd6f103fc31}"
EVENT_SLUG="${SMOKE_EVENT_SLUG:-lpb-2026}"
PASS=0
FAIL=0
PARTICIPANT_ID=""
COOKIE_JAR="$(mktemp)"

cleanup() {
  if [ -n "$PARTICIPANT_ID" ]; then
    set -a
    . /var/www/student-leader-portal/.env
    set +a
    docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" slc-postgres \
      psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
      -c "DELETE FROM participant_sessions WHERE event_participant_id='$PARTICIPANT_ID'; DELETE FROM event_participants WHERE id='$PARTICIPANT_ID';" \
      >/dev/null 2>&1 || true
  fi
  rm -f "$COOKIE_JAR"
}
trap cleanup EXIT

jqget() {
  python3 -c "import sys,json; d=json.load(sys.stdin); print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null
}

check() {
  if [ "$2" = "$3" ]; then
    echo "  OK  $1 ($2)"
    PASS=$((PASS + 1))
  else
    echo "  FAIL $1: got [$2] want [$3]"
    FAIL=$((FAIL + 1))
  fi
}

staff_login() {
  local response
  response=$(curl -sS -X POST "$BASE/api/v1/auth/login" -H "Origin: $ORIGIN" \
    -H 'Content-Type: application/json' \
    -d '{"login":"superadmin","password":"SuperAdmin!2026"}')
  ACCESS=$(printf '%s' "$response" | jqget "['data']['access_token']")
}

admin_api() {
  local method=$1 path=$2 body=${3:-}
  local args=(-sS -w '\n%{http_code}' -X "$method" "$BASE$path" \
    -H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS")
  if [ -n "$body" ]; then
    args+=(-H 'Content-Type: application/json' -d "$body")
  fi
  local response
  response=$(curl "${args[@]}")
  HTTP=$(printf '%s' "$response" | tail -n1)
  BODY=$(printf '%s' "$response" | sed '$d')
}

participant_login() {
  local method=$1 body=$2
  local response
  response=$(curl -sS -c "$COOKIE_JAR" -w '\n%{http_code}' -X POST \
    "$BASE/api/v1/events/$EVENT_SLUG/participant-auth/$method" \
    -H "Origin: $ORIGIN" -H 'Content-Type: application/json' -d "$body")
  HTTP=$(printf '%s' "$response" | tail -n1)
  BODY=$(printf '%s' "$response" | sed '$d')
}

SUFFIX="$(date +%s)-$$"
FULL_NAME="Смоук Участник $SUFFIX"
UNION_CARD="smoke-union-$SUFFIX"
SKS="smoke-sks-$SUFFIX"

echo "== EventParticipant regression: $EVENT_SLUG =="
staff_login

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants" \
  "{\"full_name\":\"$FULL_NAME\",\"birth_date\":\"2000-01-02\",\"union_card_number\":\"$UNION_CARD\",\"sks_barcode\":\"$SKS\"}"
check "create participant" "$HTTP" "201"
PARTICIPANT_ID=$(printf '%s' "$BODY" | jqget "['data']['id']")

admin_api PATCH "/api/v1/admin/contests/$CONTEST_ID/participants/$PARTICIPANT_ID" \
  "{\"full_name\":\"$FULL_NAME\",\"birth_date\":\"2000-01-02\",\"union_card_number\":\"$UNION_CARD\",\"sks_barcode\":\"$SKS\"}"
check "update participant" "$HTTP" "200"

participant_login fio "{\"full_name\":\"$FULL_NAME\",\"birth_date\":\"2000-01-02\"}"
check "login by name" "$HTTP" "200"
HTTP=$(curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' "$BASE/api/v1/participant/me")
check "participant session" "$HTTP" "200"

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants/$PARTICIPANT_ID/block"
check "block participant" "$HTTP" "200"
HTTP=$(curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' "$BASE/api/v1/participant/me")
check "block revokes session" "$HTTP" "401"
participant_login union-card "{\"value\":\"$UNION_CARD\"}"
check "blocked participant cannot login" "$HTTP" "401"

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants/$PARTICIPANT_ID/unblock"
check "unblock participant" "$HTTP" "200"
participant_login union-card "{\"value\":\"$UNION_CARD\"}"
check "login after unblock" "$HTTP" "200"

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants/$PARTICIPANT_ID/archive"
check "archive participant" "$HTTP" "200"
participant_login sks "{\"value\":\"$SKS\"}"
check "archived participant cannot login" "$HTTP" "401"

echo
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
