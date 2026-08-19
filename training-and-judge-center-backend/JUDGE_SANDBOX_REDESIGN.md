# Rediseño: dónde se ejecutan checkers y validators

**Estado**: diseño cerrado, en ejecución. Ver el plan de ejecución más abajo para el avance paso a paso.

**Qué motiva esto**: la Fase 6 del `JUDGE_VALIDATION_ROADMAP.md` implementó la compilación y ejecución de checkers/validators de forma **nativa** dentro del container del worker (`exec.Command`, sin sandbox). Al preparar la Fase 5 (infraestructura) aparecieron dos problemas serios con esa decisión, y el rediseño que resuelve los dos es el mismo.

---

## Los dos problemas

### Problema 1 — el worker no tiene recursos para lo que le pedimos

La concurrencia del worker se deriva del presupuesto del **sidecar DinD**, no del suyo (`cmd/worker/main.go`):

```go
podCPUMillis := getRequiredEnvInt64("POD_CPU_LIMIT")   // = limits.cpu del DinD = 3000
maxConcurrent := int(podCPUMillis/1000) - judgeCfg.Judge.CPUOverheadCores  // 3 - 1 = 2
```

O sea: **2 judgings en paralelo**, pero las compilaciones y ejecuciones nativas que esos 2 disparan salen del container `worker`, que tiene `limits: cpu 500m / memory 512Mi`. Dos `g++` compilando `testlib.h` a la vez dentro de 512Mi no entran.

La prueba de que ese número quedó viejo y no de que se eligió mal está en el propio `config/judge_config.yaml`:

```yaml
memoryOverheadBytes: 536870912  # 512 MiB reservado para el proceso worker
```

Esos 512Mi se dimensionaron cuando el worker **solo orquestaba**. La compilación nativa llegó en la Fase 6 y nadie volvió sobre ese número.

**Modo de falla**: si el container se pasa de memoria, el kernel mata el **proceso worker entero**, no la compilación. Se caen los judgings en vuelo y la conexión con RabbitMQ; la recuperación queda a cargo de los barridos de stale (10-20 min). Por CPU, el proceso Go hambreado puede perder heartbeats de AMQP y que le corten la conexión.

**Restricción del nodo**: `judge-pool` es `e2-standard-4` (4 CPU / 16GiB; quedan ~3.6 CPU y ~12.2GiB para el pod después de lo que reserva GKE y los DaemonSets). Hoy los límites suman 3.5 CPU / 10.5GiB. Subir el worker a `2 CPU / 2Gi` daría 5 CPU / 12GiB — la CPU sobresuscripta no es problema (es compresible), pero la memoria quedaría al filo.

### Problema 2 — el checker corre con los privilegios del worker

El código del checker/validator lo sube el problem setter. Hoy se compila y se ejecuta nativamente en el container del worker, que tiene:

- `DB_PASSWORD`, `RABBITMQ_PASSWORD` y `RABBITMQ_URL` (con la contraseña adentro) como variables de entorno
- `DOCKER_HOST` apuntando al demonio Docker privilegiado
- Workload Identity (`GKE_METADATA`) → acceso al metadata server → tokens de GCP del service account `backend`

**Verificado**: ningún subproceso nativo limpia su entorno. `exec.Command` hereda el del worker por defecto (no hay una sola aparición de `cmd.Env` en `internal/adapter/judge/`).

**Exfiltración de una línea, hoy**: subir un "validator" que imprima `environ` a stderr. El código captura ese stderr y lo devuelve como `Reason` en el reporte de validación, que sale por la respuesta HTTP del publish. Credenciales de Postgres y RabbitMQ en texto plano.

Y dos matices sobre "aseguramos la primera compilación y listo":

- **El artefacto compilado se ejecuta para siempre**, una vez por caso de prueba de cada submission real durante toda la vida del problema. Un checker que se porta bien en el publish y ataca en la submission 500 pasa cualquier control de la primera vez.
- **Compilar ya es ejecutar**: `javac` corre *annotation processors* (código arbitrario en tiempo de compilación), y `g++` con un `#include` a un archivo del sistema vuelca su contenido en el mensaje de error, que devolvemos al usuario.

---

## La propuesta

Sacar checkers y validators del container del worker y meterlos en el sandbox, con **dos pools separados** en vez de uno.

**Por qué cierra el problema 2 por completo** (no lo mitiga): los containers del pool corren con `NetworkMode: "none"` (`internal/adapter/judge/pool/pool.go`). Sin red no hay metadata server, así que ni siquiera la vía del token de GCP —que limpiar el entorno no tapaba— queda abierta. Sumado a que el container no hereda las variables del worker ni ve el socket de Docker, el checker pasa de "código confiable con los privilegios del worker" a "código sin privilegios en una caja sin salida". La diferencia es entre *asumir* confianza y *no necesitarla*.

**Por qué cierra el problema 1**: el worker vuelve a ser lo que su config ya decía que era — solo orquestación. Ese `memoryOverheadBytes: 512Mi # reservado para el proceso worker` deja de estar desactualizado.

---

## Cómo funciona hoy el pool (verificado en código)

Necesario para entender las decisiones de abajo:

- `Executor.BeginSession(ctx, lang)` hace `pool.Claim(lang)` y obtiene **un** container; la `Session` queda atada a él.
- La compilación y **todos** los `RunTestCase` de ese judging van a ese mismo container vía `docker exec`.
- `Session.Close` hace `rm -rf /sandbox/*` y luego `pool.Release(container)`. El container **no se destruye**: vuelve al pool y lo reusa la siguiente sesión, sin arrastrar estado. Solo el reaper de inactividad lo elimina. Eso evita pagar ~200-500ms de creación por judging.
- **Para el pool, compilar y ejecutar son lo mismo**: `Claim(language)` devuelve un container corriendo `sleep infinity`; qué se hace adentro con `docker exec` es asunto del llamador. El pool no sabe qué es una solución ni qué es un checker — solo entiende de lenguajes y de tamaños (`LanguageConfig{Image, CPU, MemoryBytes}`).

**Consecuencia de diseño**: la división real no es "soluciones vs checkers", es **"pesado vs liviano"**. Cada pedido elige pool según los recursos que necesita, no según qué clase de código es.

Perfiles de uso:

| Operación | Frecuencia | Peso |
|---|---|---|
| Compilar checker/validator | 1 vez por publish | pesado (testlib son ~10k líneas de C++ con templates) |
| Compilar solución | 1 vez por judging | moderado |
| Correr solución | 1 vez por caso de prueba | pesado (hasta los límites del problema) |
| Correr validator | 1 vez por caso de prueba | liviano |
| Correr checker | 1 vez por caso × solución, **en cada submission, para siempre** | liviano |

---

## Decisiones tomadas

### D1 — El artefacto compilado se sigue guardando en GCS ✅

Se mantiene el modelo de datos de la Fase 6 (`compiledKey`/`compiledAt`, `ArtifactUploader`, `JudgingArtifactWriter`). Solo cambia **dónde** ocurre la compilación.

- Publish: compilar en una sesión del sandbox → extraer el binario → subirlo a GCS → guardar `compiledKey`.
- Judging: bajar el binario de GCS → inyectarlo en el container → correrlo por cada caso.

**Alternativa descartada**: no guardar nada y recompilar el checker al abrir cada sesión de judging. Se descartó por el **modo de falla**: metería 5-10s de compilación en el camino caliente de cada submission, y peor, haría que la submission de un concursante pueda fallar por un problema del *checker del problema*. Con D1, si el checker no compila se entera el problem setter en el publish, que es donde corresponde. Además da determinismo dentro de la competencia: todos son juzgados por el mismo binario congelado.

**Argumento honesto en contra que se aceptó**: un binario guardado podría quedar inservible si la imagen del runner cambia de base. Es manejable porque las imágenes están versionadas de forma explícita en `deploy/k8s/judge/images-configmap.yaml`, así que ese cambio es deliberado y se acompaña de republicar.

**Costo real, menor al estimado**: las dos primitivas ya existen en `session.go` — `CopyToContainer` + `buildTar` (ya se usan para meter el fuente y el input) y `CopyFromContainer` + `extractFirstFile` (ya se usan para sacar `/sandbox/output.txt`). Extraer el binario compilado es un método que compone helpers existentes, no infraestructura nueva.

### D2 — El pool de checkers se indexa por lenguaje ✅

Cae directo de D1. La razón para indexar por `(problema, lenguaje)` era evitar que un container caliente tuviera compilado el checker del problema A y lo reusara para el B — pero con D1 el checker viene precompilado y se inyecta al abrir la sesión, así que el container no guarda estado específico del problema. No hay que tocar la estructura del `Pool`.

### D3 — Los puertos de pool B pasan a forma de sesión, como interfaces separadas ✅

Hoy `OutputChecker.Check` es sin estado y se llama **dentro del loop de casos de prueba**, en los dos caminos (`judge_submission.go` y `validate_solutions.go`). Y `customCheckerCompare` hace `readObject()` contra GCS en cada llamada: con 25 casos son **25 descargas del mismo binario por submission**. `ValidatorRunner.Run` tiene el mismo patrón por caso.

Los dos pasan a forma de sesión —abrir una vez, ejecutar N veces, cerrar— para amortizar el container y la descarga:

```go
// OutputChecker — usado por ValidateSolutionsUseCase y JudgeSubmissionUseCase
type OutputChecker interface {
	BeginChecking(ctx, checkerPath string, lang submission.Language, filename string) (CheckerSession, error)
}
type CheckerSession interface {
	Check(ctx, req CheckRequest) (CheckResult, error)   // por caso de prueba
	Close(ctx) error
}

// ValidatorRunner — usado por PrepareJudgingUseCase
type ValidatorRunner interface {
	BeginValidating(ctx, validatorPath string, lang submission.Language, filename string) (ValidatorSession, error)
}
type ValidatorSession interface {
	Validate(ctx, input []byte) (ValidatorRunResult, error)   // por caso de prueba
	Close(ctx) error
}
```

**Son interfaces separadas, no una compartida ni una extensión de `ExecutionSession`.** Pool A y pool B hacen cosas distintas (compilar y correr soluciones vs. ejecutar un artefacto ya compilado con argumentos de archivo), y meterle a la sesión de soluciones métodos que solo sirven para checkers rompería los puertos angostos que el proyecto ya usa: `JudgeSubmissionUseCase` cargaría con métodos que no llama y cada mock tendría que implementarlos. La reutilización ocurre en el **adapter** (helpers privados `buildTar`, `extractFirstFile`, el patrón `ExecCreate`/`ExecAttach`/`ExecInspect`), no en el puerto.

**Se descartó una tercera vía**: una sesión genérica de bajo nivel (`PutFiles` / `Exec(cmd)` / `GetFile`) sobre la que ambos casos de uso construyan. Empujaría la mecánica de containers hacia arriba — la capa de aplicación armando comandos de shell. Hoy el saber "cómo se compila en C++" vive en el adapter y en `judge_config.yaml`, que es donde corresponde.

