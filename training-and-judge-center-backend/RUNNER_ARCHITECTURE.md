# Runner — Arquitectura

> Secciones marcadas con `[ pendiente ]` se completan en discusiones posteriores.

---

## 1. Visión General

El runner es un worker independiente que consume submissions desde una cola de prioridades, ejecuta el código en contenedores Docker aislados, y escribe el veredicto en la base de datos.

```
┌─────────────────────────────────────────────────────────────────────┐
│                          K8s Cluster                                │
│                                                                     │
│  ┌─────────────┐     ┌──────────────┐     ┌──────────────────────┐  │
│  │  Backend    │────▶│  RabbitMQ    │────▶│  judge-worker (Pod)  │  │
│  │  API        │     │  (cola)      │     │                      │  │
│  └─────────────┘     └──────────────┘     │  goroutine pool      │  │
│                            ▲              │  container pool      │  │
│                       HPA watches         │    cpp20 containers  │  │
│                       queue depth         │    java17 containers │  │
│                            │              │    pythonXXX ...     │  │
│                            └──── escala ──┤                      │  │
│                                           └──────────────────────┘  │
│                                                    │                │
│                                             ┌──────▼──────┐        │
│                                             │  Postgres   │        │
│                                             │  GCS        │        │
│                                             └─────────────┘        │
└─────────────────────────────────────────────────────────────────────┘
```

### Componentes

| Componente | Responsabilidad |
|---|---|
| **Backend API** | Crea la submission, la encola en RabbitMQ |
| **RabbitMQ** | Cola de prioridades (1=contest activo, 2=postcompetición, 3=práctica, 4=bulk rejudge) |
| **judge-worker pod** | Consume mensajes, compila, ejecuta, escribe veredicto |
| **Container pool** | Pool de contenedores Docker por lenguaje dentro de cada pod |
| **HPA** | Crea/destruye pods según profundidad de la cola |

---

## 2. Modelo de Despliegue

### Dos niveles de escalado independientes

```
Nivel K8s — lento (10-30s), responde a carga sostenida
  HPA observa: mensajes sin procesar en RabbitMQ
  Acción: crear/destruir pods completos

Nivel pod — rápido (<1s), responde a bursts internos
  Pool manager observa: submissions esperando container
  Acción: crear/destruir containers de lenguaje dentro del pod
```

Estos dos niveles no interfieren entre sí. El pod no sabe que K8s existe. K8s no sabe qué hay dentro del pod.

### Pods políglotas (no especializados por lenguaje)

Cada pod puede juzgar cualquier lenguaje. Un pod no está dedicado a cpp20 ni a java17.

**Por qué políglotas y no especializados:**
- La distribución de lenguajes es impredecible y varía por concurso
- Con pods especializados, los pods cpp20 estarían saturados mientras los java17 están idle aunque ambos grupos de usuarios están esperando
- Cualquier pod libre atiende cualquier submission, maximizando la utilización
- Si en el futuro el análisis de datos muestra que un lenguaje domina consistentemente, se puede separar entonces — no antes

### Escalado por profundidad de cola

```
mensajes_pendientes > (maxConcurrent × pods_activos × threshold)
        ↓
HPA crea nuevo pod
        ↓
Nuevo pod arranca vacío (cero containers de lenguaje)
        ↓
Empieza a consumir y crea containers on demand
```

El pod no necesita señalizar explícitamente que está lleno. Si no puede procesar más rápido, la cola crece, y el HPA reacciona.

---

## 3. Modelo de Concurrencia por Pod

### Dos recursos, dos mecanismos

La concurrencia efectiva del pod está acotada por dos recursos independientes:

| Mecanismo | Recurso que protege | Cómo se deriva |
|---|---|---|
| Contabilidad de memoria en el pool | RAM — evita OOM | `POD_MEMORY_LIMIT` vía K8s Downward API |
| Semáforo `maxConcurrent` | CPU — evita over-subscription | `POD_CPU_LIMIT` vía K8s Downward API |

Ninguno se configura manualmente. Ambos se derivan de los resource limits declarados en el pod spec de K8s.

### Por qué el semáforo sigue siendo necesario

La contabilidad de memoria limita cuántos containers pueden existir, pero no acota la CPU. Con lenguajes de baja memoria como Python (1Gi), un pod de 10Gi podría alojar hasta 9 containers simultáneos. Si el pod solo tiene 4 CPU cores, ejecutar 9 judgings en paralelo produce contención severa de CPU, distorsiona los tiempos de ejecución medidos y genera veredictos TLE incorrectos.

El semáforo previene este caso acotando los judgings activos al número de CPU cores disponibles.

### Derivación del semáforo desde el pod

```
maxConcurrent = max(1, floor(POD_CPU_LIMIT) - cpuOverheadCores)   # cada judging ocupa 1 core
```

