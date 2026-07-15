#!/usr/bin/env bash
# Сквозной smoke мульти-арендности (docs/RBAC_MULTITENANCY.md §6, этапы 2-3):
#   изоляция по владению, роль MEGA_ADMIN, уровни доступа ADMIN EDIT/VIEW,
#   участниками управляет только владелец/мега, guard создания ролей.
# По умолчанию бьёт локальный стенд (http, cookie не-Secure). Параметры через env:
#   SMOKE_BASE (default http://127.0.0.1:8080), SMOKE_ORIGIN (default http://localhost:5173),
#   MEGA_LOGIN / MEGA_PASSWORD (bootstrap MEGA_ADMIN из .env).
# Требует curl + python3 + docker(psql) для генерации уникальных логинов и уборки.
set -u
BASE="${SMOKE_BASE:-http://127.0.0.1:8080}"
ORIGIN="${SMOKE_ORIGIN:-http://localhost:5173}"
MEGA_LOGIN="${MEGA_LOGIN:-admin}"
MEGA_PASSWORD="${MEGA_PASSWORD:-localdevpassword123}"
NEWPW="Smoke!Pass2026"
PASS=0; FAIL=0
SUFFIX="$(date +%s)$RANDOM"
# python3 на сервере, python локально (Windows) — берём рабочий.
PY="$(command -v python3 || command -v python)"
if ! echo '{}' | "$PY" -c 'import sys,json;json.load(sys.stdin)' >/dev/null 2>&1; then
  PY="$(command -v python)"
fi