**Los dos reciben una ruta de GCS, no bytes.** El primer borrador daba bytes al validator razonando que `PrepareJudgingUseCase` acababa de compilarlo y lo tenía en memoria. Es cierto pero irrelevante: el artefacto del validator **también queda guardado** (`problems/{slug}/validator/compiled`, con su clave persistida vía `SetValidatorCompiledKey`). Recibir una ruta habilita gratis un caso de uso futuro —cambiar los casos de prueba y re-validarlos contra el validator ya compilado, sin recompilar— y de paso **verifica el round-trip**: si la subida se corrompió o la clave quedó mal, se detecta en el publish y no meses después en medio de una competencia.

**Toca la capa de aplicación**: `ValidateSolutionsUseCase`, `JudgeSubmissionUseCase` y `PrepareJudgingUseCase` abren la sesión antes del loop.

El puerto `NativeCompiler` conserva su firma tal cual; solo cambia el adapter (de `exec.Command` al pool A).

### D4 — Dos pools: A (grande) compila y corre soluciones, B (chico) solo ejecuta checkers ✅

- **Pool A** — containers grandes. Compila **todo** (checkers, validators y soluciones) y ejecuta las soluciones.
- **Pool B** — containers chicos. **Solo ejecuta** checkers y validators ya compilados. Nunca compila.

La clave que hace que esto funcione es D1: como el artefacto queda guardado en GCS al publicar, en el judging de una submission real el checker ya viene compilado — a B solo se le inyecta el binario y se lo ejecuta. Por eso B puede dimensionarse para **ejecución pura**, que era exactamente el beneficio que motivaba separarlo. La objeción de "hay que sobredimensionar B para que aguante compilar" desaparece.

| Operación | Pool | Cuándo |
|---|---|---|
| Compilar checker/validator | **A** | solo al publicar |
| Compilar solución | **A** | publish y cada judging |
| Correr solución | **A** | publish y cada judging |
| Correr validator | **B** | solo al publicar |
| Correr checker | **B** | publish y **cada submission, siempre** |

**Los dos pools son necesarios, no solo prolijos — evitan un deadlock.** Con la forma de sesión de D3, un judging tiene **dos containers tomados a la vez**: el de la solución (abierto durante todo el loop de casos) y el de checker (también abierto durante todo el loop, para no re-descargar el binario en cada caso). Si los dos salieran del mismo pool, con capacidad C y `maxConcurrent = C`, los C judgings tomarían su container de solución y después todos pedirían uno de checker; no queda ninguno y nadie puede avanzar ni liberar. Con pools separados, cada judging pide uno de cada uno y nunca compiten entre sí.

**Corrección sobre "compilar en el mismo container que corre la solución"**: no siempre se puede, porque **cada imagen de lenguaje tiene un solo toolchain** (`cpp20.Dockerfile` instala únicamente `g++`, `java17` únicamente el JDK, `python310` únicamente `python3`). Si el checker es C++ y la solución es Java, el container de A de la solución no tiene `g++`. Así que la compilación del checker es su propio `Claim` de pool A, con el lenguaje **del checker**. Con D1 eso ocurre una sola vez por publish, así que el costo es despreciable.

**Flujo resultante:**

*Al publicar*
1. `Claim` en A (lenguaje del checker) → compilar → extraer binario → subir a GCS → `Release`
2. Ídem para el validator
3. `Claim` en B (lenguaje del validator) → inyectar → correr contra cada input → `Release`
4. Por cada solución: `Claim` en A (lenguaje de la solución) → compilar y correr; y `Claim` en B para chequear cada salida

*Al juzgar una submission*
1. `Claim` en A (lenguaje de la submission) → compilar y correr
2. `Claim` en B (lenguaje del checker) → inyectar el binario precompilado → chequear cada salida

---

### D5 — Dimensionamiento: misma CPU en ambos pools, distinta memoria, fórmula sin cambios ✅

| | CPU | Memoria |
|---|---|---|
| Pool A | `cpu: "1"` | 2 GiB |
| Pool B | `cpu: "1"` | 512 MiB de base, más para Java (la JVM) |

Los tamaños son **por lenguaje dentro de cada pool** — ver la estructura de config en D10 y los números concretos en D13.

#### Qué significa cada valor (hacen cosas distintas)

**`MemoryBytes` cumple doble función:**
1. **Techo real del container** (`Resources.Memory`, con `MemorySwap` igual para desactivar swap). Es lo que hace cumplir el kernel: al pasarse, SIGKILL → exit 137 → así se detecta MLE.
2. **Costo contable en el pool** (`allocatedBytes + MemoryBytes <= budget`): cuánto presupuesto gasta un container de ese lenguaje.

Un límite **no es un permiso para usar recursos, es un techo**. Un container capado en 2 GiB que usa 50 MB consume 50 MB de RAM real — pero el pool lo *contabiliza* como 2 GiB, porque debe asumir el peor caso.

**`CPU` (`NanoCPUs`) es solo un techo de tasa** — "como máximo N segundos de CPU por segundo". No reserva nada, y **no participa de la admisión** (`canCreate` solo mira memoria), así que no afecta cuántos containers entran.

#### Por qué la memoria es distinta y la CPU es igual

**Memoria distinta**: en pool A el número es el techo donde una solución recibe MLE, así que tiene que ser al menos tan grande como el mayor `memoryLimit` que un problema pueda declarar — no es un número libre. En pool B, 512 MiB le sobran a un checker que lee tres archivos y compara tokens. Igualarlo a 2 GiB no le daría capacidad útil: gastaría presupuesto y haría entrar menos containers en total, con lo que el LRU evictaría más seguido — justo lo contrario de lo que se busca al mantener containers calientes.

**CPU igual**: el cap de 1 CPU en pool A existe para que el **veredicto sea justo**, no para proteger la infraestructura. Como se mide tiempo de CPU acumulado, una solución multi-hilo sin techo consumiría 4 segundos de CPU por segundo de reloj en un nodo de 4 cores y llegaría al TLE cuatro veces más rápido; con el techo en 1, lanzar hilos no ayuda ni perjudica y los tiempos son comparables entre submissions.

En pool B no hay veredicto que proteger (el tiempo del checker no se compara contra nada), pero el techo **sirve como contención**: un checker con un bucle infinito no puede pasar de un core. Y como los dos containers se alternan, igualarlos no altera la cuenta:

```
max(cpuA, cpuB) = max(1, 1) = 1 CPU por judging
```

Se descartaron dos alternativas: `0.5` (número arbitrario, y frena el arranque de la JVM en checkers Java, donde cada invocación es un `docker exec` nuevo) y *sin límite* (mantiene la cuenta igual de limpia pero pierde la contención).

#### La asimetría memoria/CPU

Dentro de **un mismo judging**, A y B **nunca computan a la vez**:

```
solución corre caso 1    → A ocupado, B ocioso
checker compara salida 1 → B ocupado, A ocioso
```

No es casualidad estadística: es una **dependencia de datos** — la entrada del checker *es* la salida de la solución. De ahí que **la memoria se sume** (ambos containers están tomados todo el judging, aunque uno esté ocioso) pero **la CPU no** (solo uno computa por vez, y un techo no es una reserva).

#### La fórmula del semáforo queda igual

```
maxConcurrent = dindCPU/1000 − dockerDaemonReserveCores     ← sin cambios
```

Calcula "cuántos cores quedan para judging" y los cuenta como judgings, asumiendo **1 CPU por judging**. Con pool B agregado la demanda pico por judging sigue siendo 1 CPU, así que lo que garantiza —una solución corriendo por CPU— se cumple igual.

**Supuesto a documentar en el código**, porque pasa a ser carga estructural y hoy es implícito: la fórmula solo vale mientras los containers de pool A estén capados en `cpu: "1"`. Si alguien pusiera un lenguaje en 2 CPU, la fórmula mentiría sin que nada avise.

#### Reparto de memoria

Los números concretos y el invariante que hay que cumplir están en **D13**. El resultado adelantado: **entra en el `e2-standard-4` que ya existe** — la premisa inicial de que el rediseño obligaba a máquinas más potentes resultó falsa.

#### No hace falta pinning

`NanoCPUs` es una cuota, no una reserva: cuando A está ocioso su capacidad ya queda libre sin configurar nada. `--cpuset-cpus` sería una *restricción* (limita en qué cores puede correr), no una asignación — no crearía capacidad.

El motivo por el que un juez normalmente sí querría pinning —que la interferencia infle los tiempos medidos y produzca TLE falsos— **ya está resuelto por diseño**: se mide tiempo de CPU, no de reloj (`TimeMs: cpuTimeMs`). Una solución que consumió 800ms reporta 800ms aunque haya competido todo el tiempo.

Red de seguridad: los containers del demonio interno viven dentro del cgroup del container `dind`, así que su `limits.cpu` es un techo duro sobre la suma de todo lo de adentro. Pasarse con `maxConcurrent` produce throttling, no rotura.

#### Los dos comentarios de overhead están mal

Ambos arrastran la misma confusión — dan por hecho que el proceso worker sale del presupuesto del DinD, cuando corre en **otro container, con su propio cgroup y sus propios límites**:

```yaml
memoryOverheadBytes: 536870912  # dice "reservado para el proceso worker"
                                # en realidad protege al DEMONIO DOCKER (se resta de limits.memory del dind)
cpuOverheadCores: 1             # dice "worker + daemon Docker"
                                # en realidad reserva solo para el DEMONIO DOCKER
```

Que `memoryOverheadBytes` (512 MiB, recortado de los 10 GiB del dind) coincida en valor con `worker.limits.memory` (512 MiB, techo del container del worker) es casualidad. Conviene además renombrar `cpuOverheadCores` a `dockerDaemonReserveCores`.

### D6 — La comparación por tokens también corre en pool B, con imagen y binario propios ✅

**Nada de comparar en el worker.** El caso sin checker personalizado es la mayoría del tráfico; si se quedara en el worker, el volumen compartido (D7) no serviría para casi nada y el worker seguiría siendo el cuello de botella de CPU que motivó todo esto.

**Imagen propia y mínima.** Una comparación por tokens no tiene lenguaje —no hay checker—, así que no puede salir de una imagen de lenguaje. Se agrega una cuarta imagen, chica, dedicada a este caso. Arranca rápido y ocupa poca memoria, así que se pueden mantener varias calientes, que importa ahora que pool B es camino caliente. Se descartó fijar un lenguaje (p. ej. usar siempre `cpp20`) porque reservaría un container con un toolchain entero para comparar texto.

**Binario propio, no comando de shell.** Replica exactamente el `tokenCompare` actual, así que el comportamiento no cambia respecto de lo que hoy está en producción. Un `tr` + `cmp` no habría que construirlo, pero es fácil que difiera en casos borde (espacios iniciales, líneas vacías, `\r\n`).

**Encaja en lo que ya existe**: `cmd/compare/` en el mismo módulo Go, con su `docker/judge/compare.Dockerfile` construido con el mismo contexto que las otras tres (`build-judge-images.sh` ya usa el directorio del backend como contexto).

