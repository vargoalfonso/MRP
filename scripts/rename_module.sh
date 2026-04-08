#!/bin/bash
# rename_module.sh — initialise a new project from this template.
#
# Usage:
#   ./scripts/rename_module.sh <new-module-path> <new-service-name>
#
# Example:
#   ./scripts/rename_module.sh github.com/myorg/payment-service payment-service
#
# What it does:
#   1. Replaces every Go import path in *.go files
#   2. Updates the module declaration in go.mod
#   3. Updates SERVICE_NAME / APP_NAME in .env and ci/env.conf
#   4. Renames APP_NAME in ci/Dockerfile
#   5. Runs go mod tidy to validate

set -e

OLD_MODULE="github.com/ganasa18/go-template"
OLD_SERVICE="go-template"

NEW_MODULE="${1}"
NEW_SERVICE="${2}"

if [ -z "$NEW_MODULE" ] || [ -z "$NEW_SERVICE" ]; then
  echo "Usage: $0 <new-module-path> <new-service-name>"
  echo "Example: $0 github.com/myorg/payment-service payment-service"
  exit 1
fi

echo "Renaming module: ${OLD_MODULE} → ${NEW_MODULE}"
echo "Renaming service: ${OLD_SERVICE} → ${NEW_SERVICE}"

# ── 1. Replace import paths in all .go files ──────────────────────────────────
find . -name "*.go" -not -path "*/vendor/*" | xargs sed -i "s|${OLD_MODULE}|${NEW_MODULE}|g"
echo "  ✓ Go import paths updated"

# ── 2. Update go.mod ──────────────────────────────────────────────────────────
sed -i "s|^module ${OLD_MODULE}|module ${NEW_MODULE}|" go.mod
echo "  ✓ go.mod module declaration updated"

# ── 3. Update .env ────────────────────────────────────────────────────────────
if [ -f .env ]; then
  sed -i "s|SERVICE_NAME=\"${OLD_SERVICE}\"|SERVICE_NAME=\"${NEW_SERVICE}\"|g" .env
  sed -i "s|APP_NAME=\"${OLD_SERVICE}\"|APP_NAME=\"${NEW_SERVICE}\"|g" .env
  echo "  ✓ .env updated"
fi

# ── 4. Update ci/env.conf ─────────────────────────────────────────────────────
if [ -f ci/env.conf ]; then
  sed -i "s|SERVICE_NAME=\"${OLD_SERVICE}\"|SERVICE_NAME=\"${NEW_SERVICE}\"|g" ci/env.conf
  sed -i "s|APP_NAME=\"${OLD_SERVICE}\"|APP_NAME=\"${NEW_SERVICE}\"|g" ci/env.conf
  echo "  ✓ ci/env.conf updated"
fi

# ── 5. Update ci/Dockerfile ARG ───────────────────────────────────────────────
if [ -f ci/Dockerfile ]; then
  sed -i "s|ARG APP_NAME=${OLD_SERVICE}|ARG APP_NAME=${NEW_SERVICE}|g" ci/Dockerfile
  echo "  ✓ ci/Dockerfile updated"
fi

# ── 6. go mod tidy ────────────────────────────────────────────────────────────
echo "Running go mod tidy..."
go mod tidy
echo "  ✓ go.sum refreshed"

echo ""
echo "Done! Project is now: ${NEW_MODULE} (service: ${NEW_SERVICE})"
echo "Next steps:"
echo "  1. Update JWT_ACCESS_SECRET and JWT_REFRESH_SECRET in .env"
echo "  2. Update DB_NAME in .env"
echo "  3. Run: go build ./..."
