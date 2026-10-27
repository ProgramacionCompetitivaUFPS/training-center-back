# Roadmap: validar el Judge System end-to-end en GKE

**Objetivo final**: confirmar que el pipeline completo (API → RabbitMQ → judge-worker → veredicto) funciona de verdad, montando una competencia real con problemas, casos de prueba, un contest y submissions reales — usando el paquete ya descargado en `D:\Programming\Tesis\packages_contest`.

**Cómo usar este documento**: es una guía reproducible, no un diario. Cada fase tiene Qué/Por qué/Para qué (para entender la decisión, no solo copiarla) y el detalle técnico necesario para ejecutarla cuando llegue su turno. Se actualiza el checkbox de estado a medida que se completa cada fase — no se reescribe la fase como si "siempre hubiera estado terminada".

---

## Fase 0 — Encender el cluster ✅ COMPLETA (2026-08-09)

**Qué**: `gcloud container clusters resize training-center --zone us-east1-b --node-pool default-pool --num-nodes 2 --quiet`.

**Por qué**: rutina de ahorro de la sesión anterior había apagado `default-pool` a 0 nodos.

**Para qué**: sin esto no hay `api`/`postgres`/`rabbitmq`/`redis` corriendo — nada de lo que sigue es posible.

**Resultado**: `api`, `postgres`, `rabbitmq`, `redis` en `1/1 Running`. `judge-worker` en 0 réplicas (normal, KEDA lo levanta con el primer mensaje).

---

## Fase 1 — Auditoría: qué dice la documentación vs qué hace el código de verdad ✅ COMPLETA

**Qué**: se contrastó `CLAUDE.md` y `RUNNER_ARCHITECTURE.md` contra el código real antes de asumir nada.

**Por qué**: la memoria y `CLAUDE.md` decían que faltaban cosas (consumidor concurrente, healthcheck del pool, recuperación de submissions caídas, deploy DinD) que en realidad **ya estaban implementadas** (`git log`: `6e14f9a`, `0778577`, `551c9d9`). Ejecutar sobre documentación desactualizada hubiera hecho re-trabajo o decisiones mal informadas.

**Para qué**: tener un mapa confiable de qué falta de verdad antes de diseñar nada.

**Hallazgos confirmados (lo único que realmente falta hoy)**:
1. `POST /problems/p/{slug}/publish` no existe (`unpublish` sí).
2. El spec completo de publish (`specs/Problem management/Change problem visibility/spec.md`) exige un pipeline de validación real (compilar checker/validator, correr el validator contra los inputs, compilar y correr las soluciones contra todos los test cases) — no solo cambiar un status.
3. **Bug preexistente, no relacionado con publish en sí**: la ejecución de checkers personalizados está rota — el código (`internal/adapter/judge/output_comparator.go`) asume un binario ya compilado, pero solo se sube y guarda el código FUENTE, y la imagen del `judge-worker` (`alpine:3.21` + solo `ca-certificates`/`tzdata`) no tiene ningún compilador instalado. Ningún problema con checker personalizado podría judgearse hoy.
4. El Ingress de GKE es nativo (`networking.gke.io`, no nginx) y no tiene `BackendConfig` — el timeout del balanceador cae al default de 30s, insuficiente para una respuesta de publish que puede tardar varios minutos.
5. `GET /users/me/dashboard` y varios otros "endpoints pendientes" que mencionaba `CLAUDE.md` **ya existen** — la sección de pendientes de `CLAUDE.md` está desactualizada y se corregirá al final de este roadmap.

**El paquete de la competencia real** (`D:\Programming\Tesis\packages_contest`, 12 zips A-L) resultó ser formato **BOCA Online Contest Administrator** (`description/`, `compile/`, `run/`, `compare/`, `limits/` por lenguaje) — no el formato ICPC/Kattis que ya entiende `POST /problems/import`. El "checker" de BOCA es en realidad un script genérico de `diff` (no un checker custom real), y los test cases no vienen separados en sample/secret. La adaptación de este formato es la Fase 9 (al final), por decisión explícita: primero se arregla el judge, después se acomoda la competencia sobre un judge que ya funciona.

---

## Fase 2 — Modelar `ProblemValidation` (dominio + migración) ✅ COMPLETA (2026-08-09)

**Qué**: nuevo agregado secundario `domain/problem.ProblemValidation` (id, problemID, requestedBy, status PENDING/RUNNING/PASSED/FAILED/SYSTEM_ERROR, requestedAt, completedAt, resultJSON opaco) + tabla `problem_validations` (migración 033) + extender `JudgingFile` con `compiledKey`/`compiledAt` (para trackear el artefacto compilado del checker/validator, separado del código fuente subido).

**Por qué**: publicar un problema no es instantáneo (necesita correr en el judge-worker), así que hace falta una fila en Postgres que represente "este intento de validación está en curso" — es lo que le permite a un handler HTTP saber cuándo terminó un trabajo async. Es el mismo concepto que ya existe para `submissions` (una tabla que modela un intento de compilar+correr código con un status que avanza), adaptado a "validar un problema" en vez de "juzgar una submission".

**Para qué**: es la base de datos sobre la que se construye todo lo demás — sin esto no hay forma de que el API sepa si el worker ya terminó de validar.

**Decisión de diseño ya tomada**: `result` es JSONB (no columnas planas) porque el reporte de validación es genuinamente variable (falta un campo, o falló un test case, o no compiló el checker — formas distintas), a diferencia del veredicto de una submission que es un puñado fijo de escalares. Ya hay precedente de JSONB en la misma tabla `problems` (`checker`/`validator`/`solutions`).

**Una decisión que corrige al diseño original**: `application/judge` (la capa donde vive todo el motor del judge) **no** va a importar el agregado rico `domain/problem` para mutar el `Problem` — eso rompería una convención que el proyecto ya sigue (`application/judge` habla con `problem` solo a través de un DTO plano, `ProblemLimits`, nunca importando el dominio directo). En cambio, se usa un port angosto nuevo con métodos planos (`SetCheckerArtifact`, `SetValidatorArtifact`, `MarkPublished`) implementado con SQL directo.