**Y el pool no necesita ningún caso especial**: pool B se indexa por lenguaje igual que pool A, y el caso sin checker es una entrada más en ese mapa (`compare` → imagen mínima). `BeginChecking` con ruta vacía reclama la "lengua" `compare`.

**Consecuencia sobre el dimensionamiento**: pool B deja de ser ocasional. Lo usan **todas** las submissions, no solo las de problemas con checker personalizado. Su tamaño importa tanto como el de pool A.

**Pendiente**: agregar la cuarta imagen al init container `prepull-language-images` (hoy recorre solo `cpp20 java17 python310`) y al pipeline de build/push.

---

### D7 — Volumen compartido con directorio por judging: la salida del concursante nunca pasa por el worker ✅

**El problema que resuelve**: hoy `copyOutput` saca la salida del container por la API de Docker y la deja en memoria del worker como `RunResult.Output []byte` — hasta **64 MiB por caso de prueba** (`maxOutputBytes = 64 << 20`). El worker tiene `limits: cpu 500m`, elegidos para un worker que solo coordina. Mover decenas de MiB por caso, decodificar tar y comparar es trabajo que no le corresponde.

**Por qué no alcanzaba con mandar solo la comparación a un container**: los bytes ya están en el worker cuando se compara. Empujarlos *hacia adentro* de un container significa empaquetarlos en tar y transferirlos por la API — **más** CPU del worker, no menos. La única forma real de sacarle la carga es que esos bytes **nunca salgan del sandbox**.

#### El mecanismo

1. **Un `emptyDir` nuevo montado en los dos containers** (`dind` y `worker`) en la misma ruta. Hace falta en ambos porque el demonio Docker corre *dentro* de `dind` y resuelve las rutas origen de los bind mounts en **su** filesystem, no en el del pod; para que el worker vea esos mismos archivos, el directorio tiene que existir en los dos.

2. **Todos los containers montan la misma raíz compartida** — montaje uniforme, sin bind mounts distintos por container.

3. **El worker genera un UUID por judging**, crea `<raíz>/<uuid>/` y les pasa esa ruta a los containers que participan. Pool A escribe ahí su salida; pool B lee de ahí. Cada uno alcanza únicamente el directorio cuya ruta recibió.

4. **La raíz no puede listarse.** Acá se sostiene todo el aislamiento: en un directorio, `r` habilita *listar* y `x` habilita *atravesar*. La raíz queda dueño `root` con modo `0711` — el usuario `judge` (uid 1000, fijo en la imagen base) puede entrar a una ruta que conoce, pero `ls` no le devuelve nada. Cada directorio de UUID sí accesible para ese usuario.

5. **La salida esperada NUNCA va al volumen compartido.** Es lo único verdaderamente secreto: si el código del concursante pudiera leerla, le bastaría imprimirla para pasar todo. La salida esperada y el artefacto del checker viajan del worker al container de pool B directamente.

6. **Limpia el worker**: un `rm -rf <raíz>/<uuid>` al terminar el judging, sobre una ruta que ya conoce.

#### Por qué así y no con un directorio por container

La primera versión montaba un subdirectorio distinto en cada container, buscando aislamiento por namespace de montaje. Se descartó por tres razones:

- **No cerraba el lado de pool B**: como el emparejamiento es dinámico (cualquier container de B con cualquiera de A), B habría tenido que montar el árbol entero de A, dejando a un checker malicioso leer las salidas de todos los judgings. Con el UUID por judging y la raíz no listable, B tampoco puede enumerar: solo alcanza el UUID que le pasaron.
- **El ciclo de vida no coincidía**: los containers se reusan entre judgings, así que un directorio por container arrastraría la salida del judging anterior y habría que limpiarlo aparte. Un directorio por UUID nace y muere con el judging.
- **Era más invasivo**: obligaba al pool a armar un bind mount distinto al crear cada container.

**Tampoco alcanzaba con permisos POSIX del estilo "pool A escribe pero no lee"**: un directorio en modo `-wx` impide listar pero **no** impide abrir un archivo cuya ruta conozcas, y como todos los containers de pool A corren con el mismo uid, los permisos de archivo tampoco los separan entre sí.

#### De qué depende la seguridad

Es el mismo principio que una URL-capacidad: **la ruta es la credencial**. Se sostiene sobre dos cosas, y las dos hay que cuidarlas:

- La raíz no listable (modo `0711`, dueño `root`).
- Los UUID no pueden filtrarse — son 122 bits de entropía, imposibles de adivinar, pero no deben terminar en logs visibles al usuario ni en mensajes de error devueltos por la API.

#### Lo que se gana

- **`copyOutput` desaparece del worker**: de 64 MiB por caso de prueba a través de la API de Docker, a leer ~2 KB del filesystem para el preview de wrong answer (`Actual: truncatePreview(...)`).
- Con esto más D6 (comparación en pool B), el worker vuelve a ser lo que su config ya decía: solo coordinación. Los `500m` dejan de ser un problema.

#### Sub-decisiones pendientes

- Si la **entrada** del caso de prueba también viaja por el volumen (más barato que `CopyToContainer`) o sigue por la API.
- Limpieza: hoy `Session.Close` hace `rm -rf /sandbox/*`; hay que sumar el borrado del directorio del judging.
- `maxOutputBytes` (64 MiB) deja de presionar al worker, pero sigue acotando cuánto disco del `emptyDir` puede consumir un judging. Bajarlo a algo como 8 MiB sigue siendo razonable — para programación competitiva, 64 MiB es patológico.

### D8 — Artefacto normalizado, e invocación por lenguaje en `judge_config.yaml` ✅

**El obstáculo que había que sacar del medio**: hoy la invocación depende del **nombre del archivo subido**, así que no podía ser un comando estático.

```go
case "java17":
    className := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
    return "java", []string{"-cp", dir, className}
```

Java exige que la clase se llame como el archivo, y ese nombre está grabado dentro del `.class`. `runCmd: "java -cp /sandbox solution"` funciona para las soluciones porque siempre se escriben con nombre fijo, pero el checker llega con el nombre que le puso el problem setter. Mover eso a config habría obligado a inventar un mini-lenguaje de plantillas, con errores que solo aparecen en tiempo de ejecución.

**La salida: normalizar el nombre de destino del artefacto.**

| Lenguaje | Artefacto | Comando en config |
|---|---|---|
| cpp20 | binario → `/sandbox/checker` | `/sandbox/checker` |
| java17 | **JAR** → `/sandbox/checker.jar` | `java -jar /sandbox/checker.jar` |
| python310 | fuente → `/sandbox/checker.py` | `python3 /sandbox/checker.py` |

Para Java se empaqueta en JAR con `Main-Class` apuntando a la clase detectada — el nombre se deriva en tiempo de **compilación**, que es donde sí tenemos el filename original. A partir de ahí la invocación es uniforme.

Con eso la config no necesita plantillas: alcanza un **prefijo de comando** por lenguaje, y el adapter le agrega las tres rutas (checker) o le da stdin (validator). La misma forma que ya tiene `runCmd`.

**Dos beneficios laterales:**

- **`CheckerFilename` desaparece de los puertos de ejecución** (solo hace falta al compilar), así que `BeginChecking(ctx, checkerPath, lang)` queda más simple — y el bug de los campos que no llegaban a `judge_submission.go` **deja de existir por construcción** en vez de arreglarse.
- **Los checkers Java multi-clase funcionan.** Hoy se guarda un solo `.class`; un JAR lleva todas.

### D9 — Nombre fijo para el checker: la config queda 100% estática ✅

**Ya hay precedente en el proyecto.** Las soluciones funcionan así desde siempre:

```yaml
java17:
  compileCmd: "javac -encoding UTF-8 /sandbox/solution.java"
  runCmd: "java -cp /sandbox solution"
```

El fuente se escribe siempre como `solution.java` y se ejecuta como `java -cp /sandbox solution` — o sea que un concursante que envía Java **ya está obligado** a que su clase se llame `solution`. Lo raro sería que el checker fuera la excepción.

Imponiendo la misma convención (`checker`), desaparece la única sustitución que quedaba (`{className}`) y con ella lo último de lógica por lenguaje en Go. Los cuatro campos son datos puros en los tres lenguajes:

```yaml
cpp20:
  artifactSource:  "/sandbox/checker.cpp"
  artifactCompile: "g++ -std=c++20 -O2 -o /sandbox/checker /sandbox/checker.cpp"
  artifactPath:    "/sandbox/checker"
  artifactRun:     "/sandbox/checker"

java17:
  artifactSource:  "/sandbox/checker.java"
  artifactCompile: "javac /sandbox/checker.java && jar --create --file /sandbox/checker.jar --main-class checker -C /sandbox ."
  artifactPath:    "/sandbox/checker.jar"
  artifactRun:     "java -jar /sandbox/checker.jar"

python310:
  artifactSource:  "/sandbox/checker.py"
  artifactCompile: "python3 -m py_compile /sandbox/checker.py"
  artifactPath:    "/sandbox/checker.py"        # el artefacto es el propio fuente
  artifactRun:     "python3 /sandbox/checker.py"
```

**Los dos "casos especiales" que parecían obligar a tener lógica en Go no existen:**

- **Python no necesita rama**: su particularidad era "el artefacto es el fuente", que es simplemente `artifactPath` con el mismo valor que `artifactSource`. Nadie tiene que enterarse.
- **Java no necesita dos comandos**: es un solo string de shell con `&&`, y eso ya funciona porque los `exec` se hacen vía `sh -c` (igual que hoy en `RunTestCase`).

**El JAR se mantiene** aunque con nombre fijo `java -cp /sandbox checker` andaría directo: resuelve lo **multi-clase**. Si el checker define clases auxiliares, sin JAR el artefacto serían varios `.class` y no se podría guardar como un archivo. Con JAR es uno solo, y su `--main-class checker` es estático.

**Validación temprana**: hoy, un checker Java con la clase mal nombrada falla recién en el publish con un error críptico de `javac`. Con la convención se valida **al subir el archivo** (para Java, el nombre debe ser `checker.java`) y se devuelve un mensaje claro en el momento.

**También se normaliza el fuente en GCS.** El artefacto compilado ya tiene ruta fija (`problems/{slug}/checker/compiled`), pero el fuente se guarda hoy con el nombre original (`problems/{slug}/checker/{nombreOriginal}`), así que hay que consultar la base para saber cómo se llama. Pasa a `problems/{slug}/checker/source.{ext}` y todo el árbol queda navegable sin mirar la base.

**Detalle resuelto**: la clase va en minúscula (`checker`), consistente con el `solution` que ya se usa, para no tener dos criterios distintos dentro del mismo sistema.

#### Con esto, extraer el artefacto no necesita nada nuevo

El puerto ya devuelve los bytes:

```go
type CompileArtifactResult struct {
	Success  bool
	Log      string
	Artifact []byte
}
```

Y gracias a la normalización el artefacto es **un solo archivo con ruta conocida**, así que se saca con `CopyFromContainer` + `extractFirstFile`, que ya existen. `ExecutionSession.Compile` (soluciones) no se toca: sigue compilando en el mismo container donde va a correr, sin extraer nada.

