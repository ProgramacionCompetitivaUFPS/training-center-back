# Architecture & Conventions

This document captures all architectural decisions for this project. It is meant to be reproduced exactly on any machine — read it top to bottom before touching the code.

---

## 1. Conceptual foundation

The project follows **Hexagonal Architecture** (Alistair Cockburn, 2005) combined with **Domain-Driven Design** (Eric Evans / Vaughn Vernon).

The central idea of hexagonal architecture is the hexagon:

```
                    ┌─────────────────────────────┐
                    │                             │
 [ HTTP ] ──────────┤   application/ + domain/    ├──────────── [ Postgres ]
 (driving adapter)  │                             │  (driven adapter)
                    │   "the hexagon"             │
 [ CLI   ] ──────────┤                             ├──────────── [ Redis ]
                    │                             │
                    └─────────────────────────────┘
```

- **Inside the hexagon**: `domain/` and `application/`. Pure business logic. Zero external dependencies.
- **Outside the hexagon**: everything else — HTTP handlers, database repositories, email senders, etc.
- **Ports**: interfaces that define what the hexagon needs from the outside world, or what it offers to the outside world. They live *inside* the hexagon (in `domain/` or `application/`).
- **Adapters**: concrete implementations of those interfaces. They live *outside* the hexagon (in `adapter/`).

**Key rule**: dependencies only point inward. `adapter/` imports `application/` and `domain/`. `application/` imports `domain/`. `domain/` imports nothing from this project.

---

## 2. Folder structure

```
training-and-judge-center-backend/
│
├── cmd/
│   ├── api/main.go          ← composition root: wires all dependencies
│   ├── migrate/
│   └── seed/
│
├── config/                  ← static config files (JSON, YAML, etc.)
│   └── virtual_object.json
│
├── internal/
│   │
│   ├── config/              ← reads env vars and config files at startup
│   │   ├── config.go        ← Config struct + Load()
│   │   └── virtual_object.go
│   │
│   ├── domain/              ← the hexagon core (pure business logic)
│   │   ├── shared/          ← cross-domain primitives (UserID, CurrentUser, Role)
│   │   ├── user/
│   │   ├── problem/
│   │   ├── group/
│   │   └── material/
│   │
│   ├── application/         ← use cases (orchestration layer)
│   │   ├── shared/          ← cross-cutting ports (EmailSender, RateLimiter, TransactionManager)
│   │   ├── user/
│   │   ├── problem/
│   │   ├── group/
│   │   └── material/
│   │
│   └── adapter/             ← everything outside the hexagon
│       │
│       ├── http/            ← DRIVING adapter (initiates interaction)
│       │   ├── router.go
│       │   ├── handler/
│       │   │   ├── response.go
│       │   │   ├── auth_handler.go
│       │   │   ├── health_handler.go
│       │   │   ├── problem/
│       │   │   ├── user/
│       │   │   ├── group/
│       │   │   └── material/
│       │   └── middleware/
│       │       └── auth_middleware.go
│       │
│       ├── postgres/        ← shared DB infrastructure (used by domain adapters)
│       │   ├── connection.go
│       │   ├── querier.go
│       │   └── transaction_manager.go
│       │
│       ├── user/            ← DRIVEN adapters for user domain
│       │   ├── repository.go
│       │   ├── password_recovery_repository.go
│       │   ├── email_change_repository.go
│       │   ├── deactivation_request_repository.go
│       │   └── deactivation_audit_log_repository.go
│       │
│       ├── problem/         ← DRIVEN adapters for problem domain
│       │   ├── repository.go
│       │   ├── user_provider.go
│       │   ├── gcs_storage.go
│       │   ├── local_storage.go
│       │   ├── icpc_adapter.go
│       │   ├── icpc_parser.go
│       │   └── platform_settings.go  ← builds problem.PlatformSettings from config
│       │
│       ├── group/           ← DRIVEN adapters for group domain
│       │   ├── repository.go
│       │   ├── member_repository.go
│       │   ├── join_request_repository.go
│       │   ├── user_provider.go
│       │   └── preferences_reader.go
│       │
│       ├── material/        ← DRIVEN adapters for material domain
│       │   ├── repository.go
│       │   ├── author_provider.go
│       │   ├── group_provider.go
│       │   └── group_member_provider.go
│       │
│       ├── auth/            ← DRIVEN adapter: authentication (transversal)
│       │   ├── jwt_service.go
│       │   ├── invitation_jwt.go
│       │   └── session_invalidator.go
│       │
│       ├── email/           ← DRIVEN adapter: email (transversal)
│       │   └── smtp_sender.go
│       │
│       └── ratelimit/       ← DRIVEN adapter: rate limiting (transversal)
│           └── redis_limiter.go
│
└── pkg/                     ← shared utilities (no domain knowledge)
    ├── apperror/
    └── timeutil/
```

### Why not `platform/` + `infrastructure/`?

The previous structure had a custom rule: "if a file imports from `domain/`, it goes in `platform/`; otherwise it goes in `infrastructure/`." This rule:
- Is not from any literature
- Creates ambiguity when adding new adapters
- Forces you to answer "does this import domain?" before knowing where to put a file

The `adapter/` umbrella removes that question. Everything outside the hexagon is an adapter. Period.

### Why not `adapter/in/` and `adapter/out/`?

Cockburn's paper uses "left side" (driving) and "right side" (driven). Some books formalize this as `in/` and `out/` subdirectories. We chose not to because:
- This project has exactly one driving adapter: HTTP
- Adding `adapter/in/http/` adds a nesting level without adding information
- The distinction is still clear: `http/` receives requests, everything else responds to the application

If a second driving adapter is added in the future (e.g., a CLI or a message queue consumer), revisit this decision.

### Why does `internal/config/` stay outside `adapter/`?

`config/` reads environment variables and loads JSON files. It is not an adapter to any external service — it is bootstrap code used by `cmd/api/main.go` to wire the dependency tree. Moving it into `adapter/` would imply it implements a port, which it does not.

### Where does `platform/config/config_service.go` go?

This file builds a `*problem.PlatformSettings` (a domain type) from a `*config.VirtualObject` (raw config). It is a problem-domain-specific adapter that translates external configuration into something the domain understands. It belongs in `adapter/problem/platform_settings.go`.

### Where does `adapter/postgres/` fit?

It is shared infrastructure used by multiple domain adapters (`adapter/user/`, `adapter/problem/`, etc.). It lives at the same level as the domain adapters because it is not specific to any domain — it provides the database connection, the `Querier` interface, and the `TransactionManager` that all driven adapters depend on.

---

## 3. Dependency rules

| Layer | May import | Must NOT import |
|---|---|---|
| `domain/` | `pkg/`, `domain/shared/` | `application/`, `adapter/`, `internal/config/` |
| `application/` | `domain/`, `pkg/`, `application/shared/` | `adapter/`, `internal/config/` |
| `adapter/*` | `domain/`, `application/`, `pkg/`, `internal/config/`, `adapter/postgres/` | other `adapter/*` domains (e.g. `adapter/user/` must not import `adapter/problem/`) |
| `adapter/http/` | `application/`, `domain/shared/`, `pkg/` | `adapter/postgres/`, `adapter/user/`, etc. |
| `cmd/` | everything | — |
| `pkg/` | standard library only | `internal/` |

The `adapter/http/` handler layer specifically must not import driven adapters — it only knows about use cases from `application/`.

---

## 4. Domain layer conventions

### D1 — No internal subdirectories within a domain package

Each package under `domain/` represents one bounded context. All files for that context live flat in the package — no subdirectories.

The package itself is the encapsulation boundary in Go. Splitting into `domain/user/entities/`, `domain/user/repositories/`, etc. would be organizing by technical type rather than by domain concept, which contradicts DDD's principle of organizing around business meaning.

```
domain/user/        ✅  everything about the user bounded context lives here flat
domain/user/repos/  ❌  technical grouping, not a DDD concept
domain/user/models/ ❌  "model" is MVC vocabulary, not DDD vocabulary
```

---

### D2 — Repository file naming

Only **aggregate roots** get repositories (Evans: *"Provide repositories only for aggregate roots"*). A bounded context package may contain more than one aggregate root — for example `domain/user/` contains `User`, `EmailChangeRequest`, `PasswordRecoveryRequest`, and `DeactivationRequest`, each with its own lifecycle and identity.

Naming rule:
- The **primary aggregate** of the package → `repository.go`
- **Secondary aggregates** in the same package → `<aggregate>_repository.go`

```
domain/user/repository.go                    ← User (primary aggregate)
domain/user/email_change_repository.go       ← EmailChangeRequest (secondary aggregate)
domain/user/password_recovery_repository.go  ← PasswordRecoveryRequest (secondary aggregate)
domain/user/deactivation_repository.go       ← DeactivationRequest (secondary aggregate)
```

**Why secondary aggregates and not sub-entities?** `EmailChangeRequest`, `PasswordRecoveryRequest`, and `DeactivationRequest` each have their own ID, their own state machine, their own lifecycle independent of `User`, and they reference `User` only by ID string — never by holding a `*User` object. Vernon identifies cross-aggregate references by ID as the canonical signal of a separate aggregate.

---

### D3 — One file per domain concept

Every domain type lives in its own file, named after the concept in snake_case. This applies uniformly to all DDD building blocks — no exceptions based on complexity:

| Concept | File name |
|---|---|
| Aggregate root entity | `<aggregate>.go` — e.g. `user.go` |
| Secondary aggregate entity | `<aggregate>.go` — e.g. `email_change.go` |
| Value object | `<value_object>.go` — e.g. `email.go`, `slug.go` |
| Enum | `<enum>.go` — e.g. `status.go`, `visibility.go` |
| Domain port (non-repository) | `<interface_name>.go` — e.g. `token_service.go` |
| Primary aggregate repository | `repository.go` |
| Secondary aggregate repository | `<aggregate>_repository.go` |
| Error codes and sentinels | `errors.go` |

A simple enum (`type Status string` with constants) and a rich value object (`type Email struct`) are both domain concepts. Discriminating between them for file organization would create an arbitrary rule that requires judging "how complex" a type is. The rule is uniform: one concept, one file.

```
domain/user/
  user.go           ← User aggregate
  email.go          ← Email value object
  nickname.go       ← Nickname value object
  status.go         ← Status enum          ← same rule applies here
  request_status.go ← RequestStatus enum   ← and here
  repository.go     ← User repository
  token_service.go  ← TokenService port
  ...
```

