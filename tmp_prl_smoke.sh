#!/bin/zsh
set -euo pipefail

EMAIL_ADDR="prlcheck_$(date +%s)@example.com"
APP_USERNAME="prlcheck$(date +%s)"
APP_PASSWORD="secret123"
BASE_URL="http://localhost:8899"

REGISTER=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" -H 'Content-Type: application/json' -d "{\"username\":\"$APP_USERNAME\",\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")
LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_ADDR\",\"password\":\"$APP_PASSWORD\"}")
TOKEN=$(printf '%s' "$LOGIN" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","access_token") || ""')

CUSTOMER=$(curl -s -X POST "$BASE_URL/api/v1/customers" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"customer_name":"PT PRL Smoke Customer","phone_number":"+62-21-1112223","shipping_address":"Jl. Smoke Test No. 1","billing_address":"","billing_same_as_shipping":true,"bank_account":"BCA","bank_account_number":"11223344"}')
CUSTOMER_ID=$(printf '%s' "$CUSTOMER" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","id") || ""')
UNIQ_CODE="PRL-UNIQ-$(date +%s)"
UNIQ_BOM=$(curl -s -X POST "$BASE_URL/api/v1/uniq-boms" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"uniq_code\":\"$UNIQ_CODE\",\"product_model\":\"Camry 2026\",\"part_name\":\"Engine Mount Bracket\",\"part_number\":\"EM-PRL-001\"}")

LOOKUP_CUSTOMERS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/customers")
LOOKUP_BOMS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/uniq-boms?page=1&limit=5")
LOOKUP_PERIODS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/lookups/forecast-periods?year=2026")

BULK_CREATE=$(curl -s -X POST "$BASE_URL/api/v1/prls/bulk" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"entries\":[{\"customer_uuid\":\"$CUSTOMER_ID\",\"uniq_code\":\"$UNIQ_CODE\",\"forecast_period\":\"2026-Q1\",\"quantity\":2500},{\"customer_uuid\":\"$CUSTOMER_ID\",\"uniq_code\":\"$UNIQ_CODE\",\"forecast_period\":\"2026-Q2\",\"quantity\":1800}]}")
PRL_ID=$(printf '%s' "$BULK_CREATE" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","items",0,"id") || ""')
PRL_IDS=$(printf '%s' "$BULK_CREATE" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print JSON.generate((j.dig("data","items") || []).map{|x| x["id"]})')

LIST_PRLS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls?page=1&limit=10")
GET_PRL=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/prls/$PRL_ID")
UPDATE_PRL=$(curl -s -X PATCH "$BASE_URL/api/v1/prls/$PRL_ID" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"forecast_period":"2026-Q3","quantity":4200}')
APPROVE=$(curl -s -X POST "$BASE_URL/api/v1/prls/actions/approve" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"ids\":$PRL_IDS}")

REJECT_CANDIDATE=$(curl -s -X POST "$BASE_URL/api/v1/prls/bulk" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"entries\":[{\"customer_uuid\":\"$CUSTOMER_ID\",\"uniq_code\":\"$UNIQ_CODE\",\"forecast_period\":\"2026-Q4\",\"quantity\":900}]}")
REJECT_ID=$(printf '%s' "$REJECT_CANDIDATE" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print j.dig("data","items",0,"id") || ""')
REJECT_IDS=$(printf '%s' "$REJECT_CANDIDATE" | ruby -rjson -e 'j=JSON.parse(STDIN.read); print JSON.generate((j.dig("data","items") || []).map{|x| x["id"]})')
REJECT=$(curl -s -X POST "$BASE_URL/api/v1/prls/actions/reject" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"ids\":$REJECT_IDS}")
DELETE_REJECT=$(curl -s -X DELETE "$BASE_URL/api/v1/prls/$REJECT_ID" -H "Authorization: Bearer $TOKEN")

printf 'REGISTER=%s\nLOGIN=%s\nCUSTOMER=%s\nUNIQ_BOM=%s\nLOOKUP_CUSTOMERS=%s\nLOOKUP_BOMS=%s\nLOOKUP_PERIODS=%s\nBULK_CREATE=%s\nLIST_PRLS=%s\nGET_PRL=%s\nUPDATE_PRL=%s\nAPPROVE=%s\nREJECT_CANDIDATE=%s\nREJECT=%s\nDELETE_REJECT=%s\n' \
  "$REGISTER" "$LOGIN" "$CUSTOMER" "$UNIQ_BOM" "$LOOKUP_CUSTOMERS" "$LOOKUP_BOMS" "$LOOKUP_PERIODS" "$BULK_CREATE" "$LIST_PRLS" "$GET_PRL" "$UPDATE_PRL" "$APPROVE" "$REJECT_CANDIDATE" "$REJECT" "$DELETE_REJECT"