**Lo que sí hay que corregir es el nombre**: `NativeCompiler` deja de ser nativo, y su comentario de doc —*"compiles ... directly on the worker's own filesystem, no sandbox"*— pasa a ser falso. Renombrar a algo como `ArtifactCompiler`.

**A verificar antes de dar por hecho**: que `jar` venga incluido en `openjdk-17-jdk-headless` (debería, "headless" refiere a las librerías gráficas y no a las herramientas), y acotar el `-C /sandbox .` a los `.class` para no meter el fuente dentro del JAR.


**Corregido al ejecutar (ver Paso 3).** El nombre fijo terminó siendo **uno por rol** —`Checker` y `Validator`, en PascalCase— con un token `{name}` en los cuatro campos, y no el `checker` único que fija esta decisión. El motivo: los mismos cuatro campos compilan también el validator, así que el nombre único obligaba a que un validator Java declarara `public class checker`, que es falso de frente. El título de esta decisión ("la config queda 100% estática") queda entonces desactualizado: hay exactamente una sustitución, uniforme en los tres lenguajes y con valor tomado de un enum nuestro de dos elementos. Los dos beneficios que la decisión realmente compraba —cero lógica por lenguaje en Go, cero sorpresas en runtime— se conservan.

Y el comando de Java de esta decisión **está mal como está escrito acá**: `-C /sandbox .` mete el fuente dentro del JAR. Verificado corriéndolo. La versión correcta compila con `-d /sandbox/classes` y empaqueta desde ahí.

### D10 — `judge_config.yaml`: separar propiedades del lenguaje de dimensionamiento del pool ✅

Hoy la sección `languages:` mezcla dos cosas independientes: **propiedades del lenguaje** (imagen, extensión, comandos) y **dimensionamiento** (`cpu`, `memoryBytes`). Con dos pools eso deja de funcionar, porque el mismo lenguaje tiene tamaños distintos según el pool. Y los campos de artefacto de D9 **se reparten entre los pools**: `artifactSource`/`artifactCompile`/`artifactPath` se usan al compilar (pool A) y `artifactRun` al ejecutar (pool B).

```yaml
languages:                    # propiedades del LENGUAJE, sin importar el pool
  cpp20:
    image: "judge-runner:cpp20"
    extension: "cpp"
    compileCmd: "..."         # soluciones
    runCmd: "..."             # soluciones
    artifactSource:  "..."    # checker: usado en pool A
    artifactCompile: "..."    # checker: usado en pool A
    artifactPath:    "..."    # checker: usado en pool A
    artifactRun:     "..."    # checker: usado en pool B
  java17: ...
  python310: ...
  compare:
    image: "judge-runner:compare"   # sin comandos: solo ejecuta el binario propio

pools:                        # dimensionamiento, propiedad del POOL
  solutions:
    budgetBytes: ...
    languages:
      cpp20:     { cpu: "1", memoryBytes: 2147483648 }
      java17:    { cpu: "1", memoryBytes: 2147483648 }
      python310: { cpu: "1", memoryBytes: 1073741824 }
  checkers:
    budgetBytes: ...
    languages:
      cpp20:     { cpu: "1", memoryBytes: 536870912 }
      java17:    { cpu: "1", memoryBytes: 1073741824 }   # más, por la JVM
      python310: { cpu: "1", memoryBytes: 536870912 }
      compare:   { cpu: "1", memoryBytes: 134217728 }
```

**Por qué encaja**: `pools.<nombre>` mapea uno a uno con el `PoolConfig{MemLimitBytes, Languages map[string]LanguageConfig}` que la estructura `Pool` ya recibe. Construir dos pools es leer dos secciones y llamar dos veces al mismo constructor — sin tocar el pool.

**Resuelve el reparto de los campos de artefacto**: quedan todos juntos bajo el lenguaje, que es de quien son propiedad, y cada pool toma el que necesita. No hay que decidir "a qué pool pertenece `artifactRun`".

**A documentar**: `compare` aparece en `languages` con solo una imagen y ningún comando. Es coherente —no compila ni interpreta nada, ejecuta un binario fijo— pero conviene aclararlo para que no parezca una entrada a medio llenar.

**Los tamaños del ejemplo son ilustrativos.** El de `java17` en pool B va más alto porque una JVM en 512 MiB es ajustada, pero eso hay que medirlo (ver A3).

### D11 — Los dos bugs de memoria se arreglan dentro del rediseño ✅

Son **dos**, y el segundo hace inseguro el arreglo obvio del primero.

**Bug 1 — el `memoryLimit` del problema no se aplica.** `RunTestCase` solo pone el límite de tiempo en el comando; `req.MemoryKb` no se usa en ningún lado. El MLE salta al techo del *container* (2 GiB), no al que declaró el problema. Una solución que use 1.5 GB en un problema con límite de 256 MB pasa igual.

**Bug 2 — la memoria medida está contaminada.** Mirando la asimetría con el tiempo:

```go
cpuAfterNs, memoryKb := s.readStats(ctx)
cpuTimeMs := int((cpuAfterNs - cpuBeforeNs) / 1_000_000)   // DELTA, aísla esta corrida
...
MemoryKb: memoryKb,                                        // valor absoluto
```

y de dónde sale: `int(stats.MemoryStats.MaxUsage / 1024)` — el **pico del container desde que arrancó**. Como los containers se reusan entre casos y entre judgings (`Close` limpia `/sandbox` pero no resetea el contador del cgroup), ese número nunca baja y refleja el pico de **cualquier** corrida que haya pasado por ahí, incluidas las de otras submissions.

**Por eso el arreglo obvio del bug 1 es peligroso**: copiar lo que se hace con el tiempo (`if runResult.MemoryKb > limits.MemoryKb`) le daría MLE a una submission porque *otra anterior* consumió mucha memoria en ese mismo container.

**El arreglo**: `docker update --memory=<límite del problema>` al reclamar el container para un judging. El MLE lo hace cumplir el kernel al valor correcto y se detecta con el mecanismo que ya existe (SIGKILL → exit 137 → `exitCodeMLE`).

- Usa la detección que ya está, sin agregar comparaciones sobre un número contaminado.
- La contabilidad del pool queda **conservadora**: se reservaron 2 GiB de presupuesto y se usa menos.
- Bajar el límite de un container en marcha es seguro acá: se hace al reclamarlo, con `/sandbox` recién limpiado y sin procesos corriendo. (Docker puede rechazar una bajada si el uso actual ya la supera.)

**Se descartó `ulimit -v`**, que es lo primero que uno prueba: limita el espacio de direcciones **virtual**, no el residente, y una JVM reserva muchísimo virtual sin usarlo — mataría todo Java.

**Pendiente de definir**: qué reportar como KB consumidos. Aunque el veredicto sea correcto, el número seguiría viniendo de `MaxUsage` contaminado. Resetear el pico del cgroup entre corridas depende de la versión de cgroup y no está expuesto por la API de Docker, así que hay que elegir entre reportar algo aproximado o directamente el límite.

**Alcance**: el rediseño ya modifica `RunTestCase` (por D7, la salida deja de ir a `/sandbox/output.txt`), así que el costo marginal es bajo.

### D12 — El rediseño va en la misma rama, antes de abrir el PR ✅

`feature/judge-validation-pipeline` tiene las Fases 2-7 en un commit, rebaseada y en verde. El rediseño reescribe una parte acotada —la mecánica de ejecución en `adapter/judge` y algunos puertos— y deja intacto el grueso: dominio, migración, cola, `PublishProblemUseCase`, el poll, el guard de validación, el endpoint de última validación, la recuperación de stale.

Se decidió meter el rediseño en la misma rama porque la parte que se reescribe contiene hoy: el bug de `CheckerLanguage`/`CheckerFilename` (toda submission con checker personalizado termina en `SYSTEM_ERROR`), la ejecución nativa del checker (la exposición de seguridad que motivó todo esto), y código que vamos a borrar. Mergear a sabiendas esas tres cosas es peor que un PR grande, sobre todo estando el rediseño ya diseñado.

Se descartaron: abrir el PR ahora y hacer el rediseño después (mergea el bug y la exposición, y los revisores gastan tiempo en código que se borra), y abrir el PR bloqueando los checkers personalizados hasta que llegue el rediseño (código temporal, y pospone parte de la verificación de la Fase 8).

---

### D13 — Un invariante duro, y el resto es comodidad que degrada sola ✅

Esta decisión **disuelve la pregunta** de qué pasa si un pool está lleno cuando llega un judging o un publish. Hay que distinguir dos cosas que no son lo mismo.

#### Lo obligatorio: que nadie quede esperando

El pool tiene dos comportamientos distintos ante presión:

- Presupuesto lleno pero **con containers ociosos** → `Claim` evicta el más viejo por LRU y crea el que necesita. No bloquea.
- Solo bloquea cuando **todos están ocupados**. Y como cada judging sostiene exactamente un container de cada pool, "todos ocupados" significa `maxConcurrent` containers tomados — justo el techo del semáforo.

Basta entonces con garantizar, **en cada pool**:

```
budget ≥ maxConcurrent × max(memoryBytes entre los lenguajes de ese pool)
```

El **máximo** porque no se sabe qué lenguaje va a tocar: el peor caso es que los `maxConcurrent` judgings usen el más pesado.

**El worker valida esto al arrancar y falla rápido si no se cumple**, en vez de que aparezca como un cuelgue misterioso en medio de una competencia. Mismo criterio que ya usa `Consume`, que se niega a arrancar si falta un manejador para alguna etiqueta conocida de la cola.

**Consecuencia a tener presente**: agregar un lenguaje pesado sube el piso para todos, porque el invariante se calcula sobre el máximo. Es el precio de garantizar el peor caso.

#### Lo deseable: containers calientes, que degrada bien

Que entre un container ocioso de cada lenguaje en cada pool es **comodidad, no requisito**. El LRU existe justamente para eso: si se agregan muchos lenguajes y no entran todos, evictar es el comportamiento normal y esperado, no una falla. Se paga recrear un container de vez en cuando y nada más.

El objetivo razonable es que quepan los **lenguajes principales** con holgura. Sobre el nodo actual (budget 9.5 GiB, `maxConcurrent` = 2):

| | Mínimo obligatorio | Con los 3 lenguajes calientes |
|---|---|---|
| Pool A | 2 × 2 GiB = 4 GiB | 2 + 2 + 1 = **5 GiB** |
| Pool B | 2 × 1 GiB = 2 GiB | 0.5 + 1 + 0.5 + 0.125 = **2.125 GiB** |
| Demonio Docker | — | 0.5 GiB |
| **Total** | 6.5 GiB | **7.6 GiB ≤ 9.5** ✓ |

El `e2-standard-4` actual cumple el invariante con margen y además mantiene calientes los tres lenguajes en ambos pools. Crecer la máquina sirve solo para subir `maxConcurrent` y procesar más submissions en paralelo: es una decisión de throughput para la competencia, separada de este rediseño.

