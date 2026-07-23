# CLAUDE.md

## Commands

```bash
go run cmd/api/main.go
go run cmd/migrate/main.go up|down|status
go run cmd/seed/main.go                 # seed admin user from ADMIN_* env vars (idempotent)
go test ./...
go test -v ./internal/domain/problem/...
go test -v ./internal/application/problem/...
docker-compose --env-file .env up --build -d
bash tests/test_e2e.sh                  # requires running http
```

**Dev env vars:**
- `MOCK_AUTH=1` — reads user from `X-Mock-User` header
- `STORAGE_BACKEND=local` — local filesystem instead of GCS

## Architecture

> **Ante cualquier duda de nombrado, distribución de archivos, ubicación de tipos, o decisiones de arquitectura en general: consulta primero `training-and-judge-center-backend/ARCHITECTURE.md`.** Ese archivo es la fuente de verdad. Este CLAUDE.md es un resumen de navegación, no un sustituto.

Full conventions live in `training-and-judge-center-backend/ARCHITECTURE.md` — read it top to bottom before modifying any layer. This file is a navigational summary, not a substitute.

Go backend: DDD + Hexagonal Architecture. Dependency direction: `domain/` ← `application/` ← `adapter/` ← `cmd/`.

```
adapter/http/      ← driving adapter (handlers, middleware, router)
adapter/postgres/  ← shared DB infrastructure
adapter/user/      ← driven adapters: user domain
adapter/problem/   ← driven adapters: problem domain
adapter/group/     ← driven adapters: group domain
adapter/material/  ← driven adapters: material domain
adapter/auth/      ← driven adapter: authentication (transversal)
adapter/email/     ← driven adapter: email (transversal)
adapter/ratelimit/ ← driven adapter: rate limiting (transversal)
```

`adapter/http/` must NOT import `adapter/postgres/`, `adapter/user/`, etc. — only `application/` and `domain/shared/`.

**Migración de estructura completada** (verificado 2026-07-19): `internal/server/`, `internal/platform/` e `internal/infrastructure/` ya no existen; todo el código vive en la estructura destino (`adapter/`). El backlog de refactors de convenciones (`PENDIENTE_REFACTOR.md`) se completó y fue eliminado.

## Conventions summary

### Domain — §4 (D1–D10)

- One file per concept: `user.go`, `email.go`, `status.go`, `repository.go`, `errors.go`
- `New*()` validates and returns `apperror` on failure; `Restore*()` bypasses validation — no error return, no `New*` calls inside
- `domain/shared/` — only `UserID` and `Role` (business primitives). `CurrentUser` → `application/shared/`
- `errors.go` — string constants only (`ErrCodeEmailConflict = "EMAIL_CONFLICT"`); no sentinel `var ErrX = errors.New(...)`
- Repository ports: primary aggregate → `repository.go`; secondary aggregates → `<aggregate>_repository.go`
- Tests: `package <domain>_test`; table tests for value objects, individual functions for aggregates
- Constructors/methods needing current time receive `now time.Time`; `time.Now()` is called once in the use case, never inside domain code (D10)

### Application — §5 (A1–A10)

- `<Operation>UseCase` struct; `New<Operation>UseCase` constructor; `Execute(ctx, in) (*<Op>Output, error)` or `Execute(ctx, in) error`
- No `ports.go` — one file per port, named after the type (`user_provider.go`, `transaction_manager.go`)
- `dto.go` only when 2+ use cases in the same package share a type (actual reuse, not anticipated)
- Shared logic between use cases → descriptive filename (`permissions.go`, `pagination.go`); never `helpers.go` or `utils.go`
- Logging: `slog.ErrorContext(ctx, ...)` always; never `slog.Error(...)`
- Tests: `mocks_test.go` for shared infrastructure; `mock*` naming; `testutil.AsAdmin/AsCoach/AsContestant` for CurrentUser helpers

### Adapters — §6 (Ad1–Ad11)

