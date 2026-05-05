#!/usr/bin/env bash
# =============================================================================
# create-dynamodb-table.sh
#
# Creates the `users` DynamoDB table with a GSI on `email` for the
# oficina-auth service.
#
# Usage:
#   ./scripts/aws/create-dynamodb-table.sh [--region us-east-1] [--table users]
#
# The script is idempotent: if the table already exists it exits cleanly.
# AWS credentials must already be configured (env vars, ~/.aws/credentials, or
# an instance/task role).
# =============================================================================
set -euo pipefail

# ---------- defaults ---------------------------------------------------------
TABLE_NAME="${TABLE_NAME:-users}"
EMAIL_INDEX="${EMAIL_INDEX:-email-index}"
DOCUMENT_INDEX="${DOCUMENT_INDEX:-document-index}"
REGION="${AWS_REGION:-us-east-1}"
BILLING_MODE="${BILLING_MODE:-PAY_PER_REQUEST}"   # or PROVISIONED
ENDPOINT_URL="${DYNAMODB_ENDPOINT:-}"

# ---------- parse flags ------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --region)   REGION="$2";       shift 2 ;;
    --table)    TABLE_NAME="$2";   shift 2 ;;
    --billing)  BILLING_MODE="$2"; shift 2 ;;
    --endpoint) ENDPOINT_URL="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

AWS_CMD="aws"
if [[ -n "$ENDPOINT_URL" ]]; then
  AWS_CMD="aws --endpoint-url $ENDPOINT_URL"
fi

echo "==> Region      : $REGION"
echo "==> Table       : $TABLE_NAME"
echo "==> Email GSI   : $EMAIL_INDEX"
echo "==> Doc GSI     : $DOCUMENT_INDEX"
echo "==> Billing mode: $BILLING_MODE"
echo "==> Endpoint    : ${ENDPOINT_URL:-AWS Default}"
echo ""

# ---------- check if table already exists ------------------------------------
if $AWS_CMD dynamodb describe-table \
    --table-name "$TABLE_NAME" \
    --region "$REGION" \
    --output text \
    --query "Table.TableStatus" 2>/dev/null; then
  echo "Table '$TABLE_NAME' already exists — nothing to do."
  exit 0
fi

# ---------- create table -----------------------------------------------------
echo "==> Creating table '$TABLE_NAME'..."

$AWS_CMD dynamodb create-table \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --billing-mode "$BILLING_MODE" \
  \
  --attribute-definitions \
    AttributeName=id,AttributeType=S \
    AttributeName=email,AttributeType=S \
    AttributeName=document,AttributeType=S \
  \
  --key-schema \
    AttributeName=id,KeyType=HASH \
  \
  --global-secondary-indexes "[
    {
      \"IndexName\": \"$EMAIL_INDEX\",
      \"KeySchema\": [
        { \"AttributeName\": \"email\", \"KeyType\": \"HASH\" }
      ],
      \"Projection\": { \"ProjectionType\": \"ALL\" }
    },
    {
      \"IndexName\": \"$DOCUMENT_INDEX\",
      \"KeySchema\": [
        { \"AttributeName\": \"document\", \"KeyType\": \"HASH\" }
      ],
      \"Projection\": { \"ProjectionType\": \"ALL\" }
    }
  ]" \
  --output json

# ---------- wait until ACTIVE ------------------------------------------------
echo ""
echo "==> Waiting for table to become ACTIVE..."
$AWS_CMD dynamodb wait table-exists \
  --table-name "$TABLE_NAME" \
  --region "$REGION"

echo ""
echo "✅  Table '$TABLE_NAME' is ready."
echo ""

# ---------- describe result ---------------------------------------------------
$AWS_CMD dynamodb describe-table \
  --table-name "$TABLE_NAME" \
  --region "$REGION" \
  --query "Table.{Status:TableStatus, ItemCount:ItemCount, GSI:GlobalSecondaryIndexes[*].{Name:IndexName,Status:IndexStatus}}" \
  --output table