**Dato que falta medir** para afinar los tamaños por container: el pico real de memoria de `g++` compilando un checker testlib, de `javac`, y de una JVM ejecutando un checker en pool B. Los valores de este documento son estimaciones sin medir.

---

## Detalles a resolver al implementar

Ninguno bloquea el diseño; son elecciones que conviene hacer con el código delante.

- **Si la entrada del caso de prueba viaja por el volumen compartido** (más barato que `CopyToContainer`) o sigue por la API. La salida sí va por el volumen (D7); la entrada podría ir por cualquiera de los dos.
- **Limpieza del directorio del judging**: hoy `Session.Close` hace `rm -rf /sandbox/*`; hay que sumar el borrado de `<raíz>/<uuid>` por parte del worker.
- **`maxOutputBytes`**: hoy 64 MiB. Con D7 deja de presionar al worker, pero sigue acotando cuánto disco del `emptyDir` puede consumir un judging. Bajarlo a ~8 MiB sigue siendo generosísimo para programación competitiva.
- **Qué reportar como KB consumidos** (ver D11): el veredicto de MLE queda correcto porque lo hace cumplir el kernel, pero el número que se muestra vendría de `MaxUsage`, contaminado por corridas anteriores en el mismo container.
- ~~**Verificar que `jar` venga en `openjdk-17-jdk-headless`** y acotar el `-C /sandbox .` a los `.class`.~~ **Resuelto en el Paso 3**, verificado corriéndolo en la imagen real: `jar` viene incluido, y el `-C /sandbox .` efectivamente metía el `.java` en el JAR. Se compila con `-d /sandbox/classes` y se empaqueta desde ahí.
- **Un checker Java con la clase no pública falla tarde y disfrazado.** Verificado en `judge-runner:java17`: si la clase no es `public`, javac acepta cualquier nombre de archivo y `jar` no verifica que la main-class exista, así que la compilación pasa y revienta al ejecutar con `Could not find or load main class Checker`. Se decidió (Paso 3) no validar nada al subir, porque el mensaje nombra la clase esperada y una regex sobre fuente Java es código que se pudre. Si el caso aparece en la práctica, el lugar para atajarlo es el Paso 5, con el checker corriendo de verdad.

- **Documentar la convención de nombres de clase.** Con el Paso 3, un archivo Java debe declarar `Solution`, `Checker` o `Validator` según su rol. `README.md:133` y `specs/Judge System/README.md:161` ya documentaban `Solution.java` con mayúscula, así que la deriva contra el código se cierra sola — pero falta escribir en algún lado la convención completa, incluida la salida de la clase no pública para poder subir varias soluciones Java (ver la sección de bugs).
- **Nombre e imagen del binario de comparación** (`cmd/compare`, `docker/judge/compare.Dockerfile`), y sumarlo al init container `prepull-language-images` y al pipeline de build/push (ver D6).
- ~~**Falta una validación de arranque, sacada a propósito**: el caso inverso — un lenguaje que puede recibir soluciones y que ningún pool dimensiona.~~ **Resuelto en el Paso 3** con la alternativa que había quedado pendiente: se deriva de `runCmd`, sin tocar el dominio. Ahí está también por qué se descartó el chequeo cruzado contra `virtual_object.json`, y qué hueco deja abierto.

  Se implementó y **se revirtió**: la versión que se escribió preguntaba `submission.NewLanguage(lang)` para saber si un lenguaje era enviable, y eso metía dominio dentro de `cmd/worker` — que quedaba como el único cmd importándolo (`cmd/api` no importa dominio en absoluto) — además de usar un constructor como predicado, que no es para lo que existe.

  La alternativa que quedó pendiente de evaluar: derivarlo de la propia config, sin dominio. Un lenguaje en el que se escriben soluciones es el que declara `runCmd`, y `compare` (que llega en el Paso 4) no va a tener uno — solo `image` y `artifactRun`. Así la exención sale sola de la forma del archivo. Su punto débil: si alguien agrega un lenguaje al dominio pero olvida el `runCmd` en el YAML, el chequeo no salta.
- ~~**Partir `loadJudgeConfig`, que hoy hace tres cosas**~~ **Resuelto en el Paso 3**: `validateJudgeConfig` y `applyJudgeConfigDefaults` son funciones propias que devuelven error, y los tests las llaman directo en vez de reimplementar las reglas. Queda pendiente **mover las tres a `internal/config/judge_config.go`**, anotado más abajo con el resto de la revisión de configuración.

- **El wiring del pool depende de una invariante que no expresa**: `Image: judgeCfg.Judge.Languages[lang].Image` es seguro *solo porque* la validación de arranque ya garantizó que ese lenguaje existe. El Paso 3 lo acotó agregando una regla de que todo lenguaje declare imagen no vacía, así que el peor caso ya no es un nombre de imagen vacío; pero **el acceso al mapa sigue ignorando el `ok`**, y sigue dependiendo de que nadie reordene ni quite esa validación.

- **Tres cosas menores de `judge_config.yaml`**, sin decidir:
  - La sección `pools` usa notación inline (`{ cpu: "1", ... }`) mientras el resto del archivo es bloque. Uniformar, o asumir el inline como convención para los mapas chicos de dimensionamiento.
  - ~~El comentario que explica los nombres anteriores (`memoryOverheadBytes`/`cpuOverheadCores`) es ruido a futuro.~~ **Resuelto en el Paso 3**: los comentarios del archivo se reescribieron en inglés y en una línea, y esa explicación se fue al historial de git y a este documento.
  - `memoryBytes: 2147483648` son bytes crudos, ilegibles sin el comentario de al lado. Kubernetes acepta `2Gi` en sus manifests; acá no, porque el parser es un `int64` pelado.

- **Revisión propia de `RUNNER_ARCHITECTURE.md`**: al actualizarlo por el renombre aparecieron discrepancias **preexistentes** que no se tocaron, porque merecen su propia pasada. `syntaxCheckCmd: "pypy3 -m py_compile ..."` es un campo que **no existe en el código** y además nombra `pypy3` en vez de `python3`; y `compileCmd: "javac ... /sandbox/Solution.java"` usa mayúscula contra el `solution.java` real. Sugiere que ese documento describe un diseño previsto que la implementación no siguió en varios puntos — el `-Xmx{memoryLimit}m` (ver la sección de bugs) es el caso más consecuente.

---


- **Revisar la configuración de la plataforma entera — fuera de este rediseño, pero pedido explícitamente.** Hoy hay **tres superficies de configuración** que nadie coordina:

  | Dónde | Qué tiene | Quién lo lee | Cómo valida |
  |---|---|---|---|
  | variables de entorno → `internal/config/config.go` | 22 campos: DB, storage, JWT, SMTP, Redis, RabbitMQ, CORS, URL del frontend, clave de cifrado | `cmd/api` | `os.Exit` dentro de los getters |
  | `config/virtual_object.json` | lenguajes soportados, extensiones, límites de tiempo/memoria, tags, límites de subida | `cmd/api` | ninguna |
  | `config/judge_config.yaml` | lenguajes del judge, comandos, dimensionamiento de pools | `cmd/worker` | decode estricto + validación al arrancar |

  Los problemas concretos, para no partir de cero cuando se encare:

  - **El punto de partida: `cmd/worker/main.go` define adentro el esquema, la carga y la validación de `judge_config.yaml`** — cinco structs (`judgeConfigFile`, `judgeSection`, `judgeLanguageConfig`, `judgePoolConfig`, `judgePoolLanguageConfig`), la función `loadJudgeConfig`, cuatro constantes de default, y copias propias de `getEnv`/`getRequiredEnv`/`getRequiredEnvInt64`. El composition root debería **enchufar** piezas, no definir el formato de un archivo. Y `cmd/api` ya delega eso en `internal/config` (`Load()` + `loadVirtualObject()`), así que el worker es la asimetría. Propuesta concreta: mover esquema, carga y validación a `internal/config/judge_config.go`, espejando `virtual_object.go`, y dejar en `main.go` solo el wiring. Sus tests se mudan con él, y dejan de vivir en `package main`.

  - **`Config` es un struct único de 22 campos** que se pasa entero: ningún consumidor declara qué necesita realmente, así que no se puede saber quién usa qué sin leer todo.
  - **`cmd/worker/main.go` reimplementa `getEnv`/`getRequiredEnv`** porque no usa `internal/config`. Misma función, dos copias.
  - **El conjunto de lenguajes vive en dos archivos que tienen que coincidir y nadie cruza**: `supportedLanguages`/`languageExtensions` en el JSON y `languages` en el YAML. Es el hueco que la validación del Paso 3 deja abierto a propósito (ver ahí la opción (B) descartada).
  - **Tres formatos y tres estilos de carga** (env, JSON, YAML), con tres niveles de rigor distintos: el YAML rechaza claves desconocidas, el JSON no valida nada.
  - **`virtual_object` no dice qué contiene.** El nombre no describe ni límites de problema ni lenguajes ni tags.

## Consecuencias

### Sobre la Fase 5 del roadmap: se cae casi entera

Si el worker no compila nada, no necesita `g++` ni JDK ni `python3` en su filesystem. Por lo tanto:

- El split del `Dockerfile` en `api-final`/`worker-final` **deja de tener sentido**. La imagen sigue siendo una sola, Alpine, compartida entre API y worker, como hoy.
- Se caen con él los cambios en `docker-compose.yml`, el workflow de CI, `bootstrap.ps1`, los placeholders de los 5 manifests, y los dos nombres nuevos en Artifact Registry.

De la Fase 5 sobreviven dos cosas, ninguna relacionada con esto: el `BackendConfig` del Ingress (timeout de 600s) y el runbook para recrear la cola `submissions`.

### Sobre la Fase 6: se reescriben los adapters

**Se borra** (la ejecución nativa desaparece por completo):

| Archivo | Motivo |
|---|---|
| `adapter/judge/native_compiler.go` + `_cpp/_java/_python` + tests | la compilación pasa al sandbox (pool A) |
| `adapter/judge/validator_runner.go` + tests | pasa al sandbox (pool B) con forma de sesión |
| `adapter/judge/judging_timeouts.go` (`isTimeoutErr`) | era para `exec.CommandContext` nativo |
| `adapter/judge/artifact_invocation.go` + tests | la lógica por lenguaje se va a `judge_config.yaml` (D8/D9) |

**Se reescribe:**

| Archivo | Motivo |
|---|---|
| `adapter/judge/output_comparator.go` | pasa al pool con forma de sesión; la comparación por tokens también (D6) |
| `adapter/judge/pool/` | segunda instancia con presupuesto propio; bind mount del volumen compartido (D7) |
| `adapter/judge/session.go` | salida al volumen en vez de `copyOutput`; `docker update --memory` al reclamar (D11) |
| `adapter/judge/config.go` + `judge_config.yaml` | separar `languages` de `pools`, campos de artefacto (D9/D10) |

**Se construye nuevo:**

- Los puertos de sesión de pool B y sus adapters (D3).
- `cmd/compare/` y `docker/judge/compare.Dockerfile` — el binario de comparación por tokens y su imagen mínima (D6).
- El `emptyDir` compartido en `worker.yaml`, montado en `dind` y en `worker` (D7).
- La validación al arrancar del invariante de dimensionamiento (D13).