- Struct name = port name; no domain/tech prefix in type name — package provides context (`user.Repository`, not `user.UserRepository`)
- Repositories: field type `postgres.Querier`; call `postgres.GetQuerier(ctx, r.db)` at the start of every method — reads and writes
- All errors translated to `apperror` at the adapter boundary; never return raw `pgx` errors upstream
- `adapter/http/handler/response.go` — only `WriteJSON` and `WriteError`; no private wrappers around them
- `handler.go` — `Handler` struct with concrete `*<Op>UseCase` fields, never interfaces; one `<op>_handler.go` per method
- Handler tests: `package <domain>` (same package); `newHandlerWith<Operation>` sets only that use case, rest `nil`

### Middleware — §7 (M1–M7)

- Naming: imperative verb — `Auth(...)`, `RequireRole(...)`; no `New` prefix, no `Middleware` suffix
- Stores `shared.CurrentUser` in context (not `*user.TokenClaims`); read via `middleware.GetCurrentUser(ctx) (shared.CurrentUser, bool)`
- One file per function (`auth.go`, `require_role.go`); private `writeError` helper inside each file
- Dependencies always required — `&auth.NoOpSessionInvalidator{}` instead of nil; no nil checks in middleware body
- Logging: `slog.ErrorContext(r.Context(), ...)`
- Tests: `package middleware` (same package, context key stays private); mirrors source files; no `// Arrange / Act / Assert`

### Router — §8 (R1–R4)

- `Handlers` struct bundles all domain handlers; `Services` struct bundles middleware dependencies
- One block per access level; lowercase comment describes the level (`// public`, `// authenticated`, `// admin only`)
- `r.Group` for middleware-only grouping (no URL prefix); `r.Route` for URL prefix grouping
- Global middleware order: CORS → RequestID → Logger → Recoverer

### Composition root — §9 (C1–C5)

- Use case variables: `createProblemUseCase`, never `createProblemUC`
- Import aliases: all lowercase — `appgroup`, `appuser`, `handleruser`, `handlermaterial`, `googlestorage`
- Section comments: `// infrastructure`, `// cross-cutting services`, `// <domain> adapters`, `// <domain> use cases`
- One shared instance per stateless service; handler constructed at the end of its domain's use case block
- Repository variables: full aggregate name — `deactivationRequestRepo`, never `deactRepo`

## Non-obvious decisions

**`ListFilters.Statuses`:** empty slice = no status filter. Visibility of DRAFTs is controlled by `ViewerModifierID`. The use case must not pre-fill with all statuses.

**`domain/shared/` vs `application/shared/`:** `UserID` and `Role` are business primitives → `domain/shared/`. `CurrentUser` is request context → `application/shared/`. Never add request-lifecycle concepts to `domain/shared/`.

**Cross-domain data:** each domain defines its own display types locally (e.g., `UserDisplay` in `application/problem/user_provider.go`). Never import `domain/user` from other domains — each domain queries what it needs via its own port and adapter.

**Handler tests and `wrapWithAuth`:** handler tests build real use cases with mocked dependencies — use cases use concrete types (Ad6), so they cannot be mocked directly. `wrapWithAuth` (in `mocks_test.go`) runs the real `Auth` middleware with a mock token service, which puts `shared.CurrentUser` in context exactly as production does.

**Adapter double-logging:** the adapter logs the raw error before returning `apperror.NewInternal()`. The application layer must NOT log errors that come from adapters — they were already logged at the boundary.

**Estado de los dominios:** todos los dominios del backend (`user`, `problem`, `group`, `material`, `contest`, `team`, `submission`) tienen capa de dominio, `application/` y handlers HTTP implementados. El **Judge System** está mayormente implementado: pool de containers (`adapter/judge/pool/`), executor/session (`adapter/judge/`), use case con retries (`application/judge/`) y el binario `cmd/worker/` (RUNNER_ARCHITECTURE.md lo llama `cmd/judge` — es el mismo componente). Pendientes: consumidor concurrente con semáforo (hoy es serial), `Pool.IsHealthy()` + `os.Exit` si el pool se degrada, goroutine de recuperación (§874), y despliegue DinD en GKE.

**Endpoints pendientes principales:**
- `POST /problems/p/{slug}/publish` — requiere Judge System; bloquea DRAFT→PUBLISHED por API.
- `GET /users/me/dashboard` — dashboard de actividad cross-domain (misc-4).
