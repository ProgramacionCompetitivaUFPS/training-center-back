# Pull Request Feedback - Resoluciones

Este archivo actúa como un registro histórico del feedback recibido en los Pull Requests de la rama `feature/users-management` y detalla cómo se ha solucionado cada observación a nivel arquitectónico y de código. 

Úsalo como referencia en futuras sesiones para evitar duplicar trabajo o comprender decisiones de diseño.

***

### 1. [Invariant Breach] All fields are exported with no access control

**Descripción del issue original:**
Las entidades de dominio como `User` exponían todos sus parámetros (e.g. `user.Status = StatusDeactivated`) como públicos, lo que permitía a cualquier paquete mutar propiedades arbitrariamente burlando las reglas y métodos de negocio.

**Cómo lo solucionamos:**
1. **Encapsulamiento de Dominio:** 
   * Se hicieron estrictamente **privados** todos los campos de `User`, `EmailChangeRequest`, `PasswordRecoveryRequest`, `DeactivationRequest` y `DeactivationAuditLog`.
2. **Exposición controlada (Getters):** 
   * Se añadieron métodos *Getter* públicos (e.g., `user.Status()`) en todas estas entidades para permitir su lectura.
3. **Factorías de Restauración (`Restore*`):** 
   * Se crearon constructores "Restore" (ej. `RestoreUser`, `RestoreEmailChangeRequest`), destinados pura y exclusivamente a la infraestructura (tests y repositorios Postgres).
   * **Por qué:** Asegura que los Use Cases y lógica de negocio usen los métodos verdaderos (ej. `Deactivate()`, `Confirm()`), y que a su vez la infraestructura pueda "rehidratar" objetos salidos de bases de datos hacia estructuras puras ignorando reglas de nueva creación.
4. **Refactor de SQL y Capa de Aplicación:**
   * Se adaptaron todos los Use Cases y Tests a utilizar los *Getters*.
   * Se cambió la forma en la que los Repositorios de Postgres escanean las Queries SQL con `pgx`: ahora primero guardan en propiedades primitivas temporales, y luego las pasan a la factoría de Restauración.
5. **Aseguramiento de Calidad:**
   * Se corrió de forma exitosa y completa la suite de pruebas mediante el container (`golang:1.25`) y el cambio se incluyó bajo el commit unificado `refactor(user): encapsulate User and Domain Entities to enforce invariants`.

***

### 2. [Invariant Gap] `NewUser` does not validate plain-string fields

**Descripción del issue original:**
Los campos de tipo texto plano (`name`, `country`, `city`, `institution`) eran aceptados sin validación estricta en el dominio durante la creación mediante `NewUser`, delegando esta responsabilidad 100% a la capa de Aplicación y derivando en un *"Anemic Domain Model"*.

**Cómo lo solucionamos:**
1. **Factory Guards en Dominio:** Se modificaron las firmas de `NewUser` y `Update` en la entidad `User` para retornar un `error`. Se implementó validación de no-vacío (`strings.TrimSpace(field) == ""`) directamente en estos métodos, resguardando al *Aggregate Root* de corromperse (*Always Valid Domain Model*).
2. **Manejo de Errores en Capa de Aplicación:** Se actualizaron de forma defensiva todos los Casos de Uso que instancian o manipulan a un Usuario (`create_user`, `update_user`, `admin_update_user` y `confirm_email_change`) para capturar este error del Dominio, pero manteniendo la pre-validación a nivel de aplicación para no perder las amigables `apperror.FieldError` lanzadas al usuario final a través de la API.

***

### 3. [Invariant Gap] `Update` accepts a `*Role` but has no guard against promoting to `RoleAdmin`

**Descripción del issue original:**
La especificación establece explícitamente que el rol `ADMIN` no es mutable a través del flujo de actualización estándar. Sin embargo, el método de la entidad `User.Update(...)` aceptaba `role *Role` y cualquier llamador que armara correctamente el input podía escalar privilegios de un usuario silenciosamente, en un claro *Breach* del Invariante (dejándolo relegado exclusivamente a la capa de Aplicación).

