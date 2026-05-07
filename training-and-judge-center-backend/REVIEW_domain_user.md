# Domain User Review: Architecture Rules D1–D10

Revisión exhaustiva del dominio `internal/domain/user/` contra las reglas de arquitectura definidas en `ARCHITECTURE.md`, enfocada exclusivamente en las reglas del dominio (D1–D10).

**Fecha de revisión:** 2026-05-07  
**Estado general:** ✅ Muy bien estructurado. Una violación de D3, el resto conforme.

---

## Resumen de hallazgos

- ✅ **D1 (sin subdirectorios)**: Conforme. Todo plano en `domain/user/`.
- ✅ **D2 (repository naming)**: Conforme. Primario = `repository.go`, secundarios = `<aggregate>_repository.go`.
- ❌ **D3 (un archivo por concepto)**: **Violación**. `deactivation_request.go` contiene dos conceptos: `DeactivationStatus` (enum/value object) y `DeactivationRequest` (secondary aggregate).
- ✅ **D4 (constructores New*, Restore*)**: Conforme. `New*` valida, `Restore*` sin validación ni error.
- ✅ **D5 (domain/shared/)**: Conforme. Solo contiene `UserID` y `Role`.
- ✅ **D6 (errores con apperror)**: Conforme. Sin `fmt.Errorf`, sin `errors.New`.
- ✅ **D7 (errors.go)**: Conforme. Solo constantes de código de error, sin sentinels.
- ✅ **D8 (tipos de soporte del repositorio)**: Conforme. `SortField`, `SortOrder`, `SearchField`, `UserFilter` viven en `repository.go`.
- ✅ **D9 (convenciones de testing)**: Conforme. Package `user_test`, tabla para value objects, funciones individuales para agregados.
- ✅ **D10 (sin side effects, `now` como parámetro)**: Conforme. Todos los constructores y métodos de mutación reciben `now time.Time`.

---

## Violaciones de reglas de dominio

### ❌ **D3: Un archivo por concepto** — `deactivation_request.go`

**Ubicación:** `internal/domain/user/deactivation_request.go` (líneas 9–42)

**Problema:**
El archivo contiene dos conceptos distintos de dominio:
1. `DeactivationStatus` — enum / value object (líneas 9–41)
2. `DeactivationRequest` — secondary aggregate (líneas 48–143)

D3 es explícito: *"Every domain type lives in its own file, named after the concept in snake_case. This applies uniformly to all DDD building blocks — no exceptions based on complexity."*

Aunque `DeactivationStatus` solo se usa en `DeactivationRequest`, son conceptos separados con ciclos de vida e identidades independientes. El archivo debería reflejar eso.

**Impacto:**
- Menor: `DeactivationStatus` podría reutilizarse en otros contextos sin arrastrar `DeactivationRequest`.
- Navegación: el archivo es más largo y mezcla responsabilidades conceptuales.

**Corrección:**
Dividir en dos archivos:

```
deactivation_status.go
├─ DeactivationStatus struct + constants
├─ NewDeactivationStatus(raw string) (DeactivationStatus, error)
└─ RestoreDeactivationStatus(raw string) DeactivationStatus

deactivation_request.go
├─ DeactivationRequest struct
├─ NewDeactivationRequest(...)
├─ RestoreDeactivationRequest(...)
└─ Métodos (Confirm, RegisterFailure, IsCurrentlyBlocked, etc.)
```

**Riesgo:** Ninguno. Es una reorganización pura.

---

## Cumplimiento: Aspectos destacables

### ✅ **D4: Constructores — excelente patrón**

Todos los types siguen el patrón sin excepciones:
- **Value objects** (`Email`, `Password`, `Nickname`, `Status`, `RequestStatus`): struct con campo privado `value`, `New*` con validación, `Restore*` sin validación.
- **Aggregates** (`User`, `EmailChangeRequest`, `PasswordRecoveryRequest`, `DeactivationRequest`): `New*` recibe parámetros tipados (no strings crudos), `Restore*` no retorna error.
- **Secondary aggregates**: correctamente separados con sus propios repositorios.

Ejemplo virtuoso en `email.go`:
```go
type Email struct{ value string }  // campo privado
func NewEmail(value string) (Email, error)  // valida
func RestoreEmail(value string) Email  // sin error, sin validación
```

### ✅ **D10: Inyección de `now` — aplicado uniformemente**

Todos los constructores y métodos de mutación reciben `now time.Time`:

| Función | Recibe `now` | Ejemplo |
|---------|-------------|---------|
| `NewUser` | ✅ | `func NewUser(id string, now time.Time, email Email, ...) (*User, error)` |
| `User.Update` | ✅ | `func (u *User) Update(..., now time.Time) error` |
| `User.Deactivate` | ✅ | `func (u *User) Deactivate(..., now time.Time) error` |
| `EmailChangeRequest.MarkAsUsed` | ✅ | `func (r *EmailChangeRequest) MarkAsUsed(now time.Time) error` |
| `DeactivationRequest.RegisterFailure` | ✅ | `func (r *DeactivationRequest) RegisterFailure(now time.Time)` |

**No hay `time.Now()` ni `uuid.New()` dentro del dominio.** La I/O se concentra en la capa de aplicación.

### ✅ **D9: Testing — correctamente estratificado**

- **Package**: Todos en `user_test` (externo, fuerza interfaz pública).
- **Tabla vs. función**:
  - `TestNewEmail_Valid` (tabla, línea 9–34 en `email_test.go`)
  - `TestNewPassword_Compare` (función individual, línea 23–36 en `password_test.go`)
  - `TestUser_Update_ChangesName` (función individual, línea 69–81 en `user_test.go`)
- **Helpers**: `t.Helper()` en `makeTestUser` (línea 12 en `common_test.go`).
- **Fixtures compartidas**: `testNow` en `common_test.go` (línea 10).

---

## Análisis de conceptos secundarios

### ✅ **D2: Repository naming** — conforme

| Aggregate | Primary | File |
|-----------|---------|------|
| User | ✅ | `repository.go` |
| EmailChangeRequest | ✅ | `email_change_repository.go` |
| PasswordRecoveryRequest | ✅ | `password_recovery_repository.go` |
| DeactivationRequest | ✅ | `deactivation_request_repository.go` |
| DeactivationAuditLog | ✅ | `deactivation_audit_log_repository.go` |

### ✅ **D8: Tipos de soporte del repositorio**

En `repository.go`:
- `SortField` (enum)
- `SortOrder` (enum)
- `SearchField` (enum)
- `UserFilter` (struct)
- `NewSortField()`, `NewSortOrder()`, `NewSearchField()`, `NewUserFilter()` (validadores)

Todos correctamente colocados. D8 permite esto explícitamente: *"cada archivo de repositorio contiene la interfaz del repositorio y todos sus tipos de soporte directos."*

### ✅ **D6 & D7: Errores — patrón correcto**

`errors.go` contiene:
- Constantes de código (sin sentinel `var`): `ErrCodeEmailConflict`, `ErrCodeNicknameConflict`, etc.
- Constante de configuración de dominio: `RequestExpiryDuration = 15 * time.Minute`

Uso correcto en domain:
```go
// domain/user/user.go línea 141
return apperror.NewConflict(ErrCodeCannotUpdateDeactivated, "cannot update email of a deactivated user")
```

Sin `fmt.Errorf`, sin `errors.New`, sin sentinels. ✅

---

## Checklist de corrección

- [x] D1: Sin subdirectorios internos — ✅
- [x] D2: Repository naming primario/secundario — ✅
- [x] D3: Un archivo por concepto — ❌ **Refactorar `deactivation_request.go`**
- [x] D4: New*, Restore*, state factories — ✅
- [x] D5: domain/shared/ solo UserID, Role — ✅
- [x] D6: apperror, sin HTTP — ✅
- [x] D7: errors.go solo constantes — ✅
- [x] D8: Tipos de soporte en repository.go — ✅
- [x] D9: Testing externos, tabla vs. función — ✅
- [x] D10: Sin time.Now(), uuid.New() en el dominio — ✅

---

## Próximos pasos

1. **Refactoración de `deactivation_request.go`:**
   - Crear `deactivation_status.go` con `DeactivationStatus` enum.
   - Limpiar `deactivation_request.go` para que solo contenga `DeactivationRequest` aggregate.

2. **Sin cambios requeridos** en:
   - Value objects (Email, Password, Nickname, Status, RequestStatus) — excelente estructura.
   - Agregados principales (User, EmailChangeRequest, PasswordRecoveryRequest) — conforme a todos los patrones.
   - Repositories y ports — nombres y organización correctos.
   - Tests — estrategia adecuada (package `_test`, tabla vs. individual).
   - Manejo de errores y inyección de dependencias (`now`) — aplicado uniformemente.

**El dominio está en muy buena forma.** Una refactoración de archivo resuelve completamente los hallazgos.
