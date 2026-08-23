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

### D3 — Los puertos de pool liviano pasan a forma de sesión, como interfaces separadas ✅

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

**Son interfaces separadas, no una compartida ni una extensión de `ExecutionSession`.** Pool pesado y pool liviano hacen cosas distintas (compilar y correr soluciones vs. ejecutar un artefacto ya compilado con argumentos de archivo), y meterle a la sesión de soluciones métodos que solo sirven para checkers rompería los puertos angostos que el proyecto ya usa: `JudgeSubmissionUseCase` cargaría con métodos que no llama y cada mock tendría que implementarlos. La reutilización ocurre en el **adapter** (helpers privados `buildTar`, `extractFirstFile`, el patrón `ExecCreate`/`ExecAttach`/`ExecInspect`), no en el puerto.

**Se descartó una tercera vía**: una sesión genérica de bajo nivel (`PutFiles` / `Exec(cmd)` / `GetFile`) sobre la que ambos casos de uso construyan. Empujaría la mecánica de containers hacia arriba — la capa de aplicación armando comandos de shell. Hoy el saber "cómo se compila en C++" vive en el adapter y en `judge_config.yaml`, que es donde corresponde.

**Los dos reciben una ruta de GCS, no bytes.** El primer borrador daba bytes al validator razonando que `PrepareJudgingUseCase` acababa de compilarlo y lo tenía en memoria. Es cierto pero irrelevante: el artefacto del validator **también queda guardado** (`problems/{slug}/validator/compiled`, con su clave persistida vía `SetValidatorCompiledKey`). Recibir una ruta habilita gratis un caso de uso futuro —cambiar los casos de prueba y re-validarlos contra el validator ya compilado, sin recompilar— y de paso **verifica el round-trip**: si la subida se corrompió o la clave quedó mal, se detecta en el publish y no meses después en medio de una competencia.

**Toca la capa de aplicación**: `ValidateSolutionsUseCase`, `JudgeSubmissionUseCase` y `PrepareJudgingUseCase` abren la sesión antes del loop.

El puerto `NativeCompiler` conserva su firma tal cual; solo cambia el adapter (de `exec.Command` al pool pesado).

### D4 — Dos pools: el pesado compila y corre soluciones, el liviano solo ejecuta checkers ✅

- **Pool pesado** — containers grandes. Compila **todo** (checkers, validators y soluciones) y ejecuta las soluciones.
- **Pool liviano** — containers chicos. **Solo ejecuta** checkers y validators ya compilados. Nunca compila.

En `judge_config.yaml` y en el código se llaman `heavy` y `light` (ver el Paso 4 para por qué esos nombres y no `solutions`/`checkers`).

La clave que hace que esto funcione es D1: como el artefacto queda guardado en GCS al publicar, en el judging de una submission real el checker ya viene compilado — al liviano solo se le inyecta el binario y se lo ejecuta. Por eso puede dimensionarse para **ejecución pura**, que era exactamente el beneficio que motivaba separarlo. La objeción de "hay que sobredimensionar el liviano para que aguante compilar" desaparece.

| Operación | Pool | Cuándo |
|---|---|---|
| Compilar checker/validator | **A** | solo al publicar |
| Compilar solución | **A** | publish y cada judging |
| Correr solución | **A** | publish y cada judging |
| Correr validator | **B** | solo al publicar |
| Correr checker | **B** | publish y **cada submission, siempre** |

**Los dos pools son necesarios, no solo prolijos — evitan un deadlock.** Con la forma de sesión de D3, un judging tiene **dos containers tomados a la vez**: el de la solución (abierto durante todo el loop de casos) y el de checker (también abierto durante todo el loop, para no re-descargar el binario en cada caso). Si los dos salieran del mismo pool, con capacidad C y `maxConcurrent = C`, los C judgings tomarían su container de solución y después todos pedirían uno de checker; no queda ninguno y nadie puede avanzar ni liberar. Con pools separados, cada judging pide uno de cada uno y nunca compiten entre sí.

**Corrección sobre "compilar en el mismo container que corre la solución"**: no siempre se puede, porque **cada imagen de lenguaje tiene un solo toolchain** (`cpp20.Dockerfile` instala únicamente `g++`, `java17` únicamente el JDK, `python310` únicamente `python3`). Si el checker es C++ y la solución es Java, el container pesado de la solución no tiene `g++`. Así que la compilación del checker es su propio `Claim` de pool pesado, con el lenguaje **del checker**. Con D1 eso ocurre una sola vez por publish, así que el costo es despreciable.

**Flujo resultante:**

*Al publicar*
1. `Claim` en el pesado (lenguaje del checker) → compilar → extraer binario → subir a GCS → `Release`
2. Ídem para el validator
3. `Claim` en el liviano (lenguaje del validator) → inyectar → correr contra cada input → `Release`
4. Por cada solución: `Claim` en el pesado (lenguaje de la solución) → compilar y correr; y `Claim` en el liviano para chequear cada salida

*Al juzgar una submission*
1. `Claim` en el pesado (lenguaje de la submission) → compilar y correr
2. `Claim` en el liviano (lenguaje del checker) → inyectar el binario precompilado → chequear cada salida

---

### D5 — Dimensionamiento: misma CPU en ambos pools, distinta memoria, fórmula sin cambios ✅

| | CPU | Memoria |
|---|---|---|
| Pool pesado | `cpu: "1"` | 2 GiB |
| Pool liviano | `cpu: "1"` | 512 MiB de base, más para Java (la JVM) |

Los tamaños son **por lenguaje dentro de cada pool** — ver la estructura de config en D10 y los números concretos en D13.

#### Qué significa cada valor (hacen cosas distintas)

**`MemoryBytes` cumple doble función:**
1. **Techo real del container** (`Resources.Memory`, con `MemorySwap` igual para desactivar swap). Es lo que hace cumplir el kernel: al pasarse, SIGKILL → exit 137 → así se detecta MLE.
2. **Costo contable en el pool** (`allocatedBytes + MemoryBytes <= budget`): cuánto presupuesto gasta un container de ese lenguaje.

Un límite **no es un permiso para usar recursos, es un techo**. Un container capado en 2 GiB que usa 50 MB consume 50 MB de RAM real — pero el pool lo *contabiliza* como 2 GiB, porque debe asumir el peor caso.

**`CPU` (`NanoCPUs`) es solo un techo de tasa** — "como máximo N segundos de CPU por segundo". No reserva nada, y **no participa de la admisión** (`canCreate` solo mira memoria), así que no afecta cuántos containers entran.

#### Por qué la memoria es distinta y la CPU es igual

**Memoria distinta**: en pool pesado el número es el techo donde una solución recibe MLE, así que tiene que ser al menos tan grande como el mayor `memoryLimit` que un problema pueda declarar — no es un número libre. En pool liviano, 512 MiB le sobran a un checker que lee tres archivos y compara tokens. Igualarlo a 2 GiB no le daría capacidad útil: gastaría presupuesto y haría entrar menos containers en total, con lo que el LRU evictaría más seguido — justo lo contrario de lo que se busca al mantener containers calientes.

**CPU igual**: el cap de 1 CPU en pool pesado existe para que el **veredicto sea justo**, no para proteger la infraestructura. Como se mide tiempo de CPU acumulado, una solución multi-hilo sin techo consumiría 4 segundos de CPU por segundo de reloj en un nodo de 4 cores y llegaría al TLE cuatro veces más rápido; con el techo en 1, lanzar hilos no ayuda ni perjudica y los tiempos son comparables entre submissions.

En pool liviano no hay veredicto que proteger (el tiempo del checker no se compara contra nada), pero el techo **sirve como contención**: un checker con un bucle infinito no puede pasar de un core. Y como los dos containers se alternan, igualarlos no altera la cuenta:

```
max(cpuA, cpuB) = max(1, 1) = 1 CPU por judging
```

Se descartaron dos alternativas: `0.5` (número arbitrario, y frena el arranque de la JVM en checkers Java, donde cada invocación es un `docker exec` nuevo) y *sin límite* (mantiene la cuenta igual de limpia pero pierde la contención).

#### La asimetría memoria/CPU

Dentro de **un mismo judging**, el pesado y el liviano **nunca computan a la vez**:

```
solución corre caso 1    → pesado ocupado, liviano ocioso
checker compara salida 1 → liviano ocupado, pesado ocioso
```

No es casualidad estadística: es una **dependencia de datos** — la entrada del checker *es* la salida de la solución. De ahí que **la memoria se sume** (ambos containers están tomados todo el judging, aunque uno esté ocioso) pero **la CPU no** (solo uno computa por vez, y un techo no es una reserva).

#### La fórmula del semáforo queda igual

```
maxConcurrent = dindCPU/1000 − dockerDaemonReserveCores     ← sin cambios
```

Calcula "cuántos cores quedan para judging" y los cuenta como judgings, asumiendo **1 CPU por judging**. Con pool liviano agregado la demanda pico por judging sigue siendo 1 CPU, así que lo que garantiza —una solución corriendo por CPU— se cumple igual.

**Supuesto a documentar en el código**, porque pasa a ser carga estructural y hoy es implícito: la fórmula solo vale mientras los containers de pool pesado estén capados en `cpu: "1"`. Si alguien pusiera un lenguaje en 2 CPU, la fórmula mentiría sin que nada avise.

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

### D6 — La comparación por tokens también corre en pool liviano, con imagen y binario propios ✅

**Nada de comparar en el worker.** El caso sin checker personalizado es la mayoría del tráfico; si se quedara en el worker, el volumen compartido (D7) no serviría para casi nada y el worker seguiría siendo el cuello de botella de CPU que motivó todo esto.

**Imagen propia y mínima.** Una comparación por tokens no tiene lenguaje —no hay checker—, así que no puede salir de una imagen de lenguaje. Se agrega una cuarta imagen, chica, dedicada a este caso. Arranca rápido y ocupa poca memoria, así que se pueden mantener varias calientes, que importa ahora que pool liviano es camino caliente. Se descartó fijar un lenguaje (p. ej. usar siempre `cpp20`) porque reservaría un container con un toolchain entero para comparar texto.

**Binario propio, no comando de shell.** Replica exactamente el `tokenCompare` actual, así que el comportamiento no cambia respecto de lo que hoy está en producción. Un `tr` + `cmp` no habría que construirlo, pero es fácil que difiera en casos borde (espacios iniciales, líneas vacías, `\r\n`).

**Encaja en lo que ya existe**: `cmd/compare/` en el mismo módulo Go, con su `docker/judge/compare.Dockerfile` construido con el mismo contexto que las otras tres (`build-judge-images.sh` ya usa el directorio del backend como contexto).

**Y el pool no necesita ningún caso especial**: pool liviano se indexa por lenguaje igual que pool pesado, y el caso sin checker es una entrada más en ese mapa (`compare` → imagen mínima). `BeginChecking` con ruta vacía reclama la "lengua" `compare`.

**Consecuencia sobre el dimensionamiento**: pool liviano deja de ser ocasional. Lo usan **todas** las submissions, no solo las de problemas con checker personalizado. Su tamaño importa tanto como el de pool pesado.

**Pendiente**: agregar la cuarta imagen al init container `prepull-language-images` (hoy recorre solo `cpp20 java17 python310`) y al pipeline de build/push.

---

### D7 — Volumen compartido con directorio por judging: la salida del concursante nunca pasa por el worker ✅

**El problema que resuelve**: hoy `copyOutput` saca la salida del container por la API de Docker y la deja en memoria del worker como `RunResult.Output []byte` — hasta **64 MiB por caso de prueba** (`maxOutputBytes = 64 << 20`). El worker tiene `limits: cpu 500m`, elegidos para un worker que solo coordina. Mover decenas de MiB por caso, decodificar tar y comparar es trabajo que no le corresponde.

**Por qué no alcanzaba con mandar solo la comparación a un container**: los bytes ya están en el worker cuando se compara. Empujarlos *hacia adentro* de un container significa empaquetarlos en tar y transferirlos por la API — **más** CPU del worker, no menos. La única forma real de sacarle la carga es que esos bytes **nunca salgan del sandbox**.

#### El mecanismo

1. **Un `emptyDir` nuevo montado en los dos containers** (`dind` y `worker`) en la misma ruta. Hace falta en ambos porque el demonio Docker corre *dentro* de `dind` y resuelve las rutas origen de los bind mounts en **su** filesystem, no en el del pod; para que el worker vea esos mismos archivos, el directorio tiene que existir en los dos.

2. **Todos los containers montan la misma raíz compartida** — montaje uniforme, sin bind mounts distintos por container.

3. **El worker genera un UUID por judging**, crea `<raíz>/<uuid>/` y les pasa esa ruta a los containers que participan. Pool pesado escribe ahí su salida; pool liviano lee de ahí. Cada uno alcanza únicamente el directorio cuya ruta recibió.

4. **La raíz no puede listarse.** Acá se sostiene todo el aislamiento: en un directorio, `r` habilita *listar* y `x` habilita *atravesar*. La raíz queda dueño `root` con modo `0711` — el usuario `judge` (uid 1000, fijo en la imagen base) puede entrar a una ruta que conoce, pero `ls` no le devuelve nada. Cada directorio de UUID sí accesible para ese usuario.

5. **La salida esperada NUNCA va al volumen compartido.** Es lo único verdaderamente secreto: si el código del concursante pudiera leerla, le bastaría imprimirla para pasar todo. La salida esperada y el artefacto del checker viajan del worker al container de pool liviano directamente.

6. **Limpia el worker**: un `rm -rf <raíz>/<uuid>` al terminar el judging, sobre una ruta que ya conoce.

#### Por qué así y no con un directorio por container

La primera versión montaba un subdirectorio distinto en cada container, buscando aislamiento por namespace de montaje. Se descartó por tres razones:

- **No cerraba el lado de pool liviano**: como el emparejamiento es dinámico (cualquier container liviano con cualquiera pesado), el liviano habría tenido que montar el árbol entero del pesado, dejando a un checker malicioso leer las salidas de todos los judgings. Con el UUID por judging y la raíz no listable, el liviano tampoco puede enumerar: solo alcanza el UUID que le pasaron.
- **El ciclo de vida no coincidía**: los containers se reusan entre judgings, así que un directorio por container arrastraría la salida del judging anterior y habría que limpiarlo aparte. Un directorio por UUID nace y muere con el judging.
- **Era más invasivo**: obligaba al pool a armar un bind mount distinto al crear cada container.

**Tampoco alcanzaba con permisos POSIX del estilo "pool pesado escribe pero no lee"**: un directorio en modo `-wx` impide listar pero **no** impide abrir un archivo cuya ruta conozcas, y como todos los containers de pool pesado corren con el mismo uid, los permisos de archivo tampoco los separan entre sí.

#### De qué depende la seguridad

Es el mismo principio que una URL-capacidad: **la ruta es la credencial**. Se sostiene sobre dos cosas, y las dos hay que cuidarlas:

- La raíz no listable (modo `0711`, dueño `root`).
- Los UUID no pueden filtrarse — son 122 bits de entropía, imposibles de adivinar, pero no deben terminar en logs visibles al usuario ni en mensajes de error devueltos por la API.

#### Lo que se gana

- **`copyOutput` desaparece del worker**: de 64 MiB por caso de prueba a través de la API de Docker, a leer ~2 KB del filesystem para el preview de wrong answer (`Actual: truncatePreview(...)`).
- Con esto más D6 (comparación en pool liviano), el worker vuelve a ser lo que su config ya decía: solo coordinación. Los `500m` dejan de ser un problema.

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

Hoy la sección `languages:` mezcla dos cosas independientes: **propiedades del lenguaje** (imagen, extensión, comandos) y **dimensionamiento** (`cpu`, `memoryBytes`). Con dos pools eso deja de funcionar, porque el mismo lenguaje tiene tamaños distintos según el pool. Y los campos de artefacto de D9 **se reparten entre los pools**: `artifactSource`/`artifactCompile`/`artifactPath` se usan al compilar (pool pesado) y `artifactRun` al ejecutar (pool liviano).

```yaml
languages:                    # propiedades del LENGUAJE, sin importar el pool
  cpp20:
    image: "judge-runner:cpp20"
    extension: "cpp"
    compileCmd: "..."         # soluciones
    runCmd: "..."             # soluciones
    artifactSource:  "..."    # checker: usado en pool pesado
    artifactCompile: "..."    # checker: usado en pool pesado
    artifactPath:    "..."    # checker: usado en pool pesado
    artifactRun:     "..."    # checker: usado en pool liviano
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

**Los tamaños del ejemplo son ilustrativos.** El de `java17` en pool liviano va más alto porque una JVM en 512 MiB es ajustada, pero eso hay que medirlo (ver A3).

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

> **Corregido al preparar el Paso 6: en cgroup v2 el número no está contaminado, no existe.** Verificado corriendo el mismo decode del código (`container.StatsResponse`, mismo cliente moby) contra un container real en un Docker con `CgroupVersion: 2`: `MaxUsage = 0`, y el JSON crudo de la API **no trae la clave `max_usage`** — es v1-only. La vía de cgroup v2 es `/sys/fs/cgroup/memory.peak`, que sí funciona y no está expuesta por la API de Docker.
>
> Como el cluster es cgroup v2 (`docker:27-dind` sobre nodos COS de GKE), **hoy todo veredicto reporta 0 KB de memoria consumida**. La asimetría con el tiempo es peor de lo que este documento decía: la CPU se mide bien (`total_usage` viene poblado) y la memoria directamente no se mide.

**Por eso el arreglo obvio del bug 1 es peligroso**: copiar lo que se hace con el tiempo (`if runResult.MemoryKb > limits.MemoryKb`) compararía contra un número que no describe esta corrida — contaminado en cgroup v1, y cero en v2.

**El arreglo**: `docker update --memory=<límite del problema>` al reclamar el container para un judging. El MLE lo hace cumplir el kernel al valor correcto y se detecta con el mecanismo que ya existe (SIGKILL → exit 137 → `exitCodeMLE`).

- Usa la detección que ya está, sin agregar comparaciones sobre un número contaminado.
- La contabilidad del pool queda **conservadora**: se reservaron 2 GiB de presupuesto y se usa menos.
- Bajar el límite de un container en marcha es seguro acá: se hace al reclamarlo, con `/sandbox` recién limpiado y sin procesos corriendo. (Docker puede rechazar una bajada si el uso actual ya la supera.)

**Se descartó `ulimit -v`**, que es lo primero que uno prueba: limita el espacio de direcciones **virtual**, no el residente, y una JVM reserva muchísimo virtual sin usarlo — mataría todo Java.

**Pendiente de definir**: qué reportar como KB consumidos. Con la corrección de arriba la pregunta cambia de forma: no es *"elegir entre un número aproximado y el límite"*, es **construir una medición que hoy no existe**.

La fuente natural en cgroup v2 es `/sys/fs/cgroup/memory.peak`, leído **dentro** del container con un `docker exec` (verificado: se lee bien). Pero **no se puede resetear entre corridas**, por dos motivos independientes que hay que tener presentes al diseñarlo:

- El reset de `memory.peak` (escribirle cualquier valor) llegó en **Linux 6.11**; los nodos COS de GKE están por debajo.
- Docker monta `/sys/fs/cgroup` **de solo lectura** dentro del container, así que aunque el kernel lo soportara, el `docker exec` no podría escribirlo.

O sea que `memory.peak` sigue siendo monotónico desde que arrancó el container, y como los containers se reusan, hereda el mismo problema de contaminación que `MaxUsage` tenía en v1. Aislar una corrida necesita otra vía —medir el pico del **proceso** en lugar del cgroup (`getrusage`/`ru_maxrss` del hijo, que es lo que hacen los jueces clásicos)— y eso implica algo en la imagen o en el comando de ejecución. Queda para el Paso 6.

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

El objetivo razonable es que quepan los **lenguajes principales** con holgura. Sobre el nodo actual (10 GiB en el dind, menos 0.5 GiB de reserva del demonio = **9.5 GiB a repartir**, `maxConcurrent` = 2):

| | Mínimo obligatorio | Con los 3 lenguajes calientes |
|---|---|---|
| Pool pesado | 2 × 2 GiB = 4 GiB | 2 + 2 + 1 = **5 GiB** |
| Pool liviano | 2 × 1 GiB = 2 GiB | 0.5 + 1 + 0.5 + 0.125 = **2.125 GiB** |
| **Total** | 6 GiB | **7.125 GiB ≤ 9.5** ✓ |

**Corregido al ejecutar el Paso 4**: la versión anterior de esta tabla sumaba la reserva del demonio (0.5 GiB) *además* de los 9.5 GiB, que ya la tenían descontada, y daba 7.6. La conclusión no cambia.

> **Los números de esta tabla quedaron obsoletos con D14.** El nodo pasó a `e2-standard-8`, el dind a 24 GiB, `maxConcurrent` a 5 y el dimensionamiento a N3 (esta tabla usa N2). Se dejan porque el razonamiento —el invariante es lo obligatorio, los containers calientes son comodidad— sigue valiendo tal cual; **los valores vigentes están en D14**.

El `e2-standard-4` actual cumple el invariante con margen y además mantiene calientes los tres lenguajes en ambos pools. Crecer la máquina sirve solo para subir `maxConcurrent` y procesar más submissions en paralelo: es una decisión de throughput para la competencia, separada de este rediseño.

#### Medición del pool liviano (hecha en el Paso 4)

Reemplaza el "dato que falta medir" que decía este documento. Todo medido contra las imágenes reales del proyecto (`judge-runner:{cpp20,java17,python310}`), con el container capado igual que lo capa el pool (`--memory` = `--memory-swap`, `--cpus=1`, `--network=none`) y leyendo `/sys/fs/cgroup/memory.peak`.

**Ejecutar un validator** — input de 1.3 MB con 200 000 enteros, el validator recorriéndolos y verificando rangos:

| | pico | tamaño propuesto en D13 |
|---|---|---|
| cpp20 | 5 MiB | 512 MiB |
| python310 | 25 MiB | 512 MiB |
| java17 | 21 MiB | 1 GiB |

La JVM arrancó y terminó bien **incluso con el container en 32 MiB**: detecta el cgroup y ajusta su heap. O sea que para el validator los tamaños de D13 sobran por un factor de ~20.

**Ejecutar un checker** — el mismo algoritmo en los tres lenguajes (leer las dos salidas enteras, partir en tokens, comparar), que es lo que llega en el Paso 5:

| salida del concursante | cpp20 @ 512 MiB | python310 @ 512 MiB | java17 @ 1 GiB |
|---|---|---|---|
| 8 MiB | 119 MiB ✓ | 162 MiB ✓ | 88 MiB ✓ |
| 16 MiB | 178 MiB ✓ | 320 MiB ✓ | 154 MiB ✓ |
| 64 MiB | **OOM, exit 137** | **OOM, exit 137** | **OOM, exit 1** |

Los picos incluyen las dos salidas en el page cache del cgroup (~2× su tamaño), así que no son todos memoria del proceso.

**La conclusión que cambia el diseño**: el dimensionamiento del pool liviano **está acoplado a `maxOutputBytes`**, y hasta acá el documento los trataba como independientes. Con los 64 MiB de hoy, ningún tamaño razonable de pool liviano alcanza. Con los ~8 MiB que este mismo documento propone para el Paso 7, los tres sobran cómodos. **El Paso 5 no puede fijar los tamaños del pool liviano sin decidir antes `maxOutputBytes`.**

Los números del YAML **no se tocaron en el Paso 4**: lo único que el pool liviano ejecuta hoy es el validator, donde sobran; subirlos habría gastado presupuesto sin consumidor y la decisión correcta depende de `maxOutputBytes`.

**Y confirma el bug de MLE de Java en un lugar nuevo** (ver la sección de bugs): quedarse sin memoria da 137 en C++ y Python, pero **exit 1** en Java. Un checker Java que agota la memoria no se distingue hoy de un checker que rechaza la salida.

**Sigue sin medirse**: el pico de `g++` compilando un checker con testlib y el de `javac`, o sea el lado del pool pesado. No se pudo porque `testlib.h` no está en la imagen de C++ (ver la sección de bugs), así que no hay forma de compilar el caso que motiva la estimación.


---


---

### D14 — Dimensionamiento del nodo, el pod y los pools ✅

Cierra las dos precondiciones que el Paso 6 exigía resolver antes de escribir código, y de paso el reparto entero de la infraestructura. Todo lo de abajo está medido o derivado del código, salvo donde se dice explícitamente.

#### La máquina: `e2-standard-8`

Sube de `e2-standard-4`. **El motivo es la concurrencia, no la memoria**: `maxConcurrent` sale de `limits.cpu` del dind menos la reserva del demonio, y en la máquina chica eran 2. Con la nueva son **5**.

**El costo no fue el criterio, y conviene dejar escrito por qué**: el `judge-pool` tiene `--num-nodes 0 --min-nodes 0` y KEDA `minReplicaCount: 0`, así que el nodo del judge **solo existe mientras hay cola**. La cadena es: cola vacía → KEDA baja el worker a 0 (`cooldownPeriod: 300`) → nodo sin pods → el cluster autoscaler lo elimina (~10 min de gracia). El delta sobre `e2-standard-4` es de **~US$0.67 por competencia de 5 horas**.

El único escenario donde eso se rompe es que la cola nunca se vacíe —`activationValue: "0"` mantiene el worker en ≥1 con cualquier mensaje—, y ahí un mensaje envenenado que se reencola para siempre deja el nodo prendido 24/7 (~US$196/mes). Vale un chequeo en la Fase 8.

**Se descartó quedarse en `e2-standard-4`**: es viable —con N1 hasta sobra memoria— pero deja `maxConcurrent` en 2 y la CPU del pod al filo (3.5 de ~3.5 utilizables). Con 40 personas enviando a la vez, la cola es lo que se nota, y no se compra con memoria.

**Migración**: GKE no permite cambiar el machine type de un node pool existente. Hay que crear uno nuevo con el mismo taint, cambiar el `nodeSelector` del worker y borrar el viejo. Con 0 nodos y sin datos es reversible en minutos.

#### `requests == limits`: QoS Guaranteed, en los tres containers

Antes: dind `requests 2/8Gi` contra `limits 3/10Gi`, worker `250m/256Mi` contra `500m/512Mi`. Tres razones para igualarlos:

1. **El judge se dimensiona contra `limits`, pero el nodo reserva `requests`.** `POD_MEMORY_LIMIT` y `POD_CPU_LIMIT` leen `limits.*` del dind vía Downward API, así que `validatePoolBudgets` y `maxConcurrent` operaban sobre memoria y cores que el scheduler nunca garantizó. Funcionaba por una razón accidental: el node pool está taintado y corre un solo pod, así que no había con quién competir.
2. **QoS.** Con la brecha el pod es *Burstable*, de los primeros en ser desalojado bajo presión de memoria del nodo — con judgings en vuelo adentro.
3. **En un node pool dedicado la brecha no le sirve a nadie**: el argumento normal a favor de `requests < limits` es prestarle el sobrante a otros pods, y el taint garantiza que no hay otros.

**El `prepull-language-images` también necesita `resources`.** Kubernetes clasifica la QoS mirando **todos** los containers, init containers incluidos: sin bloque `resources` el pod queda Burstable por más que dind y worker estén igualados. Se le ponen `200m / 256Mi`, generosos para lo que hace — el `docker pull` lo ejecuta el **demonio** dentro del dind, y el container del CLI solo transporta órdenes y progreso.

**Y no le cuesta nada al pod.** El pedido efectivo de un pod con sidecars es `máx(para cada init normal: su pedido + Σ sidecars ya arrancados, Σ sidecars + Σ containers normales)`. Como `dind` es sidecar nativo (`restartPolicy: Always`) y el prepull corre antes de que el worker arranque, la franja del prepull es 6.2 CPU / 25.25 GiB, por debajo del estado estable de 7 CPU / 27 GiB. El prepull usa la franja del worker, prestada.

#### El reparto completo

Del nodo crudo a lo que queda para los pools:

```
e2-standard-8                                     8.00 CPU   32.00 GiB
  reserva del kubelet (fórmula de GKE)           −0.09      −3.66