**Se renombra**: `NativeCompiler` → `ArtifactCompiler` (deja de ser nativo, y su comentario de doc pasa a ser falso).

**Mejoras laterales que salen de paso**: los checkers Java multi-clase pasan a funcionar (el artefacto es un JAR, D9), y desaparecen las 25 descargas del mismo binario desde GCS por submission (D3).

---

## Bugs encontrados en el camino

Todos son preexistentes y ninguno lo introdujo este rediseño, pero no todos se arreglan dentro de él: los dos de memoria sí (D11), y el de `CheckerFilename` desaparece por construcción (D8/D9). El MLE de Java y la colisión de soluciones Java quedan **sin arreglar**, cada uno con su nota.


### El MLE de Java no se puede detectar — solución pendiente de rediseño

Encontrado al revisar por qué C++ y Java tenían el mismo `memoryBytes`. Son **dos** bugs; el primero ya está arreglado, el segundo **no tiene todavía una solución aceptada**.

**Bug ya corregido — Java recibía la cuarta parte de la memoria.** El `runCmd` era `java -cp /sandbox solution`, sin ninguna opción de memoria. Una JVM moderna detecta el cgroup y aplica `MaxRAMPercentage=25` por defecto, así que **en un container de 2 GiB el heap máximo era 512 MiB** (medido con la imagen real del proyecto), contra ~2 GiB para una solución en C++ con el mismo `memoryBytes`. Un problema con límite de 1 GB: la solución en C++ pasaba y la equivalente en Java moría. Se agregó `-XX:MaxRAMPercentage=75` al `runCmd` (medido: 1536 MiB). Se eligió 75 y no 90 pensando en D11: cuando el container se achique al límite real del problema, un 90% de 256 MB dejaría 26 MB para el overhead de la JVM y la mataría el cgroup.


**Hallazgo que reencuadra este bug**: `RUNNER_ARCHITECTURE.md` especificaba, en su ejemplo de configuración de lenguajes, `runCmd: "java -Xmx{memoryLimit}m Solution"` — o sea, **el diseño original sí contemplaba pasarle a la JVM el límite de memoria del problema**, con una plantilla por problema. La implementación quedó en `java -cp /sandbox solution`, sin ninguna opción de memoria. No fue un descuido de diseño: el diseño lo tenía bien y la implementación lo dejó caer.

Eso conecta directo con D11: `{memoryLimit}` implica que el `runCmd` de Java **no puede ser un string estático**, necesita sustitución por problema. Al resolver el Paso 6 conviene mirar esa especificación original antes de inventar otra.

**Otras discrepancias del mismo bloque, preexistentes y sin corregir** (se dejaron para una revisión propia de `RUNNER_ARCHITECTURE.md`, no son de este rediseño): `syntaxCheckCmd: "pypy3 -m py_compile ..."` es un campo que **no existe en el código** y además nombra `pypy3` en vez de `python3`; y `compileCmd: "javac ... /sandbox/Solution.java"` usa mayúscula contra el `solution.java` real.
**Bug sin resolver — el veredicto de MLE en Java es imposible con el mecanismo actual.**

```go
exitCodeMLE = 137 // OOM killer sent SIGKILL (128 + 9) when cgroup memory limit was exceeded
```

La JVM **nunca deja que el cgroup la mate por agotamiento de heap**: refuerza su propio tope y lanza `OutOfMemoryError`. Medido con un programa que asigna sin parar en un container de 2 GiB:

| Configuración | Exit code |
|---|---|
| `-XX:MaxRAMPercentage=75` | 1 |
| `-XX:MaxRAMPercentage=100` | 1 |
| `-Xmx1900m` | 1 |
| `-XX:+ExitOnOutOfMemoryError` | 3 |
| `-XX:+CrashOnOutOfMemoryError` | 134 |

Ningún valor produce 137. Así que **hoy todo MLE de Java se reporta como runtime error**, y sin rastro: el comando de ejecución manda stderr a `/dev/null`, así que ni el mensaje de `OutOfMemoryError` sobrevive.

**Se consideró y NO se aceptó**: usar `-XX:+ExitOnOutOfMemoryError` (exit 3 determinista) y volver la interpretación del código de salida dependiente del lenguaje. Funcionaría, pero un programa que haga `System.exit(3)` a propósito se reportaría como MLE, y sobre todo convierte una constante única (`exitCodeMLE`) en una tabla por lenguaje, esparciendo conocimiento del runtime de Java dentro de la capa de aplicación.

**Queda para el Paso 6** (D11, límites de memoria reales), donde hay que **pensar otra solución o refinar ésta**. Un punto de partida: si ahí se va a aplicar el límite del problema con `docker update --memory`, quizás la señal de MLE deba salir del propio cgroup (leyendo `memory.events`/`memory.max` después de la corrida) en vez de inferirse del código de salida — lo que además sería uniforme para los tres lenguajes en lugar de una regla por lenguaje.
### El `memoryLimit` del problema no se está aplicando

Encontrado al revisar para qué sirve `LanguageConfig.MemoryBytes`. **Arreglo decidido en D11**: `docker update --memory` al reclamar el container.

`RunTestCase` construye el comando así:

```go
cmd := fmt.Sprintf(
    "timeout --kill-after=1s %ds %s < /sandbox/input.txt > /sandbox/output.txt 2>/dev/null",
    wallBackstopSecs, s.langCfg.RunCmd,
)
```

Solo aplica el límite de **tiempo**. `req.MemoryKb` —que viene del `memoryLimit` del problema— **no se usa en ningún lado**: no hay `ulimit -v` ni equivalente. Entonces el MLE salta al techo del **container** (2 GiB), no al límite que declaró el problema.

La asimetría con el tiempo lo deja claro: para tiempo hay verificación posterior (`if runResult.TimeMs > limits.TimeLimitMs`), pero para memoria no existe — se confía solo en `exitCodeMLE`, que dispara al techo del container.

**Consecuencia**: una solución que use 1.5 GB en un problema con límite de 256 MB pasa igual.

### La memoria medida está contaminada entre submissions

Encontrado al buscar cómo arreglar el bug anterior — y es lo que hace **peligroso** el arreglo obvio. El detalle completo está en **D11**; en resumen: `MemoryKb` sale de `stats.MemoryStats.MaxUsage`, que es el pico del container **desde que arrancó**. Como los containers se reusan y `Close` no resetea el contador del cgroup, ese número refleja el pico de cualquier corrida anterior, incluidas las de otras submissions. Por eso no se puede replicar el patrón del tiempo (`if runResult.MemoryKb > limits.MemoryKb`): daría MLE por culpa de una submission ajena.

### `CheckerLanguage`/`CheckerFilename` no llegan al judging real

**Decisión explícita del usuario**: no arreglarlo aparte, porque el rediseño reescribe ese call site igual. Y con D8/D9 termina **desapareciendo por construcción**: al normalizar el artefacto, `CheckerFilename` deja de existir en los puertos de ejecución y no hay nada que se pueda olvidar de pasar.

En `application/judge/judge_submission.go` (el camino de judging de submissions **reales**) el `CheckRequest` se construye **sin** `CheckerLanguage` ni `CheckerFilename`. En `validate_solutions.go` (publish) sí se pasan. Los dos campos se agregaron en la Fase 6 y se actualizó un solo call site.

Consecuencia:

1. `CheckerLanguage` queda en su valor cero → `Language{value: ""}` → `.String()` devuelve `""`
2. `artifactInvocation` hace `switch` sobre `""` → cae en `default:` → error "unsupported language"
3. `customCheckerCompare` devuelve `apperror.NewInternal()`
4. El judging lo trata como falla de infraestructura, reintenta, vuelve a fallar → la submission termina en `SYSTEM_ERROR`

O sea: **toda submission real contra un problema con checker personalizado falla con `SYSTEM_ERROR`**. Es la misma clase de bug que la Fase 6 venía a arreglar.

Los tests no lo agarran porque `judge_submission_test.go` mockea `OutputChecker` (al mock no le importan los campos) y `output_comparator_test.go` prueba el adapter directo con los campos ya puestos. Ninguno cruza la frontera donde se pierden. **El rediseño tiene que incluir un test que cruce esa frontera**, no solo el arreglo.

### Varias soluciones Java en un mismo problema colisionan al subir — pendiente, fuera de este rediseño

Encontrado al cerrar el nombre fijo del Paso 3. **No se arregla acá**: el arreglo real toca la identidad de las soluciones en el dominio, no la mecánica de ejecución.

La cadena que lo produce:

1. Convención de la plataforma: la clase de una solución Java tiene que llamarse `Solution` (antes `solution`, ver Paso 3).
2. Regla de javac: una clase **pública** `Solution` solo compila en un archivo llamado `Solution.java`.
3. Identidad al subir: el filename. `Problem.AddSolution` **reemplaza** cuando `Filename()` coincide, y `RemoveSolution` / `DELETE /problems/{slug}/files` direccionan por `fileName`.

Un problem setter que hace lo obvio —clase pública— termina con todas sus soluciones Java llamadas `Solution.java`, así que al subir la segunda **reemplaza** la primera. El caso típico de validar un problema con varias soluciones (una correcta, una lenta, una incorrecta) queda reducido a una sola.

**La salida existe, y está verificada corriéndola en `judge-runner:java17`**: si la clase **no es pública**, javac acepta cualquier nombre de archivo. Un `BruteForce.java` que declara `class Solution { public static void main... }` compila a `Solution.class` y corre con `java -cp <dir> Solution`. O sea que el setter puede mantener nombres distintos en su máquina y subir los tres sin colisión, a costa de una convención que hay que documentar.

**Es preexistente**: hoy la convención es `solution` en minúscula con exactamente la misma mecánica. El Paso 3 le cambia la caja a una letra; no introduce ni empeora la colisión.

**Qué costaría arreglarlo de verdad**: que las soluciones dejen de identificarse por filename. Toca `AddSolution`, `RemoveSolution`, el endpoint de borrado y el contrato de la API. Es una decisión propia, no un detalle de este rediseño.

### `testlib.h` no está en la imagen de C++ — pendiente, fuera de este rediseño

Encontrado al preparar el Paso 3 y verificado adentro de `judge-runner:cpp20`: no existe `/usr/include/testlib.h`, ni ninguna copia en el repositorio. `cpp20.Dockerfile` instala únicamente `g++`.

Un checker o validator escrito con testlib —que es el caso normal en programación competitiva, y el que este mismo documento asume cuando estima el costo de compilar ("testlib son ~10k líneas de C++ con templates")— falla con `testlib.h: No such file or directory`. O sea que la funcionalidad de checkers personalizados no es usable de verdad hasta que el header esté en la imagen.

**Por qué no se arregla acá**: implica agregar el header a la imagen de C++, reconstruirla y subir `RUNNER_VERSION` en `deploy/k8s/judge/images-configmap.yaml`. Es trabajo de imágenes, no de código, y el rediseño no lo empeora — la compilación nativa tampoco lo tenía.

