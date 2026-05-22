#!/bin/bash
set -e  # exit immediately if any command fails

# Configuration
API_URL="http://localhost:8080"
PROBLEM_SLUG="e2e-test-problem-final"
# coach_john assumes the coach role according to internal/http/middleware/mock_auth.go
MOCK_USER="coach_john"
TMP_DIR="tmp_e2e_dummy_tc"
TMP_ZIP="tmp_dummy_tc.zip"

# Cleanup function: runs on exit (success or failure)
cleanup() {
  echo -e "\nCleaning up temporary files..."
  rm -rf "$TMP_DIR" "$TMP_ZIP"
}
trap cleanup EXIT

echo "=================================="
echo " Starting End-to-End Tests "
echo "=================================="

echo -e "\n1. Creating a Problem..."
curl -X POST "$API_URL/problems" \
  -H "X-Mock-User: $MOCK_USER" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "'"$PROBLEM_SLUG"'",
    "title": "E2E Test Problem",
    "timeLimit": 2000,
    "memoryLimit": 256,
    "tags": ["math"]
  }'
echo ""

echo -e "\n2. Updating the Problem..."
curl -X PUT "$API_URL/problems/$PROBLEM_SLUG" \
  -H "X-Mock-User: $MOCK_USER" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "E2E Test Problem Updated",
    "timeLimit": 3000,
    "memoryLimit": 512,
    "tags": ["graphs", "dp"]
  }'
echo ""

echo -e "\n3. Creating a dummy Zip file for File Upload test..."
mkdir -p "$TMP_DIR/data/sample"
echo "1" > "$TMP_DIR/data/sample/1.in"
echo "1" > "$TMP_DIR/data/sample/1.ans"  # Parser expects .ans instead of .out
# Create zip using PowerShell (for Windows compatibility)
# We zip the 'data' folder to maintain the structure
powershell.exe -NoProfile -Command "Compress-Archive -Path ./$TMP_DIR/data -DestinationPath ./$TMP_ZIP -Force"

echo -e "\n4. Uploading Test Cases Zip..."
curl -X POST "$API_URL/problems/$PROBLEM_SLUG/files" \
  -H "X-Mock-User: $MOCK_USER" \
  -F "file=@$TMP_ZIP" \
  -F "fileType=testcases"
echo ""

echo -e "\n5. Adding a Modifier (coach_mary)..."
curl -X POST "$API_URL/problems/$PROBLEM_SLUG/modifiers" \
  -H "X-Mock-User: $MOCK_USER" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "aaaaaaaa-0000-0000-0000-000000000003"
  }'
echo ""

echo -e "\n6. Listing Modifiers..."
curl -X GET "$API_URL/problems/$PROBLEM_SLUG/modifiers" \
  -H "X-Mock-User: $MOCK_USER"
echo ""

echo -e "\n7. Removing the Modifier..."
curl -X DELETE "$API_URL/problems/$PROBLEM_SLUG/modifiers/aaaaaaaa-0000-0000-0000-000000000003" \
  -H "X-Mock-User: $MOCK_USER"
echo ""

echo -e "\n8. Deleting the Test Cases file..."
curl -X DELETE "$API_URL/problems/$PROBLEM_SLUG/files/testcases" \
  -H "X-Mock-User: $MOCK_USER"
echo ""

echo -e "\n=================================="
echo " E2E Tests Completed Successfully!"
echo "=================================="