---

### D4 — Reglas de constructores: `New*`, `Restore*` y factorías de estado

Todo tipo de dominio (entidad, value object, enum) tiene exactamente dos caminos de construcción.

#### Representación interna: `struct{ value T }`, no type alias

Todo value object y enum de dominio usa un struct con campo privado:

```go
// ✅ campo privado — paquetes externos no pueden construir valores arbitrarios
type Status struct{ value string }

// ❌ type alias — cualquier paquete puede hacer Status("ARBITRARY") sin pasar por el constructor
type Status string
```

El campo `value` es privado (minúscula). Desde fuera del paquete, `Status{value: "..."}` no compila — el compilador lo bloquea. El único punto de entrada es `New*` o `Restore*`. Esto hace que las invariantes del tipo sean enforced por el sistema de tipos, no por convención.

Las constantes exportadas se convierten en variables (los structs no pueden ser `const` en Go):

```go
// ✅
var (
    StatusActive      = Status{value: "ACTIVE"}
    StatusDeactivated = Status{value: "DEACTIVATED"}
)

// ❌ — solo funciona con type alias
const StatusActive Status = "ACTIVE"
```

Las comparaciones con `==` siguen funcionando porque Go soporta igualdad de structs con campos comparables. `string(s)` ya no compila — usar `.String()`.

---

#### `New*(...)` — input externo, valida invariantes

Recibe tipos de dominio ya validados como parámetros (no primitivos). El aggregate no re-valida lo que cada VO ya garantiza — solo valida invariantes propios del aggregate.

```go
// ✅ recibe Email, Password, Nickname ya validados
func NewUser(email Email, password Password, nickname Nickname, name string) (*User, error)

// ❌ no recibe strings crudos
func NewUser(email string, password string, nickname string, name string) (*User, error)
```

Tipos de error según la naturaleza del fallo:

| Situación | Error a retornar |
|---|---|
| Input externo inválido (regla de negocio) | `apperror.NewValidation([]apperror.FieldError{...})` |
| Argumento requerido vacío (bug del programador) | `apperror.NewInternal()` |
| Violación de regla de negocio en método de dominio | `apperror.NewConflict(ErrCodeX, "...")`, `apperror.NewNotFound(...)`, etc. |

`fmt.Errorf` y `errors.New` están prohibidos en `domain/`.

#### `Restore*(...)` — reconstrucción desde persistencia, sin validar

Retorna el tipo directamente, **sin error**. Confía en que los datos de la DB ya eran válidos cuando se guardaron.

```go
// ✅
func RestoreUser(id string, email *string, ...) *User {
    return &User{
        email:    RestoreEmail(*email),      // llama Restore*, nunca New*
        nickname: RestoreNickname(nickname),
        ...
    }
}

// ❌ llama New* y retorna error — viola ambas reglas
func RestoreUser(...) (*User, error) {
    email, err := NewEmail(*emailStr)
    ...
}
```

Reglas de `Restore*`:
- Nunca retorna error
- Nunca llama `New*` internamente — usa `Restore*` para VOs anidados
- No ejecuta ninguna validación

#### Factorías de estado conocido — `New<Type><State>()`

Para estados internos del dominio que siempre son válidos se permiten factorías sin parámetros y sin error. Usan constantes internas como única fuente de verdad — tanto la factoría como el constructor parametrizado referencian la misma constante:

```go
const (
    statusDraft     = "DRAFT"
    statusPublished = "PUBLISHED"
)

func NewStatus(raw string) (Status, error) {
    switch raw {
    case statusDraft, statusPublished:
        return Status{value: raw}, nil
    default:
        return Status{}, apperror.NewValidation(...)
    }
}

func NewStatusDraft() Status     { return Status{value: statusDraft} }
func NewStatusPublished() Status { return Status{value: statusPublished} }
```

El string existe una sola vez. Un typo en la constante lo atrapa el compilador. No hay error que ignorar ni panic.

---

### D5 — `domain/shared/` contains only cross-cutting domain primitives

`domain/shared/` is the project's **Shared Kernel** (Vernon, IDDD) — a minimal set of domain primitives so fundamental that multiple bounded contexts reference them. The bar for adding something here is high.

**Criterion:** would this concept exist in the business model even if there were no HTTP layer, no authentication, no requests?

- `UserID` → yes. It is a business identity referenced by `problem`, `group`, and `material`.
- `Role` → yes. It is a business concept that defines what a user can do.
- `CurrentUser` → no. It packages authentication context for a request. It is tied to the request/response cycle, not the business domain.

**Rule:** if the concept only makes sense in the context of a request, a session, or an infrastructure concern, it does not belong in `domain/shared/`. It belongs in `application/shared/` (if used across multiple domains) or in the specific package that needs it.

```
domain/shared/    ← UserID, Role — business primitives
application/shared/ ← CurrentUser, EmailSender, RateLimiter — cross-cutting app concerns
```

---

### D6 — Errores del dominio: `apperror` sin HTTP

El dominio expresa sus errores usando constructores semánticos de `pkg/apperror`. No usa `fmt.Errorf` ni `errors.New` para errores que atraviesan capas.

**`pkg/apperror` no importa `net/http`.** El `AppError` usa un campo `Kind` para expresar la naturaleza del error. El mapeo de `Kind` a status HTTP vive exclusivamente en `adapter/http/`.

```go
// pkg/apperror — sin net/http
type Kind string
const (
    KindValidation      Kind = "VALIDATION"
    KindConflict        Kind = "CONFLICT"
    KindNotFound        Kind = "NOT_FOUND"
    KindForbidden       Kind = "FORBIDDEN"
    KindUnauthorized    Kind = "UNAUTHORIZED"
    KindBadRequest      Kind = "BAD_REQUEST"
    KindInternal        Kind = "INTERNAL"
    KindTooManyRequests Kind = "TOO_MANY_REQUESTS"
)

type AppError struct {
    Kind       Kind
    Code       string
    Message    string
    Details    []FieldError
    RetryAfter int    // segundos a esperar antes de reintentar — no es HTTP-específico
    cause      error
}
```

```go
// adapter/http/handler/response.go — único lugar donde vive el mapeo HTTP
func kindToStatus(k apperror.Kind) int {
    switch k {
    case apperror.KindValidation:      return http.StatusBadRequest
    case apperror.KindBadRequest:      return http.StatusBadRequest
    case apperror.KindConflict:        return http.StatusConflict
    case apperror.KindNotFound:        return http.StatusNotFound
    case apperror.KindForbidden:       return http.StatusForbidden
    case apperror.KindUnauthorized:    return http.StatusUnauthorized
    case apperror.KindTooManyRequests: return http.StatusTooManyRequests
    default:                           return http.StatusInternalServerError
    }
}
```

**Qué usa el dominio vs la aplicación:**

| Constructor | Lo usa |
|---|---|
| `NewValidation`, `NewConflict`, `NewNotFound`, `NewForbidden`, `NewBadRequest`, `NewInternal` | `domain/` y `application/` |
| `NewTooManyRequests` | solo `application/` — el rate limiting no es un concepto de dominio |
| `NewUnauthorized`, `NewServiceUnavailable` | solo `adapter/` o `application/` |

**`RetryAfter`** se queda en `AppError` porque "cuánto tiempo esperar antes de reintentar" no es exclusivo de HTTP — es metadata de error genérica. Lo establece la capa de aplicación, no el dominio.

---

### D7 — `errors.go` contiene solo constantes de código de error

Cada paquete de dominio tiene un `errors.go` con constantes `string` que identifican condiciones de error específicas del bounded context:

```go
// domain/user/errors.go
const (
    ErrCodeEmailConflict         = "EMAIL_CONFLICT"
    ErrCodeNicknameConflict      = "NICKNAME_CONFLICT"
    ErrCodeCannotSelfDeactivate  = "CANNOT_SELF_DEACTIVATE"
    ErrCodeCannotDeactivateAdmin = "CANNOT_DEACTIVATE_ADMIN"
)
```

**No sentinel errors** (`var ErrX = errors.New(...)`). Los sentinels son un patrón de Go para cuando distintas partes del código necesitan reaccionar diferente al mismo error. En este proyecto cada sentinel siempre produce exactamente el mismo `apperror` — no hay ramificación, por lo tanto no agregan valor.

Con códigos de error, el adapter traduce directamente:
```go
// adapter/user/repository.go — detecta unique constraint en email
return apperror.NewConflict(user.ErrCodeEmailConflict, "email already in use")
```

El use case recibe un `apperror` ya construido — no necesita atrapar ni convertir nada.

**Naming:** `ErrCode<PascalCase>` = `"SCREAMING_SNAKE_CASE"`.

Los códigos de error que hoy viven en `pkg/apperror/errors.go` pero son específicos de un dominio deben moverse al `errors.go` del bounded context correspondiente. `pkg/apperror` solo debe contener las constantes verdaderamente genéricas de infraestructura (`ErrCodeInternalError`, `ErrCodeValidationError`).

---

### D8 — Los tipos de soporte del repositorio pertenecen al archivo del repositorio

Los filtros de consulta (`ListFilters`, `SortField`, `SortOrder`) y los tipos de resultado (`MemberStats`) no son conceptos primarios del dominio — son el contrato de entrada y salida del repositorio. No los usa ninguna entidad ni ningún método de negocio. Fuera del repositorio no tienen contexto ni sentido.

La convención D3 ("un concepto por archivo") aplica a conceptos primarios del dominio: entidades, value objects, enums de negocio. No aplica a los tipos de soporte del repositorio.

**Regla:** cada archivo de repositorio contiene la interfaz del repositorio y todos sus tipos de soporte directos.

```
domain/group/repository.go               ← Repository, ListFilters, SortField, SortOrder, MembershipFilter
domain/group/member_repository.go        ← MemberRepository, MemberFilters, MemberStats
domain/group/join_request_repository.go  ← JoinRequestRepository, JoinRequestFilters
```

Esto resuelve también el caso de `domain/group/repository.go` que hoy agrupa tres repositorios con siete tipos de soporte en un solo archivo — no porque haya que "limpiar", sino porque cada repositorio es un contrato independiente que merece su propio archivo.

