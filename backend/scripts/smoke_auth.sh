#!/usr/bin/env bash
# Сквозной smoke авторизации против запущенного API (login→/me→refresh→неверный пароль).
# Проверяет роли и must_change_password по каждой роли. Требует curl + python3.
set -u
# Бьём через реальный https-хост: refresh-cookie помечена Secure+Domain, по http/loopback не отдаётся.
BASE="${SMOKE_BASE:-https://eazytech.ru}"
ORIGIN="https://eazytech.ru"
PASS=0; FAIL=0
jqget() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null; }
check() { # desc, actual, expected
  if [ "$2" = "$3" ]; then echo "  OK  $1 ($2)"; PASS=$((PASS+1));
  else echo "  FAIL $1: got [$2] want [$3]"; FAIL=$((FAIL+1)); fi
}

login() { # login pass -> sets ACCESS, HTTP, MCP; stores cookie in $CJAR
  local body code
  body=$(curl -s -c "$CJAR" -w '\n%{http_code}' -X POST "$BASE/api/v1/auth/login" \
    -H "Origin: $ORIGIN" -H 'Content-Type: application/json' \
    -d "{\"login\":\"$1\",\"password\":\"$2\"}")
  HTTP=$(printf '%s' "$body" | tail -n1)
  local json; json=$(printf '%s' "$body" | sed '$d')
  ACCESS=$(printf '%s' "$json" | jqget "['data']['access_token']")
  MCP=$(printf '%s' "$json" | jqget "['data']['must_change_password']")
}

me() { # uses ACCESS -> sets ROLES, MEHTTP
  local body code
  body=$(curl -s -w '\n%{http_code}' "$BASE/api/v1/auth/me" \
    -H "Authorization: Bearer $ACCESS")
  MEHTTP=$(printf '%s' "$body" | tail -n1)
  local json; json=$(printf '%s' "$body" | sed '$d')
  ROLES=$(printf '%s' "$json" | jqget "[','.join(d['data']['roles'])]" 2>/dev/null)
  ROLES=$(printf '%s' "$json" | python3 -c "import sys,json;print(','.join(json.load(sys.stdin)['data']['roles']))" 2>/dev/null)
}

echo "== 1. Суперадмин =="
CJAR=$(mktemp); login superadmin 'SuperAdmin!2026'
check "login HTTP 200" "$HTTP" "200"; check "must_change_password=False" "$MCP" "False"
me; check "/me HTTP 200" "$MEHTTP" "200"; check "роль SUPER_ADMIN" "$ROLES" "SUPER_ADMIN"

echo "== 2. Админ =="
CJAR=$(mktemp); login admin 'AdminPass!2026'
check "login HTTP 200" "$HTTP" "200"; check "must_change_password=False" "$MCP" "False"
me; check "роль ADMIN" "$ROLES" "ADMIN"

echo "== 3. Конкурсант (форс-смена) =="
CJAR=$(mktemp); login contestant 'Contestant!2026'
check "login HTTP 200" "$HTTP" "200"; check "must_change_password=True" "$MCP" "True"
me; check "роль CONTESTANT" "$ROLES" "CONTESTANT"

echo "== 4. Refresh по cookie (сессия конкурсанта) =="
RBODY=$(curl -s -b "$CJAR" -c "$CJAR" -w '\n%{http_code}' -X POST "$BASE/api/v1/auth/refresh" -H "Origin: $ORIGIN")
RHTTP=$(printf '%s' "$RBODY" | tail -n1)
NEWACCESS=$(printf '%s' "$RBODY" | sed '$d' | jqget "['data']['access_token']")
check "refresh HTTP 200" "$RHTTP" "200"
[ -n "$NEWACCESS" ] && [ "$NEWACCESS" != "None" ] && check "новый access выдан" "yes" "yes" || check "новый access выдан" "no" "yes"

echo "== 5. Неверный пароль =="
CJAR=$(mktemp); login admin 'WrongPassword!!'
check "login HTTP 401" "$HTTP" "401"

echo "== 6. Без токена /me =="
UHTTP=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/auth/me")
check "/me без токена 401" "$UHTTP" "401"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