**Decisión pendiente al hacerlo**: si el header se vendorea en el repo (reproducible, versión congelada) o se baja en el build (más simple, dependencia de red en el build).

---

## Plan de ejecución

**Regla de cada paso**: el proyecto compila y la suite queda en verde al terminarlo. Nada de estados intermedios rotos — en Go no se puede migrar media interfaz, así que cada cambio de puerto arrastra a sus llamadores y mocks en el mismo paso.

**Dos hitos que valen la pena tener presentes**: al terminar el **paso 5** queda cerrado el **problema 2** (ningún código del problem setter corre ya con privilegios del worker). Al terminar el **paso 7** queda cerrado el **problema 1** (el worker vuelve a solo coordinar). El arreglo de seguridad aterriza antes que el de rendimiento, y cada uno vale por separado.

### Paso 1 — El binario de comparación, aislado (D6) ✅ COMPLETO

`cmd/compare/main.go` replicando `tokenCompare`, con sus tests (portando los casos que ya existen), `docker/judge/compare.Dockerfile`, y sumarlo a `build-judge-images.sh` y al init container `prepull-language-images`.

Nadie lo usa todavía: es puramente aditivo. **Va primero porque tiene riesgo cero y despeja temprano la pregunta más delicada de D6** — si el binario replica *exactamente* el comportamiento que hoy está en producción.

**Además, surgido al implementarlo**: la versión de las imágenes del judge estaba **quemada como literal `v0.1.0` en cuatro archivos** (el prepull de `worker.yaml`, el comentario de prerrequisitos de `bootstrap.ps1`, el skill `recreate-environment`, y `GKE_DEPLOY_ROADMAP.md`), a diferencia de la imagen del backend que usa el placeholder `__IMAGE__`. Y una sola versión servía para las cuatro imágenes, lo que hubiera obligado a subirlas todas para corregir un detalle del comparador — que es código nuestro, no un toolchain, y evoluciona por separado.

Se resolvieron las dos cosas, y el mecanismo elegido corrige un primer intento: la versión arrancó viviendo en un `image-versions.env` sustituido con placeholders, imitando a `__IMAGE__`. Eso estaba mal por dos motivos. Primero, `__IMAGE__` existe solo porque **Kubernetes no deja parametrizar el campo `image:`** — es un rodeo a una limitación, no un patrón a copiar. Segundo, acá la versión no va en un campo `image:` sino dentro de un script de shell, donde una variable de entorno alcanza.

Quedó entonces en `deploy/k8s/judge/images-configmap.yaml`, un manifest como el resto del directorio, con `RUNNER_VERSION` (los tres toolchains, que se suben a mano) y `COMPARE_VERSION` (el comparador) independientes, inyectados al initContainer del prepull vía `configMapKeyRef`. Sin placeholders, sin parser en `bootstrap.ps1`, y bumpear una versión ya no exige re-correr el bootstrap entero.

**Y el comparador pasó al CI**, porque es código nuestro del mismo repo, no un toolchain. Se agregó un paso al workflow de deploy que ya existe —no un workflow aparte, porque los dos dispararían con el mismo tag `v*` y el `kubectl set image` del deploy podía reiniciar el worker antes de que el otro aplicara el ConfigMap, dejando al pod nuevo con la versión vieja. El paso aplica tres reglas: si `cmd/compare` cambió desde la release anterior y `COMPARE_VERSION` no se subió, **falla**; si cambió y la versión es nueva, construye y pushea; si no cambió, no hace nada. El ConfigMap se aplica siempre, para que el cluster refleje lo que declara git. Requiere `fetch-depth: 0` en el checkout, que era shallow.

### Paso 2 — Reestructurar la config (D5, D10) ✅ COMPLETO

`judge_config.yaml` pasa a `languages` + `pools`, y se renombran los dos overheads. Se actualiza el parseo en `cmd/worker/main.go`.

Se sigue construyendo **un solo pool** (desde `pools.solutions`). Sin cambio de comportamiento: es preparar el terreno.

**Alcance afinado al ejecutarlo**: el plan original metía acá también los campos de artefacto y el pool de checkers, pero **nadie los consume hasta los Pasos 3 y 4**. Aterrizan en el paso que los usa, para no dejar config parseada que ningún código lee. El Paso 2 queda entonces puramente estructural: `languages` (imagen, extensión, comandos) separado de `pools.<nombre>.languages` (cpu, memoryBytes), y los dos renombres.

**Los renombres**: `memoryOverheadBytes` → `dockerDaemonReserveBytes` y `cpuOverheadCores` → `dockerDaemonReserveCores`. Los nombres viejos decían "para el proceso worker", que era falso: los dos se recortan del presupuesto del container `dind`, donde viven el demonio Docker y los containers que crea — el worker corre en otro container, con su propio cgroup. El comentario equivocado de `PoolConfig.OverheadBytes` ("set from POD_MEMORY_OVERHEAD", cuando en realidad viene del config del judge) también se corrigió.

**Decodificación estricta, agregada al implementar**: `yaml.Unmarshal` **ignora claves desconocidas en silencio**, así que un tag de struct mal escrito dejaba el campo en cero y el presupuesto del pool salía mal sin que nada avisara — justo el riesgo de un refactor que renombra claves. `loadJudgeConfig` pasa a usar `yaml.NewDecoder` con `KnownFields(true)`: si el archivo y las structs se separan, el worker no arranca.

**Y una validación nueva al arrancar**: un pool que dimensiona un lenguaje sin imagen declarada solo fallaría cuando ese lenguaje se reclama por primera vez, en medio de un judging. Ahora se rechaza al arrancar.

**Tests**: `cmd/worker/config_test.go`, tres tests que parsean el `judge_config.yaml` real. Verificados rompiendo el archivo a propósito: una clave desconocida hace fallar el decode estricto, y un pool que dimensiona un lenguaje inexistente falla con mensaje claro.

### Paso 3 — La compilación de artefactos al sandbox (D9) ✅ COMPLETO

El puerto `NativeCompiler` pasó a `ArtifactCompiler`, con un adapter que reclama un container de pool A del lenguaje del artefacto, escribe el fuente, corre el comando de compilación vía `sh -c` y extrae el artefacto con `CopyFromContainer` + `extractFirstFile`. Se borraron once archivos del camino nativo. `PrepareJudgingUseCase` no cambió más allá de un campo del request: la firma del puerto sobrevivió, como el plan preveía.

**Con esto, el código del problem setter ya no se compila con los privilegios del worker.** Cae la superficie de ataque de tiempo de compilación (los *annotation processors* de `javac`, el `#include` de `g++` que vuelca archivos del sistema en el log de error). Queda la de ejecución, que cierran los Pasos 4 y 5.

#### Lo que se decidió en el camino

**1. Nombres por rol con un token `{name}`, en vez del nombre único de D9.** Ver la nota agregada en D9. Alternativas descartadas: **una sola palabra para los dos roles** (`verifier`, `artifact`) — obliga al autor de un checker Java a aprender una palabra que no es la de su rol, y el argumento de D9 sobre la consistencia con `solution` en realidad apunta en contra, porque `solution` funciona justamente por tener correspondencia 1 a 1 entre rol y palabra; y **ocho campos por lenguaje**, uno por rol — idénticos salvo la palabra, duplicación pura. El token preserva los dos beneficios reales de D9, y su riesgo se ataja con una regla de arranque que exige `{name}` en los cuatro campos.

**2. `solution` → `Solution`, en PascalCase como los otros dos.** Cierra de paso una deriva preexistente: `README.md:133` y `specs/Judge System/README.md:161` **ya documentaban** `Solution.java` con mayúscula contra el `solution.java` del código, así que hoy un concursante que sigue el README no compila. Es un cambio incompatible para Java (`public class solution` deja de compilar) sin datos productivos en riesgo, porque todo el pipeline es de esta rama.

**3. La clave de GCS del checker y el validator se normaliza** a `problems/{slug}/checker/Checker.<ext>`. D9 proponía `source.{ext}`; se descartó porque deja **dos nombres para el mismo archivo** —uno en el bucket, otro en el sandbox— y eso hay que explicarlo cada vez, mientras que la repetición de `checker/Checker.cpp` es fea una sola vez. El nombre que subió el problem setter **se conserva en la base** para mostrarlo. Las soluciones no se tocan: son varias por problema y se identifican por filename (ver la sección de bugs).

El nombre lo pasa quien ya sabe qué está subiendo (`handleChecker`/`handleValidator` y sus equivalentes del import ICPC), no una tabla `fileType → nombre`: un mapa devuelve `""` en silencio ante una clave inesperada y produciría `problems/abc/checker/.cpp`; un parámetro no se puede olvidar sin que el compilador avise.

**4. Ninguna validación al subir el archivo.** La que D9 proponía —exigir que el archivo se llame `checker.java`— **valida lo que no rompe**: con la normalización el nombre del archivo subido es irrelevante, y lo único que importa es el nombre de la clase pública adentro del fuente. Verificado en la imagen real: `MiChecker.java` con `public class Checker` compila perfecto, y `Checker.java` con `public class Foo` falla. Descartada también una regex sobre el fuente al subir: parseo de Java en la capa de aplicación, que el día que falle mal va a rechazar un archivo válido. El mensaje de javac ya nombra el arreglo y le llega al problem setter en el log de compilación.

**5. `Filename` sale de `CompileArtifactRequest`.** No es solo que dejaba de tener lectores: su comentario describía exactamente el comportamiento que este paso invierte. En su lugar entra `ArtifactRole`, un value object según D4 (campo privado, factorías de estado conocido, `String()`), cuyo **valor es el nombre fijo** — así no hay tabla rol → nombre de archivo que se pueda desincronizar del YAML. Lo que el value object no puede impedir es su valor cero, así que el adapter rechaza el rol vacío explícitamente.

**6. La validación de arranque que faltaba: proxy interno por `runCmd`.** Un lenguaje que declara `runCmd` es uno en el que se ejecutan soluciones, así que debe estar dimensionado por `pools.solutions` y declarar los cuatro campos de artefacto con el token. `compare` (Paso 4) no tendrá `runCmd` y queda exento sin ningún caso especial.

Se descartó el **chequeo cruzado** contra `config/virtual_object.json`, que sería el invariante de verdad: el conjunto de lenguajes que la plataforma acepta vive en ese archivo de la API, no en el del judge, así que **todo lo que validemos dentro de `judge_config.yaml` es un proxy**. Los dos archivos viajan en la misma imagen, así que es viable, pero acopla el arranque del worker a un archivo de configuración de la API y eso merece su propia decisión. El hueco que queda abierto: la API acepta un `.rs`, el worker no sabe compilarlo, y el publish explota a mitad de la validación.

Las ocho reglas quedaron: hay lenguajes; existe el pool `solutions`; ningún pool vacío; ningún pool dimensiona un lenguaje no declarado; toda entrada dimensionada tiene `cpu` y `memoryBytes` positivo; todo lenguaje tiene imagen; y todo lenguaje con `runCmd` tiene extensión, está dimensionado por `solutions`, y tiene los cuatro campos de artefacto con `{name}`.

