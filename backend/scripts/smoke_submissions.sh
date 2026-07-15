#!/usr/bin/env bash
# Сквозной smoke Этапа 4: черновик, сохранение, файл, submit+ревизия, resubmit,
# дедлайн, валидация обязательных, админ-таблица/карточка, доступ. curl+python3+docker(psql).
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

echo "== 0. Подготовка: конкурс + published-испытание с полями + конкурсант =="
login superadmin 'SuperAdmin!2026'
SLUG="sub-$(psql_exec "SELECT floor(random()*1e9)::bigint")"
api POST /api/v1/admin/contests "$ACCESS" "{\"name\":\"Submit-конкурс\",\"slug\":\"$SLUG\"}"
CID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/contests/$CID/publish" "$ACCESS"
api POST "/api/v1/admin/contests/$CID/challenges" "$ACCESS" \
  "{\"title\":\"Эссе\",\"deadline_at\":\"2030-12-31T20:00:00Z\"}"
CHID=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"title\",\"type\":\"SHORT_TEXT\",\"label\":\"Заголовок\",\"required\":true}"
api POST "/api/v1/admin/challenges/$CHID/fields" "$ACCESS" \
  "{\"key\":\"doc\",\"type\":\"FILE_GROUP\",\"label\":\"Документ\",\"required\":false,\"settings\":{\"allowed_extensions\":[\"txt\",\"pdf\"],\"max_file_size_mb\":5}}"
FDOC=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$CHID/publish" "$ACCESS"
CLOGIN="sub_cn_$(psql_exec "SELECT floor(random()*1e6)::int")"
api POST "/api/v1/admin/contests/$CID/contestants" "$ACCESS" \
  "{\"login\":\"$CLOGIN\",\"full_name\":\"Иван Тестов\",\"organization\":\"ТГУ\"}"
TEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
check "подготовка ок" "$([ -n "$CHID" ] && [ -n "$TEMP" ] && [ -n "$FDOC" ] && echo yes || echo no)" "yes"

echo "== 1. Конкурсант открывает испытание → создаётся черновик =="
login "$CLOGIN" "$TEMP"
api GET "/api/v1/challenges/$CHID/submission" "$ACCESS"
check "GET submission HTTP 200" "$HTTP" "200"
check "статус DRAFT" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "DRAFT"
check "ревизий 0" "$(printf '%s' "$BODY" | jqget "['data']['current_revision_number']")" "0"
SUBID=$(printf '%s' "$BODY" | jqget "['data']['id']")

echo "== 2. Сохранение черновика =="
api PUT "/api/v1/challenges/$CHID/submission/draft" "$ACCESS" \
  "{\"answers\":{\"title\":\"Мой черновик\"}}"
check "save draft HTTP 200" "$HTTP" "200"
check "ответ сохранён" "$(printf '%s' "$BODY" | jqget "['data']['answers']['title']")" "Мой черновик"
check "last_saved_at проставлен" "$([ "$(printf '%s' "$BODY" | jqget "['data']['last_saved_at']")" != "None" ] && echo yes || echo no)" "yes"
check "статус всё ещё DRAFT" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "DRAFT"

echo "== 3. Submit без обязательного поля → 400 =="
api POST "/api/v1/challenges/$CHID/submission/submit" "$ACCESS" "{\"answers\":{\"title\":\"\"}}"
check "пустой обязательный → 400" "$HTTP" "400"
check "код VALIDATION_ERROR" "$(printf '%s' "$BODY" | jqget "['error']['code']")" "VALIDATION_ERROR"