`cpuOverheadCores` (default 1) reserva capacidad para los otros dos consumidores de CPU del pod:
el proceso worker (descarga de test cases, comparación de salidas, escritura de veredictos) y el
daemon Docker (creación de execs, streaming). Es el análogo en CPU del `memoryOverheadBytes` que
la contabilidad de memoria ya reserva para el worker: ningún recurso del pod se reparte al 100%
entre los judgings — la casa siempre come primero.

Pod con 4 CPUs → semáforo de 3. Pod con 8 CPUs → semáforo de 7. Si se cambia el CPU limit del pod en K8s, el semáforo se ajusta en el siguiente arranque sin tocar configuración.

Sin la reserva, 4 judgings a CPU llena + worker comparando salidas + daemon creando execs
compiten por 4 cores: el scheduler roba ciclos a procesos de estudiantes mientras su reloj corre.
La reserva hace la contención *rara*; la medición por tiempo de CPU (ver EXECUTOR_ADAPTER_PLAN,
RunTestCase) hace que la contención residual sea *inofensiva para el veredicto*. Defensa en profundidad.

La concurrencia real queda acotada por el mínimo de ambos recursos:

```
concurrencia efectiva = min(
    limitada por CPU      ← semáforo derivado de POD_CPU_LIMIT,
    limitada por memoria  ← canCreate() accounting con POD_MEMORY_LIMIT
)
```

### Goroutines de worker

El consumer de RabbitMQ corre en un único goroutine y lee mensajes. Por cada mensaje, lanza un goroutine que:

```
1. Adquiere slot del semáforo (bloquea si está lleno)
2. Reclama container del pool para el lenguaje de la submission
3. Juzga la submission
4. Libera el container al pool
5. Libera el slot del semáforo
6. ACK del mensaje a RabbitMQ
```

```
RabbitMQ consumer goroutine
    │
    ├── mensaje cpp20 → goroutine A → [semáforo] → [pool cpp20] → judge
    ├── mensaje java17 → goroutine B → [semáforo] → [pool java17] → judge
    ├── mensaje cpp20 → goroutine C → [semáforo] → [pool cpp20] → judge
    └── mensaje python → goroutine D → [semáforo] → [pool python] → judge
                                         (4/4 slots ocupados — próximo goroutine bloquea)
```

---

## 4. Container Pool

### Inicialización lazy

El pod **no arranca con containers de lenguaje**. Los crea cuando llega la primera submission de ese lenguaje. Esto evita tener containers idle de todos los lenguajes configurados aunque nunca lleguen submissions de alguno de ellos.

```
Pod arranca → pool vacío (0 containers)

Primera submission cpp20 llega → pool crea container cpp20 (~300ms)
Segunda submission cpp20 llega → pool tiene container idle → lo reclama (0ms)
Primera submission java17 llega → pool crea container java17 (~300ms)
```

El cold start de 300ms solo ocurre una vez por lenguaje en la vida del pod. Después, el container se reutiliza.

### Capacidad basada en memoria del pod

En lugar de configurar `maxContainersTotal` como un número arbitrario, el pool manager lleva contabilidad real de la memoria asignada:

```
podMemoryLimit  = memoria declarada en el pod spec (K8s Downward API)
overhead        = memoria reservada para el proceso worker (configurable)
allocatedMemory = Σ(memoryLimit de cada container vivo en este pod)

canCreate(language) = (podMemoryLimit - overhead - allocatedMemory) ≥ language.memory
```

**Ejemplo:**
```
podMemoryLimit = 10Gi, overhead = 512Mi

Containers vivos:
  java17-A  → 2Gi
  java17-B  → 2Gi
  cpp20-C   → 2Gi
  python-D  → 1Gi
  ─────────────────
  allocatedMemory = 7Gi

available = 10 - 0.5 - 7 = 2.5Gi

canCreate(cpp20 / 2Gi)?    → SÍ
canCreate(java17 / 2Gi)?   → SÍ
canCreate(rust / 3Gi)?     → NO (2.5 < 3.0) → activa LRU eviction
```

**Cómo el pod conoce sus límites — K8s Downward API:**

```yaml
# pod spec
env:
  - name: POD_MEMORY_LIMIT
    valueFrom:
      resourceFieldRef:
        resource: limits.memory
  - name: POD_CPU_LIMIT
    valueFrom:
      resourceFieldRef:
        resource: limits.cpu
  - name: POD_MEMORY_OVERHEAD
    value: "536870912"   # 512Mi en bytes
```

Para Docker Compose (desarrollo local):
```yaml
judge-worker:
  environment:
    - POD_MEMORY_LIMIT=8589934592   # 8Gi en bytes
    - POD_CPU_LIMIT=4
    - POD_MEMORY_OVERHEAD=536870912
```

Si se cambian los resource limits del pod en K8s, tanto el semáforo como la contabilidad de memoria se ajustan en el siguiente arranque sin tocar configuración.