El único caso especial es `domain/user/filter.go`: los tipos ahí (`UserFilter`, `SortField`, `SortOrder`, `SearchField`) deben moverse a `repository.go` porque son el contrato del repositorio de `User`, no conceptos de dominio independientes.

---

### D9 — Convenciones de tests en la capa de dominio

**Paquete externo.** Los tests de dominio usan `package <domain>_test`. El dominio no tiene internos que necesiten testeo directo — toda su API es pública. El paquete externo fuerza a que el test use exactamente la misma interfaz que usaría `application/`.

```go
// ✅
package group_test
import "github.com/training-judge-center/backend/internal/domain/group"

// ❌
package group  // accede a internos sin necesitarlo
```

**Tabla para value objects, función individual para aggregates.** La elección depende del tipo de pregunta que responde el test:

- *¿Qué inputs acepta/rechaza el constructor?* → tabla con `t.Run`. Todos los casos tienen la misma estructura `(input → resultado esperado)`.
- *¿Qué pasa cuando ejecuto X en el estado Y?* → función individual con nombre descriptivo. Cada escenario tiene su propio setup y verifica comportamientos distintos.

```go
// ✅ value object — tabla
func TestNewEmail_Valid(t *testing.T) {
    tests := []struct{ name, input, expected string }{ ... }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}

// ✅ aggregate — función individual
func TestPublish_SetsPublishedAt(t *testing.T) { ... }
func TestPublish_AlreadyPublished_ReturnsError(t *testing.T) { ... }
```

**`t.Helper()` en toda función helper.** Aplica a cualquier capa — es una regla de Go general. Toda función que recibe `*testing.T` y no es un `Test*` debe llamar `t.Helper()` como primera línea. Hace que los mensajes de error apunten al test que llamó al helper, no a la línea interna del helper.

```go
func newGroupName(t *testing.T, s string) group.GroupName {
    t.Helper()  // ← primera línea siempre
    n, err := group.NewGroupName(s)
    if err != nil {
        t.Fatalf("NewGroupName(%q): %v", s, err)
    }
    return n
}
```

En la práctica, la mayoría de los helpers de la capa de aplicación no reciben `*testing.T` — retornan valores y usan `panic` si fallan (`newTestMaterial`, `repoWith`). La regla aplica cuando el helper sí recibe `*testing.T` y llama `t.Fatalf` o `t.Errorf` internamente.

**`common_test.go` para helpers compartidos entre archivos.** Cuando un paquete de dominio tiene dos o más archivos de test que comparten variables o funciones (ej: `testNow`, constructores de fixtures, `assertValidationField`), esas definiciones van en `common_test.go`. Cada archivo de test individual define únicamente sus helpers propios.

```go
// internal/domain/user/common_test.go
package user_test

import (
    "testing"
    "time"
    "github.com/training-judge-center/backend/internal/domain/user"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func makeTestUser(t *testing.T) *user.User {
    t.Helper()
    // ...
}
```

**Cuándo aplica:** solo cuando dos o más archivos de test usan la misma variable/función. Si `testNow` solo existe en un archivo y ningún otro lo usa, no hace falta `common_test.go`.

---

### D10 — El dominio no tiene efectos de lado ni fuentes de no determinismo

El dominio es lógica pura: dadas las mismas entradas, produce siempre las mismas salidas. Cualquier fuente de no determinismo — el reloj, un generador de UUIDs, números aleatorios, lecturas de entorno, llamadas de red — es I/O. El dominio no hace I/O.

**Regla general:** si un constructor o método de dominio necesita un valor que no puede calcular desde sus argumentos de manera determinista, ese valor se recibe como parámetro desde `application/`. El dominio recibe datos; la capa de aplicación hace el I/O.

| Fuente de no determinismo | Solución en el dominio | Quién genera el valor |
|---|---|---|
| Hora actual (`time.Now()`) | Parámetro `now time.Time` | Use case |
| ID de entidad (`uuid.New()`) | Parámetro `id string` | Use case |
| Número aleatorio (`rand.*`) | Parámetro del tipo apropiado | Use case |
| Variable de entorno (`os.Getenv`) | Parámetro de configuración | Composition root |

**Patrón para constructores:** reciben `id string` como primer parámetro y `now time.Time` cuando necesitan un timestamp de creación. El constructor valida `id == ""` con `apperror.NewInternal()` si retorna error.

```go
// ✅ — el aggregate recibe id y now como datos
func NewEmailChangeRequest(id, userID string, newEmail Email, code string, now time.Time) (*EmailChangeRequest, error) {
    if id == "" || userID == "" {
        return nil, apperror.NewInternal()
    }
    return &EmailChangeRequest{
        id:        id,
        expiresAt: now.UTC().Add(15 * time.Minute),
        createdAt: now.UTC(),
    }, nil
}

// ❌ — el aggregate genera I/O directamente
func NewEmailChangeRequest(userID string, ...) (*EmailChangeRequest, error) {
    return &EmailChangeRequest{
        id:        uuid.New().String(),  // no determinista
        createdAt: time.Now(),           // no determinista
    }, nil
}
```

**Patrón para métodos de mutación:** los métodos que actualizan timestamps también reciben `now time.Time` en lugar de llamar `time.Now()`.

```go
// ✅ — el método recibe now como dato
func (g *Group) UpdateMetadata(name *GroupName, description **string, now time.Time) {
    if name != nil { g.name = *name }
    g.updatedAt = now.UTC()
}

// ❌ — el método hace I/O directamente
func (g *Group) UpdateMetadata(name *GroupName, description **string) {
    g.updatedAt = time.Now()  // ← I/O dentro del hexágono
}
```

En la capa de aplicación — toda la I/O se concentra en un solo lugar:

```go
// ✅ — application/ hace toda la I/O de una vez
func (uc *RequestEmailChangeUseCase) Execute(ctx context.Context, in RequestEmailChangeInput) error {
    newID := uuid.New().String()   // I/O: UUID
    now   := time.Now()            // I/O: reloj
    req, err := user.NewEmailChangeRequest(newID, in.UserID, parsedEmail, code, now)
    ...
}
```

En tests — el resultado es completamente determinista:

```go
// ✅ — el test controla exactamente qué ID y qué hora se usan
var (
    testID  = "test-req-001"
    testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
)

func TestNewEmailChangeRequest_SetsExpiry(t *testing.T) {
    req, err := user.NewEmailChangeRequest(testID, "user-1", email, "code", testNow)
    require.NoError(t, err)
    assert.Equal(t, testNow.UTC().Add(15*time.Minute), req.ExpiresAt())  // ← determinista
}
```

**No existe excepción** para valores cuyo resultado importa al dominio o a los tests. Si la lógica de negocio requiere un valor no determinista, ese valor se genera en `application/` y se pasa al constructor o método. `User.Deactivate()` recibe el sufijo del nickname anónimo como parámetro — el test puede verificar exactamente qué nickname quedó registrado.

**Excepción documentada: `Password` y bcrypt**

`NewPassword` llama `bcrypt.GenerateFromPassword` internamente, que usa `crypto/rand` para generar una salt. Esta es la única excepción aceptada a D10, por tres razones:

1. **La salt no afecta ninguna regla de negocio.** Ninguna lógica de dominio ramifica sobre el valor específico del hash. A diferencia de `time.Now()` o `uuid.New()`, el valor que genera bcrypt nunca entra en comparaciones ni decisiones del dominio.

2. **El invariante del VO es "contraseña hasheada".** La invariante de `Password` no es "string con complejidad" sino "representación segura de una contraseña válida". Separar validación de hashing rompe la encapsulación y crea una API frágil donde el caller puede olvidar hashear.

3. **`Compare` es determinista.** `bcrypt.CompareHashAndPassword` dado el mismo hash almacenado y el mismo raw siempre retorna el mismo resultado — no hay no determinismo observable en las lecturas.

La regla D10 aplica cuando el valor generado importa para la lógica de negocio o los tests. Cuando la no determinidad es un detalle de implementación de seguridad sin impacto en ningún resultado observable, el dominio puede retenerla.

---

## 5. Application layer conventions

### A1 — Use case structs carry the `UseCase` suffix

Every use case is a struct named `<Operation>UseCase`. Its constructor is `New<Operation>UseCase`. This applies uniformly across all domains.

```
CreateGroupUseCase      ✅
CreateMaterialUseCase   ✅  (currently CreateMaterial — needs fix)
LoginUseCase            ✅
```

**Why the suffix?** The project is explicitly built around DDD and Hexagonal Architecture. `UseCase` is vocabulary from that literature (Cockburn, Hombergs). The suffix makes the architectural role of a type self-evident without reading its body — it is not a service, not a repository, not a helper. Hombergs (*Get Your Hands Dirty on Clean Architecture*, 2019) recommends this pattern precisely because it keeps architecture legible at a glance.

**Why not `Service`?** A `Service` groups multiple related operations in a single struct. A `UseCase` is one operation with its own dependencies. This project uses the one-struct-per-use-case pattern, so `UseCase` is the accurate term.

The entry point is always a method named `Execute` — not `Run`, `Handle`, `Do`, or the operation name itself.

```go
// ✅
type CreateGroupUseCase struct { ... }
func NewCreateGroupUseCase(...) *CreateGroupUseCase { ... }
func (uc *CreateGroupUseCase) Execute(ctx context.Context, in CreateGroupInput) (*CreateGroupOutput, error)

// ❌
type CreateGroup struct { ... }
type GroupService struct { ... }  // groups multiple operations — different pattern
func (uc *CreateGroupUseCase) CreateGroup(...) // method named after the operation
```

---

### A2 — `Execute` retorna `(*<Operation>Output, error)` o solo `error`

Cada use case tiene exactamente uno de dos contratos de salida:

```go
// cuando hay output significativo
Execute(ctx context.Context, in XxxInput) (*XxxOutput, error)

// cuando no hay output significativo (operaciones comando)
Execute(ctx context.Context, in XxxInput) error
```

**Naming:** `<Operation>Input` / `<Operation>Output` — simétrico y sin carga semántica extra. `Result` (patrón de programación funcional) y `Response` (vocabulario HTTP) están prohibidos.

**Cuatro patrones a eliminar:**

