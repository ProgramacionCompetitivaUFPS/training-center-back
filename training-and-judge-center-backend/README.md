# Training & Judge Center — Backend

## Requisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) instalado y corriendo
- [Go 1.25+](https://go.dev/dl/) (solo si deseas ejecutar sin Docker)

## Ejecución con Docker Compose

Un solo comando levanta PostgreSQL, ejecuta migraciones y arranca el backend:

```bash
docker-compose --env-file .env up --build -d
```

Verifica que todo esté corriendo:

```bash
docker ps
```

Deberías ver `training-center-db` y `training-center-api` en estado `Up`.

### Detener los servicios

```bash
docker-compose --env-file .env.example down
```

Para eliminar también los datos de PostgreSQL:

```bash
docker-compose --env-file .env.example down -v
```

## Ejecución local (sin Docker para el backend)

1. Levantar solo PostgreSQL:

```bash
docker-compose --env-file .env.example up postgres -d
```

2. Ejecutar migraciones:

```bash
go run cmd/migrate/main.go up
```

3. Iniciar el servidor:

```bash
go run cmd/api/main.go
```

## Autenticación

No hay modo mock: todos los endpoints protegidos requieren un JWT real obtenido por login.

1. Registrar un usuario (rol `CONTESTANT` por defecto):

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "coach@example.com",
    "password": "Coach1234!",
    "name": "Coach Example",
    "nickname": "coach_example",
    "country": "co",
    "city": "city",
    "institution": "institution"
  }'
```

2. Iniciar sesión para obtener el token:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "coach@example.com", "password": "Coach1234!"}'
```

La respuesta incluye `{"token": "...", "user": {...}}`. Usa ese token en cada request protegida:

```bash
curl http://localhost:8080/users/me \
  -H "Authorization: Bearer <token>"
```

Para promover un usuario a `COACH` o `ADMIN` (solo un Admin puede hacerlo):

```bash
curl -X PUT http://localhost:8080/admin/users/{id} \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "COACH"}'
```

El primer Admin se crea con `go run cmd/seed/main.go` (lee `ADMIN_EMAIL`/`ADMIN_PASSWORD` del `.env`, es idempotente por email).

## Tags válidos

Los tags se validan contra `config/virtual_object.json`:

`math`, `beginner`, `dp`, `graphs`, `greedy`, `strings`

## API Endpoints

### Health Check

```
GET /ping
```

```bash
curl http://localhost:8080/ping
```

---

### Crear Problema

```
POST /problems
```

#### Crear problema básico

```bash
curl -X POST http://localhost:8080/problems \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -d '{
    "slug": "two-sum",
    "title": "Two Sum",
    "tags": ["math", "dp"]
  }'
```

#### Crear problema completo

```bash
curl -X POST http://localhost:8080/problems \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -d '{
    "slug": "fibonacci-sequence",
    "title": "Fibonacci Sequence",
    "statement": "Given an integer N, compute the N-th Fibonacci number.",
    "timeLimit": 2000,
    "memoryLimit": 256,
    "tags": ["math", "dp", "beginner"],
    "languageOverrides": [
      {
        "language": "python310",
        "timeLimit": 4000
      },
      {
        "language": "java17",
        "timeLimit": 3000,
        "memoryLimit": 512
      }
    ]
  }'
```

#### Crear problema como admin

```bash
curl -X POST http://localhost:8080/problems \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "slug": "graph-bfs",
    "title": "BFS on Graph",
    "statement": "Perform a BFS traversal on a given graph.",
    "timeLimit": 5000,
    "memoryLimit": 512,
    "tags": ["graphs"]
  }'
```

### Respuestas esperadas

#### 201 Created — Problema creado exitosamente

```json
{
  "slug": "two-sum",
  "title": "Two Sum",
  "statement": null,
  "timeLimit": null,
  "memoryLimit": null,
  "languageOverrides": [],
  "tags": ["math", "dp"],
  "status": "DRAFT",
  "accessibility": "PRIVATE",
  "author": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "modifiers": [],
  "files": {
    "testCases": false,
    "solutions": [],
    "checker": false,
    "validator": false
  },
  "createdAt": "2026-03-09T00:03:52Z",
  "updatedAt": "2026-03-09T00:03:52Z"
}
```

#### 400 Validation Error — Datos inválidos

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    { "field": "tags", "message": "Invalid tag: invalid-tag" }
  ]
}
```

#### 403 Forbidden — Sin permisos (contestant)

```bash
curl -X POST http://localhost:8080/problems \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CONTESTANT_TOKEN" \
  -d '{
    "slug": "test",
    "title": "Test",
    "tags": ["math"]
  }'
```

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Coach and Admin users can create problems"
}
```

#### 409 Conflict — Slug duplicado

