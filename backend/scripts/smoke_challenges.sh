#!/usr/bin/env bash
# Сквозной smoke Этапа 3: CRUD испытаний, поля, reorder, preview, publish+снапшот,
# видимость для конкурсанта, переходы статуса, дедлайн. Требует curl + python3 + docker(psql).
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

echo "== 0. Подготовка: конкурс + конкурсант (SUPER_ADMIN) =="
login superadmin 'SuperAdmin!2026'
SLUG="chl-$(psql_exec "SELECT floor(random()*1e9)::bigint")"
api POST /api/v1/admin/contests "$ACCESS" "{\"name\":\"Испыт-конкурс\",\"slug\":\"$SLUG\"}"
CID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/contests/$CID/publish" "$ACCESS"
CLOGIN="chl_cn_$(psql_exec "SELECT floor(random()*1e6)::int")"
api POST "/api/v1/admin/contests/$CID/contestants" "$ACCESS" \
  "{\"login\":\"$CLOGIN\",\"full_name\":\"Конкурсант\"}"
TEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
check "подготовка: конкурс+конкурсант" "$([ -n "$CID" ] && [ -n "$TEMP" ] && echo yes || echo no)" "yes"

echo "== 1. Создание испытания (DRAFT) =="
DL="2030-12-31T20:00:00Z"
api POST "/api/v1/admin/contests/$CID/challenges" "$ACCESS" \
  "{\"title\":\"Презентация проекта\",\"deadline_at\":\"$DL\"}"
check "create HTTP 201" "$HTTP" "201"
CHID=$(printf '%s' "$BODY" | jqget "['data']['id']")
check "статус DRAFT" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "DRAFT"
check "дедлайн сохранён" "$(printf '%s' "$BODY" | jqget "['data']['deadline_at'][:10]")" "2030-12-31"
check "версия схемы = 1" "$(printf '%s' "$BODY" | jqget "['data']['current_schema_version']")" "1"

echo "== 2. Добавление полей + валидация типа =="
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"project_name\",\"type\":\"SHORT_TEXT\",\"label\":\"Название проекта\",\"required\":true}"
check "поле 1 HTTP 201" "$HTTP" "201"
F1=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"summary\",\"type\":\"LONG_TEXT\",\"label\":\"Описание\"}"
F2=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"bad\",\"type\":\"WORMHOLE\",\"label\":\"Некорректный тип\"}"
check "неизвестный тип → 400" "$HTTP" "400"
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"project_name\",\"type\":\"URL\",\"label\":\"Дубль ключа\"}"
check "дубль ключа поля → 409" "$HTTP" "409"
api GET "/api/v1/admin/challenges/$CHID/fields" "$ACCESS"
check "полей в списке = 2" "$(printf '%s' "$BODY" | jqget "['meta']['count']")" "2"

echo "== 3. Reorder полей =="
api PATCH "/api/v1/admin/challenges/$CHID/fields/reorder" "$ACCESS" \
  "{\"field_ids\":[\"$F2\",\"$F1\"]}"
check "reorder HTTP 200" "$HTTP" "200"
api GET "/api/v1/admin/challenges/$CHID/fields" "$ACCESS"
check "первым теперь summary" "$(printf '%s' "$BODY" | jqget "['data'][0]['key']")" "summary"

echo "== 4. Schema-preview =="
api GET "/api/v1/admin/challenges/$CHID/schema-preview" "$ACCESS"
check "preview HTTP 200" "$HTTP" "200"
check "preview: 2 поля" "$(printf '%s' "$BODY" | jqget "['data']['fields'].__len__()")" "2"

echo "== 5. Публикация + снапшот схемы =="
api POST "/api/v1/admin/challenges/$CHID/publish" "$ACCESS"
check "publish HTTP 200" "$HTTP" "200"
check "статус PUBLISHED" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "PUBLISHED"
SNAP=$(psql_exec "SELECT count(*) FROM challenge_schema_versions WHERE challenge_id='$CHID'")
check "снапшот версии создан" "$SNAP" "1"

echo "== 6. Правка опубликованной формы → bump версии + новый снапшот =="
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"link\",\"type\":\"URL\",\"label\":\"Ссылка\"}"
check "добавили поле в PUBLISHED (201)" "$HTTP" "201"
api GET "/api/v1/admin/challenges/$CHID" "$ACCESS"
check "версия схемы = 2" "$(printf '%s' "$BODY" | jqget "['data']['current_schema_version']")" "2"
SNAP=$(psql_exec "SELECT count(*) FROM challenge_schema_versions WHERE challenge_id='$CHID'")
check "снапшотов теперь 2" "$SNAP" "2"

echo "== 7. Второе испытание (DRAFT, не должно быть видно конкурсанту) =="
api POST "/api/v1/admin/contests/$CID/challenges" "$ACCESS" \
  "{\"title\":\"Черновик-испытание\"}"
DRAFTID=$(printf '%s' "$BODY" | jqget "['data']['id']")

echo "== 8. Конкурсант видит только PUBLISHED =="
login "$CLOGIN" "$TEMP"
api GET "/api/v1/contests/$CID/challenges" "$ACCESS"
check "список для конкурсанта HTTP 200" "$HTTP" "200"
check "видит только 1 (PUBLISHED)" "$(printf '%s' "$BODY" | jqget "['meta']['count']")" "1"
api GET "/api/v1/challenges/$CHID" "$ACCESS"
check "GET published HTTP 200" "$HTTP" "200"
check "поля отданы (3)" "$(printf '%s' "$BODY" | jqget "['data']['fields'].__len__()")" "3"
api GET "/api/v1/challenges/$DRAFTID" "$ACCESS"
check "GET draft → 404 (скрыт)" "$HTTP" "404"

echo "== 9. Переходы статуса и 409 =="
login superadmin 'SuperAdmin!2026'
api POST "/api/v1/admin/challenges/$CHID/close" "$ACCESS"
check "publish→close HTTP 200" "$HTTP" "200"
check "статус CLOSED" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "CLOSED"
api POST "/api/v1/admin/challenges/$CHID/close" "$ACCESS"
check "close→close → 409" "$HTTP" "409"
api POST "/api/v1/admin/challenges/$CHID/archive" "$ACCESS"
check "close→archive HTTP 200" "$HTTP" "200"
api POST "/api/v1/admin/challenges/$CHID/publish" "$ACCESS"
check "archive→publish → 409" "$HTTP" "409"

echo "== 10. Чужой ADMIN не имеет доступа =="
login admin 'AdminPass!2026'
api GET "/api/v1/admin/challenges/$DRAFTID" "$ACCESS"
check "чужое испытание → 403" "$HTTP" "403"

echo "== 11. Уборка =="
psql_exec "DELETE FROM contests WHERE id='$CID'" >/dev/null
psql_exec "DELETE FROM users WHERE login='$CLOGIN'" >/dev/null
echo "  очищено"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