| Patrón actual | Problema | Corrección |
|---|---|---|
| `*<Op>Result` | Nombre inconsistente | Renombrar a `*<Op>Output` |
| `*domain.Type` | Viola capas — el handler no debe recibir objetos de dominio | Envolver en `*<Op>Output` con campos primitivos o DTOs |
| `UserDTO` (naked) | Si `Execute` necesita un campo extra, hay que cambiar la firma | Envolver en `*<Op>Output{ User UserDTO }` |
| `(struct{}, error)` | En Go idiomático, void = solo `error`; `struct{}` es ruido | Eliminar el primer valor de retorno |

```go
// ✅
func (uc *CreateUserUseCase) Execute(ctx, in CreateUserInput) (*CreateUserOutput, error)
func (uc *ResetPasswordUseCase) Execute(ctx, in ResetPasswordInput) error

// ❌
func (uc *CreateUserUseCase) Execute(ctx, in CreateUserInput) (UserDTO, error)        // naked DTO
func (uc *UnpublishProblemUseCase) Execute(ctx, in ...) (*problem.Problem, error)     // domain type
func (uc *AddModifierUseCase) Execute(ctx, in AddModifierInput) (struct{}, error)     // struct{}
func (uc *ImportProblemUseCase) Execute(ctx, in ...) (*ImportProblemResult, error)    // Result
```

**Éxito parcial va en Output, no en error.**

Si un use case completa su operación principal pero falla en una operación secundaria (e.g., el password se cambió pero las sesiones no se pudieron invalidar), esa información es output — no un error. Codificarla como error (sentinel o `AppError`) mezcla el canal de errores con datos de resultado.

```go
// ✅ — el handler lee el resultado directamente
type UpdatePasswordOutput struct {
    SessionsInvalidated bool
}
func (uc *UpdatePasswordUseCase) Execute(ctx, in UpdatePasswordInput) (*UpdatePasswordOutput, error)

// handler
out, err := uc.Execute(ctx, input)
if err != nil { return respondError(err) }
if !out.SessionsInvalidated { return respond200WithWarning(...) }
return respond200(...)

// ❌ — mezcla éxito con error
var ErrSessionsNotInvalidated = errors.New("...")  // sentinel en la capa de aplicación
func (uc *UpdatePasswordUseCase) Execute(ctx, in UpdatePasswordInput) error
// el handler necesita errors.Is para saber si "el error" es en realidad un éxito
```

---

### A3 — Un archivo por puerto, nombrado igual al tipo que contiene

Cada puerto de la capa de aplicación vive en su propio archivo. El archivo lleva el nombre en `snake_case` del tipo que contiene. No existe `ports.go`.

**¿Qué es un puerto de aplicación?** Es una interfaz que define lo que el use case necesita del mundo exterior — un repositorio secundario, un proveedor de datos de otro dominio, un servicio de infraestructura. Vive dentro del hexágono (`application/`) y es implementada por un adaptador en `adapter/`.

**Regla de archivo:** una interfaz por archivo. Los tipos de soporte que solo tienen sentido en el contexto de esa interfaz (DTOs de retorno, structs de parámetros) van en el mismo archivo. Si un tipo de soporte es necesario para dos interfaces del mismo paquete, va en el archivo de la interfaz más básica.

```
application/group/
  user_provider.go          ← UserDisplay + UserProvider
  preferences_reader.go     ← PreferencesReader
  invitation_token_service.go ← InvitationClaims + InvitationTokenService

application/material/
  group_provider.go           ← GroupProvider
  group_visibility_provider.go ← GroupVisibility + GroupVisibilityProvider
  group_member_provider.go    ← GroupMemberProvider
  author_provider.go          ← AuthorDisplay + AuthorProvider

application/problem/
  user_provider.go            ← UserDisplay + UserProvider  ✓ ya existe
  problem_file_repository.go  ← ProblemFileRepository
  zip_parser.go               ← ParsedFile + ZipParser
  icpc_package_parser.go      ← ParsedPackage + ICPCPackageParser

application/shared/
  email.go                ← EmailMessage + EmailSender        ✓ ya existe
  ratelimit.go            ← RateLimiter                       ✓ ya existe
  transaction_manager.go  ← TransactionManager
```

**`application/shared/` es el lugar de los puertos transversales** — los que usan múltiples dominios de aplicación. `TransactionManager` actualmente está duplicado en `application/user/ports.go` y `application/group/ports.go`; su lugar correcto es `application/shared/transaction_manager.go`.

**Por qué no `ports.go`?**

`ports.go` es la solución de conveniencia cuando no hay una regla explícita. El problema es que crece sin criterio de corte: el día que `application/group/` tiene seis puertos, `ports.go` tiene seis interfaces con sus tipos de soporte mezclados. Navegarlo requiere leer el archivo completo. Con un archivo por puerto, el nombre del archivo ya comunica qué contrato estás leyendo — la misma razón por la que D3 aplica al dominio.

**¿Y `ports.go` en `domain/`?** La convención D3 ya lo prohíbe ahí también. Ambas capas siguen la misma lógica: un concepto, un archivo.

---

### A4 — Output types viven en el archivo del use case; `dto.go` solo para reutilización real

Dos reglas sin ambigüedad:

**Regla 1 — `<Op>Output` y sus sub-tipos van en el archivo del use case.**

El struct de output y cualquier sub-tipo que solo exista para componer ese output pertenecen al mismo archivo que el use case. No importa cuántos sub-tipos tenga el output.

```
get_group.go      → GetGroupUseCase + GetGroupInput + GetGroupOutput
                                       + LeadDisplay + GroupStatistics + UserMembership
list_problems.go  → ListProblemsUseCase + ListProblemsInput + ListProblemsOutput
                                          + ProblemSummary
```

**Regla 2 — `dto.go` solo cuando hay reutilización actual entre 2+ use cases del mismo paquete.**

Un tipo va a `dto.go` si es campo de los outputs de dos o más use cases del mismo paquete. No "podría reutilizarse" — reutilización actual. El archivo contiene el tipo y su función mapper.

```go
// application/user/dto.go  ✅ — UserDTO aparece en ListUsersOutput, UserProfileOutput,
//                                CreateUserOutput, UpdateUserOutput, AdminUpdateUserOutput
type UserDTO struct { ... }
func userToDTO(u *user.User) UserDTO { ... }

// application/material/dto.go  ✅ — MaterialData aparece en el output de Create, Get,
//                                    Pin, Publish, Unpin, Unpublish, Update
type MaterialData struct { ... }
func toMaterialData(m *material.Material) MaterialData { ... }
```

Esto elimina la pregunta "¿es suficientemente importante para ir a `dto.go`?". La pregunta es únicamente: ¿lo usan dos o más use cases?

```
// ❌ JoinRequestDetail en group/dto.go — solo lo usa ListJoinRequestsUseCase
//    Su lugar es list_join_requests.go junto al output que lo contiene
```

---

### A5 — `Execute` es el único punto de entrada de un use case

Cada `UseCase` expone exactamente un método público de negocio: `Execute`. No se permiten variantes nombradas como `GetMyProfile`, `GetUserByNickname`, `Run`, `Handle`, o el nombre de la operación.

Si una operación tiene múltiples variantes de consulta (por ID, por nickname, etc.), cada variante es un use case separado con su propio struct y su propio `Execute`.

```go
// ✅
type GetUserProfileUseCase struct { ... }
func (uc *GetUserProfileUseCase) Execute(ctx, in GetUserProfileInput) (*GetUserProfileOutput, error)

// ❌ — dos métodos de negocio en el mismo struct
type GetUserProfileUseCase struct { ... }
func (uc *GetUserProfileUseCase) GetMyProfile(ctx, userID string) (*UserProfileOutput, error)
func (uc *GetUserProfileUseCase) GetUserByNickname(ctx, ...) (*UserProfileOutput, error)
```

**Por qué?** Un use case con múltiples métodos de entrada es en realidad un *service* disfrazado. Rompe la invariante de que cada use case tiene exactamente una responsabilidad, un input y un output. También hace que los tests sean más difíciles de aislar y que las dependencias del struct crezcan sin criterio claro.

---

### A6 — `errors.go` contiene los códigos de error que solo la aplicación puede determinar

Cada paquete de aplicación que los necesite tiene un `errors.go` con constantes `string`. El naming sigue el mismo patrón que D7:

```go
// application/user/errors.go
const (
    ErrCodeInvalidCredentials  = "INVALID_CREDENTIALS"
    ErrCodeAccountDeactivated  = "ACCOUNT_DEACTIVATED"
    ErrCodeInvalidCode         = "INVALID_CODE"
    ErrCodeExpiredCode         = "EXPIRED_CODE"
    ErrCodeMaxAttemptsExceeded = "MAX_ATTEMPTS_EXCEEDED"
    ErrCodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
)
```

**Criterio de pertenencia:** un código va en `application/<domain>/errors.go` si y solo si es una condición que *el use case mismo detecta* por su lógica de orquestación — no una condición que el dominio o un adapter ya expresan por sí solos.

| Código | Origen | Dónde vive |
|---|---|---|
| `ErrCodeEmailConflict` | El adapter de repositorio detecta un unique constraint | `domain/user/errors.go` |
| `ErrCodeMaterialNotFound` | El repositorio del aggregate no encuentra el registro | `domain/material/errors.go` |
| `ErrCodeGroupNotFound` (en material) | El use case consulta un puerto externo y el grupo no existe | `application/material/errors.go` |
| `ErrCodeInvalidCredentials` | El use case compara password — ningún aggregate lo sabe | `application/user/errors.go` |
| `ErrCodeInsufficientPerms` | El use case evalúa autorización cruzando múltiples objetos | `application/material/errors.go` |

**Sin strings inline.** Todo código de error es una constante nombrada. Un use case que recibe un `apperror` ya construido desde un adapter o dominio lo propaga directamente — no lo redefine.

```go
// ✅
return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "invalid email or password")
return nil, apperror.NewConflict(user.ErrCodeEmailConflict, "email already in use")  // viene del dominio

// ❌
return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "...")   // string inline
return nil, apperror.NewConflict("EMAIL_ALREADY_EXISTS", "...")      // duplica constante del dominio
```

---

### A7 — Lógica compartida entre use cases se extrae a archivos con nombre descriptivo

Cuando dos o más use cases del mismo paquete comparten lógica no trivial que no es un puerto, un use case, ni un DTO, se extrae a un archivo con nombre descriptivo del concepto que agrupa esa lógica. Las funciones son privadas del paquete (minúscula).

