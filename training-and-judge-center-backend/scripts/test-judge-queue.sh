#!/usr/bin/env bash
# Prueba la cola RabbitMQ del worker end-to-end.
#
# Requisitos previos (en terminales separadas):
#   1. docker-compose --env-file .env up postgres rabbitmq seed -d
#   2. Desde PowerShell, con las env vars del .env:
#        $env:RABBITMQ_URL="amqp://judge:judge@localhost:5672/"
#        $env:DB_HOST="localhost"; $env:DB_PORT="5432"
#        $env:DB_USER="postgres"; $env:DB_PASSWORD="postgres"; $env:DB_NAME="training_center"
#        $env:STORAGE_BACKEND="local"; $env:STORAGE_LOCAL_DIR=".local_storage"
#        $env:POD_MEMORY_LIMIT="4294967296"
#        go run ./cmd/worker
#
# Uso (desde Git Bash, en training-and-judge-center-backend/):
#   bash scripts/test-judge-queue.sh
set -euo pipefail

# --- Cargar .env si existe ---
if [ -f .env ]; then
    set -a; source .env; set +a
fi

PG_CONTAINER="${POSTGRES_CONTAINER:-training-center-db}"
PG_USER="${POSTGRES_USER:-postgres}"
PG_DB="${POSTGRES_DB:-training_center}"
STORAGE_DIR="${STORAGE_LOCAL_DIR:-.local_storage}"
RABBITMQ_MGMT="http://localhost:15672"
RABBITMQ_USER="${RABBITMQ_DEFAULT_USER:-judge}"
RABBITMQ_PASS="${RABBITMQ_DEFAULT_PASS:-judge}"

_psql() {
    docker exec "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" "$@"
}

# UUID usando PowerShell (no requiere Python)
_uuid() {
    powershell -NoProfile -Command "[guid]::NewGuid().ToString().ToLower()" | tr -d '\r\n'
}

echo "==> 1. Obteniendo UUID del usuario admin..."
ADMIN_UUID=$(_psql -t -A -c "SELECT id FROM users WHERE role = 'ADMIN' LIMIT 1" | tr -d '[:space:]')
if [ -z "$ADMIN_UUID" ]; then
    echo "ERROR: No hay usuario admin. Ejecuta: docker-compose --env-file .env up seed -d"
    exit 1
fi
echo "    admin UUID: $ADMIN_UUID"

PROBLEM_ID="aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
TEST_CASES_KEY="${PROBLEM_ID}/test_cases.zip"

echo "==> 2. Creando problema de prueba en la DB..."
_psql -q -c "
    INSERT INTO problems (id, slug, title, status, accessibility, time_limit, memory_limit, author_id, test_cases_key)
    VALUES ('$PROBLEM_ID', 'judge-queue-test', 'Judge Queue Test', 'PUBLISHED', 'PUBLIC', 2000, 256, '$ADMIN_UUID', '$TEST_CASES_KEY')
    ON CONFLICT (id) DO UPDATE
        SET status = 'PUBLISHED', test_cases_key = EXCLUDED.test_cases_key;
"
echo "    problema: $PROBLEM_ID"

echo "==> 3. Creando ZIP de casos de prueba (2+3=5)..."
mkdir -p "$STORAGE_DIR/$PROBLEM_ID"
WIN_STORAGE_DIR=$(cygpath -w "$STORAGE_DIR/$PROBLEM_ID")
WIN_ZIP_PATH=$(cygpath -w "$STORAGE_DIR/$TEST_CASES_KEY")