### Idle timeout por container individual

Cada container es una entidad independiente con su propio timestamp de último uso. Un goroutine reaper corre en background y destruye containers individualmente:

```
cpp20-A  [idle, lastUsed: 10:00]
cpp20-B  [idle, lastUsed: 10:08]   ← terminó 8 minutos después que A
java17-C [busy]
python-D [idle, lastUsed: 10:05]

Reaper corre cada 30s con idleTimeout = 10min:

  t=10:10 → cpp20-A: 10min idle → destruido, allocatedMemory -= 2Gi
  t=10:15 → python-D: 10min idle → destruido, allocatedMemory -= 1Gi
  t=10:18 → cpp20-B: 10min idle → destruido, allocatedMemory -= 2Gi
```

Cada container muere en su propio momento, independientemente de los demás del mismo lenguaje.

### LRU Eviction

Cuando `canCreate(language)` retorna false pero hay containers idle, el pool destruye el container idle con `lastUsedAt` más antiguo para liberar memoria. Repite hasta poder crear el container solicitado:

```
Estado: 10Gi limit, 9.5Gi allocated, 0.5Gi available

  cpp20-A  [idle, lastUsed: 09:00]  ← más viejo
  cpp20-B  [idle, lastUsed: 09:45]
  java17-C [busy]
  python-D [busy]

Llega submission rust (necesita 2Gi):

  canCreate(rust / 2Gi)? → NO (0.5 < 2.0)
  LRU idle → cpp20-A → destruido → available = 2.5Gi
  canCreate(rust / 2Gi)? → SÍ
  Crear container rust
```

**Cuándo LRU no puede ayudar**: si todos los containers están busy (ninguno idle), no hay nada que evictar. El goroutine bloquea esperando a que alguno termine. La cola de RabbitMQ crece → HPA crea nuevo pod.

### Flujo completo de Claim

```
Claim(language):
  1. Buscar container idle del lenguaje → si existe, retornar (camino rápido)
  2. canCreate(language)?
       SÍ → crear container, retornar
       NO → ¿hay containers idle de cualquier lenguaje?
               SÍ → LRU eviction hasta canCreate → crear container
               NO → bloquear hasta que algún container se libere → volver al paso 1
```

### Estructura interna del Container

```
Container:
  id          string        ← ID del container en Docker
  language    string        ← "cpp20", "java17", etc.
  memoryBytes int64         ← límite de memoria de este container
  state       idle | busy
  lastUsedAt  time.Time     ← por container, independiente de sus pares del mismo lenguaje
```

---

## 5. Imágenes Docker por Lenguaje

### Una imagen por lenguaje

Cada lenguaje tiene su propia imagen Docker, construida sobre una imagen base común:

```
judge-runner:base
  Ubuntu 22.04
  Usuario judge (uid=1000, sin root)
  Directorio /sandbox creado
  Configuración de seguridad base

judge-runner:cpp20      FROM judge-runner:base + g++ 12
judge-runner:java17     FROM judge-runner:base + OpenJDK 17
judge-runner:python310  FROM judge-runner:base + PyPy 3.10
judge-runner:rust       FROM judge-runner:base + rustc   ← ejemplo futuro
```

Agregar un lenguaje nuevo = construir una imagen nueva. No requiere tocar el código del runner.

### Pool manager data-driven

El pool manager no tiene los lenguajes hardcodeados. Lee la configuración al arrancar y crea pools para lo que encuentre:

```
❌ Hardcoded — no escala con nuevos lenguajes
func (p *Pool) init() {
    p.createPool("cpp20", ...)
    p.createPool("java17", ...)
}

✅ Data-driven — agregar lenguaje = agregar entrada en YAML
func (p *Pool) init(cfg Config) {
    for lang, langCfg := range cfg.Languages {
        p.registerLanguage(lang, langCfg)
    }
}
```

### Configuración de lenguajes (YAML)

```yaml
judge:
  # maxConcurrent se deriva de POD_CPU_LIMIT al arrancar — no se configura aquí
  idleTimeoutMinutes: 10         # timeout de container idle (por container individual)
  memoryOverheadBytes: 536870912 # 512Mi reservado para el proceso worker
  cpuOverheadCores: 1            # cores reservados para worker + daemon Docker (se restan del semáforo)

  languages:
    cpp20:
      image: "judge-runner:cpp20"
      cpu: "1"
      memoryBytes: 2147483648        # 2Gi
      compileCmd: "g++ -std=c++20 -O2 -o /sandbox/solution /sandbox/solution.cpp"
      runCmd: "/sandbox/solution"
      extension: "cpp"

    java17:
      image: "judge-runner:java17"
      cpu: "1"
      memoryBytes: 2147483648        # 2Gi — más alto por el JVM
      compileCmd: "javac -encoding UTF-8 /sandbox/Solution.java"
      runCmd: "java -Xmx{memoryLimit}m Solution"
      extension: "java"

    python310:
      image: "judge-runner:python310"
      cpu: "1"
      memoryBytes: 1073741824        # 1Gi
      compileCmd: ""                 # vacío = lenguaje interpretado
      syntaxCheckCmd: "pypy3 -m py_compile /sandbox/solution.py"
      runCmd: "pypy3 /sandbox/solution.py"
      extension: "py"
```

