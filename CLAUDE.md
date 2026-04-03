# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run server
go run cmd/api/main.go

# Run migrations
go run cmd/migrate/main.go up     # apply
go run cmd/migrate/main.go down   # rollback
go run cmd/migrate/main.go status # check state

# Build binaries
go build -o api ./cmd/api
go build -o migrate ./cmd/migrate

# Run all tests
go test ./...

# Run tests for a specific package
go test -v ./internal/domain/problem/...

# Run with Docker (recommended for full integration)
docker-compose --env-file .env up --build -d

# Run E2E tests (requires a running server)
bash tests/test_e2e.sh
```

**Development environment variables (`.env`):**
- `MOCK_AUTH=1` — enables mock auth (reads user from `X-Mock-User` header)
- `STORAGE_BACKEND=local` — uses local filesystem instead of GCS
- See `.env.example` for full reference

## Architecture

This is a Go backend for a competitive programming training and judge platform. It manages coding problems: creation, file uploads (test cases, solutions, checkers), language overrides, and ICPC package imports.

### Layered Structure (DDD + Hexagonal)

```
internal/
├── domain/problem/       → Aggregate root + value objects + interfaces
├── application/problem/  → Use cases (one file per operation)
├── infrastructure/       → Concrete implementations (Postgres, GCS, parsers)
├── platform/             → App-level wiring (DB pool, storage factory, config loader)
└── server/               → HTTP layer (chi router, handlers, middleware)

cmd/
├── api/main.go           → Entrypoint: wires everything and starts the server
└── migrate/main.go       → CLI for goose migrations

config/virtual_object.json → Static platform config: supported languages, limits, valid tags
```

### Domain Layer (`internal/domain/problem/`)

The domain is pure Go — no framework or infrastructure dependencies.

- `Problem` is the aggregate root. It holds all business state and exposes methods like `UpdateInfo`, `SetStatement`, `AddLanguageOverride`, etc.
- Value objects (`Slug`, `Title`, `Statement`, `TimeLimit`, `MemoryLimit`, `Tags`, `LanguageOverride`) are constructed via `New*()` functions that validate on creation and return `AppError` on failure. They are immutable after construction.
- `Repository` and `ConfigService` are interfaces defined here — infrastructure must satisfy them.
- Domain reconstruction from persistence uses `Restore*()` constructors that bypass validation (trust the DB).

### Application Layer (`internal/application/problem/`)

Each use case is a struct with a `Run(ctx, input) (output, error)` method. Use cases:
- `CreateProblemUseCase`, `UpdateProblemUseCase`, `ImportProblemUseCase`
- `UploadProblemFilesUseCase`, `DeleteProblemFileUseCase`
- `AddModifierUseCase`, `RemoveModifierUseCase`, `ListModifiersUseCase`

External service contracts are defined in `ports.go` (`UserProvider`, `ProblemFileRepository`, `ZipParser`, `ICPCPackageParser`).

### Error Handling

All errors flow through `pkg/apperror`. Use `apperror.New(code, message)` or domain-level `NewXxx` that wrap it. HTTP handlers map error codes to status codes. Validation errors carry field-level detail.

### HTTP Layer (`internal/server/`)

- Router: `router.go` (chi v5)
- All problem routes are under `/problems` and `/problems/p/{slug}`
- Middleware: mock auth injects a user from `X-Mock-User` header when `MOCK_AUTH=1`
- Handler constructors receive use cases via dependency injection

### Testing Patterns

- Domain unit tests are table-driven (`[]struct{ name, input, wantErr, errCode }`), covering valid and invalid inputs for each value object
- Infrastructure tests (e.g., ICPC parser) use real ZIP fixtures
- E2E tests (`tests/test_e2e.sh`) use `curl` against a live server and clean up with `trap EXIT`

### Config

`config/virtual_object.json` is the authoritative source for:
- Supported languages and their compiler versions (C++20, Java17, Python310)
- Default and per-language time/memory limits
- Valid problem tags
- File upload constraints

This is loaded at startup via `internal/platform/config` and passed as a `ConfigService` to the domain.