**El nombre del archivo describe el concepto, no el rol técnico:**

```
permissions.go  ← guards de autorización compartidos entre use cases  ✅
pagination.go   ← validación de parámetros y cálculo de paginación    ✅

list_helpers.go ❌  — "helpers" no comunica nada sobre el contenido
utils.go        ❌  — mismo problema
```

**Cuándo aplica:** lógica usada por 2+ use cases, no trivial (más de 1-2 líneas), que pertenece claramente a la capa de aplicación — no es lógica de dominio (que iría en `domain/`) ni infraestructura (que iría en `adapter/`).

**Las constantes que usa el archivo van en el mismo archivo.** Si `pagination.go` define `validatePagination` con un límite máximo, la constante `MaxPageLimit` pertenece a `pagination.go`, no al use case que lo usa.

```go
// pagination.go ✅
const maxPageLimit = 100

func validatePagination(page, limit int) error { ... }
func calcTotalPages(total, limit int) int { ... }
func parseSort(...) { ... }
func parseOrder(...) { ... }
```

---

### A8 — Logging siempre con contexto: `slog.XxxContext`

En la capa de aplicación, usar siempre las variantes con contexto de `slog`. Nunca las variantes sin contexto.

```go
// ✅
slog.ErrorContext(ctx, "failed to find user by email", "error", err)
slog.WarnContext(ctx, "user display not found", "user_id", id)

// ❌
slog.Error("failed to find user by email", "error", err)
slog.Warn("user display not found", "user_id", id)
```

**Por qué:** el `ctx` de la request puede llevar valores adjuntos — request ID, trace ID, user ID — que el middleware agrega al inicio de cada llamada. `slog.ErrorContext` los extrae y los incluye en el log automáticamente, haciendo trazable toda la secuencia de eventos de una request. Con `slog.Error` esos valores se pierden. El `ctx` siempre está disponible en `Execute` porque es su primer parámetro.

---

### A9 — Tests: `mocks_test.go` centraliza toda la infraestructura de test del paquete

Cada paquete de aplicación con tests tiene un único archivo `mocks_test.go`. Contiene los dobles de test, helpers, constantes y fixtures — en ese orden:

```
1. Time fixture        var testNow + fixedClock()   ← solo si el dominio usa reloj inyectable
2. Mock types          un bloque por puerto
3. CurrentUser helpers importados de internal/testutil
4. Test constants      const testXxxID (privadas, minúscula)
5. Domain fixtures     func newXxx() con Restore*
```

**Sección 2 — Mock types.** Un bloque por puerto, con header `// ── <Puerto> mock ──`. Cada mock sigue el mismo patrón: struct con campos `fn` opcionales, métodos que delegan al `fn` si está configurado o retornan un default sensato (el camino feliz) si es `nil`. Factory functions para configuraciones recurrentes.

```go
// ── GroupProvider mock ───────────────────────────────────────────────────────

type mockGroupProvider struct {
    existsFn func(ctx context.Context, groupID string) (bool, error)
}

func (m *mockGroupProvider) Exists(ctx context.Context, groupID string) (bool, error) {
    if m.existsFn != nil {
        return m.existsFn(ctx, groupID)
    }
    return true, nil  // default: el grupo existe
}

func groupExists() *mockGroupProvider  { return &mockGroupProvider{} }
func groupNotFound() *mockGroupProvider {
    return &mockGroupProvider{
        existsFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
    }
}
```

**Sección 3 — CurrentUser helpers.** Viven en `internal/testutil/current_user.go` (archivo regular, no `_test.go`) para ser compartidos por todos los paquetes. Los tests los importan directamente.

```go
// internal/testutil/current_user.go
package testutil

func AsAdmin(id string) shared.CurrentUser      { ... }
func AsCoach(id string) shared.CurrentUser      { ... }
func AsContestant(id string) shared.CurrentUser { ... }
```

**Sección 4 — Test constants.** IDs y valores fijos con nombres semánticos, privados del paquete. Prefijos por tipo facilitan la lectura en mensajes de error: `aaa...` usuarios, `bbb...` aggregate principal.

```go
const (
    authorID   = "aaaaaaaa-0000-0000-0000-000000000001"
    modifierID = "aaaaaaaa-0000-0000-0000-000000000002"
    strangerID = "aaaaaaaa-0000-0000-0000-000000000003"
    testProbID = "bbbbbbbb-0000-0000-0000-000000000001"
)
```

**Sección 5 — Domain fixtures.** Aggregates pre-construidos en estados específicos usando `Restore*` (nunca `New*`). Se combinan con factory functions del mock repositorio para expresar "qué hay en la DB" sin ruido.

```go
func newDraftProblem() *domainProblem.Problem {
    return domainProblem.RestoreProblem(..., "DRAFT", ...).WithClock(fixedClock)
}

// en el test:
uc := NewUnpublishProblemUseCase(repoWith(newPublishedProblem()), ...)
```

**Naming:** `mock*` para todos los dobles de test. No `fake*`, no `stub*`.

**Dos niveles de archivos de test:**

`mocks_test.go` contiene únicamente lo compartido entre dos o más archivos de test del paquete. Lo específico de un use case vive en su propio archivo de test.

```
create_material_test.go  ← constructor helper, helpers de input,
                            y todos los Test* de ese use case

mocks_test.go            ← mocks, fixtures y constantes usados
                            por 2+ archivos de test del paquete
```

Si un helper solo lo usa un test, vive en el archivo de ese test — no se mueve a `mocks_test.go` por anticipación.

**Naming del constructor helper:** espeja el nombre del struct en minúscula — igual que Go distingue constructores exportados de privados. Encapsula las dependencias que siempre tienen el mismo valor en todos los tests del archivo.

```go
// producción — exportado, todas las dependencias explícitas
func NewCreateMaterialUseCase(repo, group, member, author) *CreateMaterialUseCase

// test — privado, baked-in defaults para lo que no importa configurar
func newCreateMaterialUseCase(repo *mockMaterialRepository, group *mockGroupProvider, member *mockGroupMemberProvider) *CreateMaterialUseCase {
    return NewCreateMaterialUseCase(repo, group, member, stubAuthorProvider())
}

// ❌
func newCreateUC(...)   // no dice de qué use case
func newCreateGroupUC() // abreviación inconsistente
```

Solo se define si el use case tiene tres o más dependencias y alguna siempre tiene el mismo valor en todos los tests del archivo. Si tiene una o dos dependencias, se construye inline.

---

### A10 — Uso de `TransactionManager` en use cases

`WithTx` envuelve únicamente las escrituras que deben ser atómicas entre sí — el mínimo necesario para garantizar consistencia. El use case se organiza en tres zonas:

```
ANTES del WithTx   → lecturas, validaciones, mutaciones en memoria
DENTRO del WithTx  → solo las escrituras atómicas
DESPUÉS del WithTx → efectos secundarios (emails, sesiones, llamadas externas)
```

**Por qué cada zona:**

- **Lecturas antes**: no necesitan estar en transacción salvo que sean parte de un "verificar y escribir" atómico (ej: check de membresía antes de crear un miembro — va dentro para evitar race conditions).
- **Mutaciones en memoria antes**: `req.Approve()`, `u.UpdateEmail()` son lógica pura — no tocan la DB, no tienen por qué estar dentro de la transacción.
- **Efectos secundarios después**: emails y sesiones no deben revertirse si la operación principal ya committeó. Son best-effort.

```go
// ✅ estructura correcta
u, _ := uc.userRepo.FindByID(ctx, ...)      // lectura — antes
u.UpdateEmail(newEmail)                       // mutación en memoria — antes
req.MarkAsUsed(time.Now())                   // mutación en memoria — antes

uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
    uc.userRepo.Update(txCtx, u)             // escritura atómica
    uc.emailChangeRepo.Update(txCtx, req)    // escritura atómica
    return nil
})

uc.emailSender.Send(ctx, ...)                // efecto secundario — después
```

**El parámetro del callback se llama siempre `txCtx`** — señala explícitamente que ese contexto lleva una transacción activa y que pasarlo a un repositorio lo enlista en ella. Nunca `ctx` (shadowear el contexto exterior es confuso).

```go
// ✅
WithTx(ctx, func(txCtx context.Context) error { ... })

// ❌
WithTx(ctx, func(ctx context.Context) error { ... })  // shadowing
```

---

## 6. Adapter layer conventions

### Ad1 — Adapter struct names mirror the port they implement

El struct del adapter tiene el mismo nombre que el port que implementa — sin importar si ese port vive en `domain/` (repositorios) o en `application/` (providers, readers, services). El paquete ya identifica el dominio y el contexto tecnológico; no hay razón para repetir ninguno de los dos en el nombre del tipo.

```go
// domain/user/repository.go — port en dominio
type Repository interface { ... }

// adapter/user/repository.go — implementación
type Repository struct { db postgres.Querier }
func NewRepository(q postgres.Querier) *Repository

// ❌ — repite el nombre del paquete
type UserRepository struct { ... }
// desde fuera: user.UserRepository  ← "user" aparece dos veces
```

```go
// application/problem/user_provider.go — port en aplicación
type UserProvider interface { ... }

// adapter/problem/user_provider.go — implementación
type UserProvider struct { db *pgxpool.Pool }
func NewUserProvider(db *pgxpool.Pool) *UserProvider

// ❌
type ProblemUserProvider struct { ... }
// desde fuera: problem.ProblemUserProvider  ← "problem" aparece dos veces
```

Para repositorios, la convención D2 (primario = `Repository`, secundario = `<Aggregate>Repository`) rige tanto el port como el struct del adapter:

| Port en `domain/` | Struct en `adapter/` |
|---|---|
| `user.Repository` | `user.Repository` |
| `user.EmailChangeRepository` | `user.EmailChangeRepository` |
| `user.PasswordRecoveryRepository` | `user.PasswordRecoveryRepository` |
| `user.DeactivationRequestRepository` | `user.DeactivationRequestRepository` |
| `user.DeactivationAuditLogRepository` | `user.DeactivationAuditLogRepository` |
| `group.Repository` | `group.Repository` |
| `group.MemberRepository` | `group.MemberRepository` |
| `group.JoinRequestRepository` | `group.JoinRequestRepository` |
| `material.Repository` | `material.Repository` |
| `problem.Repository` | `problem.Repository` |

**El prefijo tecnológico depende de si el paquete nombra la tecnología o el concepto:**

