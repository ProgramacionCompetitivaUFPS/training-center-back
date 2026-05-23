# A7 Violations — application/ (estado actual)

> **A7 recap:** lógica compartida entre 2+ use cases del mismo paquete se extrae a un archivo con nombre descriptivo del concepto (no `helpers.go`, no `utils.go`). Las funciones son privadas. Las constantes que usa ese archivo viven en ese mismo archivo.

> **Nota:** Este archivo refleja el estado actual del código. Trabajo previo ya completado: `list_helpers.go` → `pagination.go` en group; `appshared.ValidatePagination` y `appshared.CalcTotalPages` ya existen en `application/shared/pagination.go`; `maxPageLimit` y `DefaultPageLimit` ya están en `group/pagination.go`.

---

## VIOLACIONES CONFIRMADAS

### V1 — `application/material/get_material.go:89-105`
**Función `checkGroupAccess` definida en un use case file, usada por 2 use cases**

```
get_material.go:53   → checkGroupAccess(...)   ← la llama
get_material.go:89   → func checkGroupAccess   ← aquí está definida ❌
list_materials.go:71 → checkGroupAccess(...)   ← la llama también
```

La función tiene 16 líneas, recibe un `GroupMemberProvider` y retorna `apperror` — es lógica de autorización de acceso al grupo, no lógica de obtención de material. El paquete `group` ya tiene el patrón correcto: `permissions.go` con `requireLeadOrAdmin`. El paquete `material` no tiene ese archivo todavía.

**Fix propuesto:** mover `checkGroupAccess` de `get_material.go` a un nuevo `material/permissions.go`.

---

### V2 — `application/user/request_email_change.go:115-124`
**Función `generateSixDigitCode` definida en un use case file, usada por 3 use cases**

```
request_email_change.go:85   → generateSixDigitCode()  ← la llama
request_email_change.go:115  → func generateSixDigitCode  ← aquí está definida ❌
request_deactivation.go:60   → generateSixDigitCode()  ← la llama también
request_password_recovery.go:77 → generateSixDigitCode()  ← la llama también
```

Tres use cases distintos dependen de esta función. Es lógica de generación de código OTP usando `crypto/rand` — no pertenece al use case de cambio de email.

**Fix propuesto:** mover `generateSixDigitCode` a un nuevo `user/otp_code.go` (o `security_code.go`).

---

### V3 — `application/group/list_join_requests.go:12`
**Constante `maxRequestsPageLimit` en use case file, paquete ya tiene `pagination.go`**

```go
// list_join_requests.go:12  ❌
const maxRequestsPageLimit = 100

// pagination.go:11-14  ← el lugar correcto
const (
    maxPageLimit     = 50
    DefaultPageLimit = 20
)
```

El paquete `group` estableció `pagination.go` como el hogar de todas las constantes de paginación. Tener esta constante en el use case rompe esa convención. A7: "Las constantes que usa el archivo van en el mismo archivo" — el "archivo" para constantes de paginación en el paquete group es `pagination.go`.

**Fix propuesto:** mover `maxRequestsPageLimit = 100` a `group/pagination.go`.

---

## CASOS DEBATIBLES

### D1 — `application/group/pagination.go:13`
**`DefaultPageLimit = 20` es exportado (mayúscula)**

```go
const (
    maxPageLimit     = 50   // ✅ privado
    DefaultPageLimit = 20   // ⚠️ exportado — usado por el adapter
)
```

Usado en `adapter/http/handler/group/helpers.go:35`:
```go
limit, err = parseIntParam(q.Get("limit"), appGroup.DefaultPageLimit)
```

A7 dice "Las funciones son privadas del paquete". Se puede extender el principio a constantes. Dos posiciones:
- **Exportar está bien:** el adapter necesita el default y lo correcto es definirlo en la capa de aplicación.
- **Exportar viola el principio:** el adapter no debería depender de un detalle de paginación de la aplicación; el handler debería tener su propio `defaultPageLimit = 20`.