echo "== 4. Загрузка файла (multipart) =="
TMPF=$(mktemp --suffix=.txt); echo "hello submission" > "$TMPF"
resp=$(curl -s -w '\n%{http_code}' -X POST "$BASE/api/v1/challenges/$CHID/submission/files" \
  -H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" \
  -F "field_id=$FDOC" -F "file=@$TMPF;type=text/plain")
UHTTP=$(printf '%s' "$resp" | tail -n1); UBODY=$(printf '%s' "$resp" | sed '$d')
check "upload HTTP 201" "$UHTTP" "201"
FILEID=$(printf '%s' "$UBODY" | jqget "['data']['file_id']")
check "file_id получен" "$([ -n "$FILEID" ] && echo yes || echo no)" "yes"
# запрещённое расширение
TMPBAD=$(mktemp --suffix=.exe); echo x > "$TMPBAD"
resp=$(curl -s -w '\n%{http_code}' -X POST "$BASE/api/v1/challenges/$CHID/submission/files" \
  -H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" \
  -F "field_id=$FDOC" -F "file=@$TMPBAD;type=application/octet-stream")
check "запрещённое расширение → 400" "$(printf '%s' "$resp" | tail -n1)" "400"
rm -f "$TMPF" "$TMPBAD"

echo "== 5. Submit с обязательным полем → ревизия 1 =="
api POST "/api/v1/challenges/$CHID/submission/submit" "$ACCESS" \
  "{\"answers\":{\"title\":\"Финальный заголовок\"}}"
check "submit HTTP 200" "$HTTP" "200"
check "статус SUBMITTED" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "SUBMITTED"
check "ревизия 1" "$(printf '%s' "$BODY" | jqget "['data']['current_revision_number']")" "1"
check "submitted_at проставлен" "$([ "$(printf '%s' "$BODY" | jqget "['data']['submitted_at']")" != "None" ] && echo yes || echo no)" "yes"
check "файл в submission" "$(printf '%s' "$BODY" | jqget "['data']['files'].__len__()")" "1"

echo "== 6. Resubmit → ревизия 2, version bump =="
api POST "/api/v1/challenges/$CHID/submission/submit" "$ACCESS" \
  "{\"answers\":{\"title\":\"Обновлённый заголовок\"}}"
check "resubmit HTTP 200" "$HTTP" "200"
check "ревизия 2" "$(printf '%s' "$BODY" | jqget "['data']['current_revision_number']")" "2"
check "version = 2" "$(printf '%s' "$BODY" | jqget "['data']['version']")" "2"
REVS=$(psql_exec "SELECT count(*) FROM submission_revisions WHERE submission_id='$SUBID'")
check "в БД 2 ревизии" "$REVS" "2"

echo "== 7. Удаление файла из работы =="
api DELETE "/api/v1/challenges/$CHID/submission/files/$FILEID" "$ACCESS"
check "delete file HTTP 200" "$HTTP" "200"
api GET "/api/v1/challenges/$CHID/submission" "$ACCESS"
check "файлов больше нет" "$(printf '%s' "$BODY" | jqget "['data']['files'].__len__()")" "0"

echo "== 8. Админ: таблица работ (SITE.md §7.6) =="
login superadmin 'SuperAdmin!2026'
api GET "/api/v1/admin/challenges/$CHID/submissions" "$ACCESS"
check "admin list HTTP 200" "$HTTP" "200"
check "1 работа" "$(printf '%s' "$BODY" | jqget "['meta']['total']")" "1"
check "ФИО присоединено" "$(printf '%s' "$BODY" | jqget "['data'][0]['full_name']")" "Иван Тестов"
check "организация из metadata" "$(printf '%s' "$BODY" | jqget "['data'][0]['organization']")" "ТГУ"
check "статус SUBMITTED" "$(printf '%s' "$BODY" | jqget "['data'][0]['status']")" "SUBMITTED"
# фильтр по статусу
api GET "/api/v1/admin/challenges/$CHID/submissions?status=DRAFT" "$ACCESS"
check "фильтр DRAFT → пусто" "$(printf '%s' "$BODY" | jqget "['meta']['total']")" "0"

echo "== 9. Админ: карточка одной работы + история ревизий =="
api GET "/api/v1/admin/submissions/$SUBID" "$ACCESS"
check "admin get HTTP 200" "$HTTP" "200"
check "ответ отдан" "$(printf '%s' "$BODY" | jqget "['data']['answers']['title']")" "Обновлённый заголовок"
check "ревизий в истории 2" "$(printf '%s' "$BODY" | jqget "['data']['revisions'].__len__()")" "2"
check "новейшая ревизия сверху" "$(printf '%s' "$BODY" | jqget "['data']['revisions'][0]['revision_number']")" "2"
check "тип RESUBMIT" "$(printf '%s' "$BODY" | jqget "['data']['revisions'][0]['action_type']")" "RESUBMIT"

echo "== 10. Дедлайн истёк + без поздней подачи → 409 =="
api POST "/api/v1/admin/contests/$CID/challenges" "$ACCESS" \
  "{\"title\":\"Просроченное\",\"deadline_at\":\"2020-01-01T00:00:00Z\"}"
LCH=$(printf '%s' "$BODY" | jqget "['data']['id']")
api POST "/api/v1/admin/challenges/$LCH/fields" "$ACCESS" \
  "{\"key\":\"x\",\"type\":\"SHORT_TEXT\",\"label\":\"X\"}"
api POST "/api/v1/admin/challenges/$LCH/publish" "$ACCESS"
login "$CLOGIN" "$TEMP"
api POST "/api/v1/challenges/$LCH/submission/submit" "$ACCESS" "{\"answers\":{\"x\":\"y\"}}"
check "просроченный дедлайн → 409" "$HTTP" "409"
check "код DEADLINE_PASSED" "$(printf '%s' "$BODY" | jqget "['error']['code']")" "DEADLINE_PASSED"

echo "== 11. Доступ: посторонний не-участник → 403 =="
OTHER="sub_out_$(psql_exec "SELECT floor(random()*1e6)::int")"
login superadmin 'SuperAdmin!2026'
api POST /api/v1/admin/users "$ACCESS" \
  "{\"login\":\"$OTHER\",\"full_name\":\"Чужой\",\"role\":\"CONTESTANT\"}"
OTEMP=$(printf '%s' "$BODY" | jqget "['data']['temp_password']")
login "$OTHER" "$OTEMP"
api GET "/api/v1/challenges/$CHID/submission" "$ACCESS"
check "не-участник → 403" "$HTTP" "403"

echo "== 12. Уборка =="
psql_exec "DELETE FROM contests WHERE id='$CID'" >/dev/null
psql_exec "DELETE FROM users WHERE login IN ('$CLOGIN','$OTHER')" >/dev/null
echo "  очищено"

echo; echo "ИТОГО: PASS=$PASS FAIL=$FAIL"
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)