---

## 6. Cola de Mensajes — RabbitMQ

### Por qué RabbitMQ

| | RabbitMQ | Google Pub/Sub |
|---|---|---|
| Prioridad nativa | ✅ Cola con prioridad 1-10 | ❌ Simulada con múltiples subscriptions |
| Funciona local | ✅ Docker, sin emulador | ⚠️ Necesita emulador |
| At-least-once | ✅ ACK/NACK explícito | ✅ |
| Go SDK | ✅ `amqp091-go` | ✅ |

Los 4 niveles de prioridad del spec mapean directamente a una sola cola con prioridad numérica. Con Pub/Sub se necesitarían 4 subscriptions separadas y lógica de preferencia manual en el worker.

### Compatibilidad con GCP

RabbitMQ no es un servicio nativo de GCP pero tiene dos rutas limpias:

| Opción | Descripción |
|---|---|
| **CloudAMQP** | RabbitMQ como servicio sobre infraestructura GCP, tier gratuito disponible. Cliente Go idéntico. |
| **GKE + Helm** | `bitnami/rabbitmq` chart oficial dentro del propio cluster. |

La ventaja del diseño hexagonal: si en el futuro se migra a Pub/Sub, solo cambia el adapter del consumer. El use case no se toca.

### Formato de mensaje

```json
{
  "submissionId": "abc123-def456",
  "priority": 1,
  "enqueuedAt": "2026-01-24T10:30:00Z",
  "metadata": {
    "contestId": "contest-123",
    "problemId": "problem-456",
    "userId": "user-789",
    "language": "cpp20"
  }
}
```

### Prioridades

| Valor | Tipo |
|---|---|
| 1 | Contest ACTIVE (submission normal + rejudge) |
| 2 | Postcompetición |
| 3 | Práctica |
| 4 | Bulk rejudge |

---

## 7. Protocolo Worker ↔ Container

### El primitivo: docker exec vía Go SDK

Los containers de lenguaje solo corren `sleep infinity`. No tienen ningún proceso escuchando. El worker usa el Docker Go SDK para enviarles comandos directamente, equivalente a `docker exec` pero programático:

```
Worker (Go)
    │
    ├── docker.CopyToContainer(containerID, "/sandbox", sourceCode)
    ├── docker.ContainerExecCreate(containerID, "g++ ...")
    ├── docker.ContainerExecStart(execID)
    └── lee stdout/stderr + exit code del resultado
```

Para mover archivos: `CopyToContainer` (worker → container) y `CopyFromContainer` (container → worker).

---

### Diseño del port: sesión de ejecución

El `JudgeSubmissionUseCase` no sabe nada de Docker ni de containers. Solo habla con una interfaz. El problema de diseño: compilación y ejecución deben ocurrir en el mismo container — el binario compilado en fase 2 se usa en fase 3. El use case necesita expresar esa afinidad sin saber que existe un container.

**Solución: sesión explícita**

```go
// application/judge/executor.go

type ExecutionSession interface {
    Compile(ctx context.Context, req CompileRequest) (CompileResult, error)
    RunTestCase(ctx context.Context, req RunRequest) (RunResult, error)
    Close(ctx context.Context) error   // cleanup + devuelve container al pool
}

type Executor interface {
    BeginSession(ctx context.Context, language Language) (ExecutionSession, error)
}
```

El use case lo usa así:

```go
session, err := executor.BeginSession(ctx, submission.Language())
if err != nil { ... }
defer session.Close(ctx)   // garantiza cleanup aunque falle algo

compileResult, err := session.Compile(ctx, CompileRequest{Source: sourceCode})
// si CE → return

for _, tc := range testCases {
    result, err := session.RunTestCase(ctx, RunRequest{Input: tc.Input, Limits: limits})
    // si no es AC → break
}
```

`BeginSession` = reclamar container del pool (con LRU si es necesario).
`defer session.Close(ctx)` = garantiza que el container siempre vuelve al pool limpio.

---

### Fase de compilación

```
1. CopyToContainer: sourceCode → container:/sandbox/solution.cpp
2. exec: "g++ -std=c++20 -O2 -o /sandbox/solution /sandbox/solution.cpp"
         timeout: 30s (fijo, no configurable por problema)
         captura: stdout+stderr (log de compilación, truncado a 10KB)
3. exit code != 0 → CompileResult{Success: false, Log: ...}
   exit code == 0 → binario queda en /sandbox/solution, continúa
```