**Resultado**: construido pieza por pieza, con revisión de cada archivo contra `ARCHITECTURE.md` antes de darlo por bueno:
- `problem_validation_status.go` (Value Object, calcado de `submission/status.go`, factorías exportadas por D4) + tests de tabla.
- `problem_validation.go` (Entity/agregado secundario, `Start`/`markFinal`/`Mark*` calcado de `Submission`) + tests individuales por escenario.
- `problem_validation_repository.go` (port con 3 métodos: `Save`, `FindByID`, `FindLatestByProblemID` — se simplificó de un diseño inicial de 4 métodos al notar que el índice único ya garantiza que "la más reciente" y "la activa" son siempre la misma fila).
- `judging_file.go` extendido con `compiledKey`/`compiledAt` de solo lectura (sin setter — se escribe por SQL directo desde el worker, Fase 6).
- `internal/adapter/problem/repository.go` actualizado para persistir/leer los campos nuevos.
- Migración `034_create_problem_validations_table.sql` (renumerada desde `033` al rebasear: `develop` había usado ese número para `033_create_oauth_identities_table.sql`), probada localmente contra el Postgres de `docker-compose` (ciclo up→down→up verificado) — **no aplicada contra el cluster de GKE**, a propósito: nada en el código todavía usa la tabla (el port no tiene implementación real hasta la Fase 4), así que no había motivo para tocar el entorno real todavía.
- Todo el proyecto compila (`go build ./...`) y los tests de dominio y del adaptador pasan.

**Estado**: ✅ completa.

---

## Fase 3 — Publish "camino rápido": solo campos requeridos ✅ COMPLETA (2026-08-09)

**Qué**: `PublishProblemUseCase` + handler + ruta `POST /p/{slug}/publish`, pero **solo** el paso 1 del spec (título/statement/timeLimit/memoryLimit/testCases/solución presentes) — sin tocar Docker ni la cola todavía. Si falta algo, 400 inmediato con `missingFields`.

**Por qué**: este chequeo es pura inspección en memoria de un `Problem` ya cargado — no necesita ni Docker ni al worker. Separarlo de la parte pesada permite tener algo funcional y verificable rápido, sin esperar a que esté lista toda la máquina de compilación nativa.

**Para qué**: primer hito real y demostrable (un problema incompleto ya no puede "publicarse" silenciosamente), y valida que el modelo de `ProblemValidation` de la Fase 2 encaja antes de construir encima.

**⚠️ Huecos temporales que esta fase deja a propósito, para que la Fase 4 los cierre** (no son bugs, son incompletitud intencional documentada):
- `internal/application/problem/publish_problem.go`, `PublishProblemUseCase.Execute`: el camino "todos los campos requeridos están presentes" hoy devuelve directamente `apperror.NewServiceUnavailable(ErrCodeValidationPipelineNotReady, ...)` (503) — no hay máquina de validación real todavía. La Fase 4 reemplaza esa línea por: crear el `ProblemValidation`, encolar, y esperar el resultado.
- `internal/adapter/http/handler/problem/publish_handler.go`, `Publish`: como hoy la use case solo puede devolver error o "faltan campos", el handler asume que "si no hubo error, faltan campos" y siempre responde 400. La Fase 4 tiene que agregar la rama real: si la use case devuelve un `ValidationID` (caso completo), hacer el poll loop y responder 200/400/503 según corresponda — ya no vale asumir 400 a ciegas.

**Resultado**: `PublishProblemUseCase` (con `requiredFieldsForPublish`), handler, ruta y wiring en `cmd/api/main.go` — mismo patrón que `unpublish`. **Probado de punta a punta contra el API real corriendo localmente** (no solo `go build`): admin sembrado con `cmd/seed`, login real vía `POST /auth/login` (`MOCK_AUTH` no existe — hallazgo anotado arriba), problema de prueba creado, y las dos ramas verificadas con curl:
- Problema incompleto → `400` con `validationLogs`/`missingFields` exactos.
- Problema con título+statement+límites+test cases+solución → `503 VALIDATION_PIPELINE_NOT_READY` (el placeholder a propósito de arriba).

**Estado**: ✅ completa (con los huecos de arriba, documentados y a propósito, y confirmados en la prueba real).

---

## Fase 4 — El mecanismo síncrono-sobre-asíncrono (cola + poll) ✅ COMPLETA (2026-08-12)

**Qué**: el handler de publish, para problemas completos, encola un mensaje y **bloquea sondeando** (poll) una fila de Postgres hasta que el worker termine — respetando el timeout del request. Se reutiliza la cola `submissions` existente (no una cola nueva) agregando un discriminador `kind` (`SUBMISSION` | `PROBLEM_VALIDATION`) al mensaje. En esta fase, `ValidateProblemUseCase` (el lado del worker) solo compila y corre las soluciones contra los test cases (usando el `Executor` que YA funciona) — checker/validator se tratan como si no existieran todavía.

**Por qué**: el spec pide que `POST /publish` responda en la MISMA llamada HTTP con el resultado final (200 con logs de validación, o 400 con el detalle de qué falló) — pero el motor de ejecución (Docker) solo vive en los pods `judge-worker`, no en el API. No existe ningún precedente en el código de un handler esperando a una tarea async (se verificó por búsqueda exhaustiva) — es arquitectura nueva, deliberada.

**✅ Decisión de la cola (discutida a fondo, ya cerrada)**: una sola cola `submissions` compartida, con un discriminador `kind` (`SUBMISSION` | `PROBLEM_VALIDATION`) en el mensaje. Se descartó explícitamente una cola separada: requeriría dos consumer loops compartiendo un mismo semáforo, sin ninguna garantía real de que las validaciones "ganen" la contención frente a submissions (terminaría siendo esencialmente azaroso cuál de los dos loops agarra el próximo cupo libre) — reinventando a mano algo que el mecanismo de prioridades de RabbitMQ ya resuelve, y que ya está probado en este proyecto (`TestPublish_PriorityOrdering_HighestFirst`).

**✅ Decisión de prioridad (revisada — publish-validation queda POR ENCIMA de contest, no igual)**: se consideró primero "prioridad 4, igual que contest" para evitar tocar infraestructura, pero se revisó: publish-validation es el **único lugar de todo el sistema donde alguien queda bloqueado en la misma conexión HTTP esperando el resultado** — una submission de contest, por prioritaria que sea, nunca tiene a nadie "colgado" en vivo (el contestant consulta el resultado después, en otra llamada). Eso pesa más que evitarse un cambio de infraestructura, sobre todo estando todavía en fase de validación previa a producción — es el momento más barato de todo el proyecto para pagar ese costo. Decisión final: **prioridad 5** (nivel nuevo, por encima de contest=4). Esto obliga a subir `x-max-priority` de 4 a 5 en la declaración de la cola `submissions`, lo que a su vez requiere **borrar y recrear esa cola en el RabbitMQ de GKE** antes de desplegar esta fase (no se puede cambiar ese argumento en caliente sobre una cola que ya existe) — agregar este paso explícito a la Fase 5 (infraestructura) o como prerequisito de la Fase 8 (verificación contra el cluster real).