─────────────────────────────────────────────────────────────────────
ASIGNABLE                                         7.91      28.34
  DaemonSets (kube-proxy, fluentbit, métricas,
  pdcsi, netd) — ESTIMADO, ver pendientes        −~0.40     −~0.45
─────────────────────────────────────────────────────────────────────
DISPONIBLE PARA EL POD                           ~7.50     ~27.90 GiB

  dind        6.00 CPU / 24.00 GiB
  worker      1.00 CPU /  2.00 GiB
  prepull     (franja prestada del worker)
─────────────────────────────────────────────────────────────────────
POD                                               7.00      26.00 GiB
MARGEN                                           ~0.50      ~1.90 GiB
```

Adentro de los 24 GiB del dind:

| | GiB | bytes |
|---|---|---|
| `pools.heavy.budgetBytes` | 16.75 | 17985175552 |
| `pools.light.budgetBytes` | 6.25 | 6710886400 |
| `dockerDaemonReserveBytes` | 1.00 | 1073741824 |
| **suma = `dind.limits.memory`** | **24.00** | **25769803776** |

> **Estos números los recalculó A9 del Paso 6**, dos veces: el dind bajó de 25 a 22 GiB al descubrir que los 2.75 GiB de `java17` no servían para nada, y volvió a 24 al introducir el `memoryFactor`, que sí necesita que el container supere el límite del problema. El margen del pod quedó en **1.9 GiB**, y es lo único que protege contra la única fila del desglose que sigue estimada — los DaemonSets.

`dockerDaemonReserveBytes` sube de 512 MiB a 1 GiB: los 512 se dimensionaron para 2 judgings concurrentes y ahora son 5, con más churn de containers y más metadata en el demonio.

#### `maxMemoryLimitGlobal` se queda en 2048 MB

Se evaluó bajarlo a 1024 (la propuesta que este documento traía anotada) y **se descartó**.

**El argumento en contra que el documento pedía verificar, verificado**: los 12 paquetes BOCA de `packages_contest` —los que la Fase 9 va a convertir e importar— declaran **1024 MB los doce**, y además el mismo valor para los tres lenguajes (`limits/cpp`, `limits/java`, `limits/c` son idénticos). O sea que 1024 alcanzaría exacto para la competencia real. Pero también significa que **el tope no es lo que aprieta**: ningún problema real se acerca a 2048.

Lo que decidió: con el dimensionamiento de arriba, **2048 no le cuesta nada** — el pool pesado entra en N3 dentro de su presupuesto. Bajar a 1024 liberaría ~9 GiB que **no se pueden convertir en nada**: `maxConcurrent` lo limita la CPU, no la memoria, así que sería presupuesto muerto. Y deja el techo de la plataforma donde DOMjudge lo tiene por default.

**Se descartó también hacer `maxMemoryLimit` por lenguaje**, que era la inconsistencia anotada (`maxTimeLimit` sí lo es: 300s/400s/600s). Dos razones: `platform_settings.go` exige `maxMemoryLimit <= maxMemoryLimitGlobal`, así que un valor por lenguaje solo puede **apretar**, nunca aflojar — no sirve para "darle más a Python", que era el caso que lo motivaba, y para eso ya están los `languageOverrides` por problema. Y el dato real: los 12 paquetes no diferencian por lenguaje.

**Bajarlo después no rompe nada**: los tres call sites que validan (`create_problem`, `update_problem`, `import_problem`) lo hacen al escribir; la lectura usa `RestoreMemoryLimit`, sin validar. No hay migración.

#### Tamaños por lenguaje, y el `memoryFactor`

| Pool pesado | GiB | Por qué |
|---|---|---|
| `cpp20` | 2.00 | = el tope de problema, con `memoryFactor: 1.0` |
| `java17` | **2.375** | 2048 × 1.19: cubre el `memoryFactor: 1.15` con aire |
| `python310` | **2.00** | **corrige el bug**: tenía 1 GiB contra los 2048 que la plataforma promete |

**El techo del pool es lo máximo que un `Claim` puede pedir**, y con el `memoryFactor` de A9 un `Claim` pide `límite del problema × factor`. Para C++ y Python el factor es 1.0, así que su techo es exactamente el tope de problema. Para Java es 1.15, así que su techo tiene que ser mayor: `2048 × 1.15 = 2355 MiB`, redondeado a 2.375 GiB.

Ese techo se alcanza **en pleno** solo en dos casos: un problema que **no declara** límite de memoria (`memory_limit` es nullable, y entonces `Claim` usa `LanguageCeiling`) y la **compilación** de artefactos. Para lo segundo hay medición: `javac` sobre un checker multi-clase de 200 líneas con generics usa **60-111 MiB**, o sea que sobra por un factor de 20.

El pool liviano no se toca (`cpp20` 0.5 / `java17` 1 / `python310` 0.5): corre checkers y validators, a los que el `memoryLimit` del problema nunca se les aplica, así que `maxMemoryLimitGlobal` no lo mueve.

#### El nivel al que se dimensiona: N3

El documento tenía un solo nivel escrito (el invariante de D13). Al dimensionar aparecieron cuatro, y conviene nombrarlos porque la diferencia entre ellos son varios GiB:

| | Fórmula | Qué garantiza |
|---|---|---|
| **N1** | `maxConcurrent × mayor` | el invariante de D13: nadie se cuelga. **Lo único obligatorio** |
| **N2** | `Σ(todos los lenguajes)` | un container ocioso de cada lenguaje |
| **N3** | `(maxConcurrent − 1) × mayor + Σ(todos)` | un caliente de cada lenguaje **y** los judgings concurrentes en el más pesado |
| **N4** | `maxConcurrent × Σ(todos)` | nunca evicta; por encima, ninguna memoria extra es alcanzable |

Con `maxConcurrent = 5`, **N2 queda por debajo de N1** (6.75 contra 13.75 en el pool pesado): a concurrencia alta el invariante ya cubre "un caliente por lenguaje" de sobra. La elección real es entre N1 y N3.

```
Pesado  N3 = (5−1) × 2.375 + (2 + 2.375 + 2)  = 9.50 + 6.375 = 15.875 GiB
Liviano N3 = (5−1) × 1    + (0.5 + 1 + 0.5) =  4 + 2.00 =  6.00 GiB
                                     (6.125 cuando `compare` llegue en el Paso 7)
