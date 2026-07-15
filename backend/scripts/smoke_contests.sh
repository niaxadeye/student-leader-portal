#!/usr/bin/env bash
# Сквозной smoke Этапа 2: CRUD конкурса, scoped-доступ, участники, block/reset.
# Бьём через https-хост (cookie Secure). Требует curl + python3 + docker(psql).
set -u
BASE="${SMOKE_BASE:-https://eazytech.ru}"
ORIGIN="https://eazytech.ru"
PASS=0; FAIL=0
jqget() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null; }
check() { if [ "$2" = "$3" ]; then echo "  OK  $1 ($2)"; PASS=$((PASS+1));
  else echo "  FAIL $1: got [$2] want [$3]"; FAIL=$((FAIL+1)); fi; }

login() { # login pass -> ACCESS
  local j
  j=$(curl -s -X POST "$BASE/api/v1/auth/login" -H "Origin: $ORIGIN" \
    -H 'Content-Type: application/json' -d "{\"login\":\"$1\",\"password\":\"$2\"}")
  ACCESS=$(printf '%s' "$j" | jqget "['data']['access_token']")
}
# api METHOD PATH TOKEN [body] -> HTTP (last line), BODY (rest)
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

echo "== 1. SUPER_ADMIN создаёт конкурс =="
login superadmin 'SuperAdmin!2026'
SLUG="smoke-$(psql_exec "SELECT floor(random()*1e9)::bigint")"
api POST /api/v1/admin/contests "$ACCESS" "{\"name\":\"Смоук-конкурс\",\"slug\":\"$SLUG\"}"
check "create HTTP 201" "$HTTP" "201"
CID=$(printf '%s' "$BODY" | jqget "['data']['id']")
check "статус DRAFT" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "DRAFT"
[ -n "$CID" ] && check "id выдан" "yes" "yes" || check "id выдан" "no" "yes"

echo "== 2. ADMIN не видит чужой конкурс (scope) =="
login admin 'AdminPass!2026'
api GET /api/v1/admin/contests "$ACCESS"
SEES=$(printf '%s' "$BODY" | python3 -c "import sys,json;print(any(c['id']=='$CID' for c in json.load(sys.stdin)['data']))" 2>/dev/null)
check "не видит в списке" "$SEES" "False"
api GET "/api/v1/admin/contests/$CID" "$ACCESS"
check "GET чужого → 403" "$HTTP" "403"

echo "== 3. ADMIN не имеет доступа к реестру юзеров (только SUPER_ADMIN) =="
api GET /api/v1/admin/users "$ACCESS"
check "ADMIN → /users 403" "$HTTP" "403"

echo "== 3b. SUPER_ADMIN назначает ADMIN scope через roles-endpoint =="
login superadmin 'SuperAdmin!2026'
AID=$(psql_exec "SELECT id FROM users WHERE login='admin'")
api POST "/api/v1/admin/users/$AID/roles" "$ACCESS" \
  "{\"role\":\"ADMIN\",\"scope_type\":\"CONTEST\",\"scope_id\":\"$CID\"}"
check "assign role HTTP 200" "$HTTP" "200"
login admin 'AdminPass!2026'
api GET /api/v1/admin/contests "$ACCESS"
SEES=$(printf '%s' "$BODY" | python3 -c "import sys,json;print(any(c['id']=='$CID' for c in json.load(sys.stdin)['data']))" 2>/dev/null)
check "теперь видит конкурс" "$SEES" "True"

echo "== 4. ADMIN добавляет конкурсанта =="
CLOGIN="smoke_cn_$(psql_exec "SELECT floor(random()*1e6)::int")"
api POST "/api/v1/admin/contests/$CID/contestants" "$ACCESS" \
  "{\"login\":\"$CLOGIN\",\"full_name\":\"Смоук Конкурсант\",\"organization\":\"Тест\"}"
check "add contestant HTTP 201" "$HTTP" "201"
TEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
[ -n "$TEMP" ] && check "временный пароль выдан" "yes" "yes" || check "временный пароль выдан" "no" "yes"
api GET "/api/v1/admin/contests/$CID/contestants" "$ACCESS"
CNT=$(printf '%s' "$BODY" | jqget "['meta']['count']")
check "участник в списке" "$CNT" "1"

echo "== 5. Конкурсант входит по временному паролю (must_change) =="
login "$CLOGIN" "$TEMP"
check "login конкурсанта" "$([ -n "$ACCESS" ] && echo yes || echo no)" "yes"
MCP=$(psql_exec "SELECT must_change_password FROM users WHERE login='$CLOGIN'")
check "must_change_password=t" "$MCP" "t"

