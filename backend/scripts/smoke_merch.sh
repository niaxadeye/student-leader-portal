#!/usr/bin/env bash
# End-to-end regression for merch reserve/cancel/issue/reject and points holds.
set -u

BASE="${SMOKE_BASE:-https://eazytech.ru}"
ORIGIN="${SMOKE_ORIGIN:-https://eazytech.ru}"
CONTEST_ID="${SMOKE_EVENT_ID:-35868516-d200-40c8-b3d6-9bd6f103fc31}"
EVENT_SLUG="${SMOKE_EVENT_SLUG:-lpb-2026}"
PASS=0
FAIL=0
PARTICIPANT_ID=""
PRODUCT_ID=""
IMAGE_ID=""
COOKIE_JAR="$(mktemp)"
RACE_DIR=""
VALID_IMAGE="$(mktemp --suffix=.png)"
SPOOF_IMAGE="$(mktemp --suffix=.png)"
DOWNLOADED_IMAGE="$(mktemp --suffix=.download.png)"

cleanup() {
	if [ -n "$IMAGE_ID" ] && [ -n "${ACCESS:-}" ] && [ -n "$PRODUCT_ID" ]; then
		curl -sS -o /dev/null -X DELETE \
			"$BASE/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID/images/$IMAGE_ID" \
			-H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" || true
	fi
  if [ -n "$PARTICIPANT_ID" ]; then
    set -a
    . /var/www/student-leader-portal/.env
    set +a
    docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" slc-postgres \
      psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
      -c "BEGIN;
          ALTER TABLE points_ledger DISABLE TRIGGER trg_points_ledger_append_only;
          DELETE FROM points_ledger WHERE event_participant_id='$PARTICIPANT_ID';
          DELETE FROM points_holds WHERE event_participant_id='$PARTICIPANT_ID';
          DELETE FROM merch_order_items WHERE order_id IN
            (SELECT id FROM merch_orders WHERE event_participant_id='$PARTICIPANT_ID');
          DELETE FROM merch_orders WHERE event_participant_id='$PARTICIPANT_ID';
          DELETE FROM merch_saving_targets
            WHERE event_participant_id='$PARTICIPANT_ID' OR product_id='$PRODUCT_ID';
          DELETE FROM merch_product_images WHERE product_id='$PRODUCT_ID';
          DELETE FROM merch_products WHERE id='$PRODUCT_ID';
          DELETE FROM participant_sessions WHERE event_participant_id='$PARTICIPANT_ID';
          DELETE FROM event_participants WHERE id='$PARTICIPANT_ID';
          ALTER TABLE points_ledger ENABLE TRIGGER trg_points_ledger_append_only;
          COMMIT;" \
      >/dev/null 2>&1 || true
  fi
	rm -f "$COOKIE_JAR"
	rm -f "$VALID_IMAGE" "$SPOOF_IMAGE" "$DOWNLOADED_IMAGE"
  if [ -n "$RACE_DIR" ] && [ -d "$RACE_DIR" ]; then
    rm -f "$RACE_DIR"/*.code
    rmdir "$RACE_DIR" 2>/dev/null || true
  fi
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

participant_api() {
  local method=$1 path=$2 body=${3:-} key=${4:-}
  local args=(-sS -b "$COOKIE_JAR" -w '\n%{http_code}' -X "$method" "$BASE$path" \
    -H "Origin: $ORIGIN")
  if [ -n "$body" ]; then
    args+=(-H 'Content-Type: application/json' -d "$body")
  fi
  if [ -n "$key" ]; then
    args+=(-H "Idempotency-Key: $key")
  fi
  local response
  response=$(curl "${args[@]}")
  HTTP=$(printf '%s' "$response" | tail -n1)
  BODY=$(printf '%s' "$response" | sed '$d')
}

SUFFIX="$(date +%s)-$$"
FULL_NAME="Смоук Мерч $SUFFIX"
UNION_CARD="smoke-merch-$SUFFIX"

echo "== Merch regression: $EVENT_SLUG =="
staff_login

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants" \
  "{\"full_name\":\"$FULL_NAME\",\"birth_date\":\"2000-01-02\",\"union_card_number\":\"$UNION_CARD\",\"sks_barcode\":null}"
check "create participant" "$HTTP" "201"
PARTICIPANT_ID=$(printf '%s' "$BODY" | jqget "['data']['id']")

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/participants/$PARTICIPANT_ID/points/adjustments" \
  "{\"amount\":1000,\"reason\":\"Smoke merch balance\",\"idempotency_key\":\"smoke-points-$SUFFIX\"}"
check "seed points" "$HTTP" "201"

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/merch/products" \
  "{\"title\":\"Smoke футболка $SUFFIX\",\"description\":\"Временный товар для regression\",\"price_points\":200,\"discount_price_points\":null,\"stock_quantity\":3}"
check "create product" "$HTTP" "201"
PRODUCT_ID=$(printf '%s' "$BODY" | jqget "['data']['id']")
PRODUCT_SLUG=$(printf '%s' "$BODY" | jqget "['data']['slug']")

printf 'this is not a png' >"$SPOOF_IMAGE"
response=$(curl -sS -w '\n%{http_code}' -X POST \
	"$BASE/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID/images" \
	-H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" \
	-F "image=@$SPOOF_IMAGE;type=image/png")
HTTP=$(printf '%s' "$response" | tail -n1)
BODY=$(printf '%s' "$response" | sed '$d')
check "reject spoofed image content" "$HTTP" "400"

printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=' | base64 -d >"$VALID_IMAGE"
response=$(curl -sS -w '\n%{http_code}' -X POST \
	"$BASE/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID/images" \
	-H "Origin: $ORIGIN" -H "Authorization: Bearer $ACCESS" \
	-F "image=@$VALID_IMAGE;type=image/png")
HTTP=$(printf '%s' "$response" | tail -n1)
BODY=$(printf '%s' "$response" | sed '$d')
check "accept real png signature" "$HTTP" "201"
IMAGE_ID=$(printf '%s' "$BODY" | jqget "['data']['id']")
IMAGE_URL=$(printf '%s' "$BODY" | jqget "['data']['url']")
DOWNLOAD_HTTP=$(curl -sS --connect-timeout 10 --max-time 30 \
	-o "$DOWNLOADED_IMAGE" -w '%{http_code}' "$IMAGE_URL")
check "download image from presigned S3 URL" "$DOWNLOAD_HTTP" "200"
if cmp -s "$VALID_IMAGE" "$DOWNLOADED_IMAGE"; then
	DOWNLOAD_MATCH="yes"
else
	DOWNLOAD_MATCH="no"
fi
check "downloaded image bytes match" "$DOWNLOAD_MATCH" "yes"
admin_api DELETE "/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID/images/$IMAGE_ID"
check "delete verified image" "$HTTP" "200"
IMAGE_ID=""

admin_api POST "/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID/activate"
check "activate product" "$HTTP" "200"

response=$(curl -sS -c "$COOKIE_JAR" -w '\n%{http_code}' -X POST \
  "$BASE/api/v1/events/$EVENT_SLUG/participant-auth/union-card" \
  -H "Origin: $ORIGIN" -H 'Content-Type: application/json' -d "{\"value\":\"$UNION_CARD\"}")
HTTP=$(printf '%s' "$response" | tail -n1)
BODY=$(printf '%s' "$response" | sed '$d')
check "participant login" "$HTTP" "200"

participant_api GET "/api/v1/participant/merch/$PRODUCT_SLUG"
check "open catalog product" "$HTTP" "200"

participant_api PUT "/api/v1/participant/merch-saving-target" "{\"product_id\":\"$PRODUCT_ID\"}"
check "set saving target" "$HTTP" "200"

KEY1="smoke-order-cancel-$SUFFIX"
participant_api POST "/api/v1/participant/orders" \
  "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],\"idempotency_key\":\"$KEY1\"}" "$KEY1"
check "reserve product" "$HTTP" "201"
ORDER1=$(printf '%s' "$BODY" | jqget "['data']['order']['id']")

participant_api POST "/api/v1/participant/orders" \
  "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],\"idempotency_key\":\"$KEY1\"}" "$KEY1"
check "reserve replay" "$HTTP" "200"
check "replay marker" "$(printf '%s' "$BODY" | jqget "['data']['replayed']")" "True"

participant_api POST "/api/v1/participant/orders" \
  "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":2}],\"idempotency_key\":\"$KEY1\"}" "$KEY1"
check "idempotency conflict" "$HTTP" "409"

participant_api GET "/api/v1/participant/points"
check "balance with hold" "$(printf '%s' "$BODY" | jqget "['data']['balance']['available_points']")" "800"
check "reserved points" "$(printf '%s' "$BODY" | jqget "['data']['balance']['reserved_points']")" "200"

participant_api POST "/api/v1/participant/orders/$ORDER1/cancel"
check "cancel order" "$HTTP" "201"
participant_api GET "/api/v1/participant/points"
check "cancel releases hold" "$(printf '%s' "$BODY" | jqget "['data']['balance']['reserved_points']")" "0"

KEY2="smoke-order-issue-$SUFFIX"
participant_api POST "/api/v1/participant/orders" \
  "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],\"idempotency_key\":\"$KEY2\"}" "$KEY2"
check "reserve for issue" "$HTTP" "201"
ORDER2=$(printf '%s' "$BODY" | jqget "['data']['order']['id']")
admin_api POST "/api/v1/admin/contests/$CONTEST_ID/merch/orders/$ORDER2/issue"
check "issue order" "$HTTP" "201"
admin_api POST "/api/v1/admin/contests/$CONTEST_ID/merch/orders/$ORDER2/issue"
check "issue replay" "$HTTP" "200"

participant_api GET "/api/v1/participant/points"
check "issued ledger balance" "$(printf '%s' "$BODY" | jqget "['data']['balance']['ledger_balance']")" "800"
check "issued clears hold" "$(printf '%s' "$BODY" | jqget "['data']['balance']['reserved_points']")" "0"

KEY3="smoke-order-reject-$SUFFIX"
participant_api POST "/api/v1/participant/orders" \
  "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],\"idempotency_key\":\"$KEY3\"}" "$KEY3"
check "reserve for reject" "$HTTP" "201"
ORDER3=$(printf '%s' "$BODY" | jqget "['data']['order']['id']")
admin_api POST "/api/v1/admin/contests/$CONTEST_ID/merch/orders/$ORDER3/reject" \
  '{"reason":"Smoke regression rejection"}'
check "reject order" "$HTTP" "201"

admin_api GET "/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID"
check "issued stock decremented" "$(printf '%s' "$BODY" | jqget "['data']['stock_quantity']")" "2"
check "all reservations released" "$(printf '%s' "$BODY" | jqget "['data']['reserved_quantity']")" "0"

participant_api GET "/api/v1/participant/orders"
check "participant order history" "$HTTP" "200"
check "three orders visible" "$(printf '%s' "$BODY" | jqget "['meta']['count']")" "3"

# Two units remain. Eight simultaneous reservations must create exactly two orders.
RACE_DIR="$(mktemp -d)"
for index in 1 2 3 4 5 6 7 8; do
  (
    curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' -X POST \
      "$BASE/api/v1/participant/orders" -H "Origin: $ORIGIN" \
      -H 'Content-Type: application/json' -H "Idempotency-Key: smoke-race-$SUFFIX-$index" \
      -d "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],\"idempotency_key\":\"smoke-race-$SUFFIX-$index\"}" \
      >"$RACE_DIR/$index.code"
  ) &
done
wait
RACE_CREATED=$(grep -l '^201$' "$RACE_DIR"/*.code | wc -l | tr -d ' ')
RACE_CONFLICT=$(grep -l '^409$' "$RACE_DIR"/*.code | wc -l | tr -d ' ')
check "concurrent reservations created" "$RACE_CREATED" "2"
check "concurrent reservations rejected" "$RACE_CONFLICT" "6"
admin_api GET "/api/v1/admin/contests/$CONTEST_ID/merch/products/$PRODUCT_ID"
check "concurrent stock not oversold" "$(printf '%s' "$BODY" | jqget "['data']['reserved_quantity']")" "2"
check "sold out after race" "$(printf '%s' "$BODY" | jqget "['data']['status']")" "SOLD_OUT"

echo
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