**✅ Nuevo requisito encontrado en esta discusión — recuperación de validaciones trabadas**: el timeout de 9 minutos del handler (ver más abajo) protege al *cliente* (le corta la espera), pero no arregla el *dato* — si el pod del worker se cae a mitad de una validación, ningún timeout interno se entera (el proceso entero desaparece), y el ticket queda para siempre en `RUNNING`. Como el índice único de la Fase 2 solo permite una validación activa por problema, un ticket trabado en `RUNNING` bloquearía **para siempre** cualquier intento futuro de publicar ese problema. Hace falta un barrido periódico que mire `problem_validations`, igual al que ya existe para submissions (`RecoverStaleSubmissionsUseCase`, cada 5 minutos marca `SYSTEM_ERROR` lo que lleva demasiado tiempo en `RUNNING`) — agregar el equivalente para validaciones como parte de esta fase.

**⚠️ Segundo punto pendiente para esta fase — resiliencia ante desconexión del cliente**: si la persona que pidió publicar cierra la pestaña antes de que el handler termine de esperar, el trabajo del worker sigue igual (no depende de la conexión HTTP) y el resultado final queda guardado — pero hoy no hay forma de que el frontend consulte "¿cuál fue mi último intento de validación y por qué falló?" sin tener guardado el ID del ticket (que se perdió junto con la pestaña). Hace falta un endpoint/consulta adicional: "última validación de este problema, por slug" (no solo "¿hay una activa?", que ya estaba planeado para evitar duplicados). Esto también le da un propósito visible a `RUNNING`: si alguien recarga la página mientras el worker sigue trabajando, esta consulta se lo muestra. **Actualizar `specs/Problem management/Change problem visibility/spec.md`** con este endpoint una vez que se defina su forma exacta en esta fase — no antes, para no tener que reescribir el spec si la forma cambia en el camino.

**Para qué**: es el corazón de la feature — sin esto, no hay forma de que "publicar" dispare una validación real y el cliente se entere del resultado en la misma llamada.

**Progreso — la cola ya está lista (2026-08-10)**: construida y probada de punta a punta, con varias vueltas de diseño en vivo:
- `internal/adapter/queue/message_kind.go` — `messageKind`, un tipo cerrado (campo privado, no un `string` suelto) para las etiquetas de la cola, con `MarshalJSON`/`UnmarshalJSON` que rechaza cualquier valor que no sea uno de los conocidos — ni siquiera un mensaje malformado puede colar una etiqueta rara.
- `internal/adapter/queue/rabbitmq_queue.go` — quedó como el núcleo puramente genérico: declara/reconecta la cola, arma el sobre (`queueEnvelope`), y `Consume` recibe una **lista** de manejadores (`...payloadHandler`) en vez de parámetros fijos — verificando, antes de arrancar, que haya un manejador para cada etiqueta conocida (si falta uno, el worker no arranca, en vez de perder mensajes en silencio). El núcleo ya no importa `appsubmission` ni `appproblem` — no le hace falta saber que esos tipos existen.
- `internal/adapter/queue/rabbitmq_submission_queue.go` / `rabbitmq_validation_queue.go` — dos colas hermanas simétricas (no una "principal" y otra "de segunda"), cada una con su `Publish`, su función de parseo (testeable sola, sin RabbitMQ) y su constructor de manejador exportado.
- `maxPriority` subido a 5 (antes había quedado en 4 por error, encontrado en revisión — hubiera hecho que publish-validation compitiera como si fuera prioridad de contest, no la más alta).
- `cmd/api/main.go` y `cmd/worker/main.go` actualizados a la nueva forma. El worker tiene un placeholder explícito para el manejador de validación (`// TODO: wire ValidateProblemUseCase`) — libera el pipeline de submissions reales sin esperar a que el resto de la fase esté listo.
- Tests: `rabbitmq_queue_test.go` (integración, contra RabbitMQ real vía testcontainers) arreglado y **corrido de verdad** — pasa. Más 18 tests rápidos nuevos (`message_kind_test.go`, `rabbitmq_submission_queue_test.go`, `rabbitmq_validation_queue_test.go`, `consume_test.go`) sin necesidad de Docker.

**Progreso — el adaptador de Postgres ya está listo (2026-08-10)**: `internal/adapter/problem/problem_validation_repository.go` implementa `Save`/`FindByID`/`FindLatestByProblemID` contra la tabla `problem_validations` (upsert por `id`, detecta la violación del índice único y la traduce a `ErrCodeValidationInProgress`). Se agregó también el código de error que había quedado pendiente de escribir en la Fase 2 (`domainProblem.ErrCodeValidationInProgress`). Probado en dos niveles: 7 tests con `mockQuerier` (mismo patrón que ya usa `adapter/submission`) + una prueba real contra el Postgres local (insertar, leer, upsert de estado, y confirmar que el índice único bloquea una segunda validación activa y se libera al terminar la primera) — los mocks no pueden detectar errores de sintaxis SQL real, así que valió la pena hacer las dos.

**Progreso — `PublishProblemUseCase` ya usa la cola de verdad (2026-08-10)**: reemplaza el placeholder 503 por la lógica real — revisa si ya hay una validación activa (`FindLatestByProblemID`) para no encolar duplicados, crea el `ProblemValidation` y lo guarda, publica el mensaje con prioridad `QueuePriorityPublishValidation`. Maneja también la carrera de inserción (dos pedidos casi simultáneos): si `Save` pierde contra el índice único, busca de nuevo y reusa la validación que ganó, en vez de devolver un error confuso. `cmd/api/main.go` actualizado (el nuevo `problemValidationRepo` y `validationQueue` — con su propio `NoOpValidationQueue` para desarrollo sin RabbitMQ — construidos junto a `submissionQueue`, y `publishProblemUseCase` movido a esa sección porque ahora depende de ellos). 8 tests nuevos cubriendo cada rama (campos faltantes, ya publicado, sin permiso, encolar de cero, reusar activa, reusar tras perder la carrera, reemplazar una terminal, fallo al encolar) — todos pasan, junto con el resto del proyecto.