---

### Fase de ejecución por test case

```
1. CopyToContainer: input → container:/sandbox/input.txt

2. exec: sh -c 'timeout --kill-after=1s {limitSeconds}s
                /sandbox/solution < /sandbox/input.txt > /sandbox/output.txt 2>/dev/null'
         Go context: context.WithTimeout(timeLimit × 2)   ← safety net
         captura: exit code

3. CopyFromContainer: container:/sandbox/output.txt → worker (truncado a 64MB)

4. exec: rm /sandbox/input.txt /sandbox/output.txt
         limpia entre test cases, el binario se conserva para el siguiente
```

#### Timeout: dos capas con responsabilidades distintas

| Capa | Mecanismo | Se activa | Propósito |
|---|---|---|---|
| Primaria | `timeout` command dentro del container | A los `timeLimit` segundos | Da el veredicto TLE al usuario |
| Safety net | `context.WithTimeout(timeLimit × 2)` de Go | Si el exec no retorna | Protege la infraestructura del worker |

El safety net nunca debería activarse en operación normal. Si se activa, es señal de fallo de infraestructura — el container se marca como dañado, se destruye, y la submission recibe `SYSTEM_ERROR`.

Exit codes del `timeout` command:
```
124 → TLE (tiempo agotado)
137 → MLE (proceso matado por cgroup, OOM)
0   → éxito
otro → RUNTIME_EXCEPTION
```

#### Memoria: dos concerns distintos

| Concern | Mecanismo | Consistencia |
|---|---|---|
| **Detectar MLE** | exit code 137 (cgroup enforcement de Docker) | ✅ Determinista — siempre ocurre si se excede el límite |
| **Reportar memoryUsed** | `ContainerStats` post-ejecución (cgroup watermark) | ⚠️ Margen de pocos MB, aceptable para display |

Docker enforcea el memory limit vía cgroups en el kernel. Cuando el proceso excede el límite, el kernel lo mata con SIGKILL. No hay varianza: si el programa aloca más de lo permitido, siempre muere con exit code 137.

La memoria reportada en casos no-MLE tiene un margen de pocos MB por imprecisiones del watermark del cgroup. Es aceptable para mostrar al usuario.

---

### Fase de cleanup (Close)

```
1. exec: rm -rf /sandbox/*   ← borra binario, archivos residuales
2. container.state = idle
3. container.lastUsedAt = now
4. pool.allocatedMemory -= container.memoryBytes   ← no, la memoria sigue asignada
                                                      el container sigue vivo
```

El cleanup borra el contenido de `/sandbox` pero el container sigue corriendo (`sleep infinity`). La memoria del container sigue contabilizada en `allocatedMemory` porque el container existe. Solo se descuenta cuando el container es destruido (idle timeout o LRU eviction).

---

## 8. Bounded Contexts DDD

### Regla de separación

El dominio contiene conceptos que el negocio entiende, que persisten, y que son visibles más allá de un solo use case. Los tipos de orquestación que solo existen durante la ejecución de un caso de uso son detalles de la capa de aplicación.

### `domain/submission/` — conceptos del negocio

| Archivo | Contenido |
|---|---|
| `submission.go` | Agregado raíz `Submission` |
| `status.go` | `SubmissionStatus` value object (PENDING, RUNNING, y todos los veredictos finales) |
| `language.go` | `Language` value object (cpp20, java17, python310) |
| `repository.go` | `SubmissionRepository` port (usado por la API) |
| `errors.go` | Constantes de error del dominio |

No hay `verdict.go` separado. El veredicto final **es** el `SubmissionStatus`. Los estados finales (AC, WA, TLE, MLE, RE, CE, SYSTEM_ERROR) son el veredicto. Tener dos tipos separados duplicaría el concepto.

### Tipos internos de `application/judge/` — artefactos de orquestación

| Tipo | Por qué aquí y no en domain |
|---|---|
| `CompileResult` | Efímero. El dominio no sabe de compilación. |
| `TestCaseResult` | Veredicto por test case individual. Nunca se persiste directamente. El dominio solo ve el resultado final. |
| `ExecutionSession`, `Executor` | Ports del use case hacia Docker (§7). |

**Intuición clave:** el dominio sabe que *existe* una submission, que *está en un estado*, y que puede *transicionar* a un estado final. No sabe que hay test cases, que se compila, ni que hay un container. Eso es el *cómo* del judge, no el *qué* del negocio.

---

## 9. Agregado Submission

### Campos

```go
type Submission struct {
    id             SubmissionID
    problemID      ProblemID
    userID         shared.UserID
    contestID      *ContestID      // nil = práctica libre
    language       Language
    status         SubmissionStatus
    sourceCodePath string          // ruta en GCS

    submittedAt    time.Time
    judgedAt       *time.Time      // nil hasta que termina el juicio

    // métricas de ejecución — nil hasta veredicto final (excepto CE/SYSTEM_ERROR)
    timeMs         *int
    memoryKb       *int

    compileLog     *string         // nil excepto en CE
}
```