```json
{
  "error": "SLUG_ALREADY_EXISTS",
  "message": "A problem with that slug already exists"
}
```

---

### Actualizar Problema

```
PUT /problems/p/{slug}
```

Todos los campos son opcionales (solo se actualiza lo que se envía). El problema debe estar en estado `DRAFT`.

```bash
curl -X PUT http://localhost:8080/problems/p/two-sum \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -d '{
    "title": "Two Sum (Updated)",
    "statement": "Given an array of integers, return indices of the two numbers that add up to target.",
    "timeLimit": 3000,
    "memoryLimit": 512,
    "tags": ["math", "dp"],
    "accessibility": "PUBLIC",
    "languageOverrides": [
      {
        "language": "python310",
        "timeLimit": 6000
      }
    ]
  }'
```

**Respuestas esperadas:**

- `200 OK` — problema actualizado (mismo schema que Create)
- `403 Forbidden` — no es autor, admin ni modifier
- `404 Not Found` — slug no existe
- `400 Bad Request` — problema publicado (debe despublicarse primero)

---

### Importar Problema (paquete ICPC)

```
POST /problems/import
```

Acepta `multipart/form-data` con un `.zip` en formato ICPC (con `problem.xml` y archivos de test).

```bash
curl -X POST http://localhost:8080/problems/import \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -F "slug=icpc-problem-a" \
  -F "file=@/path/to/package.zip"
```

**Respuestas esperadas:**

- `201 Created` — problema creado (mismo schema que Create)
- `400 Bad Request` — slug o archivo faltante, ZIP inválido
- `413 Payload Too Large` — ZIP excede el límite

---

### Subir Archivos

```
POST /problems/p/{slug}/files
```

Acepta `multipart/form-data`. El campo `fileType` determina qué tipo de archivo se sube:

| `fileType`  | Descripción                        |
|-------------|------------------------------------|
| `testcases` | ZIP con casos de prueba            |
| `solution`  | Archivo de solución (C++, Java, Python) |
| `checker`   | Checker personalizado              |
| `validator` | Validador de entrada               |

```bash
# Subir casos de prueba
curl -X POST http://localhost:8080/problems/p/two-sum/files \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -F "fileType=testcases" \
  -F "file=@/path/to/tests.zip"

# Subir solución
curl -X POST http://localhost:8080/problems/p/two-sum/files \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -F "fileType=solution" \
  -F "file=@/path/to/solution.cpp"
```

**Respuesta `200 OK`:**

```json
{
  "message": "File uploaded successfully",
  "problem": { "...": "mismo schema que Create" }
}
```

---

### Eliminar Archivo

```
DELETE /problems/p/{slug}/files/{fileType}
```

Para `solution`, el filename es requerido via query param `?fileName=`.

```bash
# Eliminar casos de prueba
curl -X DELETE http://localhost:8080/problems/p/two-sum/files/testcases \
  -H "Authorization: Bearer $COACH_TOKEN"

# Eliminar una solución específica
curl -X DELETE "http://localhost:8080/problems/p/two-sum/files/solution?fileName=solution.cpp" \
  -H "Authorization: Bearer $COACH_TOKEN"

# Eliminar checker
curl -X DELETE http://localhost:8080/problems/p/two-sum/files/checker \
  -H "Authorization: Bearer $COACH_TOKEN"
```

**Respuestas esperadas:**

- `204 No Content` — archivo eliminado
- `403 Forbidden` — sin permisos
- `404 Not Found` — problema no existe

---

### Agregar Modifier

```
POST /problems/p/{slug}/modifiers
```

Solo el autor o un admin puede agregar modifiers.

```bash
curl -X POST http://localhost:8080/problems/p/two-sum/modifiers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $COACH_TOKEN" \
  -d '{
    "userNickname": "coach_mary"
  }'
```

**Respuestas esperadas:**

- `200 OK` — modifier agregado (body vacío)
- `403 Forbidden` — no es autor ni admin
- `404 Not Found` — usuario o problema no existe
- `409 Conflict` — el usuario ya es modifier

---

### Listar Modifiers

```
GET /problems/p/{slug}/modifiers
```

```bash
curl http://localhost:8080/problems/p/two-sum/modifiers \
  -H "Authorization: Bearer $COACH_TOKEN"
```

**Respuesta `200 OK`:**

```json
{
  "modifiers": ["coach_mary"]
}
```

---

### Eliminar Modifier

```
DELETE /problems/p/{slug}/modifiers/{nickname}
```

```bash
curl -X DELETE http://localhost:8080/problems/p/two-sum/modifiers/coach_mary \
  -H "Authorization: Bearer $COACH_TOKEN"
```

**Respuestas esperadas:**

- `200 OK` — modifier eliminado (body vacío)
- `403 Forbidden` — no es autor ni admin
- `404 Not Found` — modifier no estaba asignado