**Cómo lo solucionamos:**
1. **Intention-Revealing Interfaces (Split en Dominio):** Se dividió el antiguo y enorme método genérico `Update` en métodos específicos: `Update` (name, nickname, institution) para operaciones de auto-actualización, y `AdminUpdate` (name, nickname, institution, email, role).
2. **Factory Guard en AdminUpdate:** Se codificó la restricción explícita de negocio (`if role != nil && *role == RoleAdmin`) nativamente en el método `AdminUpdate` de la entidad `User`, retornando error si se intenta la escala irregular y preservando así el *Always Valid Domain Model*.
3. **Encapsulamiento del Cambio de Correo:** Se creó un nuevo comportamiento explícito `UpdateEmail(newEmail)` para servir de forma exclusiva al requerimiento de infraestructura de `ConfirmEmailChangeUseCase`.

***

### 4. [Invariant Gap] `Deactivate` can be called on an already-deactivated user

**Descripción del issue original:**
No existía un "guard" que previniera llamar a `Deactivate()` en un usuario que ya tuviera el estado `StatusDeactivated`. Una doble desactivación sobrescribiría el apodo anónimo y resetearía los timestamps, corrompiendo la auditoría.

**Cómo lo solucionamos:**
1. **Factory Guard en Deactivate:** Se modificó `User.Deactivate()` para que retorne un `error`. Se implementó una validación que verifica si el estado actual ya es `StatusDeactivated`, en cuyo caso retorna un error descriptivo.
2. **Encapsulamiento y Validación:** Esta validación en el dominio asegura que el Agregado proteja su propio invariante, independientemente de los chequeos de la capa de aplicación.
3. **Actualización de Capa de Aplicación:** Se actualizaron `admin_deactivate_user.go` y `confirm_deactivation.go` para manejar el nuevo retorno de error de `Deactivate()`, manteniendo la robustez del sistema y cumpliendo con el patrón de "Always Valid Domain Model".

***

### 5. [Critical] `RestoreUser` silently discards parse errors

**Descripción del issue original:**
`RestoreUser` descartaba con `_` los errores de todos los constructores de Value Objects internos (`NewEmail`, `NewNickname`, `NewRole`, `NewStatus`). Si la base de datos contenía datos corruptos (por una migración incorrecta, un `UPDATE` manual, o un bug previo), `RestoreUser` retornaba un `User` con campos en zero-value (`Role("")`, `Status("")`) sin señalar nada al llamador, causando comportamientos impredecibles en capas superiores.

**Cómo lo solucionamos:**
1. **Firma Explícita en Dominio:** Se cambió la firma de `RestoreUser` de `*User` a `(*User, error)`. Ahora cada Value Object que falla retorna un error contextualizado con el `id` del usuario afectado (`fmt.Errorf("restoring user %s: invalid role: %w", id, err)`).
2. **Propagación en Infraestructura:** Se actualizó `scanUser` en `user_repository.go` para capturar y propagar el error de `RestoreUser` hacia todos sus callers (`FindByID`, `FindByEmail`, `FindByNickname`, `FindAll`). Ninguno de estos callers necesitó cambios adicionales porque ya manejaban el error de `scanUser`.
3. **Actualización de Test Helpers:** Los helpers `newActiveUser()` y `newUserWithRole()` (que usan datos hardcodeados válidos) fueron actualizados para manejar el nuevo retorno con `panic`. Un `panic` es el mecanismo idiomático de Go cuando el setup de un test falla de forma no recuperable y la función helper no recibe `*testing.T`.
4. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. La suite completa vía Docker (`golang:1.24` con `GOTOOLCHAIN=auto`) devolvió **exit code 0** con todos los tests en verde.