Los veredictos por test case (falló en el caso 3 de 10) **no se persisten en esta fase**. Solo el veredicto global y las métricas globales. Se puede añadir una tabla `submission_test_results` en el futuro.

### State machine

```
PENDING ──→ RUNNING ──→ ACCEPTED
                    ──→ WRONG_ANSWER
                    ──→ TIME_LIMIT_EXCEEDED
                    ──→ MEMORY_LIMIT_EXCEEDED
                    ──→ RUNTIME_ERROR
                    ──→ COMPILATION_ERROR
                    ──→ SYSTEM_ERROR
```

Solo se permiten transiciones `PENDING → RUNNING` y `RUNNING → [veredicto final]`. Cualquier otra es un error de programación (panic o apperror interno).

### Métodos de dominio

Un método por transición de estado. Cada método valida que el aggregate esté en el estado correcto antes de transicionar y solo acepta los parámetros que ese veredicto necesita:

```go
// PENDING → RUNNING
func (s *Submission) Start(now time.Time) error

// RUNNING → veredictos finales
func (s *Submission) MarkAccepted(timeMs, memoryKb int, now time.Time) error
func (s *Submission) MarkWrongAnswer(timeMs, memoryKb int, now time.Time) error
func (s *Submission) MarkTimeLimitExceeded(timeMs int, now time.Time) error
func (s *Submission) MarkMemoryLimitExceeded(memoryKb int, now time.Time) error
func (s *Submission) MarkRuntimeError(timeMs, memoryKb int, now time.Time) error
func (s *Submission) MarkCompilationError(log string, now time.Time) error
func (s *Submission) MarkSystemError(now time.Time) error
```

Esto hace imposible, por ejemplo, pasar un `compileLog` a `MarkWrongAnswer`. Cada método documenta exactamente qué datos necesita ese veredicto.

### Constructores

```go
// Crea nueva submission — valida, retorna apperror en fallo
func NewSubmission(
    id SubmissionID,
    problemID ProblemID,
    userID shared.UserID,
    contestID *ContestID,
    lang Language,
    sourceCodePath string,
    now time.Time,
) (*Submission, error)

// Reconstruye desde DB — sin validación, sin error
func RestoreSubmission(
    id SubmissionID,
    problemID ProblemID,
    userID shared.UserID,
    contestID *ContestID,
    lang Language,
    status SubmissionStatus,
    sourceCodePath string,
    submittedAt time.Time,
    judgedAt *time.Time,
    timeMs *int,
    memoryKb *int,
    compileLog *string,
) *Submission
```

---

## 10. Ports del `application/judge/`

Los ports `Executor` y `ExecutionSession` ya están definidos en §7. Los ports restantes:

### `submission_updater.go`

Port estrecho del judge (ISP). El dominio tiene un `SubmissionRepository` más amplio que usa la API. El judge declara exactamente lo que necesita:

```go
type SubmissionUpdater interface {
    GetByID(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error)
    Update(ctx context.Context, s *submission.Submission) error
}
```

El mismo adapter (`adapter/submission/repository.go`) implementa tanto `SubmissionRepository` del dominio como `SubmissionUpdater` del judge.

### `source_code_downloader.go`

```go
type SourceCodeDownloader interface {
    Download(ctx context.Context, path string) ([]byte, error)
}
```

### `problem_provider.go`

```go
type ProblemLimits struct {
    TimeLimitMs      int
    MemoryKb         int
    HasCustomChecker bool
    CheckerPath      string   // ruta GCS al binario del checker; vacío si no tiene
}

type ProblemProvider interface {
    GetLimits(ctx context.Context, problemID problem.ProblemID) (ProblemLimits, error)
}
```

Separado de `TestCaseProvider` porque son concerns distintos: los límites son metadata del problema (pocos bytes, en Postgres), los test cases son archivos binarios en GCS. Mezclarlos en un port crea un adapter con dos backends distintos.

### `test_case_provider.go`

```go
type TestCase struct {
    Input          []byte
    ExpectedOutput []byte
}

type TestCaseProvider interface {
    GetTestCases(ctx context.Context, problemID problem.ProblemID) ([]TestCase, error)
}
```

El adapter lee los ZIPs de test cases desde GCS y los descomprime internamente.

### `output_checker.go`

```go
type CheckRequest struct {
    Input            []byte
    ExpectedOutput   []byte
    ContestantOutput []byte
    CheckerPath      string   // vacío = usar comparador default por tokens
}

type CheckResult struct {
    Accepted bool
    Message  string   // mensaje del checker para WA (opcional, puede estar vacío)
}

type OutputChecker interface {
    Check(ctx context.Context, req CheckRequest) (CheckResult, error)
}
```

