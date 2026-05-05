#!/usr/bin/env bash
# =============================================================================
# seed-test-user.sh
#
# Inserts a test user into the `users` DynamoDB table so you can validate
# the full login flow end-to-end without a real user database.
#
# The password stored is the bcrypt hash of:  TestPass123!
#
# Usage:
#   ./scripts/aws/seed-test-user.sh [--region us-east-1] [--table users]
# =============================================================================
set -euo pipefail

# ---------- defaults ---------------------------------------------------------
TABLE_NAME="${TABLE_NAME:-users}"
REGION="${AWS_REGION:-us-east-1}"
ENDPOINT_URL="${DYNAMODB_ENDPOINT:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --region)   REGION="$2";       shift 2 ;;
    --table)    TABLE_NAME="$2";   shift 2 ;;
    --endpoint) ENDPOINT_URL="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

AWS_CMD="aws"
if [[ -n "$ENDPOINT_URL" ]]; then
  AWS_CMD="aws --endpoint-url $ENDPOINT_URL"
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

USER_ID="00000000-0000-0000-0000-000000000001"
USER_EMAIL="test@oficina.dev"
USER_PASSWORD_PLAIN="${TEST_PASSWORD:-TestPass123!}"
USER_PASSWORD_HASH="${USER_PASSWORD_HASH:-}"

if [[ -z "$USER_PASSWORD_HASH" ]]; then
  echo "==> Generating password hash dynamically..."
  USER_PASSWORD_HASH=$(go run "$REPO_ROOT/scripts/aws/hash_password.go" "$USER_PASSWORD_PLAIN")
fi

echo "==> Region  : $REGION"
echo "==> Table   : $TABLE_NAME"
echo "==> Endpoint: ${ENDPOINT_URL:-AWS Default}"
echo "==> User ID : $USER_ID"
echo "==> Email   : $USER_EMAIL"
echo "==> Doc     : 58457673009"
echo "==> Pass    : $USER_PASSWORD_PLAIN"
echo ""

NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

$AWS_CMD dynamodb put-item \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --item "{
    \"id\":            {\"S\": \"$USER_ID\"},
    \"name\":          {\"S\": \"Test User\"},
    \"document\":      {\"S\": \"58457673009\"},
    \"document_type\": {\"S\": \"CPF\"},
    \"email\":         {\"S\": \"$USER_EMAIL\"},
    \"phone_number\":  {\"S\": \"11999999999\"},
    \"password\":      {\"S\": \"$USER_PASSWORD_HASH\"},
    \"roles\":         {\"L\": [{\"S\": \"USER\"}, {\"S\": \"ADMIN\"}]},
    \"created_at\":    {\"S\": \"$NOW\"},
    \"updated_at\":    {\"S\": \"$NOW\"}
  }"

echo "✅  Test user inserted."
echo ""
echo "Login credentials:"
echo "  CPF     : 58457673009"
echo "  Password: TestPass123!"
