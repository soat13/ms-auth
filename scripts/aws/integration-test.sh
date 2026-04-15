#!/usr/bin/env bash
# =============================================================================
# integration-test.sh
#
# Tests the DynamoDB UserReader end-to-end against the real AWS table.
# Runs a small Go program that:
#   1. Connects to DynamoDB via AWS SDK (uses default credential chain)
#   2. Calls GetByDocument with a known document
#   3. Validates password match
#
# Usage:
#   ./scripts/aws/integration-test.sh [--region us-east-1] [--table users]
# =============================================================================
set -euo pipefail

TABLE_NAME="${TABLE_NAME:-users}"
DOCUMENT_INDEX="${DOCUMENT_INDEX:-document-index}"
REGION="${AWS_REGION:-us-east-1}"
TEST_DOCUMENT="${TEST_DOCUMENT:-12345678909}"
TEST_PASSWORD="${TEST_PASSWORD:-TestPass123!}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --region)   REGION="$2";       shift 2 ;;
    --table)    TABLE_NAME="$2";   shift 2 ;;
    --document) TEST_DOCUMENT="$2";shift 2 ;;
    --password) TEST_PASSWORD="$2";shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "==> Region  : $REGION"
echo "==> Table   : $TABLE_NAME"
echo "==> GSI     : $DOCUMENT_INDEX"
echo "==> Document: $TEST_DOCUMENT"
echo ""

# Run the integration test program
AWS_REGION="$REGION" \
TABLE_NAME="$TABLE_NAME" \
DOCUMENT_INDEX="$DOCUMENT_INDEX" \
TEST_DOCUMENT="$TEST_DOCUMENT" \
TEST_PASSWORD="$TEST_PASSWORD" \
  go run "$REPO_ROOT/scripts/aws/integration_test_runner.go"