La regla del `cpu` no vacío importa más de lo que parece: un `cpu` vacío se parsea como *sin límite*, y eso rompe en silencio el supuesto de 1 CPU por container sobre el que se apoya la fórmula de concurrencia de D5.

**7. `validateJudgeConfig` y `applyJudgeConfigDefaults` extraídas de `loadJudgeConfig`.** Cierra el pendiente de este documento sobre partir esa función: mientras hacía `os.Exit`, ni la validación ni el defaulting se podían testear, y por eso `config_test.go` reimplementaba las reglas a mano. Ahora el test llama a la función real. **Y apareció un default que faltaba**: `idleTimeoutMinutes` no tenía piso, así que borrar esa clave del YAML dejaba el timeout en `0` y el reaper habría destruido cada container ocioso apenas se liberaba, pagando la creación en cada judging, sin que nada avisara.

**8. El empaquetado de Java, verificado corriéndolo** en `judge-runner:java17`, no deducido: `jar` **sí** viene en `openjdk-17-jdk-headless` (17.0.19); `-C /sandbox .` **sí** mete el `.java` dentro del JAR; `javac -d /sandbox/classes` crea el directorio solo y aísla los `.class`, con lo que el JAR queda limpio; y un checker multi-clase se empaqueta y corre bien.

**9. Los helpers compartidos salieron a un archivo propio.** `buildTar` y `extractFirstFile` vivían dentro de `session.go` porque hasta ahora los usaba solo `Session`. Con un segundo consumidor, Ad8 pide un archivo nombrado por el concepto que agrupan: `docker_exec.go`, junto a `docker_exec_client.go`. `truncateString` no era una operación de Docker sino una utilidad genérica, así que fue a `pkg/strutil` — y al moverla apareció un bug: `s[:maxBytes]` **parte un carácter UTF-8 al medio**, así que un log de compilación con acentos quedaba inválido en el corte. Corregido, con test.

**10. El sandbox se limpia antes de devolver el container al pool**, y si la limpieza falla el container **se destruye** en vez de devolverse. No es prolijidad: ese container vuelve al pool de soluciones, donde lo va a reclamar código de un concursante, que podría leer `/sandbox/Checker.cpp` y deducir la respuesta. La limpieza corre con su propio deadline, para que una compilación que agotó el suyo igual se limpie.

#### Tests

Cada test se verificó **rompiendo lo que prueba**: 15 mutaciones sobre la validación de config y sus defaults, 7 sobre el adapter, 2 sobre la clave de GCS. Cada una pone en rojo exactamente el test que le corresponde y ninguno más. Dos tests estaban mal escritos y los agarró justamente esa verificación: uno borraba el único lenguaje del pool y hacía saltar antes otra regla, y otro usaba un lenguaje que el pool tampoco conocía, con lo que el error podía venir del pool y no de la guarda que quería probar.

Los tests del validator runner nativo por lenguaje (`validator_runner_{cpp,java,python}_test.go`) se borraron junto con el compilador nativo: usaban `NewNativeCompiler()` para fabricar su artefacto. Sobrevive `validator_runner_test.go`, así que ese adapter conserva test hasta que el Paso 4 lo borre entero. Se descartó sostenerlos con un compilador de juguete en el archivo de test: trabajo que se tira en el paso siguiente.

**Nota para el Paso 9**: el roadmap anota que "los tests de compilación en Java se saltean por falta de JDK en el entorno de este equipo" y que con la compilación en container pasarían a poder correrse. Es cierto a medias — el test del adapter mockea el cliente de Docker, así que no ejecuta `javac` de verdad. Lo que sí se puede correr contra las imágenes reales, y se hizo a mano en este paso, es la verificación del punto 8.
### Paso 4 — El validator al sandbox (D3, D4, D13)

Puertos `ValidatorRunner`/`ValidatorSession` con forma de sesión, y su adapter sobre pool B. **Acá se construye el segundo pool** y se agrega la validación del invariante al arrancar. `PrepareJudgingUseCase` abre la sesión antes del loop de inputs. Se borra el `validator_runner.go` nativo.

### Paso 5 — El checker al sandbox (D3)

Puertos `OutputChecker`/`CheckerSession`, se reescribe `output_comparator.go` sobre pool B, y `ValidateSolutionsUseCase` y `JudgeSubmissionUseCase` abren la sesión antes de sus loops. Se borran `artifact_invocation.go` y `judging_timeouts.go`.

**La comparación por tokens se queda en el worker por ahora**: moverla sin el volumen del paso 7 empeoraría la CPU del worker en vez de mejorarla, porque habría que empujar los bytes hacia adentro del container.

Acá **desaparece por construcción** el bug de `CheckerLanguage`/`CheckerFilename`. Y hay que escribir el test que hoy no existe: uno que cruce la frontera caso de uso → adapter, porque los actuales mockean justo el punto donde se perdían los campos.

### Paso 6 — Límites de memoria reales (D11)

`docker update --memory` al reclamar el container, y decidir qué reportar como KB consumidos. Independiente del resto, se puede mover de lugar sin romper nada.

**Y acá hay que resolver el MLE de Java**, que está en la sección de bugs: la JVM nunca deja que el cgroup la mate por heap, así que el `exitCodeMLE = 137` es inalcanzable y todo MLE de Java se reporta hoy como runtime error. La vía del exit code por lenguaje se evaluó y **se descartó**; hace falta pensar otra o refinarla. Una pista: si acá el límite pasa a aplicarse con `docker update --memory`, quizás la señal deba salir del propio cgroup (`memory.events`) en vez del código de salida, lo que además sería uniforme para los tres lenguajes.

### Paso 7 — Volumen compartido y comparación por tokens a pool B (D7, D6)

El `emptyDir` en `worker.yaml` montado en `dind` y en `worker`, el UUID por judging con la raíz no listable, `RunTestCase` escribiendo al volumen, y `copyOutput` eliminado. `CheckRequest` pasa de recibir bytes a recibir una ruta, y la comparación por tokens se muda a pool B con la imagen del paso 1.

**Es el paso más difícil de verificar localmente** — los tests del pool mockean Docker, así que la prueba real cae en la Fase 8 contra el cluster. Por eso va último: si algo queda mal, no contamina los pasos anteriores.

### Paso 8 — Barrido y cierre

El barrido de código muerto de la sección siguiente, actualizar el `JUDGE_VALIDATION_ROADMAP.md` (la Fase 5 se cae casi entera y la Fase 6 se reescribe), y decidir si los dos sobrevivientes de la Fase 5 —el `BackendConfig` del Ingress y el runbook para recrear la cola— entran en este PR o van aparte.

### Paso 9 — Releer el roadmap y recuperar lo que quedó en el camino

El descubrimiento del bug nos desvió a mitad de la Fase 5, así que quedaron cosas pendientes de las fases finales que nunca se retomaron. **Releer `JUDGE_VALIDATION_ROADMAP.md` entero** buscándolas, porque están dispersas entre el cuerpo de las fases y los apéndices, no juntas en una lista.

Lo que ya identifiqué al revisarlo, para no partir de cero:

**Notas que quedaron desactualizadas**
- La Fase 4 dice *"Pendiente derivado — actualizar el spec"* con el endpoint `GET /problems/p/{slug}/validation`. **Ya se hizo** en esta sesión; hay que marcar la nota como cerrada.

**Cosas que el rediseño habilita y antes no se podían hacer**
- La Fase 6 anotó que los tests de compilación en Java *"se saltean por falta de JDK en el entorno de este equipo"*. Con la compilación dentro de un container eso deja de aplicar: **esos tests pasan a poder correrse**, vía Docker como el resto.

**Cosas que el rediseño NO arregla y siguen pendientes**
- El apéndice del mecanismo de corrección en vivo describe una **Pieza 0** (compilar a una key candidata y "promover" solo si todo el pipeline pasa). El rediseño cambia *dónde* se compila, pero `prepareChecker` sigue escribiendo directo sobre la key en vivo antes de saber si las soluciones aprueban. **Ese bug sobrevive intacto** — conviene confirmarlo y decidir si entra acá o queda para ese PR.
- Los dos ítems del apéndice de deuda técnica (las factorías privadas de `submission/status.go` contra D4, y `NewSubmission` sin validar `problemID`/`userID`).
- Los dos del apéndice de documentación en `CLAUDE.md` (la sección de "Endpoints pendientes" desactualizada y el `MOCK_AUTH` que no existe en el código).

**Cosas que hay que reescribir por el rediseño**
- El checklist de la **Fase 8** se escribió para la arquitectura vieja. Sigue valiendo en lo esencial, pero ahora hay que sumar la verificación de lo nuevo: que el checker corre efectivamente aislado, que el volumen compartido funciona en el cluster, y que un checker Java multi-clase anda.

---

## Antes de abrir el PR: barrido de código muerto

Este rediseño borra un camino de ejecución entero (el nativo) y reemplaza otro, así que es terreno fértil para dejar restos. **Antes de abrir el PR hay que revisar todo el código que queda buscando lo que dejó de usarse.**

**Ojo con esto**: el compilador de Go avisa de imports y variables locales sin usar, pero **no** de funciones, tipos, campos ni constantes privadas que quedaron huérfanas. Se compila y pasa los tests igual. Y el proyecto **no tiene linter configurado** (no hay `.golangci.yml` ni nada en el workflow), así que el barrido no sale solo.

Conviene correr una pasada de `staticcheck ./...` o `golangci-lint run --enable unused` —vía Docker, igual que los tests— aunque no se agregue al CI en este PR.

**Candidatos concretos que hay que verificar uno por uno:**

*Borrados: confirmar que nada más los referencia*
- `artifactInvocation` y sus tests
- `isTimeoutErr` y `trustedSubprocessTimeout` (`judging_timeouts.go` entero)
- `native_compiler.go` + `_cpp/_java/_python` y sus tests
- `validator_runner.go` (versión nativa) y sus tests

*Probablemente huérfanos tras los cambios*
- `tokenCompare` en `output_comparator.go` — la lógica se muda a `cmd/compare`, así que la copia en Go del adapter queda sin uso
- `copyOutput`, y `maxOutputBytes` si deja de tener llamadores (la salida pasa por el volumen)
- `CheckerFilename` en `ProblemLimits` y en `CheckRequest` — lo elimina D9
- `dbCheckerJSON.Filename` en `problem_provider.go`, si el filename ya no hace falta en tiempo de ejecución
- `ValidatorRunRequest.Artifact`, que pasa a ser una ruta
- Mocks y helpers de test de los puertos borrados (`mockNativeCompiler`, `mockValidatorRunner` y compañía en los `mocks_test.go`)

*Config*
- Restos de la forma vieja de `languages` tras la reestructuración de D10
- Referencias a `cpuOverheadCores` después del renombre a `dockerDaemonReserveCores`

*Y el barrido general*
- Campos de structs de request/result que quedaron sin escribir o sin leer
- Constructores y variables de `cmd/worker/main.go` que ya no se enchufan a nada
- Entradas de `judge_config.yaml` que ningún código lee