powershell -NoProfile -Command "
    Add-Type -Assembly 'System.IO.Compression.FileSystem'
    \$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ('judge_tc_' + [System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path (Join-Path \$tmp 'data\sample') -Force | Out-Null
    [System.IO.File]::WriteAllText((Join-Path \$tmp 'data\sample\001.in'),  \"2\`n3\`n\")
    [System.IO.File]::WriteAllText((Join-Path \$tmp 'data\sample\001.ans'), \"5\`n\")
    if (Test-Path '$WIN_ZIP_PATH') { Remove-Item '$WIN_ZIP_PATH' }
    [System.IO.Compression.ZipFile]::CreateFromDirectory(\$tmp, '$WIN_ZIP_PATH')
    Remove-Item \$tmp -Recurse -Force
"
echo "    ZIP: $STORAGE_DIR/$TEST_CASES_KEY"

echo "==> 4. Generando ID de submission y escribiendo código fuente C++..."
SUBMISSION_ID=$(_uuid)
SOURCE_PATH="${PROBLEM_ID}/${ADMIN_UUID}/general/${SUBMISSION_ID}.cpp"
mkdir -p "$STORAGE_DIR/${PROBLEM_ID}/${ADMIN_UUID}/general"

cat > "$STORAGE_DIR/$SOURCE_PATH" << 'CPPEOF'
#include <iostream>
int main() {
    int a, b;
    std::cin >> a >> b;
    std::cout << a + b << std::endl;
}
CPPEOF
echo "    submission ID: $SUBMISSION_ID"
echo "    source: $STORAGE_DIR/$SOURCE_PATH"

echo "==> 5. Insertando submission PENDING en la DB..."
NOW=$(date -u +"%Y-%m-%d %H:%M:%S+00")
_psql -q -c "
    INSERT INTO submissions
        (id, problem_id, user_id, contest_id, standing_id,
         language, compiler, status, visibility, source_code_path,
         file_hash, file_size, submitted_at, problem_title, problem_slug)
    VALUES
        ('$SUBMISSION_ID', '$PROBLEM_ID', '$ADMIN_UUID', NULL, NULL,
         'cpp20', 'g++', 'PENDING', 'PRIVATE', '$SOURCE_PATH',
         NULL, NULL, '$NOW', 'Judge Queue Test', 'judge-queue-test');
"

echo "==> 6. Publicando mensaje en RabbitMQ..."
ENQUEUED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
PAYLOAD="{\"submissionId\":\"$SUBMISSION_ID\",\"priority\":1,\"enqueuedAt\":\"$ENQUEUED_AT\",\"metadata\":{\"problemId\":\"$PROBLEM_ID\",\"userId\":\"$ADMIN_UUID\",\"language\":\"cpp20\"}}"

# Escapar las comillas del payload para meterlo como string JSON
PAYLOAD_ESCAPED=$(printf '%s' "$PAYLOAD" | sed 's/\\/\\\\/g; s/"/\\"/g')

HTTP_CODE=$(curl -s -o /tmp/rabbit_publish.json -w "%{http_code}" \
    -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
    -H "Content-Type: application/json" \
    -X POST "${RABBITMQ_MGMT}/api/exchanges/%2F/amq.default/publish" \
    -d "{\"properties\":{\"delivery_mode\":2,\"content_type\":\"application/json\",\"priority\":1},\"routing_key\":\"submissions\",\"payload\":\"$PAYLOAD_ESCAPED\",\"payload_encoding\":\"string\"}")

if [ "$HTTP_CODE" != "200" ]; then
    echo "ERROR: RabbitMQ management API respondió HTTP $HTTP_CODE"
    cat /tmp/rabbit_publish.json 2>/dev/null
    exit 1
fi

ROUTED=$(grep -o '"routed":[^,}]*' /tmp/rabbit_publish.json | grep -o 'true\|false' || echo "false")
if [ "$ROUTED" = "true" ]; then
    echo "    Mensaje publicado y enrutado a la cola 'submissions'. ✓"
else
    echo "    ADVERTENCIA: mensaje publicado pero NO enrutado."
    echo "    Asegúrate de que el worker esté corriendo ANTES de ejecutar este script"
    echo "    (el worker declara la cola al conectarse)."
fi

echo ""
echo "======================================================"
echo " Submission ID: $SUBMISSION_ID"
echo "======================================================"
echo ""
echo "El worker debería procesar el mensaje. Verifica el resultado:"
echo ""
echo "  docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB \\"
echo "    -c \"SELECT id, status, time_ms, memory_kb, compile_log FROM submissions WHERE id = '$SUBMISSION_ID'\""
