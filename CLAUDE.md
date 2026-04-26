# CLAUDE.md

## Commands

```bash
go run cmd/api/main.go                  # run server
go run cmd/migrate/main.go up|down|status
go test ./...
go test -v ./internal/domain/problem/...
go test -v ./internal/application/problem/...
docker-compose --env-file .env up --build -d
bash tests/test_e2e.sh                  # requires running server
```

**Dev env vars:**
- `MOCK_AUTH=1` — reads user from `X-Mock-User` header
- `STORAGE_BACKEND=local` — local filesystem instead of GCS

## Architecture

Go backend (DDD + Hexagonal). Layers: `domain` → `application` → `platform` / `infrastructure` → `server`.

- **`infrastructure/`** — pure technology, zero domain imports. Wraps external libraries (postgres connection, querier, transactions).
- **`platform/`** — domain-aware adapters, organized by domain (`problem/`, `user/`, `auth/`) or cross-cutting concern (`email/`, `ratelimit/`, `config/`). Implements ports defined in `domain/` and `application/`.
- **Rule:** if a file imports from `domain/` or `application/`, it goes in `platform/`. If not, it goes in `infrastructure/`.

**Key conventions:**
- Use cases: `Execute(ctx, input) (output, error)`
- Domain construction: `New*()` validates and returns `AppError` on failure
- DB reconstruction: `Restore*()` bypasses validation — trust the DB
- Ports (interfaces for external deps): viven donde los necesita quien los consume.
  - Puertos de **use cases**: en `application/<domain>/ports.go` (ej. `UserProvider`, `ZipParser` en `problem`).
  - Puertos de **dominio**: en archivos propios dentro de `domain/<domain>/` (ej. `TokenService`, `SessionInvalidator`, `TransactionManager` en `user` — son contratos que el dominio mismo define).
  - DTOs compartidos entre use cases de un mismo dominio: en `application/<domain>/dto.go` (ej. `UserDTO` en `user`).
- Handlers: `handler.go` define el struct con `*UseCase` concretos (no interfaces); un archivo `*_handler.go` por operación; `types.go` para DTOs de respuesta compartidos dentro del mismo handler package. En `main.go`, los handler packages siempre se importan con alias `handler<Domain>` (ej. `handlerProblem`, `handlerUser`) para simetría.
- Errors: always via `pkg/apperror`

## Non-obvious decisions

**`ListFilters.Statuses` (repository):** empty = no status filter. Visibility of DRAFTs is controlled by `ViewerModifierID`. Don't pre-fill with all statuses from the use case.

**HTTP responses:** `getProblemResponse` is the single problem response type. Use `buildResponse(p, display)` in `types.go` for mutation endpoints (Create/Update/Import/UploadFiles). If an endpoint needs a different shape, build it directly in the handler — don't add parameters to `buildResponse`.

**Cross-domain primitives:** `UserID` and `CurrentUser` live in `domain/shared/`. Each domain defines its own local types for display/enrichment (e.g., `application/problem/ports.go` defines `UserDisplay` locally). Platform adapters per domain implement the local port (e.g., `platform/problem/user_provider.go` queries the `users` table). Don't import `domain/user` from other domains.

**Dominios en progreso:** `domain/group/` y `domain/material/` tienen implementación de dominio + tests, pero sin application layer ni handlers todavía. `application/contest/`, `application/submission/`, `platform/mongo/`, `platform/queue/`, `platform/storage/` son directorios placeholder para trabajo futuro — están vacíos intencionalmente.