jqget() { "$PY" -c "import sys,json;d=json.load(sys.stdin);print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null; }
check() { if [ "$2" = "$3" ]; then echo "  OK  $1 ($2)"; PASS=$((PASS+1));
  else echo "  FAIL $1: got [$2] want [$3]"; FAIL=$((FAIL+1)); fi; }

# login LOGIN PASS -> ACCESS (пустой при неуспехе)
login() {
  local j
  j=$(curl -s -X POST "$BASE/api/v1/auth/login" -H "Origin: $ORIGIN" \
    -H 'Content-Type: application/json' -d "{\"login\":\"$1\",\"password\":\"$2\"}")
  ACCESS=$(printf '%s' "$j" | jqget "['data']['access_token']")
}
# api METHOD PATH TOKEN [body] -> HTTP, BODY
api() {
  local m=$1 path=$2 tok=$3 body=${4:-}
  local args=(-s -w '\n%{http_code}' -X "$m" "$BASE$path" -H "Origin: $ORIGIN" -H "Authorization: Bearer $tok")
  [ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
  local resp; resp=$(curl "${args[@]}")
  HTTP=$(printf '%s' "$resp" | tail -n1)
  BODY=$(printf '%s' "$resp" | sed '$d')
}
# create_and_activate LOGIN CREATOR_TOKEN CREATE_BODY -> ACCESS под NEWPW (сменив временный пароль)
create_and_activate() {
  local login=$1 tok=$2 body=$3
  api POST /api/v1/admin/users "$tok" "$body"
  local temp; temp=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
  login "$login" "$temp"
  curl -s -X POST "$BASE/api/v1/auth/change-password" -H "Origin: $ORIGIN" \
    -H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json' \
    -d "{\"old_password\":\"$temp\",\"new_password\":\"$NEWPW\"}" >/dev/null
  login "$login" "$NEWPW"
}
psql_exec() {
  docker exec slc-postgres psql -tA -U student_leader -d student_leader -c "$1" 2>/dev/null
}

SA="sa_$SUFFIX"; SB="sb_$SUFFIX"; VW="vw_$SUFFIX"; ED="ed_$SUFFIX"

echo "== 1. MEGA_ADMIN входит и виден как основная роль =="
login "$MEGA_LOGIN" "$MEGA_PASSWORD"; MEGA=$ACCESS
check "MEGA login" "$([ -n "$MEGA" ] && echo yes || echo no)" "yes"
api GET /api/v1/auth/me "$MEGA"
check "роль MEGA_ADMIN в /me" "$(printf '%s' "$BODY" | "$PY" -c "import sys,json;print('MEGA_ADMIN' in json.load(sys.stdin)['data']['roles'])" 2>/dev/null)" "True"

echo "== 2. MEGA создаёт двух организаторов (SUPER_ADMIN) с разными орг =="
create_and_activate "$SA" "$MEGA" "{\"login\":\"$SA\",\"full_name\":\"Орг А\",\"role\":\"SUPER_ADMIN\",\"org_name\":\"Alpha\"}"; ATA=$ACCESS
create_and_activate "$SB" "$MEGA" "{\"login\":\"$SB\",\"full_name\":\"Орг Б\",\"role\":\"SUPER_ADMIN\",\"org_name\":\"Beta\"}"; ATB=$ACCESS
check "super A активирован" "$([ -n "$ATA" ] && echo yes || echo no)" "yes"
check "super B активирован" "$([ -n "$ATB" ] && echo yes || echo no)" "yes"
check "org_name A = Alpha" "$(psql_exec "SELECT org_name FROM users WHERE login='$SA'")" "Alpha"

echo "== 3. Изоляция: A создаёт конкурс, B его не видит =="
api POST /api/v1/admin/contests "$ATA" "{\"name\":\"Конкурс А\",\"slug\":\"smoke-$SUFFIX\",\"timezone\":\"Europe/Moscow\"}"
check "A create contest 201" "$HTTP" "201"
CID=$(printf '%s' "$BODY" | jqget "['data']['id']")
check "owner_user_id = A" "$(psql_exec "SELECT owner_user_id=(SELECT id FROM users WHERE login='$SA') FROM contests WHERE id='$CID'")" "t"
api GET /api/v1/admin/contests "$ATB"
SEES=$(printf '%s' "$BODY" | "$PY" -c "import sys,json;print(any(c['id']=='$CID' for c in json.load(sys.stdin)['data']))" 2>/dev/null)
check "B не видит в списке" "$SEES" "False"
api GET "/api/v1/admin/contests/$CID" "$ATB"
check "B GET чужого → 403" "$HTTP" "403"
api GET "/api/v1/admin/contests/$CID" "$MEGA"
check "MEGA GET любого → 200" "$HTTP" "200"

echo "== 4. B не может создать SUPER_ADMIN (только MEGA) =="
api POST /api/v1/admin/users "$ATB" "{\"login\":\"hack_$SUFFIX\",\"full_name\":\"H\",\"role\":\"SUPER_ADMIN\"}"
check "B create SUPER_ADMIN → 403" "$HTTP" "403"

echo "== 5. ADMIN VIEW: читает, но не редактирует и не трогает участников =="
create_and_activate "$VW" "$ATA" "{\"login\":\"$VW\",\"full_name\":\"Вьюер\",\"role\":\"ADMIN\",\"scope_type\":\"CONTEST\",\"scope_id\":\"$CID\",\"access_level\":\"VIEW\"}"; VAT=$ACCESS
api GET "/api/v1/admin/contests/$CID" "$VAT";                                   check "VIEW GET → 200" "$HTTP" "200"
api PATCH "/api/v1/admin/contests/$CID" "$VAT" "{\"name\":\"x\",\"timezone\":\"Europe/Moscow\"}"; check "VIEW PATCH → 403" "$HTTP" "403"
api POST "/api/v1/admin/contests/$CID/challenges" "$VAT" "{\"title\":\"t\",\"slug\":\"t-$SUFFIX\"}"; check "VIEW create challenge → 403" "$HTTP" "403"
api POST "/api/v1/admin/contests/$CID/contestants" "$VAT" "{\"login\":\"z_$SUFFIX\",\"full_name\":\"Z\"}"; check "VIEW add contestant → 403" "$HTTP" "403"

echo "== 6. ADMIN EDIT: редактирует контент, но участниками — нет =="
create_and_activate "$ED" "$ATA" "{\"login\":\"$ED\",\"full_name\":\"Эдитор\",\"role\":\"ADMIN\",\"scope_type\":\"CONTEST\",\"scope_id\":\"$CID\",\"access_level\":\"EDIT\"}"; EAT=$ACCESS
api POST "/api/v1/admin/contests/$CID/challenges" "$EAT" "{\"title\":\"Испытание\",\"slug\":\"isp-$SUFFIX\"}"; check "EDIT create challenge → 201" "$HTTP" "201"
api POST "/api/v1/admin/contests/$CID/contestants" "$EAT" "{\"login\":\"z2_$SUFFIX\",\"full_name\":\"Z\"}"; check "EDIT add contestant → 403 (только владелец/мега)" "$HTTP" "403"

echo "== 7. Владелец управляет участниками =="
api POST "/api/v1/admin/contests/$CID/contestants" "$ATA" "{\"login\":\"real_$SUFFIX\",\"full_name\":\"Настоящий\"}"
check "OWNER add contestant → 201" "$HTTP" "201"

echo "== 8. Реестр пользователей изолирован по владению =="
api GET /api/v1/admin/users "$ATB"
BCOUNT=$(printf '%s' "$BODY" | "$PY" -c "import sys,json;print(len(json.load(sys.stdin)['data']))" 2>/dev/null)
check "B видит только своих (0)" "$BCOUNT" "0"

echo "== 9. Уборка =="
psql_exec "DELETE FROM contests WHERE id='$CID'" >/dev/null
psql_exec "DELETE FROM users WHERE login LIKE '%_$SUFFIX'" >/dev/null
echo "  очищено"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