```

**Se eligió N3 sabiendo que N1 alcanzaría.** Lo que se compra son los 200-500 ms de crear un container, sobre un judging que dura segundos — el propio D13 dice que evictar *"es el comportamiento normal y esperado, no una falla"*. Se eligió igual porque el margen deja ver el comportamiento bajo estrés en vez de descubrirlo en competencia, y porque con la máquina nueva sale gratis. **Si hiciera falta memoria para otra cosa, bajar a N1 libera 6 GiB sin romper ningún invariante (N1 pesado = 5 × 2.375 = 11.875 GiB, liviano = 5 × 1 = 5).**

#### El worker: 1 core / 2 GiB

Sube de `500m / 512Mi`. Los dos números se eligieron cuando el worker solo coordinaba, y desde la Fase 6 descomprime, empaqueta y compara.

**La memoria** — medido con 5 judgings concurrentes en Go 1.25, simulando el mundo post-Paso 7 (entrada por el volumen, sin duplicación de tar), reportando `runtime.MemStats.Sys`:

| Datos de prueba por problema | `Sys` |
|---|---|
| 50 MB | 525 MB |
| **100 MB** | **1006 MB** |
| 200 MB (el tope viejo) | 1614 MB |

Con 2 GiB y el tope nuevo de 100 MB queda al **52%**. La medición no incluye la línea de base real (pgx, cliente de GCS, AMQP), así que son cotas inferiores.

**La CPU** — el término dominante es descomprimir el ZIP de casos de prueba, medido a ~120 MB/s con un core:

| | 5 judgings de 100 MB a la vez |
|---|---|
| `500m` | **10.2 s** |
| `1 core` | **5.1 s** |
| 2 cores | 2.6 s |

Son segundos de latencia muerta antes de que el judging empiece. **Se descartó 2 cores**: consumiría todo el margen del nodo, y los DaemonSets corren con `system-node-critical` — no se quedarían afuera, **preemptarían al pod del judge**.

Dato que explica por qué `500m` frena tanto: **Go 1.25 nunca baja `GOMAXPROCS` de 2** aunque el cgroup diga 0.5. Con dos hilos contra medio core, el runtime quema su cuota en ~25 ms de cada período de 100 y queda frenado el resto.

**La CPU del worker no le cuesta nada a `maxConcurrent`**: son containers separados con cgroups separados, y la fórmula se deriva de `limits.cpu` **del dind**.

#### El tope de datos de prueba baja de 200 a 100 MB

`maxFileSizeTestCaseMB` en `config/virtual_object.json`, que acota el total descomprimido del ZIP. **Con 200 MB el worker no aguantaba ni un solo judging** (ver la sección de bugs). Los 100 MB dejan 3.5× de aire sobre el problema real más pesado, que son 28.8 MB.

#### Alternativas descartadas, todas medidas

- **`GOMEMLIMIT`** (decisión explícita del usuario: no por ahora). Go no lee el límite del cgroup por su cuenta, así que hoy el GC deja crecer el heap al doble de los datos vivos y **se hace matar en vez de recolectar**. Medido: con 100 MB × 5, `GOMEMLIMIT=800MiB` baja el consumo de 1005 a 797 MB. Con 200 MB × 5 **no puede honrarlo** (1614 → 1082): los datos vivos son 1000 MB y el GC no recolecta lo que sigue referenciado. Sirve donde el problema es el colchón del GC, no donde son los datos vivos.
- **Streaming de casos de prueba** (`GetTestCases` devolviendo un iterador en vez de un slice). Convierte `O(suma de casos)` en `O(caso mayor)`, pero **con estos datos no compra casi nada**: los paquetes BOCA tienen 2-3 casos y en los pesados **un solo archivo es el 99% del peso** (28.75 de 28.8 MB en el problema B). Sirve para problemas con muchos casos chicos; la palanca real acá es un tope **por archivo**, que no existe.
- **N1 en vez de N3** — ver arriba.
- **`e2-standard-4`** — ver arriba.
## Detalles a resolver al implementar

Ninguno bloquea el diseño; son elecciones que conviene hacer con el código delante.

- ~~**Si la entrada del caso de prueba viaja por el volumen compartido** o sigue por la API.~~ **Resuelto: va por el volumen.** El razonamiento (no es secreta, y es el 82% del pico de memoria del worker) está en el Paso 7.
- **Limpieza del directorio del judging**: hoy `Session.Close` hace `rm -rf /sandbox/*`; hay que sumar el borrado de `<raíz>/<uuid>` por parte del worker.
- ~~**`maxOutputBytes`**: hoy 64 MiB.~~ **Resuelto en el Paso 5**: bajó a 8 MiB. Con D7 además deja de presionar al worker, pero sigue acotando cuánto disco del `emptyDir` puede consumir un judging.
- **Los otros dos archivos que lee el checker no están acotados por `maxOutputBytes`** (encontrado en el Paso 5, sin resolver). Bajar ese límite acota la salida **del concursante**, pero el checker recibe tres archivos, y el input y la salida esperada salen de `parseTestCasesZip` (`internal/adapter/judge/test_case_provider.go`), que tiene **su propio tope, independiente y quemado como literal**: `io.ReadAll(io.LimitReader(rc, 64*1024*1024))`. O sea que un problem setter que sube un `.ans` de 60 MiB hace OOM al container del pool liviano igual, y el dimensionamiento de D13 no lo cubre. Es un camino mucho más raro que el del concursante —lo controla el setter, y el publish lo agarraría antes que una competencia— pero el número está donde nadie lo va a ver. Al encararlo hay que decidir **dos cosas distintas**: cuál es el tope por archivo de caso de prueba, y si vive junto a `maxOutputBytes` en vez de suelto en el parser del ZIP. Ojo con la asimetría al elegirlo: el pico de memoria del checker escala con la **suma** de los tres archivos, no con el mayor.

  > **Corregido al medirlo en el commit 3**: eso vale para un checker **personalizado**, que lee los archivos enteros. Para `compare` —el camino por defecto, o sea la mayoría del tráfico— es falso: streamea, nunca abre el input, y su memoria sigue al **token más grande** y no al tamaño de los archivos. Ver el punto 14 del Paso 7.

  **Y toca también al worker, no solo al container.** El Paso 5 cambia dónde viven esos bytes: hoy `customCheckerCompare` escribe los tres archivos a un directorio temporal **en disco**, y el adapter nuevo los empaqueta en un **tar en memoria** (`buildTar` arma un `bytes.Buffer`) para copiarlos por la API de Docker. Sumado a que `GetTestCases` ya trae **todos** los casos de prueba a memoria de una vez, el worker —que tiene `limits.memory: 512Mi`— queda expuesto a datos que solo el problem setter acota. El Paso 7 se lleva la salida del concursante al volumen compartido, pero **no** el input ni la salida esperada, así que este ítem le sobrevive.
- ~~**Qué reportar como KB consumidos**~~ **Diseñado y verificado; ejecuta en el Paso 7** (ver ahí la sección de A5). El veredicto de MLE queda correcto porque lo hace cumplir el kernel, pero el número que se muestra sale de `MaxUsage`, que **en cgroup v2 viene en cero**. La medición correcta es el `ru_maxrss` del proceso, con `/usr/bin/time` por fuera del `timeout`.
- **Un checker roto se reporta como wrong answer del concursante** (encontrado en el Paso 5, decidido dejarlo así). El adapter trata **cualquier** exit distinto de cero como rechazo, con el stderr como mensaje — igual que `ValidatorSession.Validate` desde el Paso 4. Eso mete dos casos ajenos en la misma bolsa:

  - **`exit 3` es `_fail` en testlib**: el checker declarando que *él* o los datos del jurado están rotos, no que la salida del concursante esté mal. La tabla completa es `0=_ok`, `1=_wa`, `2=_pe`, `3=_fail`, `7=_points`.
  - **El checker agotando la memoria del container del pool liviano**: sale con 137 en C++/Python y con **1** en Java (medido en el Paso 4, ver D13), indistinguible de un rechazo legítimo.

  Se decidió no introducir la tabla de testlib en el Paso 5 por tres razones: dejaría al checker con un criterio distinto del validator, que se acaba de cerrar en el Paso 4; **acoplaría el sistema a testlib de verdad**, cuando hoy solo lo permite —un checker que es un `main` común y corriente es válido, y un `return 3` suyo pasaría a significar algo que su autor no quiso decir—; y sobre todo **no arregla el segundo caso**, porque en Java el OOM da exit 1 y ninguna tabla de exit codes lo distingue.

  **Dónde encararlo: en el Paso 6**, que es donde se toca memoria, y con el mecanismo que ese paso ya necesita. La pista que este documento deja para el MLE de las soluciones sirve igual acá: si la señal de "se quedó sin memoria" sale del cgroup (`memory.events` / `memory.max` leídos después de la corrida) en vez del código de salida, cubre los tres lenguajes de forma uniforme y separa el OOM del rechazo **sin** tocar la interpretación del exit code. Recién con eso resuelto tiene sentido discutir si además se distingue el `3` de testlib, que es la parte chica del problema.
- ~~**Verificar que `jar` venga en `openjdk-17-jdk-headless`** y acotar el `-C /sandbox .` a los `.class`.~~ **Resuelto en el Paso 3**, verificado corriéndolo en la imagen real: `jar` viene incluido, y el `-C /sandbox .` efectivamente metía el `.java` en el JAR. Se compila con `-d /sandbox/classes` y se empaqueta desde ahí.
- **Un checker Java con la clase no pública falla tarde y disfrazado.** Verificado en `judge-runner:java17`: si la clase no es `public`, javac acepta cualquier nombre de archivo y `jar` no verifica que la main-class exista, así que la compilación pasa y revienta al ejecutar con `Could not find or load main class Checker`. Se decidió (Paso 3) no validar nada al subir, porque el mensaje nombra la clase esperada y una regex sobre fuente Java es código que se pudre. Si el caso aparece en la práctica, el lugar para atajarlo es el Paso 5, con el checker corriendo de verdad.

- **Documentar la convención de nombres de clase.** Con el Paso 3, un archivo Java debe declarar `Solution`, `Checker` o `Validator` según su rol. `README.md:133` y `specs/Judge System/README.md:161` ya documentaban `Solution.java` con mayúscula, así que la deriva contra el código se cierra sola — pero falta escribir en algún lado la convención completa, incluida la salida de la clase no pública para poder subir varias soluciones Java (ver la sección de bugs).
- **Nombre e imagen del binario de comparación** (`cmd/compare`, `docker/judge/compare.Dockerfile`), y sumarlo al init container `prepull-language-images` y al pipeline de build/push (ver D6).
- ~~**Falta una validación de arranque, sacada a propósito**: el caso inverso — un lenguaje que puede recibir soluciones y que ningún pool dimensiona.~~ **Resuelto en el Paso 3** con la alternativa que había quedado pendiente: se deriva de `runCmd`, sin tocar el dominio. Ahí está también por qué se descartó el chequeo cruzado contra `virtual_object.json`, y qué hueco deja abierto.

  Se implementó y **se revirtió**: la versión que se escribió preguntaba `submission.NewLanguage(lang)` para saber si un lenguaje era enviable, y eso metía dominio dentro de `cmd/worker` — que quedaba como el único cmd importándolo (`cmd/api` no importa dominio en absoluto) — además de usar un constructor como predicado, que no es para lo que existe.

  La alternativa que quedó pendiente de evaluar: derivarlo de la propia config, sin dominio. Un lenguaje en el que se escriben soluciones es el que declara `runCmd`, y `compare` (que llega en el Paso 4) no va a tener uno — solo `image` y `artifactRun`. Así la exención sale sola de la forma del archivo. Su punto débil: si alguien agrega un lenguaje al dominio pero olvida el `runCmd` en el YAML, el chequeo no salta.
- ~~**Partir `loadJudgeConfig`, que hoy hace tres cosas**~~ **Resuelto en el Paso 3**: `validateJudgeConfig` y `applyJudgeConfigDefaults` son funciones propias que devuelven error, y los tests las llaman directo en vez de reimplementar las reglas. Queda pendiente **mover las tres a `internal/config/judge_config.go`**, anotado más abajo con el resto de la revisión de configuración.

- **El wiring del pool depende de una invariante que no expresa**: `Image: judgeCfg.Judge.Languages[lang].Image` es seguro *solo porque* la validación de arranque ya garantizó que ese lenguaje existe. El Paso 3 lo acotó agregando una regla de que todo lenguaje declare imagen no vacía, así que el peor caso ya no es un nombre de imagen vacío; pero **el acceso al mapa sigue ignorando el `ok`**, y sigue dependiendo de que nadie reordene ni quite esa validación.
 El Paso 4 lo movió a `poolConfigFor`, así que ahora está en un solo lugar en vez de inline — arreglarlo cuesta un `ok` y un error, no una pasada por dos bloques.

- **Nada impide mal-cablear un adapter al pool equivocado** (encontrado en el Paso 4). Los dos pools tienen el mismo tipo `*pool.Pool` y dimensionan los mismos lenguajes, así que `NewValidatorRunner(heavyPool, ...)` compila, pasa los tests y funciona con poca carga. Recién produciría el **deadlock que D4 describe** con `maxConcurrent` operaciones simultáneas, en medio de una competencia. Ni el compilador ni la validación de arranque lo detectan. Se descartó darle un tipo distinto a cada pool: es sobre-ingeniería para tres líneas del composition root. Queda cubierto solo por nombres de variable inequívocos (`heavyPool` / `lightPool`).

- **Los presupuestos explícitos no siguen a un cambio de tamaño del dind** (decidido en el Paso 4). `budgetBytes` es absoluto en el YAML y `POD_MEMORY_LIMIT` viene de la Downward API, así que si alguien sube `limits.memory` del dind en `worker.yaml`, la memoria nueva queda sin usar y la validación no dice nada: `suma <= límite` se sigue cumpliendo. Se eligió lo explícito igual, porque usar de menos degrada y usar de más rompe, y porque la alternativa —derivar el presupuesto de un pool y darle el resto al otro— vuelve invisible el reparto. Si el tamaño del dind empieza a cambiar seguido, el arreglo es que la validación también avise cuando **sobra** presupuesto sin asignar.

- **Tres cosas menores de `judge_config.yaml`**, sin decidir:
  - La sección `pools` usa notación inline (`{ cpu: "1", ... }`) mientras el resto del archivo es bloque. Uniformar, o asumir el inline como convención para los mapas chicos de dimensionamiento.
  - ~~El comentario que explica los nombres anteriores (`memoryOverheadBytes`/`cpuOverheadCores`) es ruido a futuro.~~ **Resuelto en el Paso 3**: los comentarios del archivo se reescribieron en inglés y en una línea, y esa explicación se fue al historial de git y a este documento.
  - `memoryBytes: 2147483648` son bytes crudos, ilegibles sin el comentario de al lado. Kubernetes acepta `2Gi` en sus manifests; acá no, porque el parser es un `int64` pelado.

- **Revisión propia de `RUNNER_ARCHITECTURE.md`**: al actualizarlo por el renombre aparecieron discrepancias **preexistentes** que no se tocaron, porque merecen su propia pasada. `syntaxCheckCmd: "pypy3 -m py_compile ..."` es un campo que **no existe en el código** y además nombra `pypy3` en vez de `python3`; y `compileCmd: "javac ... /sandbox/Solution.java"` usa mayúscula contra el `solution.java` real. Sugiere que ese documento describe un diseño previsto que la implementación no siguió en varios puntos — el `-Xmx{memoryLimit}m` (ver la sección de bugs) es el caso más consecuente.

---


#### Pendientes que abrió el dimensionamiento de D14, sin paso asignado

- **Medir los DaemonSets antes de aplicar los números de D14.** Es la única fila del desglose que quedó estimada (~0.40 CPU / ~0.45 GiB), y de ella dependen los 0.5 CPU y 0.9 GiB de margen del pod. El orden correcto es: crear el node pool nuevo → dejar que levante un nodo → `kubectl describe node <nodo>` → leer la sección **`Allocated resources`** (que muestra lo ya pedido, con los DaemonSets arriba y el pod del judge todavía no) → ajustar `dind.limits.memory` contra el número real. Si sale más alto que lo estimado, se recorta del dind; si sale más bajo, el excedente natural es también el dind (el worker queda al 52% con el tope de 100 MB).

  **Y el margen no es capacidad ociosa**: los DaemonSets del sistema corren con `priorityClassName: system-node-critical`, así que si no entran no se quedan afuera — **preemptan al pod del judge**. El modo de falla de pedir de más no es "no arranca", es "arranca y lo matan en medio de una competencia".

  **Se decidió medirlo al desplegar, no antes.** Medir en la máquina destino **es** la migración: GKE no deja cambiar el machine type, así que tener un nodo `e2-standard-8` exige crear el node pool nuevo. Y el margen que D14 deja son 0.9 GiB contra ~0.45 estimados, o sea 2×: para que falle, los DaemonSets tendrían que pedir el doble de lo típico. El modo de falla además es benigno y ruidoso — el pod queda `Pending` y `kubectl describe pod` lo dice con todas las letras; el arreglo es bajar el `25Gi` del dind.

- **`github.com/rabbitmq/amqp091-go` está marcado `// indirect` en `go.mod` y no lo es** — `internal/adapter/queue/rabbitmq_queue.go` lo importa directamente. Apareció al correr los tests con `-mod=mod`, que lo movió solo; se revirtió por no ser de esta tarea. Un `go mod tidy` lo corrige. Sin efecto funcional: es exactitud del manifiesto de dependencias.

- **Un tope por archivo individual de caso de prueba**, separado del total. Hoy solo existe el total (`maxFileSizeTestCaseMB`, 100 MB tras D14), aplicado además como tope por archivo, así que **un solo archivo puede ocupar los 100 MB completos**. Eso importa por dos motivos distintos que conviene no mezclar:

  - **Para el worker**, es lo que hace que el streaming de casos de prueba no sirva: convertir `O(suma)` en `O(caso mayor)` no ayuda si un archivo es el 99% del peso, que es exactamente la forma de los paquetes BOCA (28.75 de 28.8 MB en el problema B).
  - **Para el container del pool liviano**, es el ítem que ya estaba anotado más arriba: el pico de un checker **personalizado** escala con la **suma** de los tres archivos que recibe, y `maxOutputBytes` solo acota uno de ellos. Para `compare` no aplica (ver el punto 14 del Paso 7).

  Un valor razonable con los datos reales sería ~32 MB (cubre los 28.75 con margen), pero hay que decidirlo mirando los dos consumidores a la vez, y decidir también **dónde vive** — hoy el tope del parser está quemado como literal (`io.ReadAll(io.LimitReader(rc, 64*1024*1024))` en `test_case_provider.go`), desincronizado de la config.

- **Cache de casos de prueba por problema.** Con `maxConcurrent = 5`, cinco submissions al mismo problema **descargan y descomprimen el mismo ZIP cinco veces**, cada una con su copia en memoria. En una competencia las submissions se agolpan sobre unos pocos problemas, así que es trabajo repetido casi en su totalidad, y ataca los dos costos a la vez:

  - **CPU**: medido, descomprimir 100 MB cuesta ~825 ms con un core; cinco a la vez tardan 5.1 s en llegar al primer caso de prueba.
  - **Memoria**: cinco copias de los mismos datos es exactamente el término que domina el consumo del worker.

  Lo que hay que resolver al encararlo: **la invalidación**. Un problem setter puede cambiar los casos de prueba, y el dominio ya tiene el campo para detectarlo (`judgingUpdatedAt`, que `SetTestCases` toca y que los tres casos de uso de rejudge ya usan). Y el canje: un cache sostiene memoria entre judgings a cambio de bajar los picos, así que su tamaño hay que dimensionarlo igual que todo lo demás — un LRU de 2-3 problemas alcanzaría para una competencia.

- **El comentario de doc de `validatePoolBudgets` quedó huérfano sobre `poolConfigFor`** (`cmd/worker/main.go`). `poolConfigFor` tiene su propio comentario justo debajo, y `validatePoolBudgets` quedó sin ninguno unas líneas más abajo. Cosmético; arreglarlo cuando se toque el archivo.

- **`CLAUDE.md` describe el consumidor del worker como serial** (*"consumidor concurrente con semáforo (hoy es serial)"*) y **ya es concurrente** desde antes de este rediseño: `Consume` usa `sem := make(chan struct{}, maxConcurrent)` con goroutines y `Qos(maxConcurrent)` (`internal/adapter/queue/rabbitmq_queue.go`). Es el mismo tipo de deriva documentación-código que el Paso 3 cerró con `Solution.java`. Va junto con los otros dos ítems de `CLAUDE.md` que el roadmap ya anota.

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
| `adapter/judge/native_compiler.go` + `_cpp/_java/_python` + tests | la compilación pasa al sandbox (pool pesado) |
| `adapter/judge/validator_runner.go` + tests | pasa al sandbox (pool liviano) con forma de sesión |
| `adapter/judge/judging_timeouts.go` (`isTimeoutErr`) | era para `exec.CommandContext` nativo |
| `adapter/judge/artifact_invocation.go` + tests | la lógica por lenguaje se va a `judge_config.yaml` (D8/D9) |

**Se reescribe:**

| Archivo | Motivo |
|---|---|
| `adapter/judge/output_comparator.go` | pasa al pool con forma de sesión; la comparación por tokens también (D6). **Hecho en el Paso 5**: quedó partido en `output_checker.go` (la factory, renombrada por Ad1), `checker_session.go` (la sesión de sandbox) y `token_comparison.go` (lo que sigue en el worker hasta el Paso 7) |
| `adapter/judge/pool/` | segunda instancia con presupuesto propio; bind mount del volumen compartido (D7) |
| `adapter/judge/session.go` | salida al volumen en vez de `copyOutput`; `docker update --memory` al reclamar (D11) |
| `adapter/judge/config.go` + `judge_config.yaml` | separar `languages` de `pools`, campos de artefacto (D9/D10) |

**Se construye nuevo:**

- Los puertos de sesión de pool liviano y sus adapters (D3).
- `cmd/compare/` y `docker/judge/compare.Dockerfile` — el binario de comparación por tokens y su imagen mínima (D6).
- El `emptyDir` compartido en `worker.yaml`, montado en `dind` y en `worker` (D7).
- La validación al arrancar del invariante de dimensionamiento (D13).

**Se renombra**: `NativeCompiler` → `ArtifactCompiler` (deja de ser nativo, y su comentario de doc pasa a ser falso).

**Mejoras laterales que salen de paso**: los checkers Java multi-clase pasan a funcionar (el artefacto es un JAR, D9), y desaparecen las 25 descargas del mismo binario desde GCS por submission (D3).

---

## Bugs encontrados en el camino

Todos son preexistentes y ninguno lo introdujo este rediseño, pero no todos se arreglan dentro de él: los dos de memoria sí (D11), y el de `CheckerFilename` desaparece por construcción (D8/D9). El MLE de Java y la colisión de soluciones Java quedan **sin arreglar**, cada uno con su nota.


### El MLE de Java no se puede detectar — ✅ RESUELTO en el Paso 6 (A4)

Encontrado al revisar por qué C++ y Java tenían el mismo `memoryBytes`. Son **dos** bugs; el primero ya está arreglado, el segundo **no tiene todavía una solución aceptada**.

**Bug ya corregido — Java recibía la cuarta parte de la memoria.** El `runCmd` era `java -cp /sandbox solution`, sin ninguna opción de memoria. Una JVM moderna detecta el cgroup y aplica `MaxRAMPercentage=25` por defecto, así que **en un container de 2 GiB el heap máximo era 512 MiB** (medido con la imagen real del proyecto), contra ~2 GiB para una solución en C++ con el mismo `memoryBytes`. Un problema con límite de 1 GB: la solución en C++ pasaba y la equivalente en Java moría. Se agregó `-XX:MaxRAMPercentage=75` al `runCmd` (medido: 1536 MiB). Se eligió 75 y no 90 pensando en D11: cuando el container se achique al límite real del problema, un 90% de 256 MB dejaría 26 MB para el overhead de la JVM y la mataría el cgroup.


**Hallazgo que reencuadra este bug**: `RUNNER_ARCHITECTURE.md` especificaba, en su ejemplo de configuración de lenguajes, `runCmd: "java -Xmx{memoryLimit}m Solution"` — o sea, **el diseño original sí contemplaba pasarle a la JVM el límite de memoria del problema**, con una plantilla por problema. La implementación quedó en `java -cp /sandbox solution`, sin ninguna opción de memoria. No fue un descuido de diseño: el diseño lo tenía bien y la implementación lo dejó caer.

Eso conecta directo con D11: `{memoryLimit}` implica que el `runCmd` de Java **no puede ser un string estático**, necesita sustitución por problema. Al resolver el Paso 6 conviene mirar esa especificación original antes de inventar otra.

**Otras discrepancias del mismo bloque, preexistentes y sin corregir** (se dejaron para una revisión propia de `RUNNER_ARCHITECTURE.md`, no son de este rediseño): `syntaxCheckCmd: "pypy3 -m py_compile ..."` es un campo que **no existe en el código** y además nombra `pypy3` en vez de `python3`; y `compileCmd: "javac ... /sandbox/Solution.java"` usa mayúscula contra el `solution.java` real.
**Bug resuelto en el Paso 6 — el veredicto de MLE en Java era imposible con el mecanismo de entonces.**

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
**Resuelto en el Paso 6 (A4), y ninguna de las dos vías que este bloque anticipaba resultó ser la buena.** La tabla de arriba, además, resultó **incompleta**: `MaxRAMPercentage=100` da 137 —no 1— cuando el programa asigna en trozos en vez de pedir un único array gigante, así que el exit code de la JVM depende del **patrón de asignación del concursante** y ninguna regla derivada del kernel puede unificarlo. La pista que este documento dejaba (`memory.events`) tampoco: con el flag adoptado la JVM se mata desde el espacio de usuario y el contador del kernel queda en cero.

Lo que se adoptó es `-XX:OnOutOfMemoryError=kill -9 %p`, que hace que la JVM produzca **el mismo 137** que el cgroup produce para los otros dos lenguajes, sin tabla de exit codes por lenguaje y sin tocar Go. El detalle completo, las mediciones y los caveats están en el Paso 6, punto 7.

### El exit 137 es ambiguo: OOM o SIGKILL tras un SIGTERM ignorado — preexistente, sin arreglar

Encontrado al cerrar A7, verificando que `timeout` no produjera 137 por su cuenta. Sí lo produce:

| Situación | exit |
|---|---|
| El OOM killer del cgroup mata al proceso | **137** |
| El proceso **ignora SIGTERM** y `timeout --kill-after=1s` recurre a SIGKILL | **137** |
| El proceso simplemente tarda de más y muere con SIGTERM | 124 |

O sea que una solución que instale un manejador de SIGTERM y no muera recibe **`MEMORY_LIMIT_EXCEEDED` en vez de `TIME_LIMIT_EXCEEDED`**. Es preexistente: la constante `exitCodeMLE = 137` siempre tuvo esta ambigüedad, y A4 no la introduce ni la agrava.

**Cuán real es**: medido que la JVM sale con SIGTERM dentro del segundo de gracia en todos los casos, incluso con el GC bajo presión al 90% del container, igual que un binario de C++. Así que solo lo dispara código que instale un manejador **a propósito** — algo que no pasa en programación competitiva, y que además no le sirve a nadie: los dos veredictos son fallos.

**Por qué no se arregló acá**: distinguirlos requeriría una señal extra, y la única disponible —el `oom_kill` del cgroup— **no sirve para Java**, porque con el flag de A4 la JVM se mata desde el espacio de usuario y el contador no se mueve (ver el Paso 6, punto 9). O sea que el arreglo cubriría C++ y Python y dejaría a Java exactamente igual de ambiguo. Si alguna vez se encara, la vía sería `-XX:+ExitOnOutOfMemoryError` (exit 3) para Java, aceptando la tabla de exit codes por lenguaje que A4 evitó.
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

### Los argumentos del checker van cruzados respecto de testlib — arreglado en el Paso 5

Encontrado al decidir si `CheckRequest` conserva `Input`. El adapter invoca al checker con `(input, salida esperada, salida del concursante)`, siguiendo al pie de la letra el spec del proyecto (`specs/Judge System/README.md:319`, `./checker input.txt expected.txt actual.txt`). **testlib liga los dos últimos al revés**:

| argv | testlib | qué es |
|---|---|---|
| `argv[1]` | `inf` | input del caso |
| `argv[2]` | `ouf` | salida del **concursante** |
| `argv[3]` | `ans` | respuesta del **jurado** |

No es cosmético, porque testlib trata los dos archivos con semánticas distintas a propósito: un `readInt()` mal formado en `ouf` es un *wrong answer* limpio (culpa del concursante), y el mismo `readInt()` mal formado en `ans` es un `_fail` — testlib asume por definición que los datos del jurado son correctos, así que si no lo son el problema está roto. Cruzados, un concursante que imprime basura hace que el checker declare que el jurado está mal. Y para el caso que justifica tener checker personalizado —problemas con múltiples respuestas válidas— el checker termina validando la respuesta del jurado como si fuera la del concursante.

**Se corrigió en el Paso 5**, que reescribe esa misma línea de invocación, y **se actualizó el spec** en el mismo commit para que no quede documentación contradiciendo al código — la misma deriva que el Paso 3 tuvo que cerrar con `Solution.java`. Sin riesgo sobre datos productivos: por D12 todo el pipeline es de esta rama y nunca se mergeó.

### `compare` quedó con el orden que el Paso 5 invirtió — ✅ RESUELTO en el Paso 0 del Paso 7

Encontrado al cargar el contexto del Paso 7, y es el mismo bug de arriba visto desde el otro archivo. `cmd/compare/main.go` se escribió en el **Paso 1**, cuando la invocación todavía era `(input, salida esperada, salida del concursante)`, y leía `args[1]` como la respuesta del jurado y `args[2]` como la del concursante. El **Paso 5** invirtió los dos últimos para seguir a testlib, pero `compare` no se tocó porque **nadie lo invoca todavía** — su primer llamador llega recién con la mudanza de la comparación por tokens al pool liviano.

Su comentario de cabecera afirmaba que tomaba *"the same three file arguments as a custom checker (input, expected, contestant)"*: cierto cuando se escribió, falso dos pasos después.

**Inofensivo hoy, y por eso peligroso.** La comparación por tokens es simétrica, así que intercambiar los dos archivos no cambia el veredicto ni el mensaje: cableado tal cual habría funcionado, con los tests en verde y el nombre equivocado adentro. Se rompe el día que `compare` se vuelva asimétrico, que es justo la mejora que su propio código anticipa (*"surfacing a checker's message is the natural thing to add later"*) y que su test de filtración ya custodia — sobre el archivo equivocado.

**Se arregló antes de empezar el Paso 7 y no dentro de él**, para que no desapareciera dentro de un diff de una docena de archivos. Se arregla `compare` y no la invocación: la invocación sigue a testlib, que es el estándar y que el Paso 5 fijó a propósito.

**Lo que lo hace verificable no es el veredicto sino el mensaje de error**, que nombra *qué* archivo no se pudo leer. Sobre esa asimetría se escribió el test que fija el orden; el veredicto, por simétrico, no puede fijarlo. Tres controles corrigieron el camino hasta ahí, y el tercero es el que vale: la primera versión del test usaba `strings.Contains(msg, "contestant")` y **pasaba con el bug puesto**, porque `t.TempDir()` nombra el directorio temporal con el nombre del subtest y la palabra buscada aparecía en la ruta del archivo faltante. Un test que pasa por el motivo equivocado. Se cerró con `HasPrefix`.


### El pool pesado le da a Python la mitad de la memoria que la plataforma promete — ✅ RESUELTO en D14

Encontrado en el Paso 5, al revisar de dónde salían los tamaños del YAML. **Arreglado en D14**: `python310` pasa a 2 GiB en `pools.heavy`. Se deja el diagnóstico completo abajo porque explica por qué el techo del pool y `maxMemoryLimitGlobal` son el mismo número visto desde dos archivos.

`config/judge_config.yaml` dimensiona el pool pesado así: `cpp20` 2 GiB, `java17` 2 GiB, **`python310` 1 GiB**. Los números vienen de la config ilustrativa de D10 —que el propio documento marcaba como ilustrativa— y pasaron al YAML en el Paso 2 sin que nadie volviera sobre el de Python.

**Por qué está mal.** D5 dice qué significa `memoryBytes` en el pool pesado: *"el número es el techo donde una solución recibe MLE, así que tiene que ser al menos tan grande como el mayor `memoryLimit` que un problema pueda declarar — no es un número libre"*. Y `config/virtual_object.json` declara `maxMemoryLimitGlobal: 2048` (MB) y, explícitamente, `{ "language": "python310", "maxMemoryLimit": 2048 }`.

O sea: un problema con `memoryLimit = 2048 MB`, una solución en Python que usa 1.5 GB → el cgroup la mata en 1 GiB → exit 137 → **MEMORY_LIMIT_EXCEEDED**, mientras la equivalente en C++ pasa. Es la misma clase de bug que el de `MaxRAMPercentage=25` que ya está en esta sección, con la config en el lugar de la JVM.

**Y el Paso 6 no lo rescata: lo empeora si no se corrige primero.** D11 aplica el límite real con `docker update --memory=<límite del problema>` al reclamar, y justifica su seguridad con *"la contabilidad del pool queda conservadora: se reservaron 2 GiB y se usa menos"*. Ese razonamiento **solo vale si el techo del pool es ≥ cualquier límite de problema**. Con Python en 1 GiB, `docker update` tendría que **subirlo**: el pool contabilizaría 1 GiB para un container que puede usar 2, y con `maxConcurrent` judgings simultáneos el dind se queda sin memoria. Sobreventa invisible.

**El arreglo del número es una línea y entra sin tocar nada más**: `python310` a 2 GiB deja el pool pesado caliente en `2+2+2 = 6 GiB ≤ 6.5` de presupuesto, y el invariante de D13 en `2 × 2 = 4 GiB ≤ 6.5`. Los presupuestos totales no se mueven.

**Lo que no es de una línea es la guarda.** Nada lo detecta al arrancar: `validatePoolBudgets` chequea que los presupuestos entren en el dind y el invariante anti-deadlock de D13, no esto. La guarda correcta —que todo lenguaje del pool pesado declare `memoryBytes ≥ maxMemoryLimitGlobal`— es exactamente el **chequeo cruzado contra `config/virtual_object.json` que el Paso 3 descartó a propósito** (*"acopla el arranque del worker a un archivo de configuración de la API y eso merece su propia decisión"*). Este hallazgo es el argumento más fuerte a favor que apareció hasta ahora: sin él, el acoplamiento existe igual, solo que implícito y sin nadie que lo verifique.

**El cuadro completo**, para un problema que declara los 2048 MB que la plataforma permite:

| | techo efectivo hoy | ¿llega a 2048 MB? |
|---|---|---|
| cpp20 | 2 GiB | sí |
| java17 | 1.5 GiB (`MaxRAMPercentage=75` sobre 2 GiB) | **no** |
| python310 | **1 GiB** | **no, la mitad** |

El de Java es deliberado y está documentado más arriba; el de Python no está justificado en ningún lado.

#### Y la pregunta que abre: ¿el tope de 2048 MB por problema es el correcto? — ✅ RESUELTA en D14

Surgió al discutir lo anterior, y **se resolvió en D14: se queda en 2048**. La propuesta que este documento traía era bajarlo a 1024; se descartó porque con el dimensionamiento de D14 los 2048 no cuestan nada y lo que se liberaría no se puede convertir en concurrencia. El análisis que sigue se conserva porque **el argumento en contra que pedía verificar ya se verificó** (los 12 paquetes BOCA declaran 1024, ver D14) y porque describe bien el acoplamiento entre los dos archivos:

**Qué controla ese número.** Es el máximo que un problem setter puede *declarar*, aplicado por **rechazo, no por recorte** (`internal/domain/problem/language_override.go:49`). Un problema normal declara 256, así que bajarlo no toca los problemas típicos: angosta el extremo que un setter podría pedir.

**Lo que le cuesta a la infraestructura, que no está escrito en ningún lado.** Por D5 el techo del container del pool pesado debe ser ≥ el mayor `memoryLimit` declarable, y ese techo entra directo en el invariante de D13:

```
budget del pool pesado  ≥  maxConcurrent × max(memoryBytes)  ≥  maxConcurrent × maxMemoryLimitGlobal
```

Con `maxConcurrent = 2` y 2048 MB, **el pool pesado no puede medir menos de 4 GiB** sobre los 9.5 GiB que hay para los dos pools. O sea: **un número del archivo de configuración de la API le fija el piso al presupuesto de memoria del judge**, y ni el código ni este documento lo decían.

**Para Java el multiplicador es peor.** Con `MaxRAMPercentage=75`, para que Java alcance de verdad un límite L el container necesita `L / 0.75 = 1.33 × L`. Para llegar a 2048 MB hacen falta **2.7 GiB de container**, y el piso de D13 salta a **5.4 GiB** — más de la mitad del presupuesto total, para servir un límite que casi ningún problema usa.

**Referencias externas**: Codeforces usa 256 MB típico con 1024 MB de máximo de plataforma; AtCoder y Kattis, 1024 MB. DOMjudge usa 2 GiB por default — es el único punto a favor del 2048 actual.

**Con 1024 MB**: containers de ~1.25 GiB para C++/Python y ~1.4 GiB para Java, piso de D13 en `2 × 1.4 = 2.8 GiB` en vez de 4 (o 5.4 con Java arreglado). Libera 1.5–2.5 GiB, que valen más como **`maxConcurrent`** —submissions en paralelo durante una competencia— que como un techo que nadie toca. Y sobre todo, hace **asequible** que los tres lenguajes lleguen de verdad al límite declarado, en vez de que la equidad entre lenguajes se pague con presupuesto.

**El argumento en contra, a verificar antes de decidir**: el import de paquetes ICPC. Un paquete real puede declarar 2048 MB, y como el exceso **rechaza** en vez de recortar, ese import fallaría. Antes de bajar el número conviene mirar qué declaran los paquetes que efectivamente se piensan importar.

**Una inconsistencia aparte, del mismo archivo**: `maxTimeLimit` es **por lenguaje** (300s C++ / 400s Java / 600s Python) — el diseño ya acepta que Python necesita más margen. `maxMemoryLimit` es **plano en los tres** (2048). La presión de memoria por lenguaje es como mínimo tan despareja como la de tiempo: una solución Python a un problema pensado para 256 MB en C++ usa fácil 3-5× eso. El mecanismo para acomodar por lenguaje ya existe en el archivo, se usa para tiempo y no para memoria.

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


### El worker no aguantaba su propio tope de datos de prueba — ✅ RESUELTO en D14

Encontrado al dimensionar el worker para el Paso 6. **Estaba activo hoy**, con `maxConcurrent = 2`, sin necesidad de ningún cambio de este rediseño.

`config/virtual_object.json` declaraba `maxFileSizeTestCaseMB: 200`, que `icpc_parser.go` aplica como **total descomprimido** del ZIP de casos de prueba, y también como tope por archivo individual. O sea que un problema legal podía traer 200 MB de datos.

El worker, por judging, sostiene:

| Componente | Tamaño | Vive |
|---|---|---|
| ZIP comprimido (`io.ReadAll` en `GetTestCases`) | el ZIP entero | solo durante `GetTestCases` |
| Todos los casos descomprimidos (`parseTestCasesZip`) | Σ inputs + answers | **todo el judging** |
| Copia tar de cada archivo que entra al container (`buildTar`) | = ese archivo | por escritura |
| Salida del concursante | ≤ 8 MiB | por caso |

Para un problema en el tope: `200 (casos) + 200 (copia tar del input) + 8 (salida) = 408 MB de datos vivos`, más la línea de base del proceso. Contra los **512Mi** que tenía el container del worker, **un solo judging ya se pasaba**. Y cuando el kernel mata al proceso worker se caen todos los judgings en vuelo y la conexión con RabbitMQ — el modo de falla que el "Problema 1" de este documento describe.

Con los datos reales no explotaba: el peor de los 12 paquetes BOCA son 28.8 MB, así que 2 judgings concurrentes daban ~130 MB. El agujero estaba entre lo que los datos reales usan y lo que la plataforma permitía declarar.

**Arreglado en D14 por los dos lados**: el tope baja a 100 MB y el worker sube a 2 GiB.

### `buildTar` duplicaba en memoria cada archivo que entra al container — ✅ RESUELTO en el Paso 6.5

Encontrado al desglosar la memoria del worker. La API de Docker tiene una sola forma de meter un archivo en un container: `CopyToContainer` recibe un `io.Reader` que debe entregar un **stream tar**. El tar es inevitable; **materializarlo entero en memoria no**.

```go
func buildTar(filename string, content []byte, mode int64) io.Reader {
	var buf bytes.Buffer      // materializa el tar COMPLETO
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(...)
	tw.Write(content)         // el archivo entero entra al buffer
	tw.Close()
	return &buf               // recién acá empieza a leerse
}
```

Devolvía un `io.Reader`, o sea que *aparentaba* ser un stream, pero para cuando devolvía ya había construido todo: el archivo existía dos veces al mismo tiempo.

**Medido** contra un archivo de 27.4 MB (el tamaño del input más grande de los paquetes reales), consumiendo el reader como lo hace el cliente de Docker:

| | Asignado durante la operación | Pico de heap |
|---|---|---|
| `buildTar` original | 27.4 MB | **55.0 MB** |
| con `io.Pipe` + goroutine | **0.0 MB** | 27.6 MB |
| con `io.MultiReader` ← **lo adoptado** | **0.0 MB** | 27.6 MB |

Por qué estaba escrito así: es la forma obvia, y hasta el Paso 5 todo lo que se copiaba era chico (el fuente ≤ 1 MB, el artefacto compilado unos MB). Los inputs de decenas de MB entraron al cuadro cuando el checker pasó a recibir sus tres archivos por la API.

**La parte que este documento marcaba como delicada quedó disuelta, no resuelta.** El plan era `io.Pipe`, y con él había que garantizar que alguien cerrara el lado lector o la goroutine escritora quedaba bloqueada para siempre. Al verificarlo resultó que el riesgo era real por dos vías —ver el Paso 6.5, punto 1— y que una tercera forma, `io.MultiReader`, da la misma propiedad de memoria sin goroutine y por lo tanto sin nada que cerrar.

**Sigue valiendo después del Paso 7**: aunque el input y la salida del concursante pasen al volumen compartido, por la API siguen viajando el fuente de la solución, el artefacto del checker y la **salida esperada** (que D7 punto 5 prohíbe poner en el volumen).

### `GOMEMLIMIT` no está configurado — decidido no hacerlo por ahora

Go **no lee el límite de memoria del cgroup por su cuenta**. Con `GOGC=100` por defecto, el pacer del GC deja crecer el heap hasta ~2× los datos vivos antes de recolectar, sin saber que hay un techo: en un container de 512Mi con 300 MB vivos, apunta a 600 MB y **el cgroup mata el proceso antes de que el GC llegue a correr**.

`grep` sobre manifests, Dockerfiles y código: cero apariciones de `GOMEMLIMIT`, `GOGC` y `GOMAXPROCS`.

Medido (5 judgings concurrentes, Go 1.25, `runtime.MemStats.Sys`):

| Datos por problema | Por defecto | Con `GOMEMLIMIT=800MiB` |
|---|---|---|
| 100 MB | 1005 MB | **797 MB** — lo honra |
| 200 MB | 1614 MB | 1082 MB — **no puede** |

La segunda fila muestra la propiedad importante: es un límite **blando** y **no puede recolectar lo que sigue referenciado**. Con 5 × 200 MB los datos vivos son 1000 MB y ninguna configuración los baja. Sirve donde el problema es el colchón del GC, no donde son los datos vivos.

**Decisión explícita del usuario: no agregarlo por ahora.** Con el dimensionamiento de D14 (tope 100 MB, worker 2 GiB) el worker queda al 52% sin necesitarlo. Queda anotado como la palanca a usar si el margen se achica.

Al medirlo apareció además un dato que explica otra cosa: **Go 1.25 nunca baja `GOMAXPROCS` de 2** aunque el cgroup declare 0.5 CPU (verificado: `cpu.max = 50000 100000` y `GOMAXPROCS = 2`). Con dos hilos contra medio core el runtime quema su cuota en ~25 ms de cada período de 100 y queda frenado el resto — es la explicación de por qué los `500m` del worker frenaban tanto la descompresión.

### El pod del judge es Burstable aunque dind y worker estén igualados — ✅ RESUELTO en D14

Kubernetes clasifica la QoS de un pod mirando **todos** sus containers, **init containers incluidos**. El `prepull-language-images` de `worker.yaml` no tiene bloque `resources` en absoluto, así que el pod queda **Burstable** —de los primeros en ser desalojado bajo presión de memoria del nodo— por más que dind y worker declaren `requests == limits`.

Es el renglón que decide si el resto del cambio de D14 sirve. Arreglado ahí, con `200m / 256Mi`.
---

## Plan de ejecución

**Regla de cada paso**: el proyecto compila y la suite queda en verde al terminarlo. Nada de estados intermedios rotos — en Go no se puede migrar media interfaz, así que cada cambio de puerto arrastra a sus llamadores y mocks en el mismo paso.

**Dos hitos que valen la pena tener presentes**: al terminar el **paso 5** queda cerrado el **problema 2** (ningún código del problem setter corre ya con privilegios del worker). Al terminar el **paso 7** queda cerrado el **problema 1** (el worker vuelve a solo coordinar). El arreglo de seguridad aterriza antes que el de rendimiento, y cada uno vale por separado.

> **Problema 2 cerrado** al completarse el Paso 5. Comprobación concreta: `os/exec` ya no aparece en ningún archivo de `internal/adapter/judge/`. Todo el código que sube un problem setter —checker y validator, al compilar y al ejecutar— corre ahora en containers del pool, con `NetworkMode: "none"`, sin las variables de entorno del worker y sin acceso al socket de Docker.

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

El puerto `NativeCompiler` pasó a `ArtifactCompiler`, con un adapter que reclama un container de pool pesado del lenguaje del artefacto, escribe el fuente, corre el comando de compilación vía `sh -c` y extrae el artefacto con `CopyFromContainer` + `extractFirstFile`. Se borraron once archivos del camino nativo. `PrepareJudgingUseCase` no cambió más allá de un campo del request: la firma del puerto sobrevivió, como el plan preveía.

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
### Paso 4 — El validator al sandbox (D3, D4, D13) ✅ COMPLETO

Los puertos `ValidatorRunner`/`ValidatorSession` pasaron a forma de sesión y su adapter corre sobre el pool liviano, que **se construye acá**, junto con la validación de arranque del invariante de dimensionamiento. `PrepareJudgingUseCase` abre la sesión antes del loop de inputs. El adapter nativo se reemplazó en su mismo archivo, así que `exec.Command` desapareció del camino del validator.

**Con esto, un validator ya no corre con las credenciales del worker.** Queda el checker, que cierra el Paso 5.

#### Lo que se decidió en el camino

**1. Los pools se llaman `heavy` y `light`, no `solutions` y `checkers`.** D10 fijaba `solutions`/`checkers` y resultaron los dos malos. `checkers` es demasiado angosto —el primer usuario del pool liviano es el **validator**, no un checker, y en el Paso 7 lo va a ser también el comparador por tokens—, y es la misma angostura que el Paso 3 ya había corregido al pasar de un nombre único a `ArtifactRole`. Pero además **`solutions` ya era inexacto**: D4 dice que el pool pesado *compila todo* (soluciones, checkers y validators) y lo único exclusivamente suyo es ejecutar soluciones.

El criterio que sí describe la división lo dice este mismo documento en "Cómo funciona hoy el pool": *"la división real no es soluciones vs checkers, es pesado vs liviano"*. Un pool **es** un presupuesto más una tabla de tamaños, así que nombrarlo por el tamaño es literal, no vago. Y aguanta el Paso 7 sin forzar nada: `compare` es liviano y entra sin explicación.

Alternativas descartadas: **`artifacts`** (usa el vocabulario del Paso 3, pero el binario de comparación es código nuestro, no el artefacto de ningún problem setter); **`runner`** (ya significa otra cosa, y significa las dos: las imágenes se llaman `judge-runner:*` y hay un `RUNNER_VERSION` en `images-configmap.yaml`); **`validation`** (colisiona con un concepto propio y distinto —la tabla `problem_validations`, `staleValidationAfterMinutes`, el endpoint de validación— y además nombraría al pool por la minoría de su tráfico, porque el checker corre en cada submission, no solo al publicar); **`judge`** (todo esto es el judge: el archivo, la sección, los paquetes); y **`verifiers`** (palabra que el Paso 3 ya había descartado por otro motivo; reusarla invita a confusión).

**2. Cada pool declara su presupuesto en el YAML, y `PoolConfig` pasa de dos campos a uno.** Los dos pools crean containers en el **mismo demonio**, dentro del mismo cgroup de 10 GiB. Pasarle a cada uno `MemLimitBytes: POD_MEMORY_LIMIT` habría hecho que cada uno se creyera dueño de los 9.5 GiB y entre los dos admitieran 19: la contabilidad del pool es local y no se entera. Además `MemLimitBytes`/`OverheadBytes` describen **el pod**, no el pool. Quedó un solo `BudgetBytes`, y quién resta qué es asunto del worker. De paso se borró el campo derivado `p.budget`, que solo guardaba el resultado de la resta.

Se descartó **derivar** el presupuesto de un pool y darle el resto al otro. Se auto-ajustaría ante un cambio de tamaño del dind y evitaría dos números que mantener sincronizados, pero vuelve invisible el reparto: para saber cuánta memoria tiene el pesado habría que reconstruir mentalmente la cuenta del liviano, y su LRU empezaría a evictar por un cambio en la config del otro. El precio de lo explícito quedó anotado en la sección de detalles pendientes.

**3. La validación de arranque quedó partida en dos funciones.** `validateJudgeConfig` mira el archivo contra sí mismo; el invariante de D13 y la suma de presupuestos necesitan `POD_MEMORY_LIMIT` y `maxConcurrent`, que salen del entorno, así que viven en `validatePoolBudgets(cfg, dindMemBytes, maxConcurrent)`, llamada desde `main` después de calcular la concurrencia. Mezclarlas habría obligado a los tests de reglas —hoy mutaciones puras sobre un struct— a poner variables de entorno. La suma recorre el mapa entero de pools y no la lista fija de dos nombres, para que un tercer pool entre en la cuenta solo; y el invariante itera ordenado, porque el orden de un mapa en Go es aleatorio y el mensaje de error cambiaría entre corridas.

**4. Una regla de arranque que faltaba, encontrada al revisar cómo se elige el pool.** El Paso 3 exigía que un lenguaje con `runCmd` estuviera dimensionado por el pool de soluciones. Con dos pools eso queda a medias: los cuatro campos de artefacto significan "en este lenguaje se puede escribir un checker o un validator", y eso implica que **el pool liviano también tiene que dimensionarlo**. Sin la regla, `Claim` falla con `unknown language` en medio de un publish. Ahora un lenguaje con `runCmd` tiene que estar dimensionado por **los dos** pools.

**5. La elección de pool se confirmó bien planteada, con un hueco anotado.** Cada adapter nace atado a un pool por argumento del constructor, en el composition root: `NewExecutor(heavyPool, ...)`, `NewArtifactCompiler(heavyPool, ...)`, `NewValidatorRunner(lightPool, ...)`. El mapeo es estático de verdad —ningún adapter necesita a veces uno y a veces el otro— y mantiene al pool ignorante de qué clase de código corre, que es lo que el diseño quiere. Se descartó un registro (`registry.Get("light")`): mete una clave string en cada adapter y reintroduce el modo de falla del typo que devuelve un pool en cero. El hueco que deja —mal-cablear un adapter al pool equivocado— está anotado en los detalles pendientes.

**6. D3 dibuja el puerto con un `filename` que D8 ya había eliminado.** El sketch de D3 dice `BeginValidating(ctx, validatorPath, lang, filename)`, pero D8 sacó el filename de los puertos de ejecución al normalizar el artefacto, y el Paso 3 lo ejecutó así. Gana D8: el parámetro no existe. `ValidatorRunRequest` desapareció entero — de sus cuatro campos, dos se fueron a la apertura de la sesión, uno es el parámetro de `Validate` y el cuarto ya no tiene función.

**7. D10 dice que `artifactPath` solo lo usa el pool pesado, y es falso.** El pool liviano lo necesita como **destino** donde escribir el artefacto que baja de GCS. Mismo campo, mismo valor, en un sentido al compilar y en el otro al ejecutar. No cambia el YAML, pero sí el wiring, y explica por qué no se partió la config de artefactos en dos structs.

**8. Un solo `ArtifactConfig` para los dos lados.** Los cuatro campos son una sola cosa —cómo se construye y se corre el artefacto de un lenguaje— y viven juntos bajo el lenguaje en el YAML. `ValidatorRunner` carga con `SourcePath` y `CompileCmd` que no usa, con el precedente exacto de `LanguageExecConfig`, que `Session` también usa a medias. La alternativa —dos structs, uno de compilación y otro de ejecución— obligaba a `main.go` a armar dos mapas de los mismos campos y a poner `artifactPath` en los dos.

**9. Un bug que el diseño habría introducido: el artefacto entraba sin permiso de ejecución.** `buildTar` tenía `Mode: 0644` quemado, y hasta acá daba igual porque lo único que se copiaba hacia adentro eran fuentes e inputs, que solo se leen. **Verificado corriéndolo**: un tar con modo 0644 aterriza como `-rw-r--r-- root root` y ejecutarlo da `Permission denied` (exit 126). Habría roto justo el caso más común, porque testlib es C++ y su artefacto es un ELF. `buildTar` pasa a recibir el modo, con constantes `modeSource`/`modeExecutable` para que no queden números sueltos en los call sites — un `buildTarExecutable` aparte se descartó porque deja el modo implícito en los call sites viejos, que es lo que hizo invisible el bug. El artefacto va en 0755 en los tres lenguajes: el `.jar` y el `.py` no lo necesitan, pero ramificar por lenguaje en Go es lo que D9 saca del código.

De paso quedó verificado que los archivos que entran por la API de Docker son de **root** (no del usuario `judge`) y que los `docker exec` corren como `judge`, uid 1000.

**10. Se borró la limpieza por caso de prueba de `Session`** (`rm -f /sandbox/input.txt /sandbox/output.txt` al final de cada `RunTestCase`). Apareció al preguntarse por qué la sesión del validator no la necesitaba, y la respuesta fue que ninguna de las dos la necesita. **Verificado corriéndolo**: la redirección `>` del shell trunca `output.txt` *antes* de ejecutar el programa, incluso cuando el binario no existe (exit 127, archivo en 0 bytes), así que nunca se puede leer una salida vieja como la de este caso; `CopyToContainer` sobreescribe `input.txt`; `Close` limpia todo; y entre casos de la misma sesión son datos de la misma submission. Era un `docker exec` por caso de prueba que no compraba nada, sin ningún test que lo cubriera. En el Paso 7 la limpieza vuelve a hacer falta, pero sobre el directorio del volumen compartido, no sobre `/sandbox`.

**11. En `BeginValidating` se baja el artefacto antes de reclamar el container.** Deja el camino de error sin un container tomado que haya que devolver. Tiene test propio, porque es un orden que se puede invertir sin que nada se rompa a la vista.

**12. `compare` no entra todavía; llega en el Paso 7.** Es de dos líneas y queda exenta de las reglas de arranque por construcción (no tiene `runCmd`), pero nadie la reclama hasta que la comparación por tokens se mude al pool liviano. Mismo criterio que el Paso 2 usó para no dejar config sin lector. **El Paso 7 tiene que agregarla a los dos lugares** —`languages` y `pools.light.languages`—: si falta el segundo, `BeginChecking` sin checker personalizado falla con `unknown language` en la primera submission.

**13. `judging_timeouts.go` no se borró**, como el plan preveía, pero su comentario nombraba a `ValidatorRunner` entre los subprocesos nativos que acota. Ahora solo queda `OutputComparator`, hasta el Paso 5.

#### Tests

Cada test se verificó **rompiendo lo que prueba**: 7 mutaciones sobre las reglas de config y el invariante de D13, más 4 sobre el `judge_config.yaml` despachado, 3 sobre el caso de uso y 9 sobre el adapter. Cada una pone en rojo exactamente el test que le corresponde.

**Y la verificación encontró un test que no probaba lo que decía**: el invariante de D13 toma el lenguaje **más grande** del pool, pero todos los pools del `validConfig` dimensionaban un solo lenguaje, así que "el más grande" y "el más chico" eran indistinguibles — una mutación que tomaba el menor pasaba en verde. Se agregó un caso donde el pool dimensiona dos lenguajes y el presupuesto alcanza para el chico pero no para el grande.

Hay además un test que corre la config despachada contra **los números reales del cluster** (10 GiB y 3 cores, copiados de `deploy/k8s/judge/worker.yaml` con un comentario que lo dice). Agarra el caso de escribir presupuestos que no entran en la máquina real; **no** agarra el inverso, que alguien agrande el dind y el YAML no lo siga.
### Paso 5 — El checker al sandbox (D3) ✅ COMPLETO

Puertos `OutputChecker`/`CheckerSession`, se reescribe `output_comparator.go` sobre pool liviano, y `ValidateSolutionsUseCase` y `JudgeSubmissionUseCase` abren la sesión antes de sus loops. Se borran `artifact_invocation.go` y `judging_timeouts.go` — verificado al cerrar el Paso 4: `artifactInvocation`, `isTimeoutErr` y `trustedSubprocessTimeout` no tienen más consumidores que `output_comparator.go`, así que los dos archivos se van enteros.

**La comparación por tokens se queda en el worker por ahora**: moverla sin el volumen del paso 7 empeoraría la CPU del worker en vez de mejorarla, porque habría que empujar los bytes hacia adentro del container.

Acá **desaparece por construcción** el bug de `CheckerLanguage`/`CheckerFilename`. Y hay que escribir el test que hoy no existe: uno que cruce la frontera caso de uso → adapter, porque los actuales mockean justo el punto donde se perdían los campos.

**Y con esto queda cerrado el problema 2**: ningún código del problem setter corre ya con los privilegios del worker.

#### `maxOutputBytes` se adelanta a este paso (decidido al cerrar el Paso 4)

El plan le asignaba esta decisión al Paso 7, pero el problema aparece acá. Antes del volumen compartido, el checker recibe input, salida esperada y salida del concursante **como bytes**, y el adapter los copia adentro del container del pool liviano. Con los 64 MiB de hoy, la medición del Paso 4 (ver D13) dice que un checker que lee las dos salidas y compara por tokens **muere por OOM en los tres lenguajes**, con cualquier tamaño razonable de pool. O sea: el Paso 5 tal como estaba planeado entregaba un checker que se cae con salidas grandes.

**Se decidió bajar `maxOutputBytes`** (una constante en `session.go`, un solo lugar). Con 8 o 16 MiB los tamaños actuales del pool liviano sobran, y el propio documento ya calificaba los 64 MiB de patológicos para programación competitiva.

Se descartó **subir los tamaños del pool liviano** para sostener los 64 MiB: obligaría a ~1 GiB para C++ y Python y más para Java, o sea a duplicar el presupuesto del pool liviano para sostener un límite que igual queremos bajar. Y se descartó **dejarlo para el Paso 7**, que es shippear un checker que se sabe que muere.

**Resuelto al ejecutar el Paso 5**: baja a **8 MiB**, y el veredicto de output limit **no entra acá** — la receta para introducirlo quedó escrita en el Paso 7, que es donde el límite pasa a aplicarse de verdad. El razonamiento completo y las alternativas descartadas están en "Lo que se decidió en el camino" de este mismo paso.

#### Advertencia para verificar este paso

Aunque el Paso 5 salga perfecto, **un checker real de testlib sigue sin poder probarse de punta a punta**: `testlib.h` no está en la imagen de C++ (ver la sección de bugs). Un checker escrito con testlib falla con `No such file or directory`, y eso **no** es una falla de este paso.

#### Lo que se decidió en el camino

**1. `maxOutputBytes` baja a 8 MiB, y el veredicto de output limit se difiere al Paso 7.** El número sale de la medición de D13 —los picos del checker a 8 MiB son 119 MiB (C++), 162 MiB (Python) y 88 MiB (Java) contra techos de 512/512/1024 MiB— y coincide con el default de `output_limit` de **DOMjudge** (8192 kB), que es el juez de referencia de ICPC. Se descartó **16 MiB** porque el pico de Python sube a 320 MiB contra un techo de 512, y esa medición usó un checker *naive*: uno de testlib que además parsea a estructuras propias se come el margen restante.

Sobre el veredicto, el argumento que decidió: **hoy `maxOutputBytes` no es un límite de salida sino un tope de lectura del worker** — el comando de `RunTestCase` redirige a `/sandbox/output.txt` sin ningún tope, así que la constante solo evita que el worker se traiga todo a memoria. Un veredicto construido sobre ella le diría al concursante que excedió un límite que el sistema nunca le impuso. Se evaluó aplicarlo de verdad ya acá con `| head -c 8M` dentro del `sh -c` y **se descartó**: el exit code del pipeline pasa a ser el de `head` y eso rompe la detección de TLE (124) y MLE (137). La receta completa quedó escrita en el **Paso 7**, con los siete lugares a tocar. Como paliativo explícito, `copyOutput` **loguea** la truncación (lee un byte más que el límite para distinguirla de una salida que justo entra).

**2. La rama del caso sin checker personalizado se queda en el adapter.** `BeginChecking` con ruta vacía devuelve una sesión sin container que compara por tokens en el worker; con ruta no vacía, la sesión de sandbox. Se descartó que **los casos de uso ramificaran** (`if limits.CheckerPath == ""`): subiría la política de comparación a la capa de aplicación, duplicada en los dos casos de uso, y el Paso 7 tendría que deshacerla en los dos lugares.

El sentinela es la **ruta vacía**, no el `HasCustomChecker` de `ProblemLimits`, que sería el intuitivo pero es incorrecto: hay un caso deliberado donde `HasCustomChecker` es `true` y `CheckerPath` queda vacío —un checker que nunca se compiló, que loguea un warning y cae a comparación por tokens en vez de romper todo judging de ese problema— y el booleano rompería ese fallback.

**Esta forma es la que sobrevive al Paso 7**: cuando la comparación por tokens se mude al pool liviano con la imagen `compare` (D6), la rama deja de devolver una sesión distinta y pasa a elegir qué lenguaje reclamar. Los casos de uso no se enteran. El comentario en el código nombra el Paso 7 explícitamente, y `token_comparison.go` se aisló en un archivo propio para que ese paso lo borre con un `git rm` en vez de desenredarlo.

**3. El orden de los argumentos del checker estaba invertido respecto de testlib; se corrigió acá.** Detalle completo en la sección de bugs. Los tres archivos entran al sandbox con nombres que dicen qué son (`input.txt`, `output.txt` del concursante, `answer.txt` del jurado) y se pasan en ese orden. **Se actualizó `specs/Judge System/README.md`** en el mismo commit, y de paso se corrigió otra cosa que ese spec decía y el código no hace: prometía `exit 1 = WRONG_ANSWER` y `exit 2 = PRESENTATION_ERROR`, cuando `PRESENTATION_ERROR` **no existe** como estado en `domain/submission/status.go` y el adapter trata cualquier no-cero como rechazo.

**4. Cualquier exit distinto de cero sigue siendo rechazo.** Se evaluó traer la tabla de testlib (`3 = _fail`, el checker declarando que el problema está roto) y se descartó; el razonamiento y la receta para encararlo en el Paso 6 están en "Detalles a resolver al implementar".

**5. Los tamaños del pool liviano no se tocaron.** Con 8 MiB los tres picos quedan holgados, y bajarlos no compra nada: el presupuesto ya cierra exacto (6.5 + 3 = 9.5 GiB disponibles) y los tres lenguajes ya entran calientes en ambos pools. En **Java** específicamente bajarlo sería peligroso: un checker que toca el techo del container **sale con exit 1** (medido en el Paso 4), indistinguible de un rechazo legítimo, así que apretarlo no produce un error visible sino *wrong answers silenciosos al concursante*. El invariante de D13 se verificó sin cambios.

**6. `OutputComparator` → `OutputChecker`, por Ad1.** Era el único adapter de `adapter/judge` cuyo nombre no coincidía con el del puerto que implementa (`Executor`→`Executor`, `ValidatorRunner`→`ValidatorRunner`).

**7. Se extrajo `artifactSession`, que es la otra mitad de D3.** Al escribir `CheckerSession` aparecieron **~54 líneas idénticas** a `ValidatorSession` —el bloque de `ExecCreate`/`ExecAttach`/`StdCopy`, la red de seguridad con su `Discard`, y `Close` entero—, no las ~20 que se habían estimado. Lo que se duplicaba no era código trivial sino **el ciclo de vida del container**: cuándo se descarta, cuándo se anula el puntero, cuándo se limpia; una tercera copia divergible de eso es donde aparecen los bugs que los tests no ven. Y D3 lo dice literal: *"La reutilización ocurre en el adapter, no en el puerto"* — los puertos separados ya estaban, el adapter compartido faltaba.

Quedó un struct privado que las dos sesiones **embeben**, con `writeFile`/`run`/`Close`. `run` se quedó además con la construcción del `timeout --kill-after=1s Ns`, que antes armaba cada llamador: así el límite adentro del container y la red de seguridad de Go —que tiene que ser más larga— no pueden separarse. Con eso `Validate` quedó en ~12 líneas y `Check` en ~20, cada una mostrando solo lo suyo.

**8. Se extrajo `downloadArtifact` a un archivo propio**, con el precedente de Ad8 que el Paso 3 usó para `buildTar`/`extractFirstFile`. Los logs pasaron de `validator_runner: compiled validator not found` a `artifact_download: compiled artifact not found` **con el path**, que identifica mejor de qué artefacto se trata (`problems/abc/checker/compiled` vs `.../validator/compiled`) que el nombre del adapter.

**9. `CheckerFilename` desapareció de los puertos, y con él `dbCheckerJSON.Filename`.** Su comentario decía que hacía falta *"to invoke a compiled Java checker — its class name must match the original uploaded filename"*, que dejó de ser cierto en el Paso 3. El nombre original **se sigue guardando en la base para mostrarlo**; lo que se fue es traerlo al judging. Borrar el campo del struct no toca la base: el JSON sigue teniendo la clave y `json.Unmarshal` la ignora.

**10. Los mocks y helpers compartidos se movieron a `mocks_test.go`.** `mockDockerExecClient`, `mockPoolDockerClient`, `fakeAttach`, `blockingAttach`, `stdcopyFrame`, `outputTar`, `statsBody`, `testPoolCfg`, `newTestPool` (renombrado desde `newTestPoolForExecutor`, que ya mentía porque también sirve al pool liviano) y `recordExecs` vivían dentro de `executor_test.go` y `artifact_compiler_test.go`. Es preexistente, pero los tres archivos de test nuevos llevaban los consumidores de 3 a 6, o sea que `executor_test.go` pasaba a ser el `mocks_test.go` del paquete con otro nombre. **A9** y **Ad11** son explícitos al respecto. De paso se unificaron `testArtifactCfg` y `testValidatorCfg`, que eran la misma fixture con un campo de diferencia.

#### Tests

Los tests siguen a lo que prueban, no al archivo donde nació el código (**M5**): los dos de `Close` que vivían en `validator_runner_test.go` se mudaron a **`artifact_session_test.go`**, porque desde la extracción prueban `artifactSession` y no el validator. Ahí se sumaron dos que faltaban: que `run` envuelve el comando en el timeout, y que la red de seguridad **descarta** el container. Y `artifact_download_test.go` cubre el guard del artefacto vacío, que hasta ahora no probaba nadie.

`checker_boundary_test.go` es el test que este documento pedía: construye el **`JudgeSubmissionUseCase` real** sobre el **`OutputChecker` real**, con solo sus vecinos mockeados. Vive en `adapter/judge` porque es el único lado posible —`adapter` importa `application`, al revés sería ciclo— y tiene precedente en **Ad11**, que hace exactamente esto para los handlers.

**Verificación por mutación: 11 mutaciones, todas en rojo en el test que les corresponde y en ninguno más.** Cuatro sobre `artifactSession` (sin timeout, red de seguridad que devuelve en vez de descartar, `Close` que no limpia, container sucio devuelto), cinco sobre el checker (orden de argv invertido, contenidos de los archivos cruzados, rama de ruta vacía desactivada, artefacto sin bit de ejecución, claim antes de la descarga) y dos sobre los casos de uso.

**La mutación que justifica el test de frontera**: reproducir el bug original —pasar `""` en vez de `limits.CheckerPath`, que compila igual que el struct incompleto de antes— pone en rojo `checker_boundary_test.go` y **deja `internal/application/judge` en verde**. Es la demostración concreta de que los tests que existían son ciegos a esta clase de bug.

**Y la verificación encontró un hueco que no habíamos previsto**: abrir la sesión y no cerrarla nunca no lo agarraba nada, y eso filtra un container del pool liviano por solución hasta agotarlo. Se agregaron dos tests —uno por caso de uso— que verifican que la sesión se abre **una vez** para N casos de prueba y se cierra **una vez**; el mismo test agarra también la mutación de abrirla adentro del loop, que es la que desharía el beneficio de D3.

### Paso 6 — Límites de memoria reales (D11) ✅ COMPLETO

`docker update --memory` al reclamar el container, y decidir qué reportar como KB consumidos. Independiente del resto, se puede mover de lugar sin romper nada.

#### Precondiciones: ✅ resueltas en D14

Este paso arrancaba con dos hallazgos del Paso 5 que había que cerrar antes de escribir código, porque cambiaban los números sobre los que opera. **Los dos se resolvieron en D14**, junto con el dimensionamiento entero de la infraestructura:

1. **`python310` tenía 1 GiB en el pool pesado** contra los 2048 MB que la plataforma promete → pasa a 2 GiB.
2. **`maxMemoryLimitGlobal`** → se queda en 2048 (se evaluó bajarlo a 1024 y se descartó; ver D14).

De paso, D14 fijó todo lo demás sobre lo que este paso opera: `maxConcurrent = 5`, los presupuestos de los dos pools, y `java17` en 2.75 GiB para que la JVM pueda entregar los 2048 declarados.

#### Lo que hay que decidir acá

**A1 — Dónde vive el `docker update`. ✅ RESUELTO, ver abajo.** `Pool.Claim(ctx, language)` solo conoce lenguajes y el diseño quiere mantenerlo ignorante de qué clase de código corre. Las dos opciones son `Executor.BeginSession` después del `Claim`, o cambiar la firma de `Claim`.

**A2 — Cómo llega el límite del problema a la sesión. ✅ RESUELTO, ver abajo.** Hoy viaja **por caso de prueba** (`RunRequest.MemoryKb`), pero es constante para todo el judging. Para hacer el `docker update` una sola vez hay que moverlo a la apertura de la sesión — exactamente lo que el Paso 5 hizo con los datos del checker en `BeginChecking`.

**A3 — Qué número exacto se le pasa al `docker update`. ✅ RESUELTO: el límite exacto, sin margen.** La premisa que este documento traía **era falsa, y se midió**.

Decía que el page cache de los archivos que el programa lee y escribe cuenta contra el cgroup y produce MLE falsos —*"un problema de 256 MB con input de 20 MB mata a una solución que reservó 240"*—. Medido en `judge-runner:cpp20`, con el comando exacto de `RunTestCase`, el umbral de OOM es **idéntico** con entradas de 0, 64 y 200 MiB:

| Reserva de la solución | input 0 MiB | input 64 MiB | input 200 MiB |
|---|---|---|---|
| 250 MiB | pasa | pasa | — |
| 255 MiB | OOM | OOM | — |
| 200 MiB | pasa | — | pasa |

El page cache **es reclamable**: al llegar al techo el kernel desaloja páginas de archivo en vez de matar el proceso. El pico del cgroup toca el límite exacto y el proceso sigue vivo.

**El overhead real, medido por búsqueda binaria del máximo reservable** (20 MiB de entrada, imágenes reales):

| Container | cpp20 | python310 | java17 |
|---|---|---|---|
| 256 MiB | **254** (99%) | **251** (98%) | 184 (72%) |
| 512 MiB | **509** (99%) | **507** (99%) | 369 (72%) |
| 1024 MiB | **1021** (99%) | **1018** (99%) | — |

C++ y Python tienen **3-6 MiB de overhead fijo**, no proporcional. Java tiene un 28% proporcional, que es territorio de **A9** y no de esta decisión.

**Se decide pasar el límite exacto, sin margen.** Esos 3-6 MiB son ~2% de un límite típico de 256 MB, y están dentro de lo que el problem setter ya asume: es el mismo modelo de Codeforces y DOMjudge, donde el límite se aplica al proceso **incluyendo su runtime**. Un problema que declara 256 MB está diciendo "tu solución entra en 256 MB, con su runtime adentro".

**Alternativas descartadas**: un **margen proporcional** —no hay nada proporcional que compensar, el overhead es fijo—; y un **margen fijo de 16 MiB**, que evitaría cualquier MLE por overhead del runtime pero compensa un problema que no existe.

**Corrección a una conclusión intermedia de esta misma sesión**: se midió primero un "50% de overhead en Java" y se dio por inválido el `java17: 2.75 GiB` de D14. **Era artefacto del test**, que reservaba un único `byte[]` gigante — el peor caso para la JVM, que necesita una región contigua. Con reservas en trozos de 1 MiB, como asigna una solución real, sube a 72%. Con eso `2816 × 0.72 = 2028 MiB` contra los 2048 declarables: D14 queda **corto por ~20 MiB, no por la mitad**. (Y A9 volvió sobre ese número por otra vía: los 2.75 GiB no servían porque el container se achica al límite del problema, y terminó en 2.375 GiB por el `memoryFactor`. Ver el punto 11.)

**Nota de método, porque casi produce la conclusión opuesta**: el primer experimento no reservaba nada. `g++ -O2` eliminó el `malloc` + `memset` porque la memoria nunca se leía después — optimización legal. Lo delató **el pico del cgroup**: 8 MiB tras "reservar" 300. Sin esa lectura de control, un barrido entero en verde habría "demostrado" que no hay overhead. Se corrigió con `volatile` y leyendo la memoria de vuelta.

**A4 — Cómo se detecta el MLE de forma uniforme en los tres lenguajes. ✅ RESUELTO, ver abajo.** Es el bug de la sección de bugs: la JVM nunca deja que el cgroup la mate por agotamiento de heap —refuerza su propio tope y lanza `OutOfMemoryError`—, así que `exitCodeMLE = 137` es inalcanzable y **todo MLE de Java se reporta hoy como runtime error**, sin rastro (el comando manda stderr a `/dev/null`). La vía del exit code por lenguaje se evaluó y se descartó. La pista que este documento deja: si la señal sale del propio cgroup (`memory.events`, campo `oom_kill`) en vez del código de salida, cubre los tres lenguajes de forma uniforme.

**A5 — Qué reportar como KB consumidos. ⏭️ DIFERIDO AL PASO 7, con el diseño hecho y verificado.** No es elegir entre un número aproximado y el límite: en cgroup v2 `MaxUsage` viene en **cero**, así que hoy todo veredicto reporta 0 KB — y peor, lo persiste como `0` y no como `NULL`, con lo que la API afirma que la solución usó 0 KB. La medición correcta es el `ru_maxrss` del proceso, y quedó diseñada, medida por lenguaje y con sus dos decisiones abiertas analizadas **en el Paso 7**. Se difirió porque lo difícil no es medir sino sacar el número del container, que es exactamente el código que ese paso reescribe: hacerlo acá era escribir esa plomería dos veces, y con la variante peor.

**A6 — Si el `docker update` aplica también al pool liviano. ✅ RESUELTO: no, y el código ya lo hace bien.** El `memoryLimit` de un problema restringe **el código del concursante**; el checker es código del jurado, y sus necesidades escalan con el tamaño de las salidas que compara, no con lo que el problema le promete al concursante. Aplicárselo sería como imponerle al compilador el límite de tiempo del problema. Y las mediciones del Paso 4 lo vuelven inviable, no solo incoherente: un checker necesita 88-162 MiB para una salida de 8 MiB, así que un problema con límite de 64 MB dejaría al checker en 64 MiB y **haría OOM en todos los casos de prueba** — que desde A7 se detecta y termina en `SYSTEM_ERROR` en vez de en wrong answers silenciosos. Los dos reclamantes del pool liviano ya piden `pool.LanguageCeiling`, así que no hay cambio de código: es una decisión a registrar para que nadie la reabra creyendo que es una inconsistencia.

**Consecuencia**: el dimensionamiento del pool liviano depende únicamente de cuántos bytes tiene que sostener el artefacto — la salida del concursante (`maxOutputBytes`) más el input y la salida esperada. Eso lo deja correcto hoy y deja el **tope por archivo de caso de prueba**, que no existe, como el único pendiente que puede volver a romperlo.

**A7 — El OOM del checker en el pool liviano. ✅ RESUELTO, ver abajo.** (abierto desde el Paso 5, mismo mecanismo). Un checker que se queda sin memoria sale con **137 en C++/Python y con 1 en Java**, indistinguible de un rechazo legítimo — o sea, *wrong answers silenciosos al concursante*. Si la solución de A4 sale del cgroup, cubre este caso también y hay que cerrarlo acá.

**A8 — La guarda de arranque que falta. ✅ RESUELTO: va en un test, no en el arranque.** que todo lenguaje del pool pesado declare `memoryBytes >= maxMemoryLimitGlobal`. Es el **chequeo cruzado contra `config/virtual_object.json`** que el Paso 3 difirió a propósito por acoplar el arranque del worker a un archivo de configuración de la API. El bug de Python es el argumento más fuerte a favor que apareció: sin la guarda, el acoplamiento existe igual, solo que implícito y sin nadie que lo verifique.

> **Verificado por mutación al aplicar D14**: revertir `python310` a 1 GiB en `judge_config.yaml` deja **toda la suite en verde**. Ninguna regla de arranque ni ningún test relaciona el techo del pool con `maxMemoryLimitGlobal`, así que el bug podría volver a entrar sin que nada avise. En cambio, las guardas que **sí** existen se comprobaron efectivas: bajar el dind a 10 GiB o subirlo a 10 cores sin tocar los presupuestos ponen en rojo `TestValidatePoolBudgets_TheShippedConfigFitsTheCluster`.

**A9 — El `runCmd` de Java deja de poder ser estático. ✅ RESUELTO: no, sigue estático — y la premisa se cayó sola.** `RUNNER_ARCHITECTURE.md` especificaba `runCmd: "java -Xmx{memoryLimit}m Solution"` — el diseño original **sí** contemplaba pasarle a la JVM el límite del problema con una plantilla. Hoy es `java -XX:MaxRAMPercentage=75 -cp /sandbox Solution`, estático, y por eso `java17` necesita 2.75 GiB de container para entregar 2048. Si el límite pasa a ser por problema, ese string choca con D9 ("la config queda 100% estática") — que el Paso 3 ya tuvo que matizar con el token `{name}`. Conviene mirar la especificación original antes de inventar otra.


#### Lo que se decidió en el camino

**1. A1 — el límite entra en la firma de `Claim`, no en un método aparte.**

```go
func (p *Pool) Claim(ctx context.Context, language string, memoryBytes int64) (*Container, error)
```

Se evaluaron tres formas y **decidió un hallazgo, no la estética**: el pool pesado lo comparten **dos** consumidores —`Executor` (juzgar) y `ArtifactCompiler` (compilar checkers y validators)— sobre los **mismos** containers, porque el pool reusa por lenguaje y no por consumidor. Entonces un container que un judging dejó en el límite de su problema le entregaría ese techo a la siguiente compilación:

```
submission a un problema de 256 MB → container en 256 MiB → Release
publish de otro problema           → ArtifactCompiler reclama ESE container
                                   → g++ -O2 con testlib en 256 MiB → OOM
```

Un checker que no compila, con un error que no se parece a la causa. Con el límite en la firma, **cada reclamante declara lo que necesita y lo que había antes deja de importar** — el estado sucio no se puede dar, no porque alguien lo limpie sino porque nadie lo lee.

**Alternativas descartadas**: *(a)* que el executor aplicara y `Session.Close` restaurara — el executor necesitaría el techo del lenguaje, que vive en `pool.LanguageConfig`, así que habría que **duplicarlo** en `ExecutorConfig` con dos números que deben coincidir y nada que lo verifique; *(b)* un `pool.SetMemoryLimit` público más restauración en `Close` — menos invasivo, pero el invariante pasa a depender de que `Close` corra y de que cada consumidor se acuerde, y hoy **tres de los cuatro** reclamantes no fijan nada.

**Y la propiedad que solo el pool puede dar**: capa cualquier valor por encima de `langCfg.MemoryBytes`. D11 justifica que bajar el límite es seguro diciendo *"la contabilidad del pool queda conservadora"*, y eso vale **solo mientras nadie suba** por encima de lo contabilizado. El pool es el único que tiene los dos números en la mano —lo que carga contra su presupuesto y lo que el kernel aplica—, así que ahí la sobreventa pasa a ser **imposible por construcción** en vez de una convención. Es el bug de Python de D14 visto desde el otro lado.

**2. Lo verificado corriéndolo, para que nadie lo re-investigue.**

- **`ContainerUpdate` con solo los campos de memoria deja intacto el `NanoCPUs`.** Era el riesgo real: el API recibe el struct `Resources` entero, y perder el techo de 1 CPU rompería la equidad del veredicto que D5 sostiene. Medido por la vía exacta del código (`Resources{Memory, MemorySwap}`): `memory.max` cambia, `cpu.max` y `memory.swap.max` quedan igual. El demonio trata los campos en cero como "no tocar".
- **Bajar y subir funcionan los dos**, sobre un container en marcha, sin recrearlo.
- **Cuesta 16.7 ms** (peor caso de 100: 19.1), contra los **221 ms** que el pool ya paga por crear un container. Y ocurre **una vez por judging**, no por caso de prueba. Además muchas veces no se paga: un container nuevo se crea directamente con el límite pedido, y uno reusado con el mismo límite se saltea el viaje.

**3. Una desviación del plan de este documento**: decía agregar `ContainerUpdate` a `dockerExecClient`. Va en **`dockerLifecycle`** (`pool/docker_client.go`), porque el dueño de la operación resultó ser el pool. `dockerExecClient` no se tocó.

**4. A2 — el límite viaja en `BeginSession`, y `RunRequest.MemoryKb` desaparece.**

```
problems.memory_limit → ProblemLimits.MemoryKb → BeginSession(ctx, lang, memoryKb)
                                               → Claim(ctx, lang, memoryKb*1024)
```

El campo no se "movió" tanto como se **borró**: `RunRequest.MemoryKb` solo se asignaba (en `judge_submission.go` y `validate_solutions.go`) y **nadie lo leía**, que es la forma en que el bug original sobrevivió sin que nada avisara.

**5. `LanguageCeiling` en vez de un `0` pelado** en los tres call sites que corren código confiable. Precedente directo: el Paso 4 introdujo `modeSource`/`modeExecutable` justamente *"para que no queden números sueltos en los call sites"*, después de que un `0644` quemado hiciera invisible el bug del artefacto sin permiso de ejecución.

**6. A3 quedó confirmado: el límite exacto, sin margen.** `BeginSession` pasa `memoryKb * 1024`. Salió así por omisión al implementar, y después se midió que es lo correcto — el detalle y las mediciones están en A3, incluida la corrección de la premisa que este documento traía sobre el page cache.

**7. A4 — el MLE de Java se detecta con el mismo 137 que los otros dos, y la pista de este documento era la equivocada.**

La pista decía sacar la señal del cgroup (`memory.events`, campo `oom_kill`). **Medido: no sirve**, y por la razón que hace difícil el problema — cuando una solución Java agota el heap **no hay ningún OOM del cgroup**: la JVM se autolimita antes y lanza `OutOfMemoryError`. El contador queda en `+0`. El cgroup habría cubierto C++ y Python, que es donde ya funcionaba, y perdido Java entero.

**También se midió, y se descartó, subir el techo del heap** para que el cgroup pase a ser el límite que ata. El exit code queda dependiendo del **patrón de asignación del concursante**:

| Patrón de asignación | `MaxRAMPercentage=75` | `MaxRAMPercentage=100` |
|---|---|---|
| un `byte[]` gigante | 1 | **1** |
| trozos de 1 MiB | 1 | 137 |
| muchos objetos chicos | 1 | 137 |

La JVM tiene dos modos de falla: si una **sola** petición no entra en el heap máximo la rechaza sin tocar memoria; si va **creciendo**, el cgroup la mata. Ninguna señal derivada del kernel puede unificarlos.

**Lo que se adoptó** es un flag en el `runCmd` de Java:

```yaml
runCmd: "java -XX:MaxRAMPercentage=75 '-XX:OnOutOfMemoryError=kill -9 %p' -cp /sandbox Solution"
```

La JVM se manda `SIGKILL` a sí misma al primer `OutOfMemoryError` → **exit 137, el mismo código que produce el cgroup**. Verificado sobre 3 patrones de asignación × 3 tamaños de container × 2 configuraciones de heap: 137 en todos. Y verificado de punta a punta tomando el string **literal del YAML despachado** y corriéndolo en `judge-runner:java17` con la forma exacta del comando de `RunTestCase`.

**Por qué gana sobre `+ExitOnOutOfMemoryError`**, que este documento había descartado: ése también da un código uniforme (el 3), pero **convierte `exitCodeMLE` en una tabla por lenguaje**, que era la objeción de fondo. Con el `kill -9` la constante sigue siendo una sola y pasa de ser una mentira a ser cierta. **Cero cambios en Go**: es una línea de configuración.

**Las comillas simples son estructurales**: `runCmd` se interpola dentro de un `sh -c`, así que sin ellas el shell parte el flag en tres argumentos y lo ignora en silencio. Medido: sin comillas, exit 1.

**Los caveats, medidos:**

- `OnOutOfMemoryError` hace `fork` para lanzar el comando, y la JVM lo documenta como *best-effort*: bajo presión extrema podría no ejecutarse. **Su modo de falla es benigno**: la JVM sigue propagando el `OutOfMemoryError` y sale con 1, o sea que en el peor caso se vuelve al comportamiento de hoy. No introduce ninguna falla nueva. No falló en ninguna de las corridas, ayudado por que con `MaxRAMPercentage=75` todavía queda un 25% del container libre cuando el heap se llena.
- Dispara aunque el programa **capture** el `OutOfMemoryError`. En programación competitiva no es un patrón, pero es un cambio de semántica.
- La detección **tarda entre 0.5 y 2.3 s** y crece con el tamaño del container, así que compite con la red de reloj de `RunTestCase`. Ver el punto 8.

**Y lo que conviene saber para calibrar la urgencia de este bug**: `get_standings.go:241` trata `RUNTIME_ERROR` y `MEMORY_LIMIT_EXCEEDED` **idénticamente** como intento fallido penalizado, así que el puntaje de la competencia nunca fue distinto por esto. Lo que se gana es **diagnóstico**: al concursante, que dejaba de buscar un crash inexistente; y sobre todo al problem setter, que en el publish recibía "runtime error" por su propia solución de referencia y podía concluir que su código tiene un bug en vez de que el límite de memoria es muy ajustado para Java — y publicar un problema que ninguna solución Java puede cumplir.

**8. La red de seguridad del worker pasa de `+2s` a `+5s`, extraída como `runGrace`.**

```
SIGTERM    en  wallBackstop
SIGKILL    en  wallBackstop + 1     ← muerte garantizada
red de Go  en  wallBackstop + 5     ← antes +2
```

Antes eran 2 segundos, y **ese margen no era margen puro**: el mismo deadline envuelve también `ExecCreate` y `ExecAttach`, dos round trips al demonio de Docker que con 5 judgings concurrentes se estiran. Y este paso lo vuelve estructural: hoy un MLE de Java sale al instante (exit 1) y nunca se acerca a la red; con el flag tarda 1-2.3 s y sí llega.

Medido, replicando las dos redes: **la red de Go no dispara en ningún caso**, ni siquiera con el margen viejo. Lo único que se degrada es la combinación de 2 GB de memoria con 1 s de tiempo, donde el backstop gana por 0.26 s y el veredicto sale **TLE en vez de MLE** — un veredicto equivocado pero inofensivo, y no `SYSTEM_ERROR`. En el caso real de los 12 paquetes (1024 MB de memoria, límites de 1 a 26 s) la detección tarda ~1 s contra un backstop de 2 s: **MLE correcto**.

Subirlo es barato: cuando esa red dispara, el container **se destruye** (`pool.Discard`, 221 ms de recreación), así que hacerlo menos sensible reduce destrucciones espurias. Y de paso **cierra una inconsistencia**: el camino del checker ya usaba `artifactRunGrace = 5 * time.Second`, con el comentario que describe exactamente para qué sirve. El de las soluciones usaba 2.

**9. A7 — el artefacto matado deja de ser un veredicto, y se detecta en `artifactSession.run`.**

`Check` y `Validate` leen cualquier exit no-cero como "el artefacto rechazó su entrada". Para un artefacto **matado** eso culpa a la salida del concursante, o al caso de prueba del setter, por nuestro propio dimensionamiento del pool liviano.

La detección va en **`run`**, la mitad compartida que el Paso 5 extrajo, y no en cada sesión: *"el artefacto murió"* es una propiedad de ejecutarlo, no de lo que significa. `Check` y `Validate` no cambian una línea.

El `artifactRun` de Java lleva el mismo flag que su `runCmd`, por la misma razón. Verificado de punta a punta con el string literal del YAML: un checker Java que se queda sin memoria sale con **137**, y uno que corre bien con **0**.

**Y acá la pista del documento se invierte del todo**: con el flag, la JVM se mata **desde el espacio de usuario**, así que el contador `oom_kill` del kernel **queda en 0**.

| | exit | `oom_kill` |
|---|---|---|
| checker Java sin el flag | 1 | +0 |
| checker Java con el flag | **137** | **+0** |

Un esquema basado en `memory.events` habría detectado C++ y Python —donde no hacía falta— y perdido exactamente el caso que motivaba A7.

**10. A9 — el `runCmd` de Java sigue siendo estático, y lo que faltaba era una segunda percentage.**

La premisa de A9 era que el `runCmd` tenía que volverse dinámico para llevar `-Xmx{memoryLimit}`, como especificaba `RUNNER_ARCHITECTURE.md`. **A1 la disolvió**: antes el container era siempre 2 GiB sin importar el problema, así que `MaxRAMPercentage=75` daba 1.5 GiB sin importar el problema. Ahora el `docker update` deja el container **en el límite del problema**, y el porcentaje escala solo. Eso era todo lo que la plantilla buscaba.

Lo que quedaba era la brecha: Java recibía el 72% de su container y C++ el 99%. Al medirla apareció que **debajo de ~250 MB de container `MaxRAMPercentage` no se aplica en absoluto** — la JVM usa `MinRAMPercentage`, cuyo default es 50%:

| Container | `MaxRAM=75` | `MaxRAM=90` | `MaxRAM=90` + **`MinRAM=90`** |
|---|---|---|---|
| 128 MiB | 47% | 47% *(lo ignora)* | **85%** |
| 256 MiB | 71% | 87% | 87% |
| 512 MiB | 72% | 86% | 86% |

Con las dos, Java queda en **85-87% uniforme en todo el rango**, con config estática y sin tocar Go. Verificado de punta a punta con el `runCmd` literal del YAML, incluyendo que un MLE siga dando 137.

**Y la plantilla del spec original resultó medidamente peor**, no solo innecesaria:

| | 128 MiB | 256 MiB | 512 MiB |
|---|---|---|---|
| `-Xmx(container−48)` | 59% | 78% | 87% |
| `MaxRAM=90` + `MinRAM=90` | **85%** | **87%** | 86% |

`RUNNER_ARCHITECTURE.md` apuntaba al problema correcto con la herramienta equivocada. **D9 queda intacto**: no hace falta ninguna sustitución nueva.

**Lo que sigue abierto y no se cierra acá**: la brecha entre el 86% de Java y el 99% de C++ es **estructural**. Los ~50 MiB no-heap de la JVM son memoria real que el proceso usa, y como el container se capa al límite del problema, no hay porcentaje que los recupere. Cerrarla exigiría que el `docker update` de Java usara `límite / 0.86`, o sea **una regla por lenguaje dentro de Go** — justo lo que A3 decidió no hacer, y que además obligaría a redimensionar los pools (el techo pasaría a ser 2048/0.86 = 2.33 GiB). Queda anotado como una asimetría conocida, no como un bug: un problema que declara 256 MB le da 254 a C++, 251 a Python y 223 a Java.

**11. La brecha del 86% se cierra con un `memoryFactor` por lenguaje, y el dimensionamiento se recalcula.**

Con el porcentaje arreglado Java seguía recibiendo el 86% de lo que el problema declara, contra el 99% de C++ y el 98% de Python. Eso no se puede cerrar con más porcentaje: los ~10% que `MaxRAMPercentage=90` le deja a lo no-heap son memoria real que el proceso usa, y como el container se capa **al límite del problema**, esa reserva sale del presupuesto del concursante.

**Lo que se adoptó** es un campo por lenguaje en `judge_config.yaml`:

```yaml
cpp20:      memoryFactor: 1.0     # un binario nativo no reserva nada propio
python310:  memoryFactor: 1.0     # el intérprete cuesta ~5 MiB, fijo y no proporcional
java17:     memoryFactor: 1.15
```

`Executor.BeginSession` pide `límite del problema × factor`. **Es dato, no lógica** — la misma forma que `runCmd`, `extension` o los cuatro campos de artefacto —, así que Go lee un número de un mapa y multiplica, sin ninguna rama por lenguaje. **D9 sigue intacto.**

Medido con el `runCmd` literal del YAML, en la imagen real:

| Límite declarado | Container = límite × 1.15 | La solución recibe |
|---|---|---|
| 128 MB | 147 MiB | **100%** |
| 256 MB | 294 MiB | **100%** |
| 512 MB | 588 MiB | **99%** |
| 1024 MB | 1177 MiB | **99%** |

**Es obligatorio para todo lenguaje con `runCmd`**, con una regla de arranque nueva, igual que los cuatro campos de artefacto. Si tuviera un default de 1.0, alguien que agregue Kotlin o Scala heredaría el bug en silencio.

**Se descartó la variante barata** de dejar el techo de `java17` en 2 GiB y aceptar que el factor quede capado por encima de problemas de ~1780 MB: los 12 paquetes reales declaran 1024, así que no se notaría, pero dejaría una asimetría silenciosa en el extremo — la plataforma diría que se puede declarar 2048 y Java recibiría 1780. Es exactamente la clase de bug que este documento viene encontrando; el de Python en D14 era eso mismo.

**El dimensionamiento pasó por dos revisiones en este mismo paso**, y conviene dejar el rastro para que los números no parezcan arbitrarios:

| | Antes de A9 | Tras quitar el 2.75 inútil | **Con el `memoryFactor`** |
|---|---|---|---|
| `java17` en `pools.heavy` | 2.75 GiB | 2 GiB | **2.375 GiB** |
| N3 del pool pesado | 17.75 | 14.00 | **15.875 GiB** |
| Presupuesto pesado | 17.75 | 14.75 | **16.75 GiB** |
| `dind.limits.memory` | 25 GiB | 22 GiB | **24 GiB** |
| Margen del pod | 0.9 GiB | 3.9 GiB | **1.9 GiB** |

Los 2.75 GiB originales salían de `2048 / 0.75` y **A1 los había dejado sin efecto** — el container se achica al límite del problema, así que ese techo solo se alcanzaba con un problema sin límite declarado o al compilar, y `javac` sobre un checker multi-clase de 200 líneas usa **60-111 MiB, medido**. Con el factor el techo vuelve a tener función, pero ahora por una razón que se puede explicar: es `2048 × 1.19`, el máximo que un `Claim` puede llegar a pedir.

**Lo que queda sin cubrir, y es de A8**: si alguien bajara el techo de `java17` por debajo de `2048 × 1.15`, el pool caparía el pedido y el factor dejaría de aplicarse para los problemas grandes. `Claim` loguea un warning al capar, pero ningún test lo detecta. La guarda que lo atraparía es *"todo lenguaje del pool pesado debe declarar `memoryBytes ≥ maxMemoryLimitGlobal × su memoryFactor"*, que es el chequeo cruzado contra `virtual_object.json` que A8 tiene pendiente. El factor le da a esa guarda una forma más precisa de la que tenía.

**12. A8 — el chequeo cruzado vive en un test, no en el arranque del worker.**

A8 pedía una guarda que ningún archivo tenía: que todo lenguaje del pool pesado declarara `memoryBytes` suficiente para el mayor límite que un problema puede declarar. El Paso 3 la había diferido porque *"acopla el arranque del worker a un archivo de configuración de la API"*.

**El dato que decidió dónde va**: el `Dockerfile` hace `COPY --from=builder /app/config /config`, y el worker **no monta ningún ConfigMap de configuración** — su único volumen es `dind-storage`. O sea que `judge_config.yaml` y `virtual_object.json` viajan **horneados en la misma imagen** y no pueden divergir en el cluster: solo pueden divergir **en el repositorio**. Y eso lo agarra un test en CI, antes de que la imagen exista.

Una validación en arranque comprobaría dos archivos que ya no pueden estar en desacuerdo, y a cambio **agregaría un modo de falla nuevo**: el worker dejaría de arrancar si ese archivo falta o no parsea, cuando hoy no lo necesita para nada más. Y si algún día `virtual_object.json` se mueve a un ConfigMap —que es justo lo que contempla el pendiente de revisar la configuración de la plataforma— el worker se rompería.

El test asevera, para cada lenguaje del pool pesado:

```
memoryBytes(lenguaje)  ≥  maxMemoryLimitGlobal × memoryFactor(lenguaje)
```

y reusa el struct `config.VirtualObject` en vez de redeclarar el esquema, así que un renombre de campo lo rompe fuerte. Con eso quedan cubiertos los tres agujeros que nadie veía: el bug original de `python310` que D14 arregló a mano, un techo de `java17` por debajo de lo que el `memoryFactor` necesita, y cualquiera que suba `maxMemoryLimitGlobal` sin redimensionar el pool.

**El aviso quedó en `internal/config/virtual_object.go`**, sobre el campo `MaxMemoryLimitGlobal`. El JSON no admite comentarios, y ese struct es donde aterriza alguien que vaya a cambiar el número.

**Y una mutación encontró un bug en el test recién escrito.** La primera versión usaba el `languageOverrides[L].maxMemoryLimit` como máximo efectivo del lenguaje, y **es incorrecto**: el `memoryLimit` **base** de un problema se valida contra el global (`memory_limit.go`) y aplica a cualquier lenguaje que se envíe; los límites por lenguaje solo capan lo que un *override* puede declarar, y `platform_settings.go` ya garantiza que sean ≤ el global. O sea que el máximo efectivo de cualquier lenguaje **es siempre el global**. Sin la mutación de subir `maxMemoryLimitGlobal`, el test habría quedado en verde sin cubrir justamente el caso que más importa.

#### Tests

Nueve tests nuevos, **los nueve verificados rompiendo lo que prueban**:

| Mutación | Tests que se ponen en rojo |
|---|---|
| `Claim` ignora el límite pedido | 1, 3, 5, 6 |
| sin el capado al techo | **solo 2** |
| crear el container con el techo en vez del límite | **solo 3** |
| sin el *skip* cuando el límite ya coincide | **solo 4** |
| el camino rápido no aplica el límite | 1, 5, 6 |
| update fallido no descarta el container | **solo 6** |
| sin la conversión KB → bytes | **solo** el del executor |
| `judge_submission` pasa `0` en vez del límite | **solo** el suyo |
| `validate_solutions` pasa `0` en vez del límite | **solo** el suyo |

Las dos últimas son las que valen: reproducen la forma exacta del bug de `CheckerLanguage` —un cero que compila perfecto— justo en el momento en que ese bug entra, porque el valor **acaba de mudarse** de un call site a otro. Los tests que ya existían mockean el executor sin mirar sus argumentos, así que son ciegos a eso.

Y el test del pool que cubre el hallazgo del punto 1 (`ContainerLeftAtLowerLimit_RestoredForNextClaimer`) es el que documenta, en código, por qué el límite está en la firma.

**Nota de método**: el primer intento de la mutación "el camino rápido no aplica el límite" rompió la compilación en vez de mutar limpio. Un build roto no prueba nada — se rehizo reemplazando solo la llamada por un `error(nil)`, y ahí sí falló en los tests correctos.

Y ocho más para A4 y A7, con sus mutaciones:

| Mutación | Tests que se ponen en rojo |
|---|---|
| `runGrace` por debajo del `--kill-after` | `SafetyNetOutlivesTheInContainerKill` |
| se quita `--kill-after` del comando | `WrapsTheCommandInTheInContainerTimeout` |
| se quita el flag de OOM del `runCmd` de Java | la guarda de config del `runCmd` |
| se quitan las comillas del flag | la misma guarda |
| se quita la detección del artefacto matado | los 3 del artefacto matado |
| `exitCodeKilled` apunta al `1` | **6**: los del artefacto matado *y* los del rechazo legítimo |
| se quita el flag del `artifactRun` de Java | la guarda de config del `artifactRun` |

La del `exitCodeKilled = 1` es la que más dice: rompe las dos familias a la vez, o sea que los tests fijan **en tensión** las dos conductas que este paso tiene que separar — un artefacto matado no es un veredicto, y un rechazo legítimo sí lo es.

**Dos mutaciones que hay que mirar aparte:**

- **`runGrace` de vuelta a 2 segundos: pasa, y está bien que pase.** El invariante testeable es *"el worker no se rinde antes del SIGKILL del container"*, y con 2 segundos se sigue cumpliendo. El 5 es una elección de calibración respaldada por medición, no una propiedad verificable. Un test que fingiera cubrirla sería un test tautológico sobre una constante.
- **Quitar `--kill-after` no rompía nada antes de este paso.** El camino del checker y el del validator aseveran su comando exacto desde el Paso 5; el de las soluciones no lo hacía. Ese flag es lo que garantiza que el proceso muera antes del deadline del worker, así que sacarlo convertía runs lentos en containers destruidos y `SYSTEM_ERROR`. El test nuevo cierra ese hueco preexistente.

**Y dos guardas sobre la configuración despachada**, con el precedente del test que ya corre `judge_config.yaml` contra los números reales del cluster. Son literales, y su valor no es lo que aseveran sino **el comentario que explica por qué**: el arreglo de A4 y A7 vive entero en un string de YAML, y quitarlo no rompe nada visible — simplemente devuelve el bug, en silencio.

**Y cuatro más para A9 y A8**, con sus mutaciones:

| Mutación | Tests que se ponen en rojo |
|---|---|
| se quita `MinRAMPercentage` del `runCmd` de Java | la guarda del reparto de Java |
| `MaxRAMPercentage` vuelve a 75 | la misma |
| `java17` vuelve a `memoryFactor: 1.0` | la misma |
| el executor ignora el `memoryFactor` | `MemoryFactorBuysBackTheRuntimeReserve` |
| se quita la regla de arranque del `memoryFactor` | `RejectsBrokenConfigs` |
| el techo de `java17` baja a 2 GiB | **el chequeo cruzado de A8** |
| se reintroduce el bug de `python310` (1 GiB) | el mismo |
| se sube `maxMemoryLimitGlobal` sin redimensionar | el mismo |
| *(control)* tocar `virtual_object.json` sin romper el invariante | ninguno — sin falsos positivos |

**Dos mutaciones más que pasan a propósito**, además de la de `runGrace`:

- **Bajar el presupuesto del pool pesado por debajo de N3 pero sobre N1 pasa.** Es correcto: la validación de arranque hace cumplir **el invariante de D13 (N1)**, no el objetivo de comodidad (N3). Bajarlo por debajo de N1 sí falla.
- **El control de A8**: editar `virtual_object.json` sin romper el invariante no rompe nada, que es lo que demuestra que el test no tiene falsos positivos.

#### Notas de método

Tres experimentos de esta tanda dieron un resultado equivocado antes de dar el correcto, y los tres habrían llevado a la conclusión opuesta:

1. **El primer experimento de overhead no reservaba memoria.** `g++ -O2` eliminó el `malloc` + `memset` porque nada leía el buffer después — optimización legal. Un barrido entero en verde habría "demostrado" que no hay overhead. **Lo delató el pico del cgroup**: 8 MiB tras "reservar" 300. Corregido con `volatile` y leyendo la memoria de vuelta.
2. **El "50% de overhead de Java" era artefacto del patrón de asignación.** Un único `byte[]` gigante es el peor caso para la JVM, que necesita una región contigua. Con trozos de 1 MiB —como asigna una solución real— el número real es 72%. Sobre el dato equivocado se llegó a dar por inválido el dimensionamiento de `java17` de D14, que en realidad está bien.
3. **Dos corridas dieron exit 1 "de la JVM" que en realidad era mi test desbordando `int`.** Un array de Java está indexado por `int`, así que `3072 << 20` da negativo y lanza `NegativeArraySizeException`. Parecía que el flag fallaba a tamaños grandes.

El patrón común: **un experimento que "pasa" no prueba que el mecanismo funcione**; hace falta un control que demuestre que el experimento sabe fallar.

### Paso 6.5 — `buildTar` deja de duplicar ✅ COMPLETO

`buildTar` pasó de materializar el tar en un `bytes.Buffer` a transmitirlo. Medido: pico de heap de 55.0 → 27.6 MB para un archivo de 27.4 MB, con **cero** asignación durante la operación. La firma no cambió, así que los seis llamadores, los puertos y los mocks quedaron intactos.

Es un paso propio porque no pertenece a ninguno de los dos vecinos: no es memoria de containers (Paso 6) ni depende del volumen compartido (Paso 7), y **sigue valiendo después del Paso 7** — el fuente de la solución, el artefacto del checker y la salida esperada siguen viajando por la API de Docker.

#### Lo que se decidió en el camino

**1. `io.MultiReader` en vez del `io.Pipe` que este documento prescribía, y el riesgo del leak resultó peor de lo anotado.**

El documento pedía verificar si el cliente de moby cierra el reader en todos los caminos de error. Verificado leyendo el código: **no**. Una vez que la petición llega a `http.Client.Do` sí —`net/http` cierra el `Body` en todos sus caminos, y `*io.PipeReader` implementa `Close()`, así que la aserción `body.(io.ReadCloser)` de `NewRequestWithContext` toma el nuestro y no un `NopCloser`—, pero hay dos caminos anteriores donde nadie lo toca: `trimID` (`container_copy.go`) devuelve error sin llegar a `putRaw`, y `http.NewRequestWithContext` puede fallar antes de asignar el body.

Y apareció una vía que el documento no anticipaba, más inmediata que las dos anteriores: **nuestros propios tests**. El mock por defecto de `CopyToContainer` devuelve sin leer nada, y el helper `firstTarEntry` lee sólo la primera entrada y suelta el reader. Medido, con control: 20 de 20 pipes abandonados dejan una goroutine bloqueada; el mismo pipe drenado por completo deja 0.

`io.MultiReader` da la misma propiedad de memoria sin goroutine: `archive/tar` escribe sólo el header a un buffer chico, y detrás van el contenido del llamador —sin copiarlo— y la cola de ceros. Como nadie empuja, no hay nada que se pueda bloquear ni nada que cerrar, y la firma sigue devolviendo `io.Reader`.

**Lo que hay que saber al leer el código**: la cola son el relleno del último bloque de 512 más los dos bloques de ceros que cierran un tar, y su tamaño depende sólo de `len(content)`. El `%512` de afuera en `(512 - len%512) % 512` es lo que evita meter un bloque de relleno entero cuando el archivo mide un múltiplo exacto.

**2. El body pasa de `Content-Length` a `Transfer-Encoding: chunked`, y el demonio lo acepta.** `http.NewRequest` sólo calcula la longitud para `*bytes.Buffer`, `*bytes.Reader` y `*strings.Reader`, así que cualquier variante en streaming —ésta o la del `io.Pipe`— cambia la codificación. Ningún test del repo puede verlo: los mocks reciben el `io.Reader` directo y nunca pasan por HTTP.

Verificado de las dos formas, con el `buildTar` real. Lo que va por el cable, leído con `httputil.DumpRequestOut`:

| | header |
|---|---|
| materializado (`*bytes.Buffer`) | `Content-Length: 27401728` |
| streaming (`buildTar`) | `Transfer-Encoding: chunked` |

Y contra un demonio real, copiando a un container de verdad y leyendo el archivo de vuelta: 5 bytes y 27.4 MB aterrizan con sha256 idéntico y con el modo correcto (0644 y 0755). **El control que le da valor a esas filas**: un tar cuyo header declara 4096 bytes de más fue **rechazado** por el demonio (`unexpected EOF`), o sea que valida el stream y el experimento sabe fallar. El chequeo se escribió como un test temporal dentro del paquete —para llamar a la función real y no a una copia— y se borró después de correrlo, igual que las verificaciones contra imágenes reales de los Pasos 3 y 4.

#### Tests

`docker_exec_test.go`, espejando el archivo fuente (M5). Dos tests, porque son dos propiedades distintas: el formato y la no-copia.

**El oráculo del formato es `archive/tar` haciendo el trabajo completo**, y el stream tiene que ser **byte por byte idéntico**. Lo intuitivo —desarmar el tar con `tar.Reader` y ver que el archivo salga bien— **no sirve**: `tar.Reader` da `io.EOF` limpio cuando el stream se termina, así que acepta un tar sin marcador de fin. Ese oráculo pasaría en verde justo con la mutación que más importa.

Cinco mutaciones, cada una en rojo exactamente donde corresponde:

| Mutación | Casos en rojo |
|---|---|
| el tail sin los dos bloques de fin de archivo | los **7** de la tabla |
| el tail sin el relleno del último bloque | **5** de 7 |
| se cae el `%512` de afuera | **solo 2**: `empty file` y `exactly one record` |
| vuelve el `bytes.Buffer` | **solo** `DoesNotCopyTheContent` |
| el contenido antes del header | los 7, **y además** `DoesNotCopyTheContent` |

Tres cosas se leen de ahí. La segunda mutación deja dos casos en verde y **está bien**: son justo aquellos donde el relleno vale cero, así que no hay nada que romper. La tercera pone en rojo sólo esos mismos dos, que es la demostración de que las filas de 0 y 512 bytes se ganan el lugar en la tabla. Y la cuarta es la que más dice: volver al `bytes.Buffer` deja el test de igualdad **en verde**, porque la implementación vieja produce un tar perfectamente válido — sólo que copiando. El test del formato, solo, nunca habría notado una regresión al comportamiento que este paso elimina.

### Paso 7 — Volumen compartido y comparación por tokens a pool liviano (D7, D6) ← SIGUIENTE

El `emptyDir` en `worker.yaml` montado en `dind` y en `worker`, el UUID por judging con la raíz no listable, `RunTestCase` escribiendo al volumen, y `copyOutput` eliminado. `CheckRequest` pasa de recibir bytes a recibir una ruta, y la comparación por tokens se muda a pool liviano con la imagen del paso 1.

**Es el paso más difícil de verificar localmente** — los tests del pool mockean Docker, así que la prueba real cae en la Fase 8 contra el cluster. Por eso va último: si algo queda mal, no contamina los pasos anteriores.

#### El paso se ejecuta en seis commits

Salió de cargar el contexto con el código delante, y cada uno deja el proyecto compilando y la suite en verde.

| | | |
|---|---|---|
| 0 | el orden de los argumentos de `compare` | ✅ |
| 1 | la plomería del volumen | ✅ |
| 2 | el corazón: la salida deja de pasar por el worker | ✅ |
| 3 | la comparación por tokens al pool liviano | ✅ |
| 4 | A5: la memoria consumida se mide de verdad | |
| 5 | el veredicto `OUTPUT_LIMIT_EXCEEDED` | |

**Y el encuadre de "difícil de verificar localmente" quedó corregido**: es cierto para la suite —los tests del pool mockean Docker— pero **no para los mecanismos**. Todo lo riesgoso de este paso se puede medir con Docker real, incluido levantar un `dind` de verdad y ver qué alcanza un container creado por el demonio de adentro. Lo que sí queda para la Fase 8 es la parte de Kubernetes propiamente dicha: que el `emptyDir` se comparta entre los dos containers del pod.

#### Lo que se decidió en el camino

**1. La topología de `docker-compose` no es la de Kubernetes, y el volumen tiene que servir a las dos.**

D7 razona sobre el pod de Kubernetes y nunca menciona el entorno local, donde `docker-compose.yml` **no tiene sidecar `dind`**: el worker monta el socket del demonio del host, así que los containers del sandbox son sus **hermanos** y el demonio resuelve el origen del montaje en el filesystem **del host**. Una ruta `/judging` que existe adentro del worker no existe ahí.

Medido con Docker real, levantando un `dind` y comparando las dos topologías:

| | origen = ruta `/judging` | origen = nombre de volumen |
|---|---|---|
| demonio adentro de `dind` (Kubernetes) | **funciona** | — |
| demonio del host (compose) | **0 archivos** | **funciona** |

**El modo de falla es el peor posible y es lo que fija todo lo demás**: Docker **crea el origen que no encuentra como un directorio vacío**, sin error y con exit 0. Medido. O sea que un montaje mal armado no rompe: hace que el checker lea archivos vacíos y que **todas las submissions den wrong answer**.

De ahí las tres decisiones:

- **El origen es configuración** (`JUDGE_VOLUME_SOURCE`), **requerida y sin default**: `/judging` en `worker.yaml`, el nombre del volumen en el compose. Un default habría hecho que un compose sin configurar cayera justo en el modo silencioso. Es la misma clase de valor que `DOCKER_HOST`, que ya vale distinto en cada topología y por esta misma razón.
- **La ruta es una constante en Go** (`pool.SharedVolumePath`), porque no cambia entre topologías ni adentro del worker ni adentro del sandbox. Se descartó ponerla en `judge_config.yaml`: ese archivo va **horneado en la imagen**, así que tendría el mismo valor en las dos topologías, que es justo lo que no puede ser.
- **El worker verifica el directorio al arrancar y no lo crea si falta.** Que falte significa que el volumen no está montado, que *es* el bug; crearlo lo taparía, exactamente el error que comete Docker. De paso le aplica el `0711` de D7.

**2. El montaje va en la config del pool, no en `Claim`.** Los montajes son **inmutables después de crear el container** y el pool reusa containers entre judgings. Es el espejo exacto de A1 del Paso 6: el límite de memoria *sí* se puede cambiar en caliente y por eso viaja en `Claim`; el montaje no, y por eso es del pool. Coincide con lo que D7 ya pedía — montaje uniforme, igual en todos los containers de un pool.

**3. El pool liviano monta en sólo lectura.** El experimento verificó los permisos de D7 y de paso marcó su límite: con la raíz en `0711` un sandbox **no puede enumerar** (`ls` da *Permission denied*), pero si conoce una ruta puede leerla **y escribirla**. Leer es el contrato que D7 acepta —la ruta es la credencial—, pero escribir significa poder **alterar el veredicto de otra submission**. El liviano nunca escribe en el volumen (lee el input y la salida del concursante; la respuesta del jurado le llega por la API), así que un `:ro` cierra esa mitad gratis. No rompe la uniformidad: cada pool tiene su config y adentro de cada uno el montaje sigue siendo idéntico.

Sube además la apuesta sobre la regla que D7 ya trae: **los UUID no pueden aparecer nunca en un log visible ni en un mensaje de error de la API**.

**4. Un hueco de tests que el barrido de mutaciones encontró.** La primera tanda dejó pasar en verde la mutación *"`poolConfigFor` no marca el liviano como sólo lectura"*: el test del pool probaba que el bind **respeta** la bandera, pero nada probaba que el composition root la **pone**. Es la forma exacta del bug de `CheckerLanguage` y del `RunRequest.MemoryKb` — un valor que no se pasa y que compila perfecto. Cubierto ahora en `cmd/worker/shared_volume_test.go`.

**Pendiente que abre este paso, para el commit 2**: la verificación **de punta a punta** al arrancar —escribir un marcador, crear un container y comprobar que lo ve— es lo único que atraparía un `JUDGE_VOLUME_SOURCE` con un valor *equivocado*, contra ausente, que sí se atrapa. No entró en el commit 1 porque necesita una imagen y un container y porque hasta el commit 2 el volumen no lo usa nadie. En el commit 2 el modo de falla se vuelve real.

**5. Lo que cruza la capa de aplicación es un token opaco, no rutas (commit 2).** Las dos sesiones —la del pool pesado y la del checker— salen de adapters distintos y tienen que apuntar al mismo directorio, y el único que conoce a las dos es el caso de uso. Se le pasa un UUID que **nunca interpreta**; el layout (`input.txt`, `output.txt`) es saber del adapter, compartido entre dos archivos del mismo paquete, que es lo que D3 pide: *"la reutilización ocurre en el adapter, no en el puerto"*. Se descartó pasar rutas explícitas: meten filesystem en la capa de aplicación y las acercan a los logs, que es justo lo que D7 prohíbe. Y se descartó un puerto propio con el patrón de `TransactionManager`: su ventaja real —la limpieza garantizada— se consigue igual desde `Session.Close`, que los dos casos de uso ya difieren.

Con eso **`CheckRequest` desapareció entero**, con el precedente exacto del Paso 4: de sus tres campos, dos se mudaron al volumen y `Check` quedó recibiendo sólo la respuesta del jurado. `RunResult.Output` pasó a `OutputPreview`.

**El emparejamiento pasa a ser implícito** —`Check` mira lo que la última `RunTestCase` dejó— y eso lo hace **más** seguro, no menos: antes el caso de uso tenía que pasar los bytes del mismo caso que acababa de correr, y `testCases[j]` por error era un bug silencioso. Ahora no hay nada que pasar. Es el argumento de D8: el bug deja de existir por construcción.

**6. La respuesta del jurado se queda en la API, y la alternativa quedó medida (commit 2).** Se propuso meterla también en el volumen, bajo un segundo nombre no adivinable (`<uuid>/<secreto>/answer.txt`). **El esquema funciona** —verificado: con la raíz del judging en `0111` un sandbox no puede listar, así que el secreto anidado es el mismo modelo de capacidad que el UUID de afuera—, pero el tipo de cambio no cierra: sobre los paquetes reales **un solo archivo son 28.75 de 28.8 MB**, y ése es el input, así que todas las respuestas juntas son unos 50 KB. Mover la respuesta compra ~50 KB de transferencia a cambio de degradar la garantía de D7 de *"no existe ninguna ruta que el container del concursante pueda alcanzar"* a *"existe, y la protege un nombre"*.

**Si algún día el número cambia, la vía correcta no es un secreto anidado sino un montaje que el pool pesado no tenga**: la respuesta en un volumen que sólo monta el liviano. El ahorro sería el mismo y no habría secreto que guardar. Cuesta un segundo volumen y un segundo valor de configuración en las dos topologías.

**7. La cuenta de "~70 a ~33 MB" de este documento ya no vale, y el desglose real es otro (commit 2).** Esa medición atribuía el ahorro a mandar el input por el volumen, pero contaba la duplicación de `buildTar`, que el **Paso 6.5 ya eliminó**. Con el código de hoy, sobre el problema real más pesado:

| Qué se mueve al volumen | Memoria del worker | CPU / API |
|---|---|---|
| la salida del concursante | **hasta 8 MiB por judging** | dos cruces por caso |
| el input | ninguna — `GetTestCases` lo sostiene igual | **2 × 28.75 MB por caso** |
| la respuesta del jurado | ninguna — mismo motivo | ~50 KB por judging |

O sea que del commit 2 el ahorro de **memoria** lo trae la salida, y el de **CPU** lo trae el input.

**8. El layout del directorio quedó más apretado de lo planeado, y salió de la misma discusión (commit 2).** El plan era que el directorio fuera del uid 1000 para que el `>` del shell pudiera crear `output.txt`. Medido, eso le da al concursante más de lo necesario: podía listar el directorio, crear archivos y **borrar su propio input**. Con el directorio de `root` en `0111` y `output.txt` **pre-creado** a nombre del uid 1000, la redirección sigue funcionando y el sandbox pierde las tres capacidades:

| | uuid del concursante (`0755`) | uuid de root (`0111`), `output.txt` pre-creado |
|---|---|---|
| `ls` del directorio | input.txt | Permission denied |
| redirigir a `output.txt` | OK | **OK** |
| crear un archivo nuevo | OK | Permission denied |
| borrar el input | OK | Permission denied |

**Y el worker sigue pudiendo escribir adentro porque es root**, verificado corriendo las mismas llamadas de Go que hace el adapter (`os.Mkdir` con el modo intacto tras el umask, `os.WriteFile`, `os.Chown`, `os.RemoveAll`). El control, como uid 1000, falla en todas — incluida la de crear un directorio propio bajo la raíz, así que un concursante tampoco puede plantar señuelos.

**9. La raíz del volumen pasó a ser un argumento del constructor (commit 2).** `judgingDir` empezó usando la constante absoluta `/judging` y eso deja **sin poder testear** todo el commit: ningún test puede escribir ahí, y en la máquina de un desarrollador sin root tampoco. `NewExecutor` y `NewOutputChecker` la reciben, el composition root les pasa `pool.SharedVolumePath`, y los tests usan `t.TempDir()`.

**10. `Session.Close` borra el directorio ANTES del guard del container (commit 2).** `Close` sale temprano cuando `s.container == nil`, que es lo que deja la red de seguridad al descartar un container. Con la limpieza detrás de ese guard, **cada judging que toca la red de seguridad dejaba un directorio para siempre** en el `emptyDir`. Tiene test propio.

**11. Dos huecos más que encontró el barrido de mutaciones (commit 2).**

- El test del layout aseveraba el modo contra **`judgingDirMode`, la constante que la mutación cambia**, así que ponerla en `0755` pasaba en verde. Es el mismo error que ya había aparecido en el paso 0 con `exitRejected`. Ahora compara contra el literal `0o111`.
- **Nada cubría que el nombre del directorio fuera aleatorio**, que es donde se apoya toda la seguridad de D7: una mutación a un nombre fijo pasaba. El test nuevo fija las dos mitades —que dos judgings no compartan directorio, y que el nombre **no sea el id de la submission**, que la API entrega—, y las dos mutaciones correspondientes quedan en rojo.

**12. El volumen habilita un streaming más fuerte que el que este documento descartó (commit 2, pendiente de evaluar).** El documento descartó el streaming de casos de prueba razonando que *"convierte O(suma) en O(caso mayor)"*, que no ayuda cuando un solo archivo **es** la suma. Eso vale para un iterador que materializa un caso a la vez; **escribiendo del ZIP directo a un archivo no se materializa ninguno**, y pasa a ser `O(buffer)`. Es la palanca que subiría el tope de datos de prueba, que hoy lo fija el worker sosteniendo todo en memoria.

**No hace falta el volumen para conseguirlo**: el ZIP trae el tamaño descomprimido en su header y `buildTar` sólo necesita el tamaño por adelantado, así que una variante que reciba un `io.Reader` más el tamaño deja pasar la respuesta del jurado por la API igual de sin-materializar. El streaming y el destino son decisiones independientes. **Queda para evaluar al terminar el paso.**

**13. `compare` declara su comando en el YAML, contra lo que D10 decía (commit 3).** D10 fija que *"`compare` aparece en `languages` con solo una imagen y ningún comando"*, y esa línea quedó desactualizada: se escribió antes de que existiera el mecanismo de invocación. Poner `/usr/local/bin/compare` en Go reintroduce exactamente lo que D9 sacó del código — lógica por lenguaje. Con `artifactRun: "/usr/local/bin/compare"` la maquinaria existente lo corre sin ninguna rama nueva: `withArtifactName` sobre un string sin `{name}` no hace nada, y las reglas de arranque que exigen los cuatro campos lo eximen **por construcción**, porque sólo aplican a lenguajes con `runCmd`.

La entrada queda deliberadamente parcial —imagen y `artifactRun`, nada más— y el YAML lo explica en el lugar, que es lo que D10 pedía documentar.

**14. `compare` en el pool liviano son 128 MiB, y el número dejó de ser ilustrativo (commit 3).** Medido contra la imagen real, con el container capado como lo capa el pool:

| | |
|---|---|
| salidas de 8 MiB, tokens normales | funciona en **16 MiB** de container |
| salidas de **64 MiB**, tokens normales | funciona en **16 MiB** |
| input de 64 MiB | irrelevante: **`compare` nunca abre el input** |
| un solo token de 8 MiB | 64 MiB sí, 48 MiB **OOM** |
| un token de 16 MiB o más | **exit 3** en 64 MiB, nunca OOM |

**Lo que esto corrige del documento**: el pendiente del tope por archivo dice que *"el pico de memoria del checker escala con la suma de los tres archivos"*. **Es falso para el camino por defecto**, que es la mayoría del tráfico. `compare` streamea con un buffer de 64 KB, así que su memoria sigue al **token más grande**, no a los archivos — y ese token está acotado por `maxTokenBytes` (16 MiB), una constante nuestra. Sigue siendo cierto para checkers personalizados, que leen los archivos enteros: la medición de D13 era de uno escrito así.

**Consecuencia para el futuro**: si alguna vez sube el tope de la respuesta, hay que redimensionar `cpp20`/`java17`/`python310` del pool liviano y el worker, pero **no `compare`**. Su techo está atado a `maxTokenBytes`, no a la configuración de la plataforma.

Los 128 MiB son 2× el piso medido del peor caso. Bajarlo a 64 no compra nada: el presupuesto liberado no se convierte en concurrencia, que la limita la CPU.

**15. El `exit 3` de `compare` se sigue leyendo como rechazo, y su comentario era el que estaba mal (commit 3).** `cmd/compare` distingue `0/1/3` a propósito y su comentario prometía que pasarse de `maxTokenBytes` *"is reported as a checker failure rather than silently as a wrong answer"*. `CheckerSession.Check` trata cualquier no-cero como rechazo, así que esa promesa era falsa en cuanto se cableara.

Se evaluó distinguirlo y **se descartó por un argumento que apareció al mirarlo**: el tamaño del token lo controla el concursante, y `Check` devolviendo error dispara el **reintento** — o sea que sería un amplificador, tres judgings completos por submission a pedido. Se corrigió el comentario en vez del comportamiento.

**Y de paso se cayó una hipótesis mía**: creí que distinguir el 3 atraparía el montaje roto del volumen. No lo hace — con el montaje roto el archivo existe pero **vacío**, así que `compare` sale con 1, no con 3. Eso sólo lo atrapa la sonda del punto 16.

**16. Dos guardas nuevas (commit 3).**

- **`compare` no puede faltar en la config.** Las reglas existentes no lo cubrían: alguien lo borra de `pools.light.languages` y la config sigue siendo válida, hasta que la primera submission sin checker personalizado falla con `unknown language` — en competencia, en el peor momento. Ahora `validateJudgeConfig` exige que esté declarado con `artifactRun` y dimensionado por el pool liviano.
- **La sonda de punta a punta del volumen**, que el commit 2 quedó debiendo. Al arrancar, el worker escribe un marcador en el volumen, reclama un container `compare`, lo lee desde adentro y compara. Es lo único que puede ver un `JUDGE_VOLUME_SOURCE` con un valor equivocado. Vive en el paquete del adapter y `main.go` la llama con una línea. Costo: ~220 ms una vez. **Canje aceptado**: el worker pasa a depender de Docker al arrancar, cosa que antes no hacía — en Kubernetes el `startupProbe` del dind ya lo garantiza, en local exige tener las imágenes construidas.


#### La entrada del caso de prueba SÍ viaja por el volumen — sub-decisión de D7, resuelta

D7 dejaba explícitamente abierto *"si la entrada del caso de prueba también viaja por el volumen o sigue por la API"*. **Va por el volumen**, por dos razones que aparecieron al desglosar la memoria del worker:

- **No hay problema de seguridad.** El argumento de D7 para excluir la salida esperada es que es secreta: si el código del concursante pudiera leerla, le bastaría imprimirla. **La entrada no lo es** — el programa del concursante la recibe por stdin por definición, así que ponerla en el directorio del judging no le filtra nada que no vaya a leer igual. La salida esperada sigue viajando por la API, como D7 punto 5 exige.
- **Es el 82% del pico de memoria del worker** en el peor problema real. Hoy el input se copia **dos veces** por caso: una al container pesado (`RunTestCase`) y otra al liviano (`Check`), y cada copia paga además la duplicación de `buildTar`. Sobre el problema B de los paquetes reales (input de 28.75 MB), el pico por judging baja de ~70 MB a ~33 MB solo con esto.

**El volumen tiene que estar respaldado por disco, nunca `medium: Memory`.** Un `emptyDir` normal vive en el almacenamiento efímero del nodo, y leerlo pone sus páginas en el page cache — que **sí** se le cargan al cgroup, pero son **reclamables**, y por eso no le quitan memoria a la solución (ver A3). Un `emptyDir: {medium: Memory}` es un **tmpfs**: sus páginas también se cargan al cgroup y **no son reclamables** (solo podrían irse a swap, que está desactivado a propósito para que el MLE sea determinista).

Medido, con un input de 64 MiB en un container de 256 MiB:

| Dónde vive el input | La solución llega a reservar |
|---|---|
| disco / overlayfs (`emptyDir` normal) | **254 MiB** |
| tmpfs (`emptyDir: {medium: Memory}`) | **190 MiB** |

Pierde exactamente el tamaño del archivo. Si alguien declarara el volumen como tmpfs buscando velocidad, **cada byte de entrada y de salida pasaría a descontarse del límite de memoria del concursante**, y el veredicto quedaría atado al tamaño de los casos de prueba. El `dind-storage` que ya existe es `emptyDir: {}`, así que el precedente está bien; falta que quede escrito antes de agregar el volumen nuevo.

**Lo que el volumen NO elimina**: los casos de prueba se siguen descomprimiendo enteros en memoria del worker, porque el worker es quien lee el ZIP de GCS y escribe los archivos al volumen. Después de este paso ese pasa a ser el **único** término dominante de su consumo, y lo acota el tope de 100 MB de D14.

#### Lo demás que este paso arrastra

- ~~**`compare` hay que agregarlo a `languages` Y a `pools.light.languages`**~~ **Hecho en el commit 3**, y el modo de falla que preocupaba —`unknown language` en la primera submission— pasó a ser una regla de arranque (ver el punto 16). El presupuesto del pool liviano que fijó D14 lo contemplaba: 6.25 GiB contra los 6.125 que N3 pide con `compare` adentro.
- **La apertura de sesión de pool liviano sigue duplicada** entre `OutputChecker.BeginChecking` y `ValidatorRunner.BeginValidating`. El Paso 5 extrajo la mitad de abajo (`artifactSession`) y la descarga (`downloadArtifact`) pero no ésta, porque este paso vuelve a tocar los dos archivos. Mirarla al terminar.


#### Acá entra la medición de memoria consumida (A5, diferido desde el Paso 6)

**Qué está roto hoy.** `Session.readStats` devuelve `int(stats.MemoryStats.MaxUsage / 1024)`, y **`MaxUsage` viene en cero en cgroup v2** — la clave `max_usage` directamente no aparece en el JSON de la API de Docker, es de v1. El cluster es cgroup v2 (`docker:27-dind` sobre nodos COS), así que **todo veredicto reporta 0 KB**.

Y se guarda peor de lo que parece: `s.memoryKb = &memoryKb` toma la dirección del parámetro, así que un `0` se persiste como **0, no como NULL**. La API devuelve `"memoryKb": 0`, que el concursante lee como *"tu solución usó 0 KB"*, no como *"no lo medimos"*. Las únicas transiciones que dejan NULL son `MarkTimeLimitExceeded` y `MarkCompilationError`, que no tocan el campo.

**Por qué aterriza en este paso y no en el 6.** Medir no es el problema: el mecanismo está diseñado y verificado abajo. El problema es **sacar el número del container**, que es exactamente el código que este paso reescribe — hacerlo en el Paso 6 significaba escribir esa plomería dos veces, y con la variante peor.

##### El mecanismo, ya verificado

**`/usr/bin/time -f %M`**, que reporta el `ru_maxrss` del proceso: el pico de RSS, aislado por corrida. Es lo que hacen los jueces clásicos.

**Los exit codes se preservan** — el obstáculo que mató la idea del `| head -c` del Paso 5. Medido con y sin el envoltorio: `0`, `124` (TLE) y `137` (MLE) llegan intactos en las dos formas de anidarlo.

**`time` va POR FUERA del `timeout`**, no adentro:

```
/usr/bin/time -f %M -o <archivo>  timeout --kill-after=1s Ns  CMD < in > out 2>/dev/null
```

Con `time` adentro, un TLE mata también a `time` y **no se escribe ninguna medición**. Con `time` afuera sobrevive y mide en los tres casos (éxito, TLE, MLE). Medido.

**La exactitud, por lenguaje** (container de 512 MiB, reservas de 20/50/100 MiB):

| Lenguaje | `ru_maxrss` − reserva real |
|---|---|
| cpp20 | **1 MiB**, constante |
| python310 | **7 MiB**, constante (el intérprete) |
| java17 | 38-62 MiB, creciente (la JVM) |

**Y por qué no sirve el pico del cgroup**, medido en la misma corrida: marcó 176-269 MiB sin importar lo que la corrida reservara, porque venía contaminado por el `g++`/`javac` que había compilado en ese mismo container. Es la contaminación que D11 describe, vista en vivo. Sumado a que `memory.peak` **no se puede resetear** (reset desde Linux 6.11, y Docker monta `/sys/fs/cgroup` de solo lectura), la vía del cgroup está cerrada.

##### Las dos decisiones que quedan, con su análisis hecho

**1. Cómo llega la herramienta al container.** `/usr/bin/time` **no está en ninguna de las tres imágenes** (son Ubuntu; el paquete es `time`).

| | `apt install time` | binario propio, copiado al sandbox |
|---|---|---|
| Imágenes | 3 Dockerfiles + `RUNNER_VERSION` + push | ninguna |
| Despliegue | coordinado: las imágenes antes que el código | normal |
| Código | solo el comando | un `cmd/` nuevo, embeberlo y copiarlo por sesión |
| Precedente | `testlib.h`, pendiente por exactamente esto | `cmd/compare`, que ya existe |

El binario propio sería un programa mínimo que hace `fork` + `wait4` y reporta `ru_maxrss`; estático (`CGO_ENABLED=0`) corre igual en las imágenes Ubuntu del judge y en la Alpine del backend.

**2. Cómo vuelve el número al worker.** Acá es donde este paso lo abarata:

- Un `CopyFromContainer` extra **por caso de prueba** (~17 ms × N; con 25 casos, ~425 ms por judging).
- Rutearlo por el stream de stderr del exec, guardando el exit code a mano (`C=$?; cat ...; exit $C`). Sin round trip extra, pero es gimnasia de shell dentro del template del comando.
- **Escribirlo al directorio del judging en el volumen compartido y leerlo de ahí: gratis**, cero llamadas a Docker. Es la opción que este paso habilita, y la razón por la que A5 se difirió hasta acá.

##### La superficie que cambia

`memoryKb` **no alimenta ninguna lógica**: fuera del judging solo se transporta al JSON. Las standings y el ranking ICPC miran el **status**, nunca el consumo; la única comparación en todo el backend es el máximo entre casos de `judge_submission.go`. O sea que es puramente informativo, y el cambio no puede alterar ningún resultado.

Se muestra en 4 tipos y 5 endpoints:

| Endpoint | Tipo |
|---|---|
| `GET /submissions/{id}` | `handler/submission/get_submission_handler.go` |
| `GET /users/me/submissions` | `handler/submission/types.go` |
| `GET /problems/p/{slug}/submissions` | idem |
| `GET /groups/{id}/contests/{id}/submissions` | `handler/contest/types.go` |
| `GET /users/me/dashboard` | `handler/user/dashboard_handler.go` |

Los cinco declaran el campo como `*int`, así que **aceptan `null` sin cambio de contrato**.

##### Por qué no se puso `NULL` mientras tanto

Se evaluó dejar el campo en `NULL` en vez de en `0` hasta que la medición exista, para quitar de la API la afirmación falsa. **Se descartó**: los cuatro `Mark*` que tocan memoria reciben `memoryKb int`, no `*int`, así que dejar NULL exige cambiar la firma del dominio, sus call sites y sus tests — un cambio de la API del dominio, más un paso intermedio que este paso deshace, para un campo que acá pasa a tener un valor real. El `0` queda como está hasta entonces.
#### Acá entra el veredicto `OUTPUT_LIMIT_EXCEEDED` (diferido desde el Paso 5)

**Por qué acá y no en el Paso 5.** Hasta este paso, `maxOutputBytes` **no es un límite de salida sino un tope de lectura del worker**: el comando de `RunTestCase` redirige a `/sandbox/output.txt` sin ningún tope, así que un programa que imprime sin parar llena el archivo igual y la constante solo evita que el worker se traiga todo a memoria. Un veredicto construido sobre ese número le diría al concursante "excediste el límite de salida" cuando el sistema en realidad nunca se lo impuso. En el Paso 7 la salida pasa al volumen compartido y el límite **hay que aplicarlo de verdad** —el `emptyDir` es finito y lo comparten todos los judgings—, así que recién ahí el veredicto significa lo que dice.

**Se evaluó y se descartó aplicarlo ya en el Paso 5** con `| head -c 8M` dentro del `sh -c`: el exit code del pipeline pasa a ser el de `head`, y eso rompe la detección de TLE (124) y MLE (137), que es de lo que dependen los dos casos de uso para dar veredicto. Cualquier solución que no rompa eso es del tamaño del Paso 7.

**Mientras tanto (desde el Paso 5) la truncación se loguea** con `slog.WarnContext` en `copyOutput`, para que el caso deje evidencia en vez de aparecer como un wrong answer inexplicable. Es un paliativo explícito, no la solución.

**Receta concreta, para no re-investigar.** El veredicto **no necesita migración**: `submissions.status` es `TEXT NOT NULL` sin CHECK ni tipo enum (`cmd/migrate/migrations/012_create_submissions_table.sql:9`). Lugares a tocar:

1. `internal/domain/submission/status.go` — la constante `statusOutputLimitExceeded`, la factoría privada `newStatusOutputLimitExceeded`, y **los dos `switch`**: el de `NewStatus` y el de `IsFinal`. Olvidar el de `IsFinal` deja el veredicto fuera de "terminado" y lo agarraría el barrido de stale.
2. `internal/domain/submission/submission.go` — `MarkOutputLimitExceeded`, siguiendo la forma de `MarkWrongAnswer`.
3. `internal/application/judge/judge_submission.go` — detectar el exceso en el resultado de la corrida y marcar.
4. `internal/application/judge/validate_solutions.go` — un `FailureKind` nuevo; si no, la solución del propio setter que imprime de más se reporta como wrong answer.
5. `internal/application/problem/solution_validator.go` — el `SolutionFailureKind` espejo del anterior.
6. **`internal/application/contest/get_standings.go`** — la lista literal de veredictos que cuentan como intento penalizado (`case "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", ...`). Es el que se olvida: si no se agrega, un OLE no penaliza y nada avisa. **Buscar por el literal, no por el tipo**, porque son strings sueltos.
7. La detección misma: en el Paso 7 sale de acotar la escritura en el container, no de mirar cuánto leyó el worker.

Fuera de este repo queda el frontend, que muestra los veredictos.

**Dos cosas que hay que decidir acá y que este documento todavía no plantea:**

**1. Con qué mecanismo se acota la escritura.** El punto 7 de arriba dice "sale de acotar la escritura en el container" sin decir cómo, y la vía obvia ya está descartada: `| head -c 8M` convierte el exit code del pipeline en el de `head` y rompe la detección de TLE (124) y MLE (137). Un candidato que no tiene ese problema es **`ulimit -f`** dentro del `sh -c`: el kernel manda `SIGXFSZ` al proceso que se pasa del tamaño, lo que da un exit code propio (128+25 = 153) en vez de pisar el del comando, y no introduce ningún pipeline. Hay que verificarlo corriéndolo —incluido qué hace la JVM con esa señal, que es donde esta clase de cosas se rompe— antes de darlo por bueno.

**2. ~~Si 8 MiB sigue siendo el número correcto.~~ ✅ RESUELTA en el commit 3: se queda en 8 MiB.** Se volvió a mirar con el cambio de significado encima —pasa de tope de lectura interno del worker a límite real impuesto al concursante— y ahora hay el dato que faltaba: **medido sobre los 12 paquetes reales de la competencia**, la respuesta esperada más grande es la del problema H con **2 068 944 B ≈ 1.97 MiB**; el resto están por debajo de 120 KB. O sea que 8 MiB dan **4× de margen sobre la peor salida correcta real**, y hay un problema que de verdad usa el 25% del límite: no es un número teórico.

Se sostiene por tres razones a la vez: ese 4×, la coincidencia con el default de DOMjudge (`output_limit` 8192 kB), y que es el valor más alto que mantiene cómodos a los checkers **personalizados** con el dimensionamiento actual del pool liviano — a 16 MiB el pico de Python sube a 320 MiB contra un techo de 512 (D13), que es lo que hizo descartar 16 en el Paso 5.

**El caveat**: el margen es 4×, no 100×. Si aparece un problema cuya salida correcta pase de ~2 MiB de forma significativa, éste es el número a revisar, y no se revisa solo — subirlo obliga a redimensionar `cpp20`/`java17`/`python310` en el pool liviano, que sí escalan con el archivo. `compare` no (punto 14).

*(Como control de la medición: el input más grande dio 28.75 MiB en el problema B, exactamente lo que este documento ya tenía anotado por otra vía.)*

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
- ~~`artifactInvocation` y sus tests~~ **borrado en el Paso 5**
- ~~`isTimeoutErr` y `trustedSubprocessTimeout` (`judging_timeouts.go` entero)~~ **borrado en el Paso 5**. Con eso `os/exec` **no aparece más en `adapter/judge`**, que es la prueba concreta de que ningún código del problem setter corre ya en el filesystem del worker.
- ~~`native_compiler.go` + `_cpp/_java/_python` y sus tests~~ **borrado en el Paso 3**
- ~~`validator_runner.go` (versión nativa) y sus tests~~ **reemplazado en el Paso 4**

*Probablemente huérfanos tras los cambios*
- `tokenCompare` — **NO queda huérfano en el Paso 5**: la comparación por tokens sigue en el worker hasta el Paso 7. Vive aislada en `token_comparison.go` junto a `tokenCheckerSession`, para que el Paso 7 borre el archivo entero.
- `copyOutput`, y `maxOutputBytes` si deja de tener llamadores (la salida pasa por el volumen)
- ~~`CheckerFilename` en `ProblemLimits` y en `CheckRequest`~~ **eliminado en el Paso 5**