***

### 6. [Strength / Normalization Gap] `Email` VO stores raw input instead of canonical address

**Descripción del issue original:**
`NewEmail` usaba `mail.ParseAddress` como guardia de validación pero descartaba el resultado (`_`), guardando en `Email.value` el string de entrada original (`trimmed`). `mail.ParseAddress` acepta el formato RFC 5322 *name-addr* (ej. `"John Doe <john@example.com>"`), por lo que una entrada con display name pasaba la validación pero quedaba almacenada con el nombre completo, haciendo que dos instancias que representan el mismo correo real (`NewEmail("John Doe <john@example.com>")` vs `NewEmail("john@example.com")`) tuvieran `value` distintos — rompiendo la propiedad de comparabilidad por valor exigida a todo Value Object.

**Cómo lo solucionamos:**
1. **Normalización canónica en `NewEmail`:** Se modificó el constructor para capturar el resultado de `mail.ParseAddress` y usar `parsed.Address` (siempre de la forma `local@domain`) en lugar de `trimmed` como valor interno del VO. El cambio es de una sola línea en la llamada de retorno.
2. **Determinismo completo:** Con este fix, sin importar si el input es bare address (`user@example.com`) o name-addr (`John Doe <user@example.com>`), el VO siempre almacena la representación canónica. La normalización existente (trim + lowercase) se mantiene intacta y complementa este paso.
3. **Cobertura de tests:** Se añadieron dos casos al cuadro `TestNewEmail_Valid`: `"display name stripped"` y `"display name normalized"`, que verifican explícitamente que el display name es descartado y que la normalización a minúsculas sigue aplicando sobre la dirección extraída.
4. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. Los 10 tests de `internal/domain/user` pasaron de forma nativa (`go test ./internal/domain/... -v -count=1`), incluyendo los dos casos nuevos.

***

### 7. [Critical] Silent failure — `Expire` error is discarded, making rate limiting unreliable

**Descripción del issue original:**
En `RedisRateLimiter.Allow`, la llamada a `r.client.Expire(ctx, key, window)` se ejecutaba de forma *fire-and-forget*: el resultado `*redis.IntCmd` era descartado silenciosamente. Si `Expire` fallaba (timeout de Redis, blip de red), la clave creada por `Incr` quedaba **sin TTL**, produciendo dos escenarios catastróficos:
- **Bloqueo permanente**: el contador crecía indefinidamente, bloqueando al usuario para siempre.
- **Bypass del rate limit**: si Redis evictaba la clave (política `allkeys-lru`), un `Incr` posterior la recreaba desde 1 sin TTL, efectivamente anulando el rate limit.

**Por qué la solución sugerida (Decr rollback) no es suficiente:**
La sugerencia del PR proponía revertir el `Incr` con un `Decr` si `Expire` fallaba. Sin embargo, si Redis está caído (el escenario más probable de fallo), el `Decr` también fallaría, dejando el mismo estado huérfano. El rollback es *best-effort*, no garantizado.

**Cómo lo solucionamos — Lua Script atómico:**
1. **Atomicidad real en Redis:** Se reemplazaron los dos comandos separados (`Incr` + `Expire`) por un único script Lua ejecutado con `redis.NewScript`. Redis ejecuta scripts Lua de forma **monohilo y sin interrupciones**: ningún cliente puede ver un estado intermedio entre `INCR` y `EXPIRE`.
2. **Arquitectura del fix:** `rateLimitScript` se declara como variable de paquete (se inicializa una sola vez). `go-redis` calcula su SHA1 y usa `EVALSHA` en todas las llamadas subsiguientes — más eficiente que enviar el script completo cada vez.
3. **Simplificación del código:** El método `Allow` pasó de 2 RTTs (viajes de red) y múltiples bloques `if` a 1 solo RTT y un único cheque de error, eliminando además la necesidad de rollback.
4. **Tests unitarios con `miniredis`:** Se creó `redis_limiter_test.go` usando `github.com/alicebob/miniredis/v2` — una implementación de Redis en memoria que corre dentro del proceso de tests, sin infraestructura externa. Se cubren 8 casos: intentos bajo el límite, exactamente en el límite, superando el límite, verificación de TTL tras la primera llamada (regresión directa del bug), que llamadas subsiguientes no reinician el TTL, error cuando Redis está caído, borrado de clave en `Reset`, y comportamiento correcto tras un `Reset`.
5. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. Los 8 tests del paquete `internal/platform/ratelimit` pasaron vía Docker (`golang:1.24` con `GOTOOLCHAIN=auto`) con **exit code 0**. La suite completa también quedó en verde.