- Si el paquete nombra la **tecnología** (`adapter/postgres/`), el tipo usa el nombre genérico del concepto. El prefijo sería redundante — igual que en la stdlib de Go (`http.Client`, `bytes.Buffer`, `sql.DB`).
- Si el paquete nombra el **concepto** (`adapter/email/`, `adapter/ratelimit/`), el tipo incluye la tecnología. El paquete no la dice, y podrían existir múltiples implementaciones.

```go
// adapter/postgres/ — paquete = tecnología → nombre genérico
type TransactionManager struct { ... }      // ✅
type PostgresTransactionManager struct { ... }  // ❌ redundante

// adapter/email/ — paquete = concepto → nombre incluye tecnología
type SMTPSender struct { ... }     // ✅ distingue de SendGridSender, etc.
type Sender struct { ... }         // ❌ no dice cómo envía

// adapter/ratelimit/ — paquete = concepto → nombre incluye tecnología
type RedisRateLimiter struct { ... }   // ✅
type RateLimiter struct { ... }        // ❌ no dice con qué tecnología
```

**Un struct puede implementar múltiples ports.** Cuando dos ports definen contratos sobre el mismo recurso subyacente y distintos use cases los consumen por separado (ISP), el adapter puede implementarlos en un solo struct. En ese caso el struct toma el nombre del port principal.

```go
// application/material/ — ports separados porque los consumen use cases distintos
type GroupProvider interface {
    Exists(ctx context.Context, groupID string) (bool, error)
}
type GroupVisibilityProvider interface {
    FindVisibility(ctx context.Context, groupID string) (GroupVisibility, bool, error)
}

// adapter/material/group_provider.go — un struct, dos ports, misma tabla
type GroupProvider struct { db *pgxpool.Pool }
func (p *GroupProvider) Exists(...)          // satisface GroupProvider
func (p *GroupProvider) FindVisibility(...)  // satisface GroupVisibilityProvider
```

El criterio para fusionar dos ports en una sola interfaz no es "el adapter los implementa juntos", sino "los mismos use cases siempre necesitan los dos métodos". Si distintos use cases usan subconjuntos distintos, los ports permanecen separados aunque el adapter los agrupe.

### Ad2 — Error translation at the adapter boundary

El adapter es la única capa donde existen los errores externos. Todo error que cruza desde un adapter hacia `application/` debe ser un `apperror` — nunca un error crudo de librería (`pgx.ErrNoRows`, `pgconn.PgError`, `fmt.Errorf`) ni un sentinel de dominio.

**Reglas de traducción para adapters de Postgres:**

| Condición externa | `apperror` a retornar |
|---|---|
| `pgx.ErrNoRows` | `apperror.NewNotFound(domain.ErrCodeXxx, "...")` |
| Unique constraint (`postgres.UniqueViolation`) | `apperror.NewConflict(domain.ErrCodeXxx, "...")` |
| Cualquier otro error inesperado | log + `apperror.NewInternal()` |

Los códigos de error deben ser siempre constantes de dominio (de `domain/<paquete>/errors.go`) — nunca códigos genéricos como `apperror.ErrCodeNotFound`.

El código de unique violation de Postgres es una constante en `adapter/postgres/`:

```go
// adapter/postgres/errors.go
const UniqueViolation = "23505"  // PostgreSQL unique_violation
```

```go
// ✅ adapter/material/repository.go
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, apperror.NewNotFound(material.ErrCodeMaterialNotFound, "material not found")
    }
    slog.ErrorContext(ctx, "database error in FindByID", "error", err, "material_id", id)
    return nil, apperror.NewInternal()
}

// ❌ retorna error crudo — nunca debe cruzar la frontera del adapter
return nil, fmt.Errorf("failed to find material: %w", err)

// ❌ retorna sentinel del dominio — el use case no debería necesitar errors.Is
return domainUser.ErrEmailConflict
```

**Regla de logging:** el adapter logea el error técnico original antes de cada `NewInternal()` — es el único lugar donde ese detalle existe. Una vez convertido a `apperror.NewInternal()`, la causa queda deliberadamente oculta (anti-corruption layer).

La capa de aplicación **no vuelve a loguear** errores que vienen del adapter. Solo logea condiciones que ella misma detecta.

```go
// ✅ application/material/create_material.go
if err := uc.repo.Save(ctx, m); err != nil {
    return nil, err  // propaga — el adapter ya logueó
}

// ❌ double logging — el adapter ya logueó el mismo error
if err := uc.repo.Save(ctx, m); err != nil {
    slog.ErrorContext(ctx, "failed to save material", "error", err)
    return nil, err
}
```

### Ad3 — Repositorios usan `postgres.Querier`, no `*pgxpool.Pool`

Los repositorios de Postgres solo necesitan tres operaciones: `Exec`, `Query`, `QueryRow`. Las tres están definidas en `postgres.Querier`. Guardar `*pgxpool.Pool` expone más de lo necesario — el repositorio no necesita saber que existe un pool, ni sus estadísticas, ni su ciclo de vida de conexiones.

```go
// ✅ campo del tipo mínimo necesario
type Repository struct { db postgres.Querier }
func NewRepository(db postgres.Querier) *Repository

// ❌ expone más de lo necesario
type Repository struct { db *pgxpool.Pool }
```

En `main.go` no cambia nada: `*pgxpool.Pool` implementa `postgres.Querier`, así que `dbPool` se puede pasar directamente.

**Todo método llama `postgres.GetQuerier(ctx, r.db)` al inicio — lecturas y escrituras.**

`GetQuerier` retorna la transacción activa si hay una en el contexto, o el pool si no. Esto garantiza que todas las operaciones del repositorio participen en la misma transacción cuando el use case lo requiere.

```go
// ✅ lectura dentro de una transacción ve los cambios uncommitted
func (r *Repository) FindByID(ctx context.Context, id string) (*user.User, error) {
    q := postgres.GetQuerier(ctx, r.db)
    row := q.QueryRow(ctx, query, id)
    ...
}

// ❌ la lectura no ve los cambios uncommitted de la transacción activa
func (r *Repository) FindByID(ctx context.Context, id string) (*user.User, error) {
    row := r.db.QueryRow(ctx, query, id)  // bypasea la transacción
    ...
}
```

El mecanismo completo:

```
*pgxpool.Pool  ──implementa──▶  postgres.Querier  (Exec, Query, QueryRow)
pgx.Tx         ──implementa──▶  postgres.Querier  (mismos tres métodos)

GetQuerier(ctx, defaultQuerier):
  ctx tiene tx  →  retorna la tx    (es un Querier)
  ctx sin tx    →  retorna el pool  (es un Querier)
```

El repositorio nunca sabe con cuál está hablando. La transacción es un detalle del contexto, invisible para el repositorio.

**Interfaces locales privadas que reimplementan `GetQuerier` están prohibidas.** `memberDBQuerier`, `joinRequestDBQuerier` y patrones similares reinventan exactamente lo que ya hace `postgres.GetQuerier` — deben eliminarse.

### Ad9 — `response.go` es el único lugar con lógica de respuesta HTTP

`adapter/http/handler/response.go` expone exactamente dos funciones públicas: `WriteJSON` y `WriteError`. No existen wrappers privados que las envuelvan — los handlers las llaman directamente.

```go
// ✅ el handler llama directo
handler.WriteJSON(w, http.StatusOK, resp)
handler.WriteError(w, err)

// ❌ wrapper privado que no agrega nada
func respondJSON(w http.ResponseWriter, status int, data any) {
    WriteJSON(w, status, data)
}
```

`response.go` también contiene `kindToStatus` — el único lugar del proyecto donde `apperror.Kind` se mapea a un código HTTP. Ningún otro archivo importa `net/http` para hacer este mapeo.

```go
func kindToStatus(k apperror.Kind) int {
    switch k {
    case apperror.KindValidation:      return http.StatusBadRequest
    case apperror.KindNotFound:        return http.StatusNotFound
    case apperror.KindConflict:        return http.StatusConflict
    case apperror.KindForbidden:       return http.StatusForbidden
    case apperror.KindUnauthorized:    return http.StatusUnauthorized
    case apperror.KindTooManyRequests: return http.StatusTooManyRequests
    default:                           return http.StatusInternalServerError
    }
}
```

### Ad8 — Lógica compartida entre handler packages vive en `adapter/http/handler/`

El paquete raíz `adapter/http/handler/` ya es importado por todos los sub-paquetes de dominio. Las funciones utilitarias que usan dos o más paquetes de handler van ahí, en archivos con nombre descriptivo del concepto.

```
adapter/http/handler/
  response.go    ← WriteJSON, WriteError (ya existe)
  pagination.go  ← ParsePagination, ParseIntParam, WriteBadPagination
```

**Regla:** si una función de utilidad HTTP es necesaria en 2+ paquetes de handler → sube al paquete raíz. Si es específica de un dominio → se queda en el paquete de ese dominio.

Las funciones compartidas se parametrizan para no importar conceptos de dominio:

```go
// adapter/http/handler/pagination.go — sin imports de dominio
func ParsePagination(q url.Values, w http.ResponseWriter, defaultLimit int) (page, limit int, ok bool)
func ParseIntParam(raw string, defaultVal int) (int, error)

// cada handler aporta su propio default — el dominio no contamina el paquete compartido
page, limit, ok := handler.ParsePagination(r.URL.Query(), w, appGroup.DefaultPageLimit)
```

Las utilidades genéricas sin conocimiento HTTP (e.g., conversiones de string/pointer) van en `pkg/`.

No existe `helpers.go` — el nombre no comunica nada sobre el contenido. Los archivos de lógica compartida dentro de un paquete se nombran por el concepto que agrupan (`pagination.go`, no `helpers.go`).

### Ad7 — Naming de tipos de request y response en handlers

Los tipos HTTP del handler layer usan sufijos según su rol:

| Rol | Sufijo | Ejemplo |
|---|---|---|
| Top-level request (body o params que el handler deserializa) | `*Request` | `createGroupRequest`, `updateProblemRequest` |
| Top-level response (struct que el handler serializa a JSON) | `*Response` | `getGroupResponse`, `listProblemsResponse` |
| Sub-componente (tipo anidado dentro de un request o response) | sin sufijo | `pagination`, `author`, `langOverride` |

