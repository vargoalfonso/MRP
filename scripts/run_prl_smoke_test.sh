#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8899"
EMAIL_ADDR="prltest_$(date +%s)@example.com"
APP_USERNAME="prltestuser$(date +%s)"
APP_PASSWORD="secret123"
IMPORT_FILE="/tmp/prl-import-test.xlsx"

json_get() {
  local expression="$1"
  ruby -rjson -e "j=JSON.parse(STDIN.read); v=${expression}; case v; when NilClass then print ''; when String then print v; else print JSON.generate(v); end"
}

REGISTER=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$APP_USERNAME\",\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")

LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")
TOKEN=$(printf '%s' "$LOGIN" | json_get 'j.dig("data","access_token")')

CUSTOMER=$(curl -s -X POST "$BASE_URL/api/v1/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"customer_name":"PT PRL Smoke Customer","phone_number":"+62-21-7778889","shipping_address":"Jl. PRL Customer No. 1","billing_address":"","billing_same_as_shipping":true,"bank_account":"BCA","bank_account_number":"11223344"}')
CUSTOMER_UUID=$(printf '%s' "$CUSTOMER" | json_get 'j.dig("data","id")')
CUSTOMER_CODE=$(printf '%s' "$CUSTOMER" | json_get 'j.dig("data","customer_id")')

UNIQ_CODE="PRL-UNIQ-$(date +%s)"
UNIQ_BOM=$(curl -s -X POST "$BASE_URL/api/v1/uniq-boms" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"uniq_code\":\"$UNIQ_CODE\",\"product_model\":\"Camry 2026\",\"part_name\":\"Engine Mount Bracket\",\"part_number\":\"EM-PRL-001\"}")
UNIQ_BOM_UUID=$(printf '%s' "$UNIQ_BOM" | json_get 'j.dig("data","id")')
UNIQ_CODE_ACTUAL=$(printf '%s' "$UNIQ_BOM" | json_get 'j.dig("data","uniq_code")')

LOOKUP_CUSTOMERS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/customers")
LOOKUP_UNIQ_BOMS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/uniq-boms?page=1&limit=10")
LOOKUP_PERIODS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/forecast-periods?year=2026")

BULK_CREATE=$(curl -s -X POST "$BASE_URL/api/v1/prls/bulk" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"entries\":[{\"customer_uuid\":\"$CUSTOMER_UUID\",\"uniq_code\":\"$UNIQ_CODE_ACTUAL\",\"forecast_period\":\"2026-Q1\",\"quantity\":2500},{\"customer_uuid\":\"$CUSTOMER_UUID\",\"uniq_code\":\"$UNIQ_CODE_ACTUAL\",\"forecast_period\":\"2026-Q2\",\"quantity\":3000}]}")
PRL_UUID=$(printf '%s' "$BULK_CREATE" | json_get 'j.dig("data","items",0,"id")')
PRL_IDS_JSON=$(printf '%s' "$BULK_CREATE" | json_get 'j.dig("data","items")&.map{|x| x["id"] }')

LIST_PRLS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls?page=1&limit=10")
GET_PRL=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/$PRL_UUID")
UPDATE_PRL=$(curl -s -X PATCH "$BASE_URL/api/v1/prls/$PRL_UUID" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"forecast_period":"2026-Q3","quantity":4200}')
APPROVE_PRLS=$(curl -s -X POST "$BASE_URL/api/v1/prls/actions/approve" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"ids\":$PRL_IDS_JSON}")

REJECT_CANDIDATE=$(curl -s -X POST "$BASE_URL/api/v1/prls/bulk" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"entries\":[{\"customer_uuid\":\"$CUSTOMER_UUID\",\"uniq_code\":\"$UNIQ_CODE_ACTUAL\",\"forecast_period\":\"2026-Q4\",\"quantity\":1800}]}")
REJECT_PRL_UUID=$(printf '%s' "$REJECT_CANDIDATE" | json_get 'j.dig("data","items",0,"id")')
REJECT_IDS_JSON=$(printf '%s' "$REJECT_CANDIDATE" | json_get 'j.dig("data","items")&.map{|x| x["id"] }')
REJECT_PRLS=$(curl -s -X POST "$BASE_URL/api/v1/prls/actions/reject" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"ids\":$REJECT_IDS_JSON}")
DELETE_PRL=$(curl -s -X DELETE "$BASE_URL/api/v1/prls/$REJECT_PRL_UUID" \
  -H "Authorization: Bearer $TOKEN")

EXPORT_HEADERS=$(curl -s -D - -o /tmp/prl-export-test.xlsx \
  -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/api/v1/prls/export")

CUSTOMER_CODE="$CUSTOMER_CODE" IMPORT_FILE="$IMPORT_FILE" /Users/vargoalfonso/MRP/.venv/bin/python <<'PY'
from openpyxl import Workbook
import os

customer_code = os.environ["CUSTOMER_CODE"]
output_path = os.environ["IMPORT_FILE"]

wb = Workbook()
ws = wb.active
ws.append(["customer_code", "uniq_code", "forecast_period", "quantity"])
ws.append([customer_code, "LV7-001", "2026-Q1", 1200])
ws.append([customer_code, "LV7-002", "2026-Q2", 900])
wb.save(output_path)
PY

IMPORT_PRLS=$(curl -s -X POST "$BASE_URL/api/v1/prls/import" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@$IMPORT_FILE;type=application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

printf 'REGISTER=%s\n' "$REGISTER"
printf 'LOGIN=%s\n' "$LOGIN"
printf 'CUSTOMER=%s\n' "$CUSTOMER"
printf 'UNIQ_BOM=%s\n' "$UNIQ_BOM"
printf 'LOOKUP_CUSTOMERS=%s\n' "$LOOKUP_CUSTOMERS"
printf 'LOOKUP_UNIQ_BOMS=%s\n' "$LOOKUP_UNIQ_BOMS"
printf 'LOOKUP_PERIODS=%s\n' "$LOOKUP_PERIODS"
printf 'BULK_CREATE=%s\n' "$BULK_CREATE"
printf 'LIST_PRLS=%s\n' "$LIST_PRLS"
printf 'GET_PRL=%s\n' "$GET_PRL"
printf 'UPDATE_PRL=%s\n' "$UPDATE_PRL"
printf 'APPROVE_PRLS=%s\n' "$APPROVE_PRLS"
printf 'REJECT_CANDIDATE=%s\n' "$REJECT_CANDIDATE"
printf 'REJECT_PRLS=%s\n' "$REJECT_PRLS"
printf 'DELETE_PRL=%s\n' "$DELETE_PRL"
printf 'EXPORT_HEADERS=%s\n' "$EXPORT_HEADERS"
printf 'IMPORT_PRLS=%s\n' "$IMPORT_PRLS"