***

### 8. [Invariant Gap] No character-set constraint on `Nickname`

**Descripción del issue original:**
El constructor `NewNickname` aplicaba normalización a minúsculas y validación de longitud (3–30), pero aceptaba cualquier carácter: espacios internos, símbolos Unicode, caracteres de control, y caracteres especiales de HTML/SQL (`<`, `>`, `"`, `'`). En una plataforma de programación competitiva donde el nickname se usa en URLs públicas y se muestra en rankings, esto representa un invariante de negocio real no expresado en el Dominio.

**Cómo lo solucionamos:**
1. **Invariante de charset en el Value Object:** Se declaró la variable de paquete `validNicknameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)` en `nickname.go`. Al ser una `var` de paquete con `regexp.MustCompile`, el patrón se compila exactamente una vez en el inicio del proceso (sin costo en cada llamada), y hace `panic` en startup si el patrón literal fuera inválido — comportamiento idiomático en Go para patrones estáticos conocidos.
2. **Orden de validaciones (cascada informativa):** La nueva validación se ubica después de la de longitud. Así cada error retornado es el más informativo posible: primero se descarta el caso vacío, luego el de longitud, y solo entonces el de charset.
3. **Momento de aplicación:** La regex se ejecuta sobre `trimmed` (ya en minúsculas), por lo que el patrón solo necesita cubrir `[a-z0-9_-]` sin ambigüedad.
4. **Cobertura de tests ampliada:** Se añadieron 4 casos válidos nuevos (`with hyphen`, `with underscore`, `alphanumeric mix`, `hyphen and underscore combined`) y 8 casos inválidos nuevos (`internal space`, `<`, `>`, `"`, `'`, `ñ` Unicode, `@`, `.`), documentando explícitamente las decisiones de negocio (ej. que la plataforma usa ASCII únicamente).
5. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. Los 21 tests de `internal/domain/user` (9 válidos + 12 inválidos en Nickname, más Email y Password) pasaron nativamente con **exit code 0**.

***

### 10. [Invariant Gap] `Role` and `Status` are `string` aliases — direct construction bypasses constructors

**Descripción del issue original:**
`Role` y `Status` están definidos como `type X string`, lo que permite a cualquier código construir valores arbitrarios sin pasar por `NewRole`/`NewStatus` mediante un simple cast (`user.Role("SUPERUSER")`). Esto es una limitación fundamental del enfoque de string alias para enumeraciones en Go: el constructor provee un camino validado pero no puede forzar su uso.

**Por qué elegimos la mitigación pragmática (no el struct unexported):**
La alternativa "fuerte" (struct no exportado con sentinel values exportados) habría requerido refactorizar la serialización JSON/DB, los métodos `String()`, y todos los sitios de comparación en el código. La mitigación con `IsValid()` cierra el gap en las fronteras del agregado con impacto mínimo, preservando la legibilidad del código existente.