echo "== 6. Переход статуса DRAFT→ACTIVE (publish) =="
login admin 'AdminPass!2026'
api POST "/api/v1/admin/contests/$CID/publish" "$ACCESS"
check "publish HTTP 200" "$HTTP" "200"
check "статус ACTIVE" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "ACTIVE"
api POST "/api/v1/admin/contests/$CID/finish" "$ACCESS"
check "finish → FINISHED" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "FINISHED"
api POST "/api/v1/admin/contests/$CID/publish" "$ACCESS"
check "недопустимый переход → 409" "$HTTP" "409"

echo "== 7. reset-password и block конкурсанта =="
CUID=$(psql_exec "SELECT id FROM users WHERE login='$CLOGIN'")
api POST "/api/v1/admin/users/$CUID/reset-password" "$ACCESS"
check "reset HTTP 200" "$HTTP" "200"
NEWTEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
[ -n "$NEWTEMP" ] && check "новый temp выдан" "yes" "yes" || check "новый temp выдан" "no" "yes"
api POST "/api/v1/admin/users/$CUID/block" "$ACCESS"
check "block HTTP 200" "$HTTP" "200"
login "$CLOGIN" "$NEWTEMP"
check "заблокированный не входит" "$([ -z "$ACCESS" ] && echo yes || echo no)" "yes"

echo "== 9. Import/export конкурсантов (ADMIN, CSV) =="
login admin 'AdminPass!2026'
IMPLOGIN="smoke_imp_$(psql_exec "SELECT floor(random()*1e6)::int")"
CSV=$(printf 'login,full_name,organization\n%s,Импорт Один,ОргА\n%s_2,Импорт Два,ОргБ\n,БезЛогина,X' "$IMPLOGIN" "$IMPLOGIN")
RESP=$(curl -s -w '\n%{http_code}' -X POST "$BASE/api/v1/admin/contests/$CID/contestants/import" \
  -H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" -H 'Content-Type: text/csv' --data-binary "$CSV")
HTTP=$(printf '%s' "$RESP" | tail -n1); BODY=$(printf '%s' "$RESP" | sed '$d')
check "import HTTP 200" "$HTTP" "200"
check "created=2" "$(printf '%s' "$BODY" | jqget "['meta']['created']")" "2"
check "failed=1 (пустой login)" "$(printf '%s' "$BODY" | jqget "['meta']['failed']")" "1"
EXP=$(curl -s "$BASE/api/v1/admin/contests/$CID/contestants/export" -H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS")
HDR=$(printf '%s' "$EXP" | head -1)
check "export заголовок CSV" "$HDR" "login,full_name,organization,status,joined_at"
ROWS=$(printf '%s' "$EXP" | grep -c "$IMPLOGIN")
check "export содержит импортированных (2)" "$ROWS" "2"

echo "== 10. User CRUD и roles (SUPER_ADMIN) =="
login superadmin 'SuperAdmin!2026'
ULOGIN="smoke_usr_$(psql_exec "SELECT floor(random()*1e6)::int")"
api POST /api/v1/admin/users "$ACCESS" "{\"login\":\"$ULOGIN\",\"full_name\":\"Новый Юзер\",\"role\":\"ADMIN\"}"
check "create user 201" "$HTTP" "201"
NUID=$(printf '%s' "$BODY" | jqget "['data']['user_id']")
api POST /api/v1/admin/users "$ACCESS" "{\"login\":\"$ULOGIN\",\"full_name\":\"Дубль\"}"
check "дубль логина → 409" "$HTTP" "409"
api GET "/api/v1/admin/users?search=$ULOGIN" "$ACCESS"
check "поиск находит (total=1)" "$(printf '%s' "$BODY" | jqget "['meta']['total']")" "1"
api GET "/api/v1/admin/users/$NUID" "$ACCESS"
check "деталь: роль ADMIN назначена" "$(printf '%s' "$BODY" | jqget "['data']['roles'][0]['code']")" "ADMIN"
api PATCH "/api/v1/admin/users/$NUID" "$ACCESS" "{\"full_name\":\"Обновлён\"}"
check "update → новое имя" "$(printf '%s' "$BODY" | jqget "['data']['full_name']")" "Обновлён"
api DELETE "/api/v1/admin/users/$NUID/roles?role=ADMIN&scope_type=GLOBAL" "$ACCESS"
check "remove role 200" "$HTTP" "200"
api GET "/api/v1/admin/users/$NUID" "$ACCESS"
check "ролей не осталось" "$(printf '%s' "$BODY" | jqget "['data']['roles']")" "[]"

echo "== 11. Уборка =="
psql_exec "DELETE FROM contests WHERE id='$CID'" >/dev/null
psql_exec "DELETE FROM users WHERE login LIKE '$IMPLOGIN%' OR login='$CLOGIN' OR login='$ULOGIN'" >/dev/null
psql_exec "DELETE FROM user_roles WHERE scope_id='$CID'" >/dev/null
echo "  очищено"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