```go
// ✅
type createGroupRequest struct { ... }         // top-level: recibe el handler
type listGroupsResponse struct {               // top-level: escribe el handler
    Groups     []groupListItem `json:"groups"` // sub-componente: sin sufijo
    Pagination pagination      `json:"pagination"`
}

// ❌
type paginationResp struct { ... }   // sufijo inconsistente
type requestJoinBody struct { ... }  // sufijo incorrecto
```

Si al quitar el sufijo de un sub-componente el nombre queda ambiguo o poco descriptivo, se usa un nombre más específico — no es una excepción a la regla, sino una decisión de naming. `langOverrideRequest` y `langOverrideResp` son casi idénticos: se fusionan en `langOverride` (un solo tipo).

El sufijo `*Body` no existe en este proyecto.

### Ad6 — `handler.go`: struct, campos y constructor

`handler.go` define el struct `Handler` con todos los use cases que el paquete necesita.

**Campos:** nombre = camelCase del use case sin el sufijo `UseCase`. Tipo = el concreto `*app<Domain>.<Operation>UseCase`, nunca una interfaz.

```go
// ✅
type Handler struct {
    createGroup  *appGroup.CreateGroupUseCase
    listGroups   *appGroup.ListGroupsUseCase
    joinGroup    *appGroup.JoinGroupUseCase
}

// ❌ — abreviación que oculta qué hace el campo
type Handler struct {
    createUC  *appGroup.CreateGroupUseCase
    listUC    *appGroup.ListGroupsUseCase
}
```

**Por qué tipos concretos y no interfaces:** el handler es la capa más externa — ya es el adaptador. No hay nadie más afuera que necesite sustituir la implementación del use case. Una interfaz de un único método sin variantes posibles es ruido sin valor.

**Constructor:** los parámetros llevan el mismo nombre que los campos.

```go
func NewHandler(
    createGroup  *appGroup.CreateGroupUseCase,
    listGroups   *appGroup.ListGroupsUseCase,
) *Handler {
    return &Handler{
        createGroup: createGroup,
        listGroups:  listGroups,
    }
}
```

### Ad5 — Un archivo `<operación>_handler.go` por método de handler

Cada método de handler vive en su propio archivo. El nombre del archivo es el snake_case del nombre del método, con sufijo `_handler.go`. No se incluye el nombre del dominio en el archivo — el paquete ya lo da.

```
adapter/http/handler/user/
  create_handler.go                  ← Handler.Create
  request_deactivation_handler.go    ← Handler.RequestDeactivation
  confirm_deactivation_handler.go    ← Handler.ConfirmDeactivation
  request_email_change_handler.go    ← Handler.RequestEmailChange
  confirm_email_change_handler.go    ← Handler.ConfirmEmailChange
```

Un archivo con dos métodos de handler agrupa dos operaciones sin justificación — cada use case tiene su propio `Execute`, cada handler tiene su propio archivo. La misma razón por la que A5 prohíbe múltiples métodos de negocio en un use case aplica acá.

```
// ❌ dos operaciones en un archivo
user/deactivation_handler.go → RequestDeactivation + ConfirmDeactivation

// ✅ una operación por archivo
user/request_deactivation_handler.go  → RequestDeactivation
user/confirm_deactivation_handler.go  → ConfirmDeactivation
```

### Ad4 — HTTP handler structs se llaman `Handler` dentro de su paquete de dominio

El struct principal del handler de un dominio se llama `Handler`. El paquete ya identifica el dominio — repetirlo en el nombre del tipo es redundante, igual que en Ad1.

```go
// ✅
package user
type Handler struct { ... }
func NewHandler(...) *Handler

// desde main.go — el alias del import da el contexto
handlerUser.NewHandler(...)

// ❌ — "user" aparece dos veces
type UserHandler struct { ... }
// handlerUser.UserHandler
```

Esta convención aplica a los paquetes de dominio bajo `adapter/http/handler/`: `user/`, `problem/`, `group/`, `material/`. Los handlers que viven en `adapter/http/handler/` directamente (sin sub-paquete) pueden llevar nombre descriptivo si hay más de uno en el mismo paquete.

### Ad10 — Swagger: todo endpoint HTTP tiene anotaciones completas

Todo método de handler que corresponde a un endpoint HTTP debe tener un bloque de anotaciones `swaggo` inmediatamente antes de la firma del método.

**Anotaciones obligatorias:**

```go
// @Summary      Create group
// @Tags         groups
// @Produce      json
// @Success      201 {object} groupResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /groups [post]
```

**Anotaciones condicionales** (solo cuando aplican):

```go
// @Accept       json          ← solo si el endpoint recibe body
// @Security     BearerAuth    ← solo si requiere autenticación
// @Param        id   path   string true "Group ID"   ← por cada path param
// @Param        body body   createGroupRequest true "..."  ← si hay body
// @Param        page query  int false "Page number"  ← por cada query param
```

`@Description` es opcional — se agrega solo cuando `@Summary` no alcanza para describir un comportamiento no obvio.

---

### Ad11 — Tests de handlers: integración handler + use case

Los handlers usan tipos concretos para los use cases (Ad6), no interfaces. Por tanto los tests no pueden mockear el use case — construyen use cases reales con dependencias mockeadas.

**Estructura de archivos** — análoga a A9:

```
adapter/http/handler/user/
  mocks_test.go                         ← mocks, wrapWithAuth, fixtures compartidas
  update_password_handler_test.go       ← newHandlerWithUpdatePassword + Test*
  confirm_deactivation_handler_test.go  ← newHandlerWithConfirmDeactivation + Test*
  ...
```

**`mocks_test.go` contiene:**
1. Mocks de dependencias de dominio (repositorios, servicios) usados por 2+ archivos de test
2. `wrapWithAuth` — inyecta `TokenClaims` en el contexto de la request simulando el middleware de auth
3. Fixtures de dominio compartidas entre 2+ archivos de test

```go
func wrapWithAuth(h http.Handler, claims *domainuser.TokenClaims) http.Handler {
    tokenSvc := &mockTokenService{validateFn: func(_ string) (*domainuser.TokenClaims, error) {
        return claims, nil
    }}
    return middleware.Auth(tokenSvc, nil)(h)
}
```

**Cada `<operación>_handler_test.go` contiene:**
- Constructor `newHandlerWith<Operation>` — `Handler` con solo ese use case wired, el resto `nil`. Si un test llega accidentalmente a un use case nil, Go entra en pánico — señal correcta de que el test está ejerciendo el camino equivocado.
- Fixtures específicas de esa operación
- Todos los `Test*` del handler

```go
func newHandlerWithUpdatePassword(uc *appuser.UpdatePasswordUseCase) *Handler {
    return &Handler{updatePassword: uc}
}
```

**Package:** `package <domain>` (mismo paquete, no `_test`) — necesario para acceder a los tipos de request no exportados.

**Naming:** `mock*` sin prefijos adicionales — igual que A9. No `mockHandler*`, no `stub*`.

**Qué verifican los tests:** el contrato HTTP (status codes, forma del response body) y comportamientos de negocio observables a través de HTTP. No re-testean lógica que ya cubre el test del use case.

---

## 7. Middleware layer conventions

### M1 — Logging siempre con contexto: `slog.XxxContext`

En middleware HTTP, usar siempre las variantes con contexto de `slog`. El `r.Context()` siempre está disponible y puede llevar request ID, trace ID u otros valores adjuntos por middleware anterior.

```go
// ✅
slog.ErrorContext(r.Context(), "session revocation check failed", "user_id", id, "error", err)

// ❌
slog.Error("session revocation check failed", "user_id", id, "error", err)
```

---

### M2 — Naming: verbo imperativo sin sufijo

Las funciones de middleware (fábricas que retornan `func(http.Handler) http.Handler`) se nombran con verbo imperativo. No llevan prefijo `New` ni sufijo `Middleware`. El paquete ya dice que son middleware.

```go
// ✅
func Auth(tokenService user.TokenService, sessionInvalidator user.SessionInvalidator) func(http.Handler) http.Handler
func RequireRole(required shared.Role) func(http.Handler) http.Handler

// ❌
func NewAuth(...) func(http.Handler) http.Handler          // prefijo New — no es un constructor
func AuthMiddleware(...) func(http.Handler) http.Handler   // sufijo redundante con el paquete
```

Sigue el mismo patrón de la stdlib de Go (`http.StripPrefix`) y frameworks como Chi.

---

### M3 — El middleware guarda `shared.CurrentUser` en el contexto

`Auth` guarda `shared.CurrentUser` en el contexto — no `*user.TokenClaims`. La función `GetCurrentUser` vive en el mismo paquete `middleware` y retorna `(shared.CurrentUser, bool)`.

```go
// ✅ — guarda el tipo de cross-cutting concern de aplicación
ctx := context.WithValue(r.Context(), currentUserKey, shared.CurrentUser{
    ID:   shared.UserID(claims.UserID),
    Role: claims.Role,
})

// handler — solo importa middleware, no domain/user
cu, ok := middleware.GetCurrentUser(r.Context())
if !ok { ... }
```

**Por qué no `*user.TokenClaims`:** `TokenClaims` incluye campos específicos de JWT (`IssuedAt`, raw email) que los handlers no necesitan. Guardar `CurrentUser` elimina la conversión en cada handler y reduce el acoplamiento al paquete `domain/user`.

**Por qué `GetCurrentUser` vive en `middleware` y no en `application/shared/`:** la context key es un detalle del adaptador HTTP. Moverla al interior del hexágono haría que `application/shared/` conozca detalles de implementación del adaptador — viola la regla de dependencias.

---

### M4 — Un archivo por función de middleware

Cada función de middleware vive en su propio archivo. El nombre del archivo es el snake_case de la función, sin sufijo `_middleware`. El paquete ya dice `middleware`.

```
adapter/http/middleware/
  auth.go           ← Auth + GetCurrentUser + currentUserKey + writeError
  require_role.go   ← RequireRole
```

Los helpers privados que pertenecen exclusivamente a una función van en el mismo archivo. Si se agrega un middleware `Logger` o `CORS`, cada uno tiene su propio archivo.

---

### M5 — Tests: mismo paquete, espeja source files, sin comentarios AAA

Los tests de middleware usan el mismo paquete (`package middleware`, no `package middleware_test`). La context key es privada — exportarla solo para los tests sería peor que usar el mismo paquete.

