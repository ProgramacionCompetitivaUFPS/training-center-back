#!/bin/bash
set -e  # exit immediately if any command fails

# Configuration
API_URL="http://localhost:8080"
PROBLEM_SLUG="e2e-test-problem-final"
TMP_DIR="tmp_e2e_dummy_tc"
TMP_ZIP="tmp_dummy_tc.zip"

# Load .env if present (for ADMIN_EMAIL / ADMIN_PASSWORD, seeded by cmd/seed)
if [ -f .env ]; then
  set -a; source .env; set +a
fi
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin1234!}"
ADMIN_NICKNAME="${ADMIN_NICKNAME:-admin}"

# Unique identity per run so registration never collides with a previous run
RUN_ID=$(date +%s)
COACH_EMAIL="e2e_coach_${RUN_ID}@example.com"
COACH_NICKNAME="e2e_coach_${RUN_ID}"
COACH_PASSWORD="E2eCoach123!"

# Cleanup function: runs on exit (success or failure)
cleanup() {
  echo -e "\nCleaning up temporary files..."
  rm -rf "$TMP_DIR" "$TMP_ZIP"
}
trap cleanup EXIT

# Extracts a top-level string field ("field":"value") from a compact JSON response.
extract_field() {
  local field="$1"
  local json="$2"
  echo "$json" | grep -o "\"${field}\":\"[^\"]*\"" | head -1 | sed 's/.*:"\(.*\)"/\1/'
}

echo "=================================="
echo " Starting End-to-End Tests "
echo "=================================="

echo -e "\n0a. Registering a fresh Coach candidate ($COACH_NICKNAME)..."
curl -sf -X POST "$API_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$COACH_EMAIL"'",
    "password": "'"$COACH_PASSWORD"'",
    "name": "E2E Coach",
    "nickname": "'"$COACH_NICKNAME"'",
    "country": "test",
    "city": "test",
    "institution": "test"
  }' > /dev/null
echo "    registered."

echo -e "\n0b. Logging in as seeded admin..."
ADMIN_LOGIN_RESP=$(curl -sf -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "'"$ADMIN_EMAIL"'", "password": "'"$ADMIN_PASSWORD"'"}')
ADMIN_TOKEN=$(extract_field "token" "$ADMIN_LOGIN_RESP")
if [ -z "$ADMIN_TOKEN" ]; then
  echo "ERROR: could not log in as admin ($ADMIN_EMAIL). Run 'go run cmd/seed/main.go' first."
  exit 1
fi
echo "    admin token acquired."

echo -e "\n0c. Looking up the new user's id..."
LOOKUP_RESP=$(curl -sf -X GET "$API_URL/admin/users?searchField=nickname&searchTerm=$COACH_NICKNAME" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
COACH_ID=$(extract_field "id" "$LOOKUP_RESP")
if [ -z "$COACH_ID" ]; then
  echo "ERROR: could not find the newly registered user via GET /admin/users."
  exit 1
fi
echo "    id: $COACH_ID"

echo -e "\n0d. Promoting the new user to COACH..."
curl -sf -X PUT "$API_URL/admin/users/$COACH_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "COACH"}' > /dev/null
echo "    promoted."

echo -e "\n0e. Logging in as the new Coach..."
COACH_LOGIN_RESP=$(curl -sf -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "'"$COACH_EMAIL"'", "password": "'"$COACH_PASSWORD"'"}')
TOKEN=$(extract_field "token" "$COACH_LOGIN_RESP")
if [ -z "$TOKEN" ]; then
  echo "ERROR: could not log in as the newly promoted coach."
  exit 1
fi
echo "    coach token acquired."

echo -e "\n1. Creating a Problem..."
curl -X POST "$API_URL/problems" \
  -H "Authorization: Bearer $TOKEN" \
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
curl -X PUT "$API_URL/problems/p/$PROBLEM_SLUG" \
  -H "Authorization: Bearer $TOKEN" \
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
curl -X POST "$API_URL/problems/p/$PROBLEM_SLUG/files" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@$TMP_ZIP" \
  -F "fileType=testcases"
echo ""

echo -e "\n5. Adding a Modifier ($ADMIN_NICKNAME)..."
curl -X POST "$API_URL/problems/p/$PROBLEM_SLUG/modifiers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "userNickname": "'"$ADMIN_NICKNAME"'"
  }'
echo ""

echo -e "\n6. Listing Modifiers..."
curl -X GET "$API_URL/problems/p/$PROBLEM_SLUG/modifiers" \
  -H "Authorization: Bearer $TOKEN"
echo ""

echo -e "\n7. Removing the Modifier..."
curl -X DELETE "$API_URL/problems/p/$PROBLEM_SLUG/modifiers/$ADMIN_NICKNAME" \
  -H "Authorization: Bearer $TOKEN"
echo ""

echo -e "\n8. Deleting the Test Cases file..."
curl -X DELETE "$API_URL/problems/p/$PROBLEM_SLUG/files/testcases" \
  -H "Authorization: Bearer $TOKEN"
echo ""

echo -e "\n=================================="
echo " E2E Tests Completed Successfully!"
echo "=================================="
