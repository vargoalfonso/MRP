#!/bin/bash
set -euo pipefail

BASE_URL="http://localhost:8899"
EMAIL_ADDR="prlexcel_$(date +%s)@example.com"
APP_USERNAME="prlexcel$(date +%s)"
APP_PASSWORD="secret123"
IMPORT_FILE="/tmp/prl_import_test.xlsx"
EXPORT_FILE="/tmp/prl_export_test.xlsx"
EXPORT_HEADERS="/tmp/prl_export_headers.txt"

REGISTER=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" -H 'Content-Type: application/json' -d "{\"username\":\"$APP_USERNAME\",\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")
LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")
TOKEN=$(printf '%s' "$LOGIN" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","access_token") || ""')
CUSTOMER=$(curl -s -X POST "$BASE_URL/api/v1/customers" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"customer_name":"PT PRL Excel Customer","phone_number":"+62-21-4445556","shipping_address":"Jl. Excel Smoke Test No. 1","billing_address":"","billing_same_as_shipping":true,"bank_account":"BCA","bank_account_number":"44556677"}')
CUSTOMER_CODE=$(printf '%s' "$CUSTOMER" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","customer_id") || ""')

cd /Users/vargoalfonso/MRP
go run ./tmp/prlimport "$CUSTOMER_CODE" "$IMPORT_FILE" >/tmp/prl_import_gen.out 2>&1

EXPORT_STATUS=$(curl -sS -D "$EXPORT_HEADERS" -o "$EXPORT_FILE" -w '%{http_code}' -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/export")
IMPORT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/prls/import" -H "Authorization: Bearer $TOKEN" -F "file=@$IMPORT_FILE")
LIST_IMPORTED=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls?search=$CUSTOMER_CODE&page=1&limit=10")

printf 'REGISTER=%s\nLOGIN=%s\nCUSTOMER=%s\nEXPORT_STATUS=%s\nEXPORT_HEADERS=%s\nEXPORT_FILE_SIZE=%s\nIMPORT_RESPONSE=%s\nLIST_IMPORTED=%s\n' \
  "$REGISTER" "$LOGIN" "$CUSTOMER" "$EXPORT_STATUS" "$(tr '\n' ';' < "$EXPORT_HEADERS")" "$(wc -c < "$EXPORT_FILE" | tr -d ' ')" "$IMPORT_RESPONSE" "$LIST_IMPORTED"