### `standing_updater.go`

```go
type RecordVerdictRequest struct {
    UserID      shared.UserID
    ContestID   submission.ContestID
    ProblemID   problem.ProblemID
    Verdict     submission.SubmissionStatus
    SubmittedAt time.Time
}

type StandingUpdater interface {
    RecordVerdict(ctx context.Context, req RecordVerdictRequest) error
}
```

### Resumen de todos los ports

| Archivo | Interface(s) |
|---|---|
| `executor.go` | `Executor`, `ExecutionSession` |
| `submission_updater.go` | `SubmissionUpdater` |
| `source_code_downloader.go` | `SourceCodeDownloader` |
| `problem_provider.go` | `ProblemProvider` |
| `test_case_provider.go` | `TestCaseProvider` |
| `output_checker.go` | `OutputChecker` |
| `standing_updater.go` | `StandingUpdater` |

---

## 11. Algoritmo de Comparación de Output

### Comparador default — token-based

La comparación no es byte-a-byte. Todos estos outputs se consideran equivalentes:

```
Expected:    "3 5\n10\n"
Contestant:  "3 5\r\n10\r\n"   ← Windows line endings
             "3 5\n10\n   "    ← trailing space
             "3  5\n10\n"      ← doble espacio entre tokens
```

Algoritmo:
```
1. Dividir ambos outputs en tokens por whitespace (space, tab, \n, \r\n)
2. Filtrar tokens vacíos
3. Comparar token a token
4. AC si todos coinciden y los conteos coinciden
5. WA si algún token difiere o los conteos difieren
```

Si un problema necesita comparación exacta (raro), se usa un custom checker que implementa esa lógica.

### Custom checker — subprocess del worker

El custom checker es un binario del problem setter — código confiable, no del contestante. No necesita sandboxing Docker.

```
Worker descarga checker de GCS → lo escribe a un archivo temporal
exec.Command(checkerPath, inputFile, expectedOutputFile, contestantOutputFile)
timeout: timeLimitMs del problema
exit 0  → AC
exit != 0 → WA (el checker puede escribir un mensaje en stderr)
```

### Implementación del adapter

El adapter `adapter/judge/output_comparator.go` implementa el port `OutputChecker` y decide internamente:

```
CheckRequest.CheckerPath vacío → comparación por tokens (Go puro)
CheckRequest.CheckerPath no vacío → descarga binario de GCS, exec.Command
```

El use case no sabe qué estrategia se usa — solo llama `checker.Check(ctx, req)`.

---

## 12. Manejo de Errores Transitorios y Permanentes

### Clasificación

| Tipo | Ejemplos | Acción |
|---|---|---|
| **Veredicto determinista** | CE, WA, TLE, MLE, RE | ACK. Veredicto final. No reintentar. |
| **Error transitorio** | GCS no disponible, Postgres caído, timeout de red | NACK. Submission sigue en PENDING. Otro worker reintenta. |
| **Error permanente de infraestructura** | Safety net activado, checker corrupto, código fuente no existe en GCS | ACK. Submission → SYSTEM_ERROR. |

### Mecanismo de reintentos — delegado a RabbitMQ

El judge no implementa reintentos manualmente. Los delega a RabbitMQ:

```
Error transitorio detectado
        ↓
NACK del mensaje (sin requeue inmediato)
        ↓
Dead Letter Exchange → cola de espera (TTL: 30s)
        ↓
Después de 30s → mensaje vuelve a la cola principal
        ↓
Otro worker lo reintenta
```

Si el mensaje falla `maxRetries` veces (configurable, por ejemplo 3), RabbitMQ lo mueve a una dead letter queue final para inspección manual. La submission queda en SYSTEM_ERROR.

Reintentar en el mismo worker es contraproducente: si el worker tiene un problema local (container pool lleno, memoria baja), el reintento local va a fallar de nuevo.

### Tabla de decisión ACK/NACK

| Situación | Acción RabbitMQ | Estado Submission |
|---|---|---|
| Error al leer de GCS / Postgres | NACK | Sin cambio (PENDING) |
| Safety net del container activado | ACK | SYSTEM_ERROR |
| Checker binario corrupto | ACK | SYSTEM_ERROR |
| Submission ya no está en PENDING | ACK | Sin cambio (ignorar) |
| Veredicto determinista (CE, WA, etc.) | ACK | Veredicto final |

### Recuperación de workers caídos

Si un worker muere después de marcar `RUNNING` pero antes de terminar, la submission queda en `RUNNING` indefinidamente.

**Solución: goroutine de recuperación en background** dentro de `cmd/judge/main.go`. Corre cada 5 minutos y busca submissions en estado `RUNNING` por más de N minutos (configurable, por ejemplo 10 min). Las marca `SYSTEM_ERROR`.