**Progreso — consultar el estado de una validación ya existe (2026-08-10)**: puerto angosto `ProblemStatusProvider` (`GetStatus`) en vez de agregar `FindByID` al `problem.Repository` completo — evita forzar a todos los implementadores del repositorio a cargar con un método que solo hace falta acá. `GetProblemValidationStatusUseCase` busca la validación por ID; si no es terminal devuelve `Terminal:false`, si lo es decodifica `resultJSON` en los DTOs del spec (`ValidationReport`, `FailedTestCase`, `CompilationErrors`, `FailedInput`) y adjunta el status actual del problema.

**Progreso — el poll loop ya está escrito, y vive en `application/`, no en el handler (2026-08-10)**: el primer borrador ponía el sondeo (`awaitValidation`, con `time.Ticker` y `context.WithTimeout`) directamente en `publish_handler.go`, con el interval/timeout como campos de `Handler` para poder acortarlos en tests. En revisión con el usuario se identificó que eso era orquestación de negocio, no traducción HTTP, y que forzaba a un struct compartido por 19 métodos a cargar con dos `time.Duration` que solo le servían a uno. Se movió a un caso de uso nuevo: `AwaitProblemValidationUseCase` (`application/problem/await_problem_validation.go`), que reusa `GetProblemValidationStatusUseCase` para cada chequeo y sondea cada 750ms hasta que la validación termina o pasan 9 minutos. Distingue un timeout real (devuelve `apperror.NewServiceUnavailable(ErrCodeValidationTimedOut, ...)`) de que el `ctx` padre ya venía cancelado (lo devuelve tal cual, sin envolverlo). El handler queda delgado: llama `Execute` una sola vez y, si hay error, revisa `r.Context().Err()` para decidir si el cliente se desconectó (no escribe nada) o si debe traducir el error a HTTP. `GetProblemValidationStatusUseCase` sigue existiendo aparte — sirve para el futuro endpoint no bloqueante de "última validación por slug" (ver más abajo).

12 tests nuevos: `TestGetProblemValidationStatus_*` (6, chequeo único), `TestAwaitProblemValidation_*` (5, incluye sondeo repetido, timeout real, y desconexión del cliente — todos deterministas, sin esperar tiempo real gracias a que `AwaitProblemValidationUseCase` es de mismo paquete que sus tests), y `TestPublish_ClientDisconnected_WritesNothing` a nivel handler (prueba que la delegación al caso de uso está bien conectada).

**Progreso — el lado del worker ya está completo (2026-08-11/12)**: se dividió en dos casos de uso, en dos paquetes distintos, para no crear el primer import cruzado entre paquetes de `application/`:
- `application/judge/validate_solutions.go` (`ValidateSolutionsUseCase`) — mecánica pura, no sabe qué es un ticket. Reusa las 4 mismas dependencias que ya usa `JudgeSubmissionUseCase` (`Executor`, `SourceCodeDownloader`, `TestCaseProvider`, `OutputChecker`) más un port nuevo, `SolutionProvider` (lee las soluciones del problema — nadie lo necesitaba hasta ahora, una submission trae su propio código). Compila y corre cada solución contra cada caso de prueba, parando en la primera falla (decisión explícita: más simple, encaja con el formato del spec). Reintenta por solución ante fallos transitorios de infra (mismo patrón que `judgeAttempt`). El comparador es siempre por tokens (`OutputChecker` con `CheckerPath: ""`) — el checker personalizado queda para la Fase 6, pero el código ya está preparado: cuando exista un `CheckerPath` real, alcanza con empezar a pasarlo.
- `application/problem/validate_problem.go` (`ValidateProblemUseCase`) — dueño del ciclo de vida del ticket: lo marca `RUNNING`, arma el `ValidationReport` (`ValidationSummary`/`FailedTestCases`/`CompilationErrors`, con los campos exactos que pide el spec — incluyendo `expected`/`actual` para wrong answer y el límite configurado para timeout, que se agregaron retroactivamente a `SolutionFailure`), y marca `FAILED`/`SYSTEM_ERROR`, o `PASSED` + `MarkPublished` **juntos en una sola transacción** (si `MarkPublished` falla, el ticket queda intacto en `RUNNING` para que lo agarre el barrido de recuperación, en vez de quedar en un estado a medias).
- El puente entre ambos paquetes: `application/problem/solution_validator.go` define el port `SolutionValidator` con tipos propios (nunca importa `judge`); `adapter/problem/solution_validator.go` es el único archivo de todo el sistema que importa los dos paquetes de aplicación a la vez, y solo traduce.
- `adapter/problem/problem_publisher.go` — `UPDATE` SQL directo de una sola columna (no carga el agregado rico `Problem`, que ni siquiera tiene `FindByID`).
- Salvaguarda agregada en revisión: `Expected`/`Actual`/`CompileLog` se truncan a 2000 bytes antes de construir el `SolutionFailure` — sin esto, un caso de prueba de varios MB terminaría completo en la fila de Postgres y en la respuesta HTTP del publish.
- `cmd/worker/main.go` — reemplazado el placeholder; el consumidor de validación ahora ejecuta `ValidateProblemUseCase.Execute` de verdad.
- Corrección al spec (`specs/Problem management/Change problem visibility/spec.md`): el ejemplo de runtime error usaba `status`/`details` en vez de `verdict`, inconsistente con wrong-answer/timeout — se alineó a `verdict: "RUNTIME_ERROR"`.
- Tests: 12 nuevos en `ValidateSolutionsUseCase` (incluye 3 de truncado), 6 en `ValidateProblemUseCase`, 7 en el adaptador `SolutionProvider`, 3 en el adaptador `ProblemPublisher`, 3 en el traductor `SolutionValidator` — todos pasan (corridos vía Docker, WDAC bloqueó los binarios nativos de forma persistente en esta parte de la sesión).

**Progreso — barrido de recuperación de validaciones trabadas (2026-08-12)**: mirror exacto de `RecoverStaleSubmissionsUseCase`, en `application/problem` (no en `judge` — es el ciclo de vida del ticket, igual que `ValidateProblemUseCase`): `StaleValidationRecoverer` (port) + `RecoverStaleValidationsUseCase` + `adapter/problem/stale_validation_recoverer.go` (`UPDATE ... SET status = 'SYSTEM_ERROR' WHERE status = 'RUNNING' AND requested_at < cutoff` — `problem_validations` no tiene `updated_at`, así que se usa `requested_at`, razonable porque `Start()` pasa a `RUNNING` casi inmediatamente después de crear el ticket). Umbral propio, `staleValidationAfterMinutes` (default 20 min) en `judge_config.yaml` — deliberadamente más alto que `staleRunningAfterMinutes` (10 min, para submissions), porque el timeout del cliente HTTP en `AwaitProblemValidationUseCase` ya es de 9 minutos: reusar el mismo umbral de submissions hubiera dejado solo 1 minuto de margen antes de marcar `SYSTEM_ERROR` una validación que todavía corre de verdad. Conectado en `cmd/worker/main.go` con el mismo patrón de goroutine+ticker de 5 minutos que ya existe para submissions. 5 tests nuevos (2 del caso de uso, 3 del adaptador).

