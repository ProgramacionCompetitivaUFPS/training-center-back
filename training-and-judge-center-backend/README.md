# Training & Judge Center â€” Backend

## Requisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) instalado y corriendo
- [Go 1.25+](https://go.dev/dl/) (solo si deseas ejecutar sin Docker)

## EjecuciÃ³n con Docker Compose

Un solo comando levanta PostgreSQL, ejecuta migraciones y arranca el backend:

```bash
docker-compose --env-file .env up --build -d
```

Verifica que todo estÃ© corriendo:

```bash
docker ps
```

DeberÃ­as ver `training-center-db` y `training-center-api` en estado `Up`.

### Detener los servicios

```bash
docker-compose --env-file .env.example down
```

Para eliminar tambiÃ©n los datos de PostgreSQL:

```bash
docker-compose --env-file .env.example down -v
```

## EjecuciÃ³n local (sin Docker para el backend)

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

## AutenticaciÃ³n

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

2. Iniciar sesiÃ³n para obtener el token:

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

## Tags vÃ¡lidos

Los tags se validan contra `config/virtual_object.json`:

`math`, `beginner`, `dp`, `graphs`, `greedy`, `strings`

# Swagger

Para actualizar swagger usar: 
```powershell
swag init -g cmd/api/main.go -o docs
```
