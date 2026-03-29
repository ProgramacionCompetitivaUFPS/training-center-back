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

3. Iniciar el servidor con mock auth:

```bash
MOCK_AUTH=1 go run cmd/api/main.go
```

En PowerShell:

```powershell
$env:MOCK_AUTH="1"; go run cmd/api/main.go
```

## Mock Auth

Con `MOCK_AUTH=1`, el middleware inyecta un usuario simulado vía el header `X-Mock-User`.

Usuarios disponibles:

| Header Value   | Rol         |
|----------------|-------------|
| `admin`        | ADMIN       |
| `coach_john`   | COACH       |
| `coach_mary`   | COACH       |
| `contestant`   | CONTESTANT  |

Si no envías el header, se usa `coach_john` por defecto.

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
  -H "X-Mock-User: coach_john" \
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
  -H "X-Mock-User: coach_john" \
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
  -H "X-Mock-User: admin" \
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
  -H "X-Mock-User: contestant" \
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