**Progreso — cerrado el segundo punto pendiente: endpoint "última validación por slug" (2026-08-13)**: `GET /problems/p/{slug}/validation`. Nuevo `GetLatestProblemValidationUseCase` (`application/problem/get_latest_problem_validation.go`) — usa el mismo modelo de permisos que `Update`/`UploadFiles`/`DeleteFile` (`CanBeEditedBy`, porque es información de un problema DRAFT, no pública), busca la validación más reciente con `FindLatestByProblemID`, y si existe compone `GetProblemValidationStatusUseCase` (el mismo caso de uso que ya usaba `AwaitProblemValidationUseCase`) en vez de duplicar la decodificación de `resultJSON` — una lectura extra a la base de datos, aceptada a cambio de no repetir esa lógica. Si no hay ningún intento todavía, devuelve `Found:false` sin error (no es un 404 — es información legítima: "nunca se intentó publicar"). El handler (`get_latest_validation_handler.go`) siempre responde 200 (es una consulta de estado, no un endpoint bloqueante como `/publish`). 5 tests en el caso de uso + 4 a nivel handler (401, 404 de problema, `found:false`, encontrada con reporte decodificado) — todos pasan, junto con el resto de la suite.

**Pendiente derivado — actualizar el spec**: ahora que la forma del endpoint está definida, falta actualizar `specs/Problem management/Change problem visibility/spec.md` con `GET /problems/p/{slug}/validation` (según la nota de la línea de arriba, esto se dejó pendiente a propósito hasta tener la forma final).

**Estado**: ✅ completa — todo el camino síncrono-sobre-asíncrono (API, cola, worker, publish real, recuperación de validaciones trabadas, consulta de última validación) está terminado y probado de punta a punta. Suite completa del proyecto corrida y en verde.

---

## Fase 5 — Infraestructura: Dockerfile, recursos del worker, timeout del Ingress ⏳ PENDIENTE

**Qué**:
- Separar el `Dockerfile` en dos imágenes finales: `api-final` (liviana, como hoy) y `worker-final` (con `g++`, JDK, `python3` para compilar checkers/validators nativamente). Implica actualizar `deploy/k8s/bootstrap.ps1` y el pipeline de CI/CD para construir y pushear 2 imágenes en vez de 1.
- Subir los límites de recursos del container `worker` en `deploy/k8s/judge/worker.yaml` (de `cpu:500m/memory:512Mi` a `cpu:2/memory:2Gi`) — la compilación nativa corre DENTRO de ese container, no en el sidecar DinD.
- Agregar un `BackendConfig` de GKE (`deploy/k8s/ingress/backendconfig.yaml`, `timeoutSec: 600`) referenciado desde el Service `api`, para que el balanceador no corte una respuesta de publish que tarde varios minutos.
- Borrar y recrear la cola `submissions` en el RabbitMQ de GKE (`x-max-priority` sube de 4 a 5 por la prioridad nueva de publish-validation, decidida en la Fase 4 — RabbitMQ no permite cambiar ese argumento en caliente sobre una cola existente).

**Por qué**: son tres gaps de infraestructura descubiertos en la Fase 1 que, sin arreglarse, harían fallar la feature en producción aunque el código Go esté perfecto (imagen sin compiladores, container sin memoria/CPU para compilar, balanceador cortando la conexión a los 30s).

**Para qué**: sin esto, todo lo diseñado en las Fases 2-4 y 6 funcionaría en local pero fallaría (o se degradaría silenciosamente) en el cluster real de GKE — que es justamente lo que estamos tratando de validar.

**Decisión ya tomada con el usuario**: separar en dos imágenes (no bloatear la única imagen actual), y soportar los 3 lenguajes (cpp20/java17/python310) desde el día uno en la Fase 6.

**Estado**: pendiente — puede ejecutarse en paralelo a la Fase 4 (es infra pura, no depende del código Go de esa fase).

**Orden de ejecución (decidido 2026-08-12)**: se hace la Fase 6 primero. `NativeCompiler` corre sin Docker, directo en el filesystem del pod worker — se puede escribir y probar en local sin que esta fase (Fase 5) esté terminada, siempre que la máquina de desarrollo tenga `g++`/`javac`/`python3` instalados. Esta fase solo es un bloqueante real para el *despliegue* en GKE. Con eso, conviene juntar las Fases 5 y 6 y hacer una sola verificación real contra el cluster (Fase 8) al final de ambas, en vez de subir y bajar el cluster dos veces.

---

## Fase 6 — Arreglar la compilación/ejecución nativa de checker y validator ✅ COMPLETA (2026-08-13)

**Qué**: un compilador nativo (`NativeCompiler`, `exec.Command`, sin Docker — el checker es código confiable del problem setter) para checker/validator en los 3 lenguajes, que sube el artefacto compilado a storage; arreglar `output_comparator.go` para invocar ese artefacto según el lenguaje (`java -cp`, `python3 script.py`, o el binario directo); un `ValidatorRunner` que corre el validator contra cada test case por stdin (convención estándar de validators tipo testlib). Con esto, `ValidateProblemUseCase` queda completo (pasos 3-7 del spec).

**Por qué**: es el arreglo del bug encontrado en la Fase 1 — hoy la ejecución de checkers personalizados está completamente rota (asume un binario que nunca se compila, en una imagen sin compiladores). Este arreglo sirve para DOS cosas a la vez: completar el pipeline de publish, Y arreglar el judging real de cualquier submission futura contra un problema con checker personalizado (que hoy fallaría siempre).

**Para qué**: sin esto, publish nunca podría aprobar un problema con checker/validator personalizado, y aunque lo aprobara "a la fuerza", el judging real de submissions contra ese problema seguiría roto.

**Nota sobre Python (no es un lenguaje compilado)**: para python310, "compilar" significa correr una verificación de sintaxis (`python3 -m py_compile checker.py`), no generar un binario. El "artefacto" que queda en `compiledKey` para Python es el propio código fuente, guardado recién después de pasar esa verificación — así quien ejecuta el checker más adelante no necesita tratar a Python distinto de cpp/java, siempre hay "algo listo para ejecutar" en `compiledKey` sin importar el lenguaje.

