#!/usr/bin/env bash
# Сквозной smoke Этапа 5: транзакционный outbox. Проверяет, что submit/resubmit
# создают событие в той же транзакции, а сохранение черновика — нет. Доставка в
# Telegram (PENDING→SENT) проверяется только если задан TELEGRAM_BOT_TOKEN.
# curl + python3 + docker(psql).
set -u
BASE="${SMOKE_BASE:-https://eazytech.ru}"
ORIGIN="https://eazytech.ru"
PASS=0; FAIL=0
jqget() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null; }
check() { if [ "$2" = "$3" ]; then echo "  OK  $1 ($2)"; PASS=$((PASS+1));
  else echo "  FAIL $1: got [$2] want [$3]"; FAIL=$((FAIL+1)); fi; }
login() {
  local j
  j=$(curl -s -X POST "$BASE/api/v1/auth/login" -H "Origin: $ORIGIN" \
    -H 'Content-Type: application/json' -d "{\"login\":\"$1\",\"password\":\"$2\"}")
  ACCESS=$(printf '%s' "$j" | jqget "['data']['access_token']")
}
api() {
  local m=$1 path=$2 tok=$3 body=${4:-}
  local args=(-s -w '\n%{http_code}' -X "$m" "$BASE$path" -H "Origin: $ORIGIN" -H "Authorization: Bearer $tok")
  [ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
  local resp; resp=$(curl "${args[@]}")
  HTTP=$(printf '%s' "$resp" | tail -n1)
  BODY=$(printf '%s' "$resp" | sed '$d')
}
psql_exec() {
  set -a; . /var/www/student-leader-portal/.env; set +a
  docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" slc-postgres \
    psql -tA -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1" 2>/dev/null
}
ecount() { # события по submission id и типу
  psql_exec "SELECT count(*) FROM outbox_events WHERE aggregate_id='$1' AND event_type='$2'" | tr -d ' '
}

echo "== 0. Подготовка: конкурс + published-испытание + конкурсант =="
login superadmin 'SuperAdmin!2026'
SLUG="obx-$(psql_exec "SELECT floor(random()*1e9)::bigint")"
api POST /api/v1/admin/contests "$ACCESS" "{\"name\":\"Outbox-конкурс\",\"slug\":\"$SLUG\"}"
CID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/contests/$CID/publish" "$ACCESS"
api POST "/api/v1/admin/contests/$CID/challenges" "$ACCESS" \
  "{\"title\":\"Питч\",\"deadline_at\":\"2030-12-31T20:00:00Z\"}"
CHID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"title\",\"type\":\"SHORT_TEXT\",\"label\":\"Заголовок\",\"required\":true}"
api POST "/api/v1/admin/challenges/$CHID/publish" "$ACCESS"
CLOGIN="obx_cn_$(psql_exec "SELECT floor(random()*1e6)::int")"
api POST "/api/v1/admin/contests/$CID/contestants" "$ACCESS" \
  "{\"login\":\"$CLOGIN\",\"full_name\":\"Пётр Питчев\",\"organization\":\"МФТИ\"}"
TEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
check "подготовка ок" "$([ -n "$CHID" ] && [ -n "$TEMP" ] && echo yes || echo no)" "yes"

echo "== 1. Черновик НЕ создаёт событие =="
login "$CLOGIN" "$TEMP"
api GET "/api/v1/challenges/$CHID/submission" "$ACCESS"
SUBID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api PUT "/api/v1/challenges/$CHID/submission/draft" "$ACCESS" \
  "{\"answers\":{\"title\":\"Черновик\"},\"version\":1}"
check "save draft HTTP 200" "$HTTP" "200"
check "событий после черновика 0" "$(ecount "$SUBID" submission.submitted)" "0"

echo "== 2. Submit создаёт PENDING submission.submitted =="
api POST "/api/v1/challenges/$CHID/submission/submit" "$ACCESS" \
  "{\"answers\":{\"title\":\"Финал\"},\"version\":1}"
check "submit HTTP 200" "$HTTP" "200"
check "1 событие submitted" "$(ecount "$SUBID" submission.submitted)" "1"
ST=$(psql_exec "SELECT status FROM outbox_events WHERE aggregate_id='$SUBID' AND event_type='submission.submitted'" | tr -d ' ')
# при выключенном Telegram событие остаётся PENDING; при включённом — уедет в SENT
check "статус события PENDING или SENT" "$([ "$ST" = PENDING ] || [ "$ST" = SENT ] && echo yes || echo no)" "yes"
check "payload содержит revision=1" \
  "$(psql_exec "SELECT (payload->>'revision') FROM outbox_events WHERE aggregate_id='$SUBID' AND event_type='submission.submitted'" | tr -d ' ')" "1"

echo "== 3. Resubmit (тот же /submit; сервис выбирает RESUBMIT по номеру ревизии) =="
api POST "/api/v1/challenges/$CHID/submission/submit" "$ACCESS" \
  "{\"answers\":{\"title\":\"Финал 2\"},\"version\":2}"
check "resubmit HTTP 200" "$HTTP" "200"
check "1 событие resubmitted" "$(ecount "$SUBID" submission.resubmitted)" "1"
check "payload resubmit revision=2" \
  "$(psql_exec "SELECT (payload->>'revision') FROM outbox_events WHERE aggregate_id='$SUBID' AND event_type='submission.resubmitted'" | tr -d ' ')" "2"

echo "== 4. Резолвер уведомления собирает читаемые поля =="
# Прямая проверка джойна резолвера (то, что диспетчер отправит в Telegram).
RESOLVED=$(psql_exec "SELECT ct.name||'|'||ch.title||'|'||u.full_name||'|'||COALESCE(u.organization,'') \
  FROM submissions s JOIN contest_challenges ch ON ch.id=s.challenge_id \
  JOIN contests ct ON ct.id=ch.contest_id JOIN users u ON u.id=s.contestant_user_id \
  WHERE s.id='$SUBID'" | sed 's/^ *//;s/ *$//')
check "резолвер: конкурс|испытание|ФИО|орг" "$RESOLVED" "Outbox-конкурс|Питч|Пётр Питчев|МФТИ"

echo "== 5. (опц.) Доставка в Telegram, если включена =="
ENABLED=$(set -a; . /var/www/student-leader-portal/.env 2>/dev/null; set +a; echo "${TELEGRAM_NOTIFICATIONS_ENABLED:-false}")
if [ "$ENABLED" = "true" ]; then
  echo "  (Telegram включён — ждём доставку до 20с)"
  DELIVERED=no
  for _ in $(seq 1 10); do
    sleep 2
    S=$(psql_exec "SELECT status FROM outbox_events WHERE aggregate_id='$SUBID' AND event_type='submission.submitted'" | tr -d ' ')
    if [ "$S" = SENT ]; then DELIVERED=yes; break; fi
    if [ "$S" = DEAD ]; then DELIVERED=dead; break; fi
  done
  check "событие доставлено (SENT)" "$DELIVERED" "yes"
else
  echo "  (Telegram выключен — проверка доставки пропущена; события остаются PENDING)"
fi

echo "== 6. Уборка =="
psql_exec "DELETE FROM contests WHERE id='$CID'" >/dev/null
psql_exec "DELETE FROM users WHERE login='$CLOGIN'" >/dev/null
echo "  очищено"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