Esto cubre el caso de pod crasheado sin necesidad de lógica especial en el worker principal.

---

## 13. Standing Updates

### Condiciones para actualizar el standing

No toda submission afecta el standing:

```
1. ¿La submission pertenece a un contest? (contestID != nil)
         NO → práctica libre, no hay standing que actualizar

2. ¿El contest estaba activo al momento de la submission?
         NO → postcompetición, no afecta el standing oficial

3. ¿El veredicto es final? (no PENDING, no RUNNING)
         SÍ → continúa

4. ¿El usuario ya tiene AC en este problema en este contest?
         SÍ y veredicto == AC → ignorar (no mejora la posición)
         SÍ y veredicto != AC → contar intento de todas formas (penalización ICPC)
```

Los intentos antes del primer AC cuentan para la penalización. Los intentos después del AC se ignoran.

### Atomicidad — transacción única

El judge hace dos writes que deben ser atómicos:

```
1. Update(submission)      → Postgres: veredicto final
2. RecordVerdict(standing) → Postgres: standing actualizado
```

Si el worker muere entre ambos writes, el veredicto quedaría sin reflejarse en el standing. La solución es una **transacción única**:

```go
// Dentro de JudgeSubmissionUseCase.Execute:
tx := txManager.Begin(ctx)
    submissionUpdater.Update(ctx, submission)      // dentro de tx
    if hayQueActualizarStanding(submission) {
        standingUpdater.RecordVerdict(ctx, req)    // dentro de tx
    }
tx.Commit()

// Solo después del commit:
rabbitmq.ACK(message)
```

El ACK va **después** del commit. Si el commit falla, el mensaje vuelve a la cola y otro worker reintenta. Ambas tablas están en el mismo Postgres, por lo que la transacción es directa.

Se usa el `TransactionManager` de `application/shared/` que ya existe en el proyecto.

---

## 14. Docker Compose para Desarrollo Local

### Nuevos servicios

El proyecto ya tiene `docker-compose.yml`. El judge añade RabbitMQ y el worker:

```yaml
rabbitmq:
  image: rabbitmq:3-management
  ports:
    - "5672:5672"      # AMQP — el worker conecta aquí
    - "15672:15672"    # Management UI — para inspeccionar la cola
  environment:
    RABBITMQ_DEFAULT_USER: judge
    RABBITMQ_DEFAULT_PASS: judge

judge-worker:
  build:
    context: .
    dockerfile: cmd/judge/Dockerfile
  depends_on:
    - rabbitmq
    - postgres
  environment:
    # Límites del "pod" simulado (equivalente a K8s Downward API)
    POD_MEMORY_LIMIT: 8589934592     # 8Gi en bytes
    POD_CPU_LIMIT: 4
    POD_MEMORY_OVERHEAD: 536870912   # 512Mi en bytes
    # Conexiones
    RABBITMQ_URL: amqp://judge:judge@rabbitmq:5672/
    DATABASE_URL: postgres://...
    STORAGE_BACKEND: local
  volumes:
    # Docker socket mount (DooD) — el worker habla con el Docker daemon del host
    - //./pipe/docker_engine://./pipe/docker_engine   # Windows
    # En Linux/Mac: - /var/run/docker.sock:/var/run/docker.sock
```

### Docker socket mount vs Docker-in-Docker

El worker necesita hablar con Docker para crear y gestionar los containers del pool. Hay dos enfoques:

| | Docker socket mount (DooD) | Docker-in-Docker (DinD) |
|---|---|---|
| Complejidad | Baja — un volume mount | Alta — requiere modo privilegiado |
| Containers de lenguaje | Hermanos del worker en el host | Hijos del worker |
| Debug | `docker ps` del host los muestra | Requiere entrar al worker |
| Desarrollo | ✅ Recomendado | ❌ Sobreingeniería para dev |

Se usa **Docker socket mount** para desarrollo local. Los containers de lenguaje (cpp20, java17, etc.) aparecen directamente en `docker ps` del host, lo que facilita el debugging.

### Imágenes de lenguajes

Las imágenes `judge-runner:cpp20`, `judge-runner:java17`, etc. **no son servicios** de Docker Compose — son imágenes que el worker usa internamente cuando crea containers del pool. Se construyen una vez:

```bash
# scripts/build-judge-images.sh
docker build -t judge-runner:base -f docker/judge/base.Dockerfile .
docker build -t judge-runner:cpp20 -f docker/judge/cpp20.Dockerfile .
docker build -t judge-runner:java17 -f docker/judge/java17.Dockerfile .
docker build -t judge-runner:python310 -f docker/judge/python310.Dockerfile .
```

Agregar un lenguaje nuevo = nueva imagen + entrada en la config YAML + ejecutar el script. Cero cambios de código.

---

## Secciones pendientes

- [ ] Estructura del `cmd/judge/` y composition root