**Progreso — todo construido, pieza por pieza, con revisión en el camino (2026-08-12/13)**:
- `application/judge/native_compiler.go` (port) + `adapter/judge/native_compiler.go` (despachador por mapa) + `native_compiler_cpp.go`/`_java.go`/`_python.go` — compila sin Docker, directo en el filesystem del worker (el checker es código de confianza del problem setter). Java aprovecha que el nombre de archivo debe coincidir con el de la clase pública — no hace falta rastrear el nombre de clase en ningún lado nuevo. Python "compila" con un chequeo de sintaxis y el artefacto es la fuente misma.
- `application/judge/judging_source_provider.go` (port) + su adapter — lee el checker/validator **fuente** (fileKey+lenguaje) de un problema, si existe.
- `application/problem/judging_artifact_writer.go` (port) + su adapter — guarda el `compiledKey`/`compiledAt` en la columna JSONB del checker/validator, sin cargar el agregado rico `Problem`.
- `application/judge/artifact_uploader.go` (port) + `gcs_writer.go`/`local_writer.go` — sube el artefacto compilado a storage (mismo patrón intercambiable que `gcsReader`/`localReader`; se confirmó que `internal/adapter/storage/` es un placeholder vacío, no hay nada ahí para reusar — cada dominio tiene su propio adapter de storage, patrón ya establecido).
- `application/judge/validator_runner.go` — corre el validator compilado contra un input por stdin (convención testlib, exit 0 = aceptado).
- `application/judge/prepare_judging.go` (`PrepareJudgingUseCase`) — la orquestación: compila checker/validator si existen, sube el artefacto, corre el validator contra cada input de los test cases, fail-fast en el primer problema. Usa el `ProblemID` (no el slug) para todo excepto el *path* de storage del artefacto compilado — el slug viaja desde `PublishProblemUseCase` a través de `ValidationQueueMessage` → RabbitMQ → `cmd/worker/main.go` → `ValidateProblemInput`, solo para eso.
- `application/problem/judging_preparer.go` (port) + su traductor en `adapter/problem/` — mismo patrón que `SolutionValidator`, conecta `ValidateProblemUseCase` con `PrepareJudgingUseCase` sin que los paquetes de aplicación se importen entre sí.
- `ValidateProblemUseCase` — el pipeline completo: prepara checker/validator → persiste sus claves compiladas → valida soluciones (ahora con el checker real, no siempre token-compare) → publica.
- **Arreglo importante encontrado en el camino**: `ProblemProvider.GetLimits` (usado por `JudgeSubmissionUseCase`, submissions reales) leía el `fileKey` del checker — la fuente, no el compilado. O sea que el bug de la Fase 1 afectaba también a submissions reales contra problemas con checker personalizado, no solo a publish. Se corrigió para leer `compiledKey` + lenguaje + filename.
- **Refactor encontrado en revisión**: `output_comparator.go` iba a reimplementar por tercera vez la lógica de "cómo invocar un artefacto compilado según su lenguaje" (ya estaba en `ValidatorRunner`). Se extrajo a `adapter/judge/artifact_invocation.go`, una sola función compartida — esto permitió borrar `validator_runner_cpp.go`/`_java.go`/`_python.go` por completo, dejando `ValidatorRunner.Run` en un solo archivo sin despachador.
- Truncado a 2000 bytes (ya usado en `ValidateSolutionsUseCase`) también aplicado a `Log`/`Reason` en `PrepareJudgingFailure`.
- Se corrigió una inconsistencia en el spec (`specs/Problem management/Change problem visibility/spec.md`): el ejemplo de runtime error usaba `status`/`details` en vez de `verdict`.
- Tests: alrededor de 60 nuevos entre `judge` y `problem` (compiladores reales corridos vía Docker para C++ y Python — Java se saltea por falta de JDK en el entorno de este equipo, pero queda listo para cuando corra en la imagen `worker-final` de la Fase 5).

**Hallazgo posterior (2026-08-13), disparado por una pregunta sobre la Fase 7 ("¿el guard de edición puede bloquear para siempre?")**: la respuesta llevó a revisar qué pasa si un checker/validator se cuelga de verdad, y aparecieron dos bugs reales:
1. **Ni `NativeCompiler.Compile` ni `ValidatorRunner.Run` tenían timeout propio** — el `ctx` que reciben viene del worker (`cmd/worker/main.go`), que solo se cancela por `SIGTERM`/`SIGINT`, nunca por tiempo. Un checker/validator con un bug (loop infinito, esperar stdin que nunca llega) colgaba ese goroutine del worker para siempre — no eran solo 20 minutos de guard bloqueando ediciones, era un cupo de concurrencia perdido permanentemente. Se agregó `trustedSubprocessTimeout` (30s, `adapter/judge/judging_timeouts.go`), reusando el mismo criterio que ya existía como `checkerTimeout` en `output_comparator.go` (ahora unificado ahí). El timeout es un campo del struct (no una constante de paquete), mismo patrón que `AwaitProblemValidationUseCase`, para poder acortarlo en tests.
2. **Al agregar el timeout, un segundo bug (preexistente) salió a la luz**: `exec.CommandContext` mata el proceso por señal cuando el contexto se cumple, y eso hace que `cmd.Run()` devuelva un `*exec.ExitError` común — indistinguible de un proceso que terminó solo con código de error. Los tres archivos (`native_compiler_*.go`, `validator_runner.go`, y `output_comparator.go`, este último ya con el bug desde antes de esta sesión) trataban CUALQUIER `*exec.ExitError` como veredicto legítimo — un proceso matado por timeout se reportaba como "el checker rechazó esto" en vez de "falló la infraestructura". Se agregó `isTimeoutErr(ctx, err)` (mismo archivo, `judging_timeouts.go`) que todos los call sites consultan antes de interpretar un `*exec.ExitError`. Confirmado con un test que cuelga un validator Python a propósito (`time.sleep(60)`) contra un timeout acortado — falló primero (probando que el bug era real), pasó después del arreglo.

**Estado**: ✅ completa.

---

## Fase 7 — Cerrar la carrera entre editar un problema y una validación en curso ✅ COMPLETA (2026-08-13)

**Qué**: bloquear `UpdateProblem`/`UploadProblemFiles` con 409 si hay una validación PENDING/RUNNING para ese problema.

**Por qué**: sin esto, alguien podría re-subir el checker de un problema mientras el worker todavía está compilando/corriendo la versión anterior — corrompiendo silenciosamente el resultado de esa validación.