**Cómo lo solucionamos:**
1. **`IsValid()` en los Value Objects:** Se añadió `func (r Role) IsValid() bool` a `role.go` y `func (s Status) IsValid() bool` a `status.go`. Cada método usa un `switch` sobre las constantes válidas — la misma lógica de `NewRole`/`NewStatus` — para que la definición de "válido" viva en un solo lugar canónico.
2. **Guard en la frontera del agregado (`AdminUpdate`):** En `user.go`, `AdminUpdate` es el único método del agregado que acepta un `*Role` proveniente del exterior. Se añadió `if role != nil && !role.IsValid()` **antes** del chequeo de `RoleAdmin`, para rechazar en el dominio cualquier valor construido por cast arbitrario. `NewUser` y `RestoreUser` no necesitan el guard: el primero hardcodea `RoleContestant`/`StatusActive`, y el segundo ya pasa por `NewRole`/`NewStatus` que validan y retornan error.
3. **Tests de cobertura completa:** Se crearon `role_test.go` y `status_test.go` cubriendo: casos válidos de `NewRole`/`NewStatus`, casos inválidos (vacío, lowercase, valores desconocidos, coincidencia parcial), y casos de `IsValid()` incluyendo el escenario específico del bypass por cast (`Role("SUPERUSER")`, `Status("SUSPENDED")`, `Role("admin")`).
4. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. Los 32 tests de `internal/domain/user` (incluyendo los 15 nuevos de role y status) pasaron nativamente con **exit code 0**.

***

### 9. [Critical] `deactRepo.Update` error silently discarded — state machine broken

**Descripción del issue original:**
En `ConfirmDeactivationUseCase.Execute`, los errores de `deactRepo.Update` en tres paths de retorno temprano eran descartados con `_ =`. Esto hacía que las mutaciones de estado (expirar una solicitud, registrar un intento fallido, bloquear al usuario) se aplicaran en memoria pero nunca se persistieran si la escritura en DB fallaba. El efecto crítico estaba en el path del código incorrecto: si `Update` fallaba al persistir el contador de intentos, el atacante nunca acumulaba intentos en la DB (pues `FindPendingByUserID` releía el registro original cada llamada), anulando completamente el mecanismo anti-fuerza-bruta.

**Análisis de impacto por path:**
- **Path de expiración:** La omisión es técnicamente menos grave porque `expiresAt` es un timestamp fijo en la DB; `now.After(req.ExpiresAt())` siempre será `true` independientemente del campo `status`. Sin embargo, se corrigió por consistencia y robustez defensiva frente a futuros cambios en `FindPendingByUserID`.
- **Path del código incorrecto (no bloqueante):** Bug auténtico y crítico — el contador nunca crece en DB, el anti-brute-force está muerto.
- **Path de bloqueo (exceso de intentos):** Bug crítico — el estado `BLOCKED` nunca se persiste, el atacante no queda bloqueado.

**Cómo lo solucionamos:**
1. **Propagación explícita de errores:** Se reemplazaron los tres `_ = uc.deactRepo.Update(ctx, req)` con `if err := uc.deactRepo.Update(ctx, req); err != nil { return apperror.NewInternal() }`. Cuando la escritura en DB falla, el Use Case retorna `500 Internal Server Error` en lugar de continuar con estado inconsistente — comportamiento correcto para un error de infraestructura no recuperable.
2. **Tests de regresión:** Se añadieron 3 casos a `confirm_deactivation_test.go`: `TestConfirmDeactivation_ExpiredCode_UpdateFails_ReturnsInternal`, `TestConfirmDeactivation_InvalidCode_UpdateFails_ReturnsInternal`, y `TestConfirmDeactivation_BlockedState_UpdateFails_ReturnsInternal`. Cada uno simula un fallo de `deactRepo.Update` y verifica que el Use Case retorna un `AppError` con `StatusCode 500`.
3. **Aseguramiento de Calidad:** `go build ./...` compiló limpio. Los 6 tests de `TestConfirmDeactivation_*` pasaron vía Docker (`golang:1.24` con `GOTOOLCHAIN=auto`). La suite completa también quedó en verde con **exit code 0**.