Los archivos de test espejean los source files:

```
auth.go           → auth_test.go
require_role.go   → require_role_test.go
mocks_test.go     ← mocks y helpers compartidos por 2+ archivos de test
```

`mocks_test.go` contiene:
1. Mock types usados por 2+ archivos de test (`mockTokenService`, `mockSessionInvalidator`)
2. Helpers compartidos (`okHandler`, `validClaims`)

Cada `<función>_test.go` contiene helpers y fixtures específicos de esa función, además de todos sus `Test*`.

No se usan comentarios `// Arrange`, `// Act`, `// Assert` — la secuencia del código es suficientemente clara.

---

### M6 — Las dependencias de middleware son siempre requeridas; no-op explícito

Ninguna dependencia de un middleware acepta `nil`. Si un comportamiento es opcional (ej: verificación de sesión revocada), existe una implementación no-op que se pasa explícitamente desde el caller.

```go
// ✅ — el caller decide explícitamente
Auth(tokenSvc, &auth.NoOpSessionInvalidator{})

// ❌ — nil check en el cuerpo: condición que no debería ocurrir en wiring correcto
if sessionInvalidator != nil {
    revoked, err := sessionInvalidator.IsSessionRevoked(...)
}
```

El no-op vive junto a la implementación real en `adapter/auth/`:

```go
// adapter/auth/noop_session_invalidator.go
type NoOpSessionInvalidator struct{}

func (n *NoOpSessionInvalidator) IsSessionRevoked(_ context.Context, _ string, _ time.Time) (bool, error) {
    return false, nil
}
func (n *NoOpSessionInvalidator) InvalidateAllUserSessions(_ context.Context, _ string, _ time.Time) error {
    return nil
}
```

---

### M7 — Las respuestas de error del middleware son escritas por un helper privado

El middleware escribe sus propias respuestas de error con un `writeError` privado. No importa `adapter/http/handler/` para usar `WriteError` — crearía acoplamiento entre middleware y handler dentro del mismo adaptador HTTP.

```go
// ✅ — autónomo, sin dependencias sobre handler/
func writeError(w http.ResponseWriter, status int, code, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    fmt.Fprintf(w, `{"error":%q,"message":%q}`, code, msg)
}
```

Los errores que escribe el middleware son fijos y bien definidos (401, 403, 503). La duplicación mínima con `handler.WriteError` es preferible al acoplamiento.

---

## 8. Router conventions

### R1 — `Handlers` y `Services`: dos structs que agrupan las dependencias de `NewRouter`

`NewRouter` recibe dos structs en lugar de una lista larga de parámetros:

```go
type Handlers struct {
    Problem  *problem.Handler
    User     *handlerUser.Handler
    Auth     *handler.AuthHandler
    Group    *group.Handler
    Material *handlerMaterial.Handler
}

type Services struct {
    TokenService       user.TokenService
    SessionInvalidator user.SessionInvalidator
}

func NewRouter(h *Handlers, s *Services, allowedOrigins []string) *chi.Mux
```

**`Handlers`** — un campo por dominio HTTP. El nombre del campo es el dominio en PascalCase; el tipo es el `*Handler` concreto del paquete del dominio (Ad4).

**`Services`** — las dependencias de infraestructura que el router necesita para configurar middleware. No son dependencias de handlers específicos — los handlers reciben sus dependencias a través de los use cases, no del router.

**`allowedOrigins []string`** queda fuera de `Services` porque es configuración estática de arranque, no un servicio con comportamiento.

---

### R2 — Un grupo por nivel de acceso; el comentario describe el nivel

Las rutas se organizan en bloques según su requisito de acceso. Cada bloque tiene exactamente un comentario en minúscula que describe el nivel de acceso — no el contenido de las rutas:

```go
// public
r.Post("/users", ...)
r.Route("/auth", ...)

// authenticated
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))
    r.Route("/groups", ...)
    r.Route("/problems", ...)
    r.Get("/users/me", ...)
})

// admin only
r.Route("/admin", func(r chi.Router) {
    r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))
    r.Use(middleware.RequireRole(shared.RoleAdmin))
    r.Get("/users", ...)
})
```

**Un solo bloque por nivel de acceso.** Si dos grupos tienen el mismo middleware de acceso, se fusionan. Tener dos grupos `authenticated` separados duplica la configuración sin ninguna ganancia.

El comentario describe el requisito — no el contenido (`"Group and Problem routes"` describe contenido, `"authenticated"` describe acceso).

---

### R3 — `r.Route` para prefijos de URL; `r.Group` para middleware sin prefijo

- **`r.Route("/prefix", ...)`** — agrupa rutas que comparten un prefijo de URL. Aplica a cualquier sub-recurso: `/admin`, `/groups/{id}/requests`, `/problems/p/{slug}/files`.
- **`r.Group(...)`** — agrupa rutas que comparten middleware pero no un prefijo de URL. El caso canónico es el bloque `authenticated`: `/users/me`, `/groups`, `/problems` no comparten prefijo, solo el requisito de auth.

```go
// r.Group — solo middleware, sin prefijo
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(...))
    r.Get("/users/me", ...)    // prefijos distintos entre sí
    r.Route("/groups", ...)
})

// r.Route — prefijo compartido
r.Route("/admin", func(r chi.Router) {
    r.Get("/users", ...)       // → GET /admin/users
    r.Put("/users/{id}", ...)  // → PUT /admin/users/{id}
})
```

---

### R4 — Orden del middleware global

```go
r.Use(cors.Handler(...))   // 1. CORS primero — resuelve OPTIONS preflight antes de cualquier auth
r.Use(chimw.RequestID)     // 2. asigna el ID cuanto antes para que lo usen Logger y handlers
r.Use(chimw.Logger)        // 3. logea con el RequestID ya disponible en el contexto
r.Use(chimw.Recoverer)     // 4. captura panics en handlers — siempre después del logger
```

**Por qué CORS primero:** las peticiones `OPTIONS` de preflight no deben llegar a los middlewares de autenticación ni a los handlers.

**Por qué RequestID antes de Logger:** `chimw.RequestID` pone el ID en el contexto. Si Logger se registra antes, corre sin ese valor y el ID no aparece en los logs.

**Por qué Recoverer al final:** captura panics que ocurran en cualquier handler. Registrarlo antes de Logger significa que un panic sería capturado antes de que Logger pueda registrarlo.

---

## 9. Composition root conventions (`cmd/api/main.go`)

`main.go` es la raíz de composición: el único lugar del proyecto donde se instancian todas las dependencias y se cablea el grafo de objetos. No contiene lógica de negocio — solo construcción y wiring.

### C1 — Variables de use case llevan el sufijo `UseCase`

Todas las variables que almacenan un use case usan el sufijo completo `UseCase`, sin abreviaciones.

```go
// ✅
createProblemUseCase := appproblem.NewCreateProblemUseCase(...)
loginUseCase         := appuser.NewLoginUseCase(...)

// ❌
createProblemUC := ...   // abreviación
loginUC         := ...   // abreviación
```

El sufijo hace que la variable sea inequívocamente identificable como use case en un archivo largo con muchos tipos de objetos (repos, servicios, handlers, use cases).

---

### C2 — Aliases de import en minúsculas

Los aliases de import siguen la convención idiomática de Go: todo minúsculas, sin separadores. El patrón es `<capa><dominio>`:

```go
// ✅
appgroup    "...application/group"
appuser     "...application/user"
appmaterial "...application/material"
appproblem  "...application/problem"

handlergroup    "...handler/group"
handleruser     "...handler/user"
handlermaterial "...handler/material"
handlereproblem "...handler/problem"
```

Los paquetes externos que necesitan alias para evitar ambigüedad usan un nombre corto descriptivo en minúsculas:

```go
googlestorage "cloud.google.com/go/storage"   // "storage" sería ambiguo con fileStorage local
chimw         "github.com/go-chi/chi/v5/middleware"  // convención establecida de Chi
```

---

### C3 — Estructura uniforme de secciones con comentarios en minúscula

El archivo se organiza en bloques con un comentario de sección en minúscula. El orden es fijo:

```
// infrastructure       ← conexiones a sistemas externos (DB, Redis, GCS)
// cross-cutting services  ← servicios usados por múltiples dominios (txManager, JWT, email, etc.)
// <domain> adapters    ← repositorios y providers de ese dominio
// <domain> use cases   ← use cases del dominio (inmediatamente seguidos de su handler)
```

Un bloque por dominio, en el orden en que aparecen en el router. El handler se construye al final de su bloque de use cases — no existe una sección `// handlers` separada.

```go
// problem adapters
problemRepo      := ...
settingsProvider := ...
fileStorage      := ...   // switch de backends va aquí

// problem use cases
createProblemUseCase := ...
...
problemHandler := handlerProblem.NewHandler(createProblemUseCase, ...)

// user adapters
userRepo := ...
...
```

---

### C4 — Una sola instancia de cada servicio compartido

Los servicios de infraestructura sin estado per-dominio (`TransactionManager`, etc.) se instancian una vez y se comparten. No hay instancias duplicadas del mismo servicio con el mismo `dbPool`.

```go
// ✅ — una instancia, pasada a todos los use cases que la necesiten
txManager := infrapostgres.NewTransactionManager(dbPool)

// ❌ — dos instancias idénticas sin razón
txManager      := infrapostgres.NewTransactionManager(dbPool)
groupTxManager := infrapostgres.NewTransactionManager(dbPool)
```

---

### C5 — Variables de repositorio usan el nombre completo del aggregate

Los nombres de variables de repositorio siguen el patrón `<aggregateName>Repo` sin abreviaciones.

```go
// ✅
deactivationRequestRepo  := platformuser.NewDeactivationRequestRepository(dbPool)
deactivationAuditLogRepo := platformuser.NewDeactivationAuditLogRepository(dbPool)

// ❌
deactRepo := ...
auditRepo := ...
```

---

## 10. Literature references

- Cockburn, A. (2005). *Hexagonal Architecture*. https://alistair.cockburn.us/hexagonal-architecture/
- Evans, E. (2003). *Domain-Driven Design: Tackling Complexity in the Heart of Software*. Addison-Wesley.
- Vernon, V. (2013). *Implementing Domain-Driven Design*. Addison-Wesley.
- Hombergs, T. (2019). *Get Your Hands Dirty on Clean Architecture*. Packt.