**Para qué**: cierra un hueco real de integridad de datos que la Fase 2 ya deja fácil de detectar (reusa la misma consulta "¿hay una validación activa para este problema?").

**Progreso (2026-08-13)**: se amplió el alcance a `DeleteProblemFileUseCase` también — borrar un archivo (ej. el checker) mientras el worker lo compila es el mismo riesgo que resubirlo. La lógica compartida vive en `application/problem/validation_guard.go` (`ensureNoActiveValidation`, mismo criterio que `permissions.go`: lógica de negocio idéntica reusada por varios casos de uso, no un helper genérico parametrizado). `cmd/api/main.go` reordenado: `problemValidationRepo` se construye antes de los tres casos de uso para poder inyectarlo. 8 tests nuevos (5 de `ensureNoActiveValidation` en aislado, 3 de integración confirmando que cada caso de uso quedó bien conectado).

**Estado**: ✅ completa.

---

## Fase 8 — Verificación end-to-end contra el cluster real ⏳ PENDIENTE

**Qué**: checklist manual contra GKE — publicar un problema incompleto (400 rápido), uno completo sin checker (ver la fila pasar PENDING→RUNNING→PASSED mientras el HTTP request está bloqueado, confirmar que KEDA escala el worker de 0→1), uno con checker que no compila, uno con validator que rechaza un input, y una submission real contra un problema publicado con checker personalizado (la prueba de que el bug de la Fase 1 quedó realmente resuelto). Medir tiempos contra el timeout de 600s. Matar el worker a mitad de una validación y confirmar que el cliente recibe un timeout claro (no un cuelgue infinito) y que un reintento funciona limpio.

**Por qué**: todo lo anterior son tests unitarios y diseño — esta es la prueba real de que el judge system funciona de punta a punta en el entorno real, que es el objetivo original de toda esta sesión.

**Para qué**: es la confirmación de que se puede pasar a la Fase 9 con confianza.

**Estado**: pendiente.

---

## Fase 9 — Adaptar el paquete BOCA y montar la competencia real ⏳ PENDIENTE (fase final)

**Qué**: convertir los 12 problemas de `D:\Programming\Tesis\packages_contest` (formato BOCA) al formato que `POST /problems/import` ya entiende (ICPC/Kattis: `problem.yaml`, `data/sample/`+`data/secret/`, statement en texto/LaTeX). Esto requiere, por cada zip:
- Extraer título de `description/problem.info` (`fullname=`).
- Convertir el statement (hoy un PDF en `description/`) a texto/LaTeX — probablemente extracción de texto del PDF más ajuste manual, ya que el backend no acepta PDFs como statement.
- Los límites de tiempo/memoria en BOCA vienen de un SCRIPT (`limits/cpp`) que hay que ejecutar (o leer) para sacar los valores, no de un archivo estático.
- Los test cases (`input/`/`output/`) no distinguen sample/secret — hay que decidir un criterio (p. ej., los primeros N casos como sample, o los que aparezcan mencionados en el PDF como ejemplo) y separarlos en las carpetas que espera el importador.
- El "checker" de BOCA es un `diff` genérico — mapea directo a "sin checker personalizado" (comparación exacta) en el modelo del backend, no hace falta portar nada de `compare/*`.
- Las soluciones (si existen en el paquete BOCA — no confirmado todavía, no se vieron en la exploración inicial) se ubican y suben como `solution` para que el publish pueda validarlas.

Con los 12 problemas convertidos e importados (`DRAFT`), publicarlos uno por uno (validando que el pipeline de las Fases 2-6 los aprueba), crear un contest, agregarlos, y hacer submissions reales (aciertos y fallos a propósito) para confirmar veredictos correctos.

**Por qué**: es el objetivo final explícito de todo este trabajo — no alcanza con que el judge funcione en abstracto, hay que demostrarlo con una competencia real.

**Para qué**: cierre del ciclo completo API → RabbitMQ → judge-worker → veredicto, con datos reales, no sintéticos.

**Nota**: el diseño detallado de esta fase (¿escribir un script de conversión en Go/Python? ¿hacerlo a mano para 12 problemas? ¿cómo se decide qué casos son "sample"?) se deja para cuando lleguemos a esta fase — según lo pedido, primero se arregla el judge con las Fases 2-8.

**Estado**: pendiente — última fase.

---

## Apéndice — corrección pendiente de documentación

Al cerrar este roadmap, actualizar `CLAUDE.md`:
- La sección "Endpoints pendientes principales" y la lista de pendientes del Judge System están desactualizadas (ver Fase 1) — deben reflejar el estado real post-Fase 8/9.
- La sección "Dev env vars" documenta `MOCK_AUTH=1` ("reads user from `X-Mock-User` header") — **no existe en el código** (grep completo sobre `MOCK_AUTH`/`X-Mock-User`: cero resultados). Detectado al intentar probar el endpoint de publish localmente (Fase 3). Quitar esa línea, o implementarla de verdad si en algún momento hace falta para testing local — mientras tanto, probar localmente requiere login real (`POST /auth/login`) o el usuario admin de `cmd/seed`.

---

## Apéndice — deuda técnica encontrada en el camino (no es parte de esta feature, pero hay que corregirla)

- **`internal/domain/submission/status.go` no sigue `ARCHITECTURE.md` D4 al pie de la letra**: las factorías de estado conocido (`newStatusPending()`, `newStatusRunning()`, etc.) son privadas (minúscula), pero D4 documenta explícitamente que deben ser `New<Type><State>()` — exportadas — y el propio `internal/domain/problem/status.go` (`NewStatusDraft()`, `NewStatusPublished()`) sí lo hace bien. Detectado al construir `problem_validation_status.go` (Fase 2) y notarlo al comparar contra la convención documentada. Corregir en algún momento: renombrar las 9 factorías privadas de `submission/status.go` a exportadas, sin cambiar su comportamiento.
- **`internal/domain/submission.NewSubmission` no valida `problemID`/`userID` vacíos**, solo `id`. Detectado al escribir `NewProblemValidation` (Fase 2) y decidir, a diferencia de `Submission`, sí validar `problemID` — un argumento requerido vacío es "bug del programador" según `ARCHITECTURE.md` D4, sin importar de qué campo se trate. Corregir en algún momento: agregar la misma validación a `NewSubmission` (y revisar si otros constructores del dominio tienen el mismo hueco).

---