**Pregunta clave:** ¿el valor default `20` es una regla de negocio (pertenece a application) o una convención HTTP del endpoint (pertenece al handler)?

---

### D2 — `application/user/list_users.go:111-121` → CONFIRMADA COMO FIX
**Paginación inline con números mágicos, no usa `appshared.ValidatePagination`**

```go
page := input.Page
if page < 1 {
    page = 1          // clamping silencioso
}
limit := input.Limit
if limit < 1 {
    limit = 20        // magic number
}
if limit > 100 {
    limit = 100       // magic number
}
```

El spec decía "capped at 100" pero eso era una descripción de la implementación existente, no un requerimiento de diseño consciente. La inconsistencia con el resto de la API (todos los otros endpoints retornan 400 para parámetros inválidos) es un problema real de UX para clientes que consumen múltiples endpoints.

**Decisión:** estandarizar — usar `appshared.ValidatePagination(page, limit, maxPageLimit)`.

**Fix:** reemplazar el bloque de clamping por `appshared.ValidatePagination`, agregar `const maxPageLimit = 100` con nombre, y actualizar el spec (FR-031/FR-032 y el edge case).

---

### D3 — `application/contest/list_contests.go:14`
**`const maxLimit = 100` en use case file — nombre no descriptivo**

```go
// list_contests.go
const maxLimit = 100   // ¿máximo de qué?
```

No es A7 estrictamente (solo 1 list UC en contest, no hay lógica compartida). Pero:
- El nombre `maxLimit` no dice "paginación" — otros paquetes usan `maxPageLimit`
- Si contest eventualmente tiene más list UCs, habría que crear `pagination.go`

---

### D4 — `application/material/list_materials.go:12`
**`const maxLimit = 100` — mismo caso que D3**

Solo 1 list UC en material. Mismo análisis.

---

---

## RESUMEN

| ID | Archivo | Violación | Severidad | Fix |
|----|---------|-----------|-----------|-----|
| V1 | `material/get_material.go:89` | `checkGroupAccess` en use case, usada por 2 UCs | **Confirmada** | Extraer a `material/permissions.go` |
| V2 | `user/request_email_change.go:115` | `generateSixDigitCode` en use case, usada por 3 UCs | **Confirmada** | Extraer a `user/otp_code.go` |
| V3 | `group/list_join_requests.go:12` | `maxRequestsPageLimit` fuera de `pagination.go` | **Confirmada** | Mover a `group/pagination.go` |
| D1 | `group/pagination.go:13` | `DefaultPageLimit` exportado hacia adapter | Debatible | ¿Negocio o convención HTTP? |
| D2 | `user/list_users.go:111` | Clamping silencioso con magic numbers | **Confirmada** | Usar `appshared.ValidatePagination`; actualizar spec |
| D3 | `contest/list_contests.go:14` | `maxLimit` — nombre no descriptivo, 1 solo list UC | Estético | Renombrar a `maxPageLimit` |
| D4 | `material/list_materials.go:12` | `maxLimit` — mismo caso | Estético | Renombrar a `maxPageLimit` |

## Archivos que NO violan A7

- `application/group/permissions.go` — nombre descriptivo, función compartida entre UCs ✅
- `application/group/pagination.go` — nombre descriptivo, lógica compartida entre 3 list UCs ✅
- `application/shared/pagination.go` — utilidades de paginación cross-domain ✅
- `application/problem/icpc_package_parser.go` — es un puerto (interfaz), no lógica compartida ✅
- `application/contest/dto.go` — `buildOutput` es un mapper de DTO compartido, mismo patrón que `group/dto.go` ✅
- `application/material/list_materials.go:145` — `uniqueAuthorIDs` solo usada en ese archivo ✅
- `application/contest/create_contest.go:183` — `deduplicateSlugs` solo usada en ese archivo ✅
- `application/contest/update_contest.go:282` — `hasAnyField` y `deduplicateBySlug` solo usadas en ese archivo ✅
