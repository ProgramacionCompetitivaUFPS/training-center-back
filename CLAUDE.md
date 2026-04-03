# CLAUDE.md

## Commands

```bash
go run cmd/api/main.go                  # run server
go run cmd/migrate/main.go up|down|status
go test ./...
go test -v ./internal/domain/problem/...
docker-compose --env-file .env up --build -d
bash tests/test_e2e.sh                  # requires running server
```

**Dev env vars:**
- `MOCK_AUTH=1` — reads user from `X-Mock-User` header
- `STORAGE_BACKEND=local` — local filesystem instead of GCS

## Architecture

Go backend (DDD + Hexagonal). Layers: `domain` → `application` → `infrastructure/server`.

**Key conventions:**
- Use cases: `Execute(ctx, input) (output, error)`
- Domain construction: `New*()` validates and returns `AppError` on failure
- DB reconstruction: `Restore*()` bypasses validation — trust the DB
- Ports (interfaces for external deps): `application/problem/ports.go`
- Errors: always via `pkg/apperror`

## Non-obvious decisions

**`ListFilters.Statuses` (repository):** empty = no status filter. Visibility of DRAFTs is controlled by `ViewerModifierID`. Don't pre-fill with all statuses from the use case.

**HTTP responses:** `getProblemResponse` is the single problem response type. Use `buildResponse(p, display)` in `types.go` for mutation endpoints (Create/Update/Import/UploadFiles). If an endpoint needs a different shape, build it directly in the handler — don't add parameters to `buildResponse`.

**Cross-domain UserID:** `UserID` lives temporarily in `domain/problem/user_id.go`. Move to `domain/shared/` when the User domain branch merges. `domain/user/user.go` is a stub (only `CurrentUser` and `Display`) — not the real User domain.