## Apéndice — mecanismo de corrección en vivo (checker/validator/test cases/solución durante una competencia activa) — diseñado, PR separado

**No es parte de este PR** — este roadmap ya quedó grande con las Fases 2-7. Queda documentado acá, diseñado, para ejecutarlo en un PR aparte cuando corresponda.

**Motivación**: hoy, si en medio de una competencia se descubre que un checker, un validator, o los test cases de un problema ya `PUBLISHED` están mal, no hay ningún mecanismo para corregirlo. Todo camino de edición (`UpdateProblemUseCase`, `UploadProblemFilesUseCase`, `DeleteProblemFileUseCase`) exige que el problema esté `DRAFT`, y `UnpublishProblemUseCase` bloquea con 409 (`ErrCodeProblemInActiveContest`) si el problema está en un contest activo. No hay salida.

**Diseño descartado**: permitir un "unpublish de emergencia" (Admin y/o el Coach dueño del contest, saltándose el bloqueo de contest activo) para poder editar y volver a publicar. Se descartó porque implica downtime real — mientras el problema está `DRAFT`, las submissions a ese problema se rechazan con 400 `ErrCodeProblemNotPublished` (`submit_contest_solution.go:129-132`).

**Diseño elegido**: un caso de uso **independiente**, que no toca el status del problema:
1. Recibe checker, validator, test cases y/o solución de referencia nuevos para un problema que sigue `PUBLISHED`.
2. Corre exactamente el mismo pipeline de validación que un publish normal (compilar checker/validator, correr el validator contra los inputs, correr la solución contra los test cases).
3. Si todo pasa: promueve los archivos nuevos a las keys en vivo, y toca `judgingUpdatedAt` — el campo que **ya existe** en el dominio (`domain/problem/problem.go:188-247`, tocado por `SetChecker`/`SetValidator`/`SetTestCases`/`AddSolution`) y que los 3 casos de uso de rejudge que **ya existen** (`RejudgeSubmissionsUseCase`, `RejudgeContestSubmissionsUseCase`, `AdminRejudgeSubmissionsUseCase`) ya usan para filtrar "submissions anteriores a este cambio". Es decir: el "stop" que hace falta ya está construido, este caso de uso solo necesita tocarlo.
4. Dispara automáticamente un rejudge de las submissions de los **contests activos** que usan ese problema (no las de práctica — decisión explícita, para no rejuzgar de más sin que nadie lo haya pedido). Cualquier submission juzgada DESPUÉS de la promoción ya usa los datos nuevos automáticamente — confirmado por rastreo completo: `ProblemProvider.GetLimits` lee la BD en vivo en cada llamada, y `OutputComparator.customCheckerCompare` descarga el checker desde storage en cada ejecución — no hay cache en ningún nivel del pipeline de judging real.
5. Si algo falla en el camino, no se toca nada en vivo — el problema sigue sirviendo con los datos viejos, sin downtime en ningún momento.

**Bug real encontrado que hay que arreglar primero (prerequisito, Pieza 0)**: `PrepareJudgingUseCase.prepareChecker`/`prepareValidator` (`application/judge/prepare_judging.go:132-167`) compilan el checker y **ya escriben la key en vivo** (`problems/{slug}/checker/compiled`) — y `ValidateProblemUseCase` persiste ese `compiledKey` en la tabla `problems` — **antes** de saber si las soluciones lo van a aprobar. Hoy es inofensivo porque un publish normal corre con el problema todavía `DRAFT` (sin submissions reales posibles). Pero si este mismo pipeline se reutiliza tal cual para el mecanismo de arriba —que corre con el problema **ya `PUBLISHED`, en competencia real**—, en cuanto el checker candidato compila ya queda pisando el que usan submissions reales en ese instante, sin ninguna forma de revertir si la validación termina fallando después (por ejemplo, si las soluciones ya no pasan con el checker nuevo).

**Arreglo, necesario independientemente de este mecanismo nuevo**: compilar a una key candidata (no la key fija en vivo), correr TODO el pipeline de validación contra esa candidata — incluyendo `ValidateSolutionsUseCase`, que hoy obtiene el `CheckerPath` exclusivamente vía `ProblemProvider.GetLimits` (lectura en vivo de la BD) y necesitaría un override explícito para apuntar a la key candidata en vez de a la que está en la BD — y solo si absolutamente todo pasa, "promover": escribir la key en vivo + persistir `compiledKey` en la BD, en un único paso final. Ya hay precedente de este patrón en el propio código: `UploadProblemFilesUseCase.handleTestCases` sube cada test case nuevo a una ruta con un UUID único y solo al final, si todo salió bien, actualiza el puntero del problema (`p.SetTestCases(basePath, now)`) — el checker/validator deberían seguir la misma idea. Este arreglo beneficia también al publish normal (hoy, si un publish falla a mitad de camino, ya deja el `compiledKey` de la BD apuntando a un binario a medio confirmar — no es un bug visible porque el problema sigue en `DRAFT`, pero es la misma causa raíz).

**Decisiones ya tomadas con el usuario**:
- Alcance del rejudge automático: solo submissions de contests **activos** en este momento, no las de práctica.
- Alcance de datos que este mecanismo puede corregir: checker, validator, test cases, **y** la solución de referencia.
- Permisos: propuesto usar la misma regla que cualquier otra edición (`CanBeEditedBy` — autor, admin, o modifiers), en vez de restringirlo a admin/lead del contest, ya que a diferencia del diseño descartado este mecanismo nunca genera downtime — **pendiente de confirmación explícita del usuario**, no se cerró del todo antes de decidir posponer esto a otro PR.

**Secuencia de piezas para cuando se ejecute**:
1. Pieza 0 — refactor "candidato + promote" en `PrepareJudgingUseCase`/`ValidateProblemUseCase`/`ValidateSolutionsUseCase` (prerequisito compartido, beneficia también al publish normal).
2. Pieza 1 — el caso de uso nuevo (nombre sin decidir todavía).
3. Pieza 2 — endpoint HTTP + wiring.
4. Pieza 3 — actualizar `specs/Problem management/Change problem visibility/spec.md` con la forma final.

**Estado**: diseñado, no implementado. Ejecutar en un PR separado del de este roadmap.

---

## Apéndice A — detalle técnico (se completa fase por fase, al momento de ejecutar cada una)

*(vacío por ahora — se llena cuando arranquemos la Fase 2, con los tipos Go exactos, SQL de migraciones, y archivos nuevos/modificados de cada fase, para no inflar el documento con detalle que puede cambiar mientras se itera sobre el roadmap)*
