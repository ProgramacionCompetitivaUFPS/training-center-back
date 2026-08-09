# Roadmap de despliegue en GKE (Google Kubernetes Engine)

Objetivo: llevar el stack completo (API + judge-worker + Postgres + Redis + RabbitMQ + GCS + SMTP)
a un cluster de Kubernetes en Google Cloud, usando el crédito de prueba de $300 USD (90 días).

Cada fase termina en un estado funcional verificable. No avanzar a la siguiente sin verificar la actual.

---

## Decisiones previas (leer antes de empezar)

| Decisión | Elección | Por qué |
|---|---|---|
| Modo de GKE | **Standard** (no Autopilot) | El judge-worker necesita un sidecar Docker-in-Docker **privilegiado** para el pool de contenedores. Autopilot prohíbe contenedores privilegiados. |
| Tipo de cluster | **Zonal** (una sola zona) | El free tier de GKE exime la cuota de gestión ($0.10/h ≈ $73/mes) para **un** cluster zonal. Regional la cobra y triplica nodos. |
| Región | `us-east1` o `us-central1` | Las más baratas y con buena latencia desde Colombia. `southamerica-east1` (São Paulo) baja latencia pero cuesta ~30-40% más. |
| Postgres | **In-cluster** (StatefulSet + PVC) | Cloud SQL cuesta $10-30/mes extra y complica la conexión (proxy). Para la tesis, in-cluster con disco persistente es suficiente y más didáctico. |
| Redis / RabbitMQ | In-cluster, imágenes oficiales | Mismo criterio. Nota: los charts `bitnami/*` gratuitos quedaron congelados en 2025 (movidos a `bitnamilegacy`); usar manifests propios con imágenes oficiales (`postgres:16-alpine`, `redis:7-alpine`, `rabbitmq:3-management`) — es lo mismo que ya corre en docker-compose. |
| Storage de archivos | **GCS** (`STORAGE_BACKEND=gcs`) | El adapter ya existe. Autenticación vía Workload Identity — sin ficheros de credenciales. |
| Email | Gmail SMTP con App Password | Ya documentado en `.env.example`. |
| TLS / dominio | Ingress de GKE + certificado gestionado por Google | Reemplaza al par nginx+certbot del compose. El dominio DuckDNS sirve: solo tiene que apuntar a la IP estática del LB. |

### Presupuesto estimado (24/7)

| Recurso | Costo/mes aprox. |
|---|---|
| 2 × `e2-standard-2` (2 vCPU / 8 GB c/u) | ~$98 |
| Load Balancer HTTP(S) | ~$18 |
| Discos persistentes (~20 GiB) | ~$2 |
| GCS + egress + Artifact Registry | < $5 |
| Cuota de gestión GKE (cluster zonal, free tier) | $0 |
| **Total** | **~$120/mes** |

El crédito da ~2.5 meses encendido 24/7. **Truco clave**: cuando no estés trabajando,
baja los nodos a cero (`gcloud container clusters resize ... --num-nodes=0`). El cluster
y los discos quedan (centavos), los nodos dejan de cobrar. Con esa rutina el crédito
sobra para los 90 días del trial.

> El trial no cobra al agotarse el crédito: la cuenta se pausa hasta que actives
> facturación manualmente. Aun así, configura alertas (Fase 0) para vigilar el ritmo.

---

## Fase 0 — Cuenta, CLI y presupuesto

**Estado final: `gcloud` funcionando contra tu proyecto, con alertas de presupuesto activas.**

1. Crear proyecto en https://console.cloud.google.com (ej. `training-center-502916`).
2. Verificar que el billing account del trial está vinculado al proyecto.
3. **Presupuesto**: Billing → Budgets & alerts → crear budget por el monto del crédito con alertas al 25/50/75/90/100%.
   - **⚠️ El importe va en la moneda de la cuenta de facturación**, no en USD. Con cuenta en COP,
     el crédito de "$300 USD" aparece como **$1,031,258 COP** (tasa congelada ~3,437.5) — ese es el
     importe correcto del presupuesto. Escribir "300" en una cuenta COP crea un presupuesto de
     300 pesos (~US$0.09) y las alertas se disparan el primer día.
   - Período **personalizado** (fechas del trial), no mensual — mide el crédito total, no el gasto del mes.
   - En "Ahorro", **desmarcar "Otros ahorros"** (promociones): si se resta el crédito del trial, el gasto neto siempre es $0 y las alertas nunca suenan. "Programas de ahorros" (descuentos reales) sí queda marcado.
   - Los presupuestos solo avisan, no detienen el gasto; los costos tardan hasta 24 h en registrarse.
4. Instalar herramientas locales:
   - `gcloud` CLI: https://cloud.google.com/sdk/docs/install
   - `kubectl`: `gcloud components install kubectl gke-gcloud-auth-plugin`
5. Autenticación y proyecto por defecto:

```bash
gcloud auth login
gcloud config set project training-center-502916
gcloud config set compute/region us-east1
gcloud config set compute/zone us-east1-b
```

6. Habilitar APIs:

```bash
gcloud services enable container.googleapis.com \
  artifactregistry.googleapis.com \
  storage.googleapis.com \
  compute.googleapis.com
```

**⚠️ Gotchas de entorno Windows** (si replicas en una máquina con execution policy / WDAC / Smart App Control):
- PowerShell puede bloquear `gcloud.ps1` por execution policy → trabajar desde **cmd** (usa `gcloud.cmd`, sin políticas).
- WDAC (Device Guard) puede bloquear el `kubectl` de Chocolatey si gana en el PATH → desinstalar `minikube` + `kubernetes-cli` de Chocolatey para que `kubectl` resuelva a la copia de Docker Desktop o de Cloud SDK.
- **Smart App Control puede bloquear `gke-gcloud-auth-plugin.exe`** (el binario de Google no está
  firmado; SAC puede bloquearlo en cualquier momento aunque antes funcionara, y no admite excepciones
  puntuales). Solución: un wrapper `%USERPROFILE%\.kube\gke-auth.cmd` que obtiene el token con
  `gcloud auth print-access-token` y emite el JSON `ExecCredential`; apuntar el kubeconfig a él:
  ```
  kubectl config set-credentials gke_training-center-502916_us-east1-b_training-center --exec-command="C:\Users\RYZEN7~1\.kube\gke-auth.cmd" --exec-api-version=client.authentication.k8s.io/v1beta1
  ```
  (ruta corta 8.3 porque el espacio de "Ryzen 7" se pierde al guardarse). **⚠️ Si algún día se re-ejecuta
  `gcloud container clusters get-credentials`, sobrescribe esta configuración y kubectl vuelve a fallar
  con "Una directiva de Control de aplicaciones bloqueó este archivo" → re-correr el comando de arriba.**

---

## Fase 1 — Imagen en Artifact Registry

**Estado final: la imagen del backend (con los 4 binarios) publicada en un registry privado de GCP.**

El Dockerfile ya construye `api`, `migrate`, `seed` y `worker` en una sola imagen con
entrypoints distintos — perfecto para K8s: un solo push, cada Deployment/Job elige su `command`.

```bash
gcloud artifacts repositories create training-center \
  --repository-format=docker --location=us-east1

gcloud auth configure-docker us-east1-docker.pkg.dev

docker build -t us-east1-docker.pkg.dev/training-center-502916/training-center/backend:v0.1.0 .
docker push us-east1-docker.pkg.dev/training-center-502916/training-center/backend:v0.1.0
```

Convención: tag por versión (`v0.1.0`, `v0.1.1`…), nunca `latest` — permite rollback con `kubectl rollout undo`.

---

## Fase 2 — Cluster GKE Standard

**Estado final: cluster zonal corriendo, `kubectl get nodes` responde.**

```bash
gcloud container clusters create training-center \
  --zone us-east1-b \
  --num-nodes 2 \
  --machine-type e2-standard-2 \
  --disk-size 30 \
  --workload-pool=training-center-502916.svc.id.goog

gcloud container clusters get-credentials training-center --zone us-east1-b
kubectl get nodes
```

- `--workload-pool` habilita Workload Identity (lo usa la Fase 4 para GCS).
- **Sin autoscaling por ahora** (deliberado): con autoscaling activo, el `resize 0` de la rutina
  de ahorro pelea contra el autoscaler — ve los pods Pending y revive nodos. Tamaño fijo =
  apagado determinista. El cluster autoscaler se activa en la Fase 7 junto con KEDA,
  donde de verdad aporta (escalar el pool del judge según la cola).

### Rutina de apagado/encendido (el seguro del crédito)

```bash
# al terminar cada sesión de trabajo:
gcloud container clusters resize training-center --zone us-east1-b --num-nodes 0 --quiet

# al volver:
gcloud container clusters resize training-center --zone us-east1-b --num-nodes 2 --quiet

# verificar que de verdad quedó apagado (lista vacía = nada facturando cómputo):
gcloud compute instances list
```

**Qué pasa al apagar** (por qué es seguro):
- Las VMs se **destruyen** (no se pausan) junto con sus discos de arranque → cómputo a $0.
- El **control plane sigue vivo y gratis**: todo lo declarado (Deployments, Services, Secrets,
  ConfigMaps) permanece en etcd. Los pods mueren, pero un pod es la materialización de una
  declaración — la declaración sobrevive.
- Los **PVCs** (datos de Postgres/RabbitMQ desde la Fase 3) son discos independientes de las VMs:
  sobreviven y cuestan centavos (~$0.10/GB/mes).

**Qué pasa al encender**: nacen 2 VMs vírgenes, el scheduler ve "pods declarados ≠ pods corriendo"
y re-materializa todo; Postgres reencuentra sus datos en su PVC. ~3 min, sin intervención.

**Letras pequeñas:**
- Lo que estaba en vuelo se interrumpe: judgings a medias → mensaje sin ack vuelve a la cola
  (el diseño lo absorbe); Redis pierde contadores de rate-limit (descartables por diseño).
- Desde la Fase 5, el **Load Balancer factura (~$0.60/día) aunque los nodos estén apagados** —
  es un recurso aparte. Decidir en Fase 5 si la rutina también lo elimina o si se acepta el costo
  por no reconfigurar DNS en cada sesión.
- La primera vez tras encender, los nodos re-descargan las imágenes (pull) — el primer arranque
  de cada pod tarda un poco más.

Organización de manifests en el repo (nuevo directorio):

```
deploy/k8s/
  namespace.yaml
  infra/        # postgres, redis, rabbitmq
  app/          # configmap, secrets (plantilla), api, jobs
  judge/        # worker + dind
  ingress/      # ip, certificado, ingress
```

---

## Fase 3 — Infraestructura in-cluster (Postgres, Redis, RabbitMQ)

**Estado final: los tres servicios `Running`, con datos que sobreviven reinicios de pod.**

1. Namespace — declarativo, como todo lo que no sea secret:

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl config set-context --current --namespace=training-center   # default local, adiós -n
```
2. **Secrets** primero (equivalente al `.env`):

```bash
kubectl create secret generic app-secrets \
  --from-literal=DB_PASSWORD='...' \
  --from-literal=JWT_SECRET='...' \
  --from-literal=SMTP_PASSWORD='...' \
  --from-literal=ADMIN_PASSWORD='...' \
  --from-literal=RABBITMQ_PASSWORD='...'
```

3. **Postgres** (`deploy/k8s/infra/postgres.yaml`): StatefulSet con `postgres:16-alpine`,
   `volumeClaimTemplates` de 10 GiB (GKE aprovisiona un Persistent Disk real — sobrevive al `resize 0`),
   Service **headless** `postgres` → el `DB_HOST` de la app es `postgres`, igual que en compose.
   Gotchas ya resueltos en el manifest: `PGDATA` apunta a un subdirectorio (el `lost+found` del disco
   recién formateado impide inicializar en la raíz); probes con `pg_isready`; password vía `secretKeyRef`.

```bash
kubectl apply -f deploy/k8s/infra/postgres.yaml
kubectl get pods -w        # Pending → ContainerCreating → Running 1/1 (~1 min la primera vez)
kubectl get pvc            # el disco: STATUS Bound
kubectl exec postgres-0 -- pg_isready -U postgres -d training_center
```
4. **Redis** (`deploy/k8s/infra/redis.yaml`): Deployment con `redis:7-alpine`, sin volumen
   (solo rate limiting, datos descartables → pod intercambiable), Service normal (con VIP).
5. **RabbitMQ** (`deploy/k8s/infra/rabbitmq.yaml`): StatefulSet con `rabbitmq:3-management`,
   PVC de 2 GiB (cola durable + mensajes persistent necesitan disco; el hostname estable evita
   huérfanos en la DB interna de Rabbit), Service headless con puertos 5672 (AMQP) y 15672
   (management UI — solo vía port-forward, nunca expuesta). Usuario `judge` visible; contraseña
   vía `secretKeyRef` (adiós al judge/judge del compose). Probes pacientes (Erlang arranca lento).

```bash
kubectl apply -f deploy/k8s/infra/redis.yaml -f deploy/k8s/infra/rabbitmq.yaml
kubectl get pods -w
kubectl exec deploy/redis -- redis-cli ping
kubectl exec rabbitmq-0 -- rabbitmq-diagnostics -q ping
kubectl get pvc
```

**Prueba de fuego de la persistencia** (validación final de la fase):

```bash
kubectl exec postgres-0 -- psql -U postgres -d training_center -c "CREATE TABLE prueba_persistencia (id int); INSERT INTO prueba_persistencia VALUES (42);"
kubectl delete pod postgres-0          # simula crash; el StatefulSet lo recrea con SU disco (~30s)
kubectl get pods -w                    # esperar 1/1 de nuevo
kubectl exec postgres-0 -- psql -U postgres -d training_center -c "SELECT * FROM prueba_persistencia;"   # debe devolver 42
kubectl exec postgres-0 -- psql -U postgres -d training_center -c "DROP TABLE prueba_persistencia;"      # limpieza
```

> Equivalencias compose → K8s: `services` → Deployment/StatefulSet, `volumes` → PVC,
> nombre del servicio como hostname → Service ClusterIP, `depends_on` → no existe
> (la app debe reintentar la conexión, o usar initContainers que esperen).

---

## Fase 4 — API, migraciones, seed y GCS

**Estado final: la API responde vía `kubectl port-forward`, con datos en Postgres y archivos en GCS.**

1. **Bucket GCS**:

```bash
gcloud storage buckets create gs://training-center-502916-uploads \
  --location=us-east1 --uniform-bucket-level-access
```

2. **Workload Identity** — la API y el worker leen/escriben GCS sin ficheros de credenciales:

```bash
kubectl create serviceaccount backend

gcloud storage buckets add-iam-policy-binding gs://training-center-502916-uploads \
  --role=roles/storage.objectAdmin \
  --member="principal://iam.googleapis.com/projects/54498548428/locations/global/workloadIdentityPools/training-center-502916.svc.id.goog/subject/ns/training-center/sa/backend"
```

   (Los pods usan `serviceAccountName: backend`; la librería de GCS detecta las credenciales sola.)

   **⚠️ Gotcha**: en proyectos GCP nuevos, la cuenta de servicio de cómputo por defecto nace SIN
   permisos → los nodos reciben `403 Forbidden` al hacer pull del Artifact Registry
   (`ImagePullBackOff` en todos los pods con imagen propia). Arreglo — lectura sobre el
   repositorio para la identidad de los nodos:

```bash
gcloud artifacts repositories add-iam-policy-binding training-center --location=us-east1 \
  --member="serviceAccount:54498548428-compute@developer.gserviceaccount.com" \
  --role="roles/artifactregistry.reader"
```

   Nota la distinción de identidades: el pull de imágenes lo hace el **kubelet del nodo** (SA de
   cómputo), no el pod; Workload Identity (KSA `backend`) es solo para lo que la app hace en runtime.

3. **ConfigMap** (`deploy/k8s/app/configmap.yaml`) — la config no sensible. `RABBITMQ_URL` NO está
   aquí: lleva la contraseña incrustada, así que se compone en el Deployment con expansión
   `$(RABBITMQ_PASSWORD)` (definida antes en la lista `env`, el orden importa).
4. **Jobs** — `migrate-job.yaml` + `seed-job.yaml` (separados; imagen `__IMAGE__` parametrizable,
   se resuelve al último tag `v*` del registro al bootstrapear — nada quemado).
   - El seed es un **upsert de contraseña**: si el admin existe, la actualiza (`admin password updated`).
     Se corre UNA VEZ al bootstrap (no re-corre en cada deploy; el CD solo re-corre migrate).
   - Cambiar la clave del admin después: patch del secret + re-correr seed. El patch sin peleas
     de comillas: `Set-Content` (PowerShell) del JSON a un archivo + `kubectl patch secret
     app-secrets --type merge --patch-file ...` + borrar el archivo.
5. **API** (`deploy/k8s/app/api.yaml`): Deployment (1 réplica) + Service. Claves:
   `serviceAccountName: backend` (activa Workload Identity), probes `httpGet /ping`,
   sin `command` (usa el ENTRYPOINT `/bin/api`).

```powershell
# resolver el último tag v* del registro (nada quemado en ningún manifest)
$REG = "us-east1-docker.pkg.dev/training-center-502916/training-center/backend"
$LATEST = gcloud artifacts docker images list $REG --include-tags --filter="tags~^v" --sort-by="~UPDATE_TIME" --format="value(tags)" --limit=1 2>$null
$IMG = "${REG}:${LATEST}"
kubectl apply -f deploy/k8s/app/configmap.yaml
kubectl delete job migrate seed -n training-center --ignore-not-found
# manifests con la imagen del backend: sustituir __IMAGE__ por la última versión y aplicar
foreach ($f in "api","migrate-job","seed-job") {
  (Get-Content "deploy/k8s/app/$f.yaml") -replace '__IMAGE__', $IMG | kubectl apply -f -
}
kubectl wait --for=condition=complete job/migrate job/seed -n training-center --timeout=180s
kubectl port-forward svc/api 8080:8080
# navegador: http://localhost:8080/ping  y  /swagger/index.html → POST /auth/login con el admin
```

Comprobación de éxito: el login del admin devuelve 200 con un token JWT. (La subida de archivos a
GCS vía Workload Identity se ejercita con la primera creación de un problema real.)

---

## Fase 5 — Exposición pública con HTTPS

**Estado final: `https://tu-dominio.duckdns.org/...` responde con certificado válido.**

Esto reemplaza a nginx + certbot del compose: el Load Balancer de Google hace de nginx
y el certificado gestionado hace de certbot.

1. IP estática global y DNS:

```bash
gcloud compute addresses create training-center-ip --global
gcloud compute addresses describe training-center-ip --global --format='value(address)'
# apuntar el dominio DuckDNS a esa IP
```

2. **Dominio** (ej. `trainingjudgecenter.com`, registrado en Hostinger). Crear un registro `A`:
   `api` → la IP reservada (panel DNS del registrador). Reparto: `api.` → GKE; apex y `www.` → Vercel (frontend).
3. Manifests (`deploy/k8s/ingress/ingress.yaml`): `ManagedCertificate` (dominio literal),
   `FrontendConfig` (redirect HTTP→HTTPS), `Ingress` con 3 anotaciones que referencian **por nombre**:
   la IP (`training-center-ip`, recurso GCP), el cert y el frontendconfig (recursos del mismo archivo).
   Además: anotación `cloud.google.com/neg: '{"ingress": true}'` en el Service `api`
   (balanceo container-nativo — el LB apunta directo a los pods, y permite Service ClusterIP con Ingress).
4. CORS actualizado en el ConfigMap con los dominios reales (+ `rollout restart deployment/api`).

```bash
kubectl apply -f deploy/k8s/app/configmap.yaml -f deploy/k8s/app/api.yaml -f deploy/k8s/ingress/ingress.yaml
kubectl rollout restart deployment/api
# verificar que DNS e IP coinciden (requisito del cert):
gcloud compute addresses describe training-center-ip --global --format="value(address)"
nslookup api.trainingjudgecenter.com
# vigilar (15-60 min): Provisioning → Active
kubectl describe managedcertificate api-cert
kubectl get ingress api        # el ADDRESS aparece a los ~5 min
# prueba final: https://api.trainingjudgecenter.com/ping (candado verde)
```

---

## Fase 6 — Judge worker (el paso más delicado)

**Estado final: worker `2/2 Running` en el judge-pool, pool de lenguajes precargado, escuchando la cola.**
Manifest: `deploy/k8s/judge/worker.yaml`.

**El problema**: en compose, el worker monta el socket Docker del host. En GKE los nodos
usan containerd — **no existe socket Docker en el nodo**. La solución preserva el diseño
de `RUNNER_ARCHITECTURE.md` (pool de contenedores por pod, el pod no sabe que K8s existe).

### Prerrequisito 1 — node pool dedicado del judge

```bash
# autoescalado 0..2, con un taint que solo el worker tolera → aislamiento del pool
gcloud container node-pools create judge-pool --cluster training-center --zone us-east1-b \
  --machine-type e2-standard-4 --disk-size 50 --num-nodes 0 \
  --enable-autoscaling --min-nodes 0 --max-nodes 2 --node-taints judge=true:NoSchedule

gcloud container node-pools list --cluster training-center --zone us-east1-b   # verificar
```

### Prerrequisito 2 — permiso de lectura del registro para la KSA

```bash
# la KSA backend puede LEER el registro (lo usa dind vía token del metadata server, no el nodo)
gcloud artifacts repositories add-iam-policy-binding training-center --location=us-east1 \
  --role=roles/artifactregistry.reader \
  --member="principal://iam.googleapis.com/projects/54498548428/locations/global/workloadIdentityPools/training-center-502916.svc.id.goog/subject/ns/training-center/sa/backend"
```

### Prerrequisito 3 — construir y subir las imágenes de lenguaje

`scripts/build-judge-images.sh` construye las imágenes **localmente** con nombres cortos
(`judge-runner:base`, `judge-runner:cpp20`, `judge-runner:java17`, `judge-runner:python310`).
Para que dind pueda descargarlas hay que subirlas al registro con el nombre de registro
(el initContainer las re-etiqueta de vuelta al nombre corto al arrancar el pod).

```bash
# 1. construir localmente (requiere Docker corriendo)
bash scripts/build-judge-images.sh

# 2. etiquetar cada lenguaje con el nombre de registro y subir (bucle sobre los 3)
#    NOTA: no se sube judge-runner:base — es solo capa intermedia; las de lenguaje ya la incluyen.
REG=us-east1-docker.pkg.dev/training-center-502916/training-center
for lang in cpp20 java17 python310; do
  docker tag  "judge-runner:$lang" "$REG/judge-runner-$lang:v0.1.0"
  docker push "$REG/judge-runner-$lang:v0.1.0"
done

# 3. verificar que quedaron en el registro
gcloud artifacts docker images list "$REG" --include-tags | grep judge-runner
```

> Convención de nombres: el registro no admite `:` en el path, así que `judge-runner:cpp20`
> (nombre corto local) se sube como `judge-runner-cpp20:v0.1.0` (repo distinto, tag de versión).
> El initContainer `prepull-language-images` deshace esta traducción dentro de dind.

**El pod (dos containers):**
- **`dind`** (`docker:27-dind`, `privileged: true`, native sidecar con `restartPolicy: Always` +
  `startupProbe` de `docker info`): daemon Docker privado; el pool de lenguajes vive aquí.
  Privileged es la razón de GKE Standard, no Autopilot.
- **`worker`**: habla al daemon con `DOCKER_HOST=tcp://localhost:2375` (el cliente usa `client.FromEnv`).

**Piezas que lo hacen funcionar:**
- **Downward API → dind**: `POD_CPU_LIMIT`/`POD_MEMORY_LIMIT` se leen del límite del container
  `dind`. Para CPU, **`divisor: 1m`** (millicores) — con divisor 1, K8s redondea 3.5→4 y sobrevende
  el semáforo. Resultado esperado en los logs: `max_concurrent=2` derivado de los 3 cores de dind.
- **initContainer `prepull-language-images`** (`docker:27-cli`): token de Workload Identity del
  metadata server → `docker login` a Artifact Registry → `pull` + `tag` de las 3 imágenes a los
  nombres cortos de `judge_config.yaml`. Resuelve auth del registro privado (dind es un daemon
  aislado sin credenciales), mismatch de nombres y cold-start de un golpe.
- **Aislamiento**: `nodeSelector: gke-nodepool: judge-pool` (exige) + `toleration` del taint
  `judge=true` (permite). Ambos: worker solo en judge-pool, judge-pool solo workers.
- **Recursos**: dind Burstable (request 2 CPU, limit 3) para schedulear con holgura pero alimentar
  el semáforo desde el limit; worker modesto (250m/512Mi). `terminationGracePeriodSeconds: 120`
  para que el consumidor concurrente drene los judgings en vuelo ante SIGTERM.
- Watchdog de salud del pool (`IsHealthy` cada 30s → `os.Exit` si dind cae → K8s reinicia).

```powershell
# worker.yaml usa __IMAGE__ en el contenedor worker (dind/prepull son de terceros, no se tocan):
# sustituir por la última versión al aplicar. El CD lo actualiza después con `kubectl set image`.
$REG = "us-east1-docker.pkg.dev/training-center-502916/training-center/backend"
$LATEST = gcloud artifacts docker images list $REG --include-tags --filter="tags~^v" --sort-by="~UPDATE_TIME" --format="value(tags)" --limit=1 2>$null
(Get-Content deploy/k8s/judge/worker.yaml) -replace '__IMAGE__', "${REG}:${LATEST}" | kubectl apply -f -
kubectl get pods -w        # Pending → autoscaler crea nodo judge → Init:1/2 (dind) → Init:2/2 (prepull) → 2/2
kubectl logs -l app=judge-worker -c worker --tail=20      # "listening for submissions", max_concurrent=2
kubectl exec deploy/judge-worker -c dind -- docker images judge-runner   # las 3 imágenes
```

Dimensionamiento inicial: 1 réplica con limits ~2 CPU / 4-6 GiB (ajustar según `judge_config.yaml`).
Si el nodo `e2-standard-2` queda corto al correr todo junto, subir el node pool a `e2-standard-4`
(4 vCPU / 16 GB, ~$98/mes el nodo) y compensar apagando el cluster fuera de horas de trabajo.

Verificación end-to-end: submission por API → mensaje en RabbitMQ → worker compila y ejecuta → veredicto en la DB.

---

## Fase 7 — Autoscaling por profundidad de cola (KEDA)

**Estado final: los pods del worker escalan solos cuando la cola de RabbitMQ crece.**

El HPA nativo de K8s solo ve CPU/memoria; para escalar por profundidad de cola
(el diseño de `RUNNER_ARCHITECTURE.md` §2) se usa **KEDA**. El node pool dedicado ya se creó
en la Fase 6 (`judge-pool`, autoescalado 0..2 con taint); KEDA es quien crea los *pods* que lo
disparan. Cadena completa: cola crece → KEDA sube réplicas del worker → pods `Pending` →
cluster autoscaler crea nodo en `judge-pool` → cola vacía → KEDA baja réplicas a 0 →
nodo vacío → autoscaler lo elimina (~10 min de gracia) → $0.

### 1. Instalar el operador de KEDA

```bash
# --server-side: los CRDs de KEDA exceden el límite de tamaño del apply normal
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.20.1/keda-2.20.1.yaml
kubectl get pods -n keda        # keda-operator, keda-metrics-apiserver, keda-admission → Running
```

### 2. Secret con el host de RabbitMQ para KEDA

KEDA necesita credenciales para *consultar* la cola. **El host debe ser el FQDN**
`rabbitmq.training-center.svc.cluster.local`: el operador sondea desde el namespace `keda`,
no desde `training-center`, así que el nombre corto resolvería mal. Crear el secret leyendo
la contraseña del secret existente (PowerShell, para no reteclearla):

```powershell
$pass = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String((kubectl get secret app-secrets -n training-center -o jsonpath='{.data.RABBITMQ_PASSWORD}')))
kubectl create secret generic keda-rabbitmq --namespace training-center --from-literal=host="amqp://judge:$pass@rabbitmq.training-center.svc.cluster.local:5672/"
Remove-Variable pass   # limpiar la contraseña de la sesión
```

### 3. Aplicar el ScaledObject + TriggerAuthentication

`deploy/k8s/judge/keda.yaml`. Parámetros clave: `minReplicaCount: 0` (reposo = $0),
`maxReplicaCount: 2` (= max nodos del judge-pool), `value: 10` (2º pod solo con backlog
sostenido — escalar un nodo para un pico chico es inútil, la VM tarda ~2 min), `activationValue: 0`
(cualquier mensaje despierta 0→1). Con AMQP + `QueueLength` cuenta mensajes *ready*, no los
*unacked* en vuelo — por eso el consumidor fija `prefetch = maxConcurrent`.

```bash
kubectl apply -f deploy/k8s/judge/keda.yaml
kubectl get scaledobject -n training-center     # READY=True, ACTIVE=False (cola vacía)
```

### Verificación

Con la cola vacía, KEDA baja el worker a 0 solo (toma el control del replica count).
Publicar un mensaje → KEDA escala 0→1 y el pod arranca por demanda.

```bash
# publicar un mensaje de prueba (PowerShell; $pass extraído como en el paso 2)
kubectl exec -n training-center rabbitmq-0 -- rabbitmqadmin -u judge -p $pass publish routing_key=submissions payload="{}"
kubectl get pods -l app=judge-worker -w         # 0 → 1 pod (Init dind+prepull → Running)
```

> **Nota sobre demos sintéticas**: un mensaje `{}` malformado se descarta en microsegundos,
> más rápido de lo que el HPA reacciona → un solo pod drena un burst antes de que haga falta un 2º.
> Para *ver* el 1→2 dirigido por cola hace falta carga real (submissions que tardan segundos).
> Para forzar el 2º nodo y ver al autoscaler comprar la VM: anotar
> `autoscaling.keda.sh/paused-replicas="2" --overwrite` (y quitarla con el sufijo `-` al terminar).

**Rutina de apagado con KEDA**: una vez KEDA gestiona el `judge-worker`, NO lo escales a mano —
con la cola vacía ya queda en 0 solo. La rutina de apagado se simplifica al `resize 0` del pool
default (ver chuleta al final).

---

## Fase 8 — Operación y cierre

**Estado final: sabes qué gasta, dónde ver logs, y cómo apagar/encender todo.**

- **Logs**: `stdout` de los pods llega solo a Cloud Logging (Console → Logging → filtrar por namespace). `slog` en JSON se indexa por campos.
- **Monitoreo**: Console → Kubernetes Engine → Workloads muestra CPU/memoria por pod sin configurar nada.
- **Gasto**: Billing → Reports, agrupado por SKU, cada pocos días. Comparar contra la tabla de esta guía.
- **Rutina diaria**: resize a 0 al terminar, resize a 2 al volver (Fase 2). Es la diferencia entre 90 días de crédito y 30.
- **Onboarding de colaboradores**: para que un compañero pueda operar el despliegue igual que tú
  (desplegar, gestionar cluster, montar IAM/WIF) sin poder borrar el proyecto ni tocar billing,
  darle `editor` + `resourcemanager.projectIamAdmin` a nivel de proyecto:

```bash
gcloud projects add-iam-policy-binding training-center-502916 --member="user:<correo>" --role="roles/editor"
gcloud projects add-iam-policy-binding training-center-502916 --member="user:<correo>" --role="roles/resourcemanager.projectIamAdmin"
```
  Luego cada uno hace su setup local de la Fase 0 (gcloud/kubectl, `auth login`, `get-credentials`).
  Estos accesos humanos NO se gestionan en Terraform a propósito (así un `terraform destroy` no
  revoca a nadie) — ver `deploy/gcp/iam.tf`.

### Mejoras opcionales — orden recomendado de aplicación

Regla del orden: redes de seguridad primero, el paso disruptivo (recrear el cluster) cuando ya
estás cubierto, y lo que depende de él al final. Dependencia dura: **NetworkPolicy necesita un
enforcer, que solo se activa gratis al recrear con Terraform (Dataplane V2)**.

```
1. Backup (red de seguridad)  →  2. CI/CD (automatización)  →  3. Terraform+DPv2 (fundación)  →  4. NetworkPolicy (hardening)
```

**1. Backup de Postgres** (`deploy/k8s/infra/backup-cronjob.yaml`). El más pequeño y autocontenido.
   `CronJob` diario: initContainer `dump` (`postgres:16-alpine`, `pg_dump | gzip` → emptyDir) →
   container `upload` (`google/cloud-sdk`, `gcloud storage cp` al bucket con la KSA `backend`). El
   orden init→main garantiza volcar-antes-de-subir sin sincronización. Solo corre con el cluster
   encendido; `pg_dump` usa snapshot MVCC (no bloquea la app). Deja un respaldo *antes* de la
   recreación del paso 3.

```bash
kubectl apply -f deploy/k8s/infra/backup-cronjob.yaml
# backup a demanda (p. ej. antes del Terraform):
kubectl create job -n training-center --from=cronjob/postgres-backup backup-manual
kubectl logs -n training-center job/backup-manual -c dump && kubectl logs -n training-center job/backup-manual -c upload
gcloud storage ls gs://training-center-502916-uploads/backups/
# restaurar un dump:
# gunzip -c dump.sql.gz | psql -h postgres -U postgres -d training_center
```

**2. CI/CD para GKE** (`.github/workflows/deploy-gke.yml`). Build → push a Artifact Registry →
   migraciones → `kubectl set image`. Auth **sin llaves** vía Workload Identity Federation.
   Disparadores: `tags: ['v*']` (releases, imagen `backend:vX.Y.Z`) + `workflow_dispatch` (pruebas
   desde cualquier rama, imagen `backend:{rama}-{sha}`). **El workflow debe vivir en la rama por
   defecto (`main`)** para que el dispatch aparezca y los triggers de tag funcionen.

   *Montar la federación (una vez, gcloud):*
```bash
# SA dedicada con menor privilegio
gcloud iam service-accounts create github-deployer --display-name="GitHub Actions deployer for GKE"
gcloud artifacts repositories add-iam-policy-binding training-center --location=us-east1 \
  --member="serviceAccount:github-deployer@training-center-502916.iam.gserviceaccount.com" --role="roles/artifactregistry.writer"
gcloud projects add-iam-policy-binding training-center-502916 \
  --member="serviceAccount:github-deployer@training-center-502916.iam.gserviceaccount.com" --role="roles/container.developer"
# federación OIDC restringida al repo (la condición es obligatoria — evita que otros repos impersonen)
gcloud services enable iamcredentials.googleapis.com sts.googleapis.com
gcloud iam workload-identity-pools create github-pool --location=global --display-name="GitHub Actions pool"
gcloud iam workload-identity-pools providers create-oidc github-provider --location=global \
  --workload-identity-pool=github-pool --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository" \
  --attribute-condition="assertion.repository == 'ProgramacionCompetitivaUFPS/training-center-back'"
gcloud iam service-accounts add-iam-policy-binding github-deployer@training-center-502916.iam.gserviceaccount.com \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/54498548428/locations/global/workloadIdentityPools/github-pool/attribute.repository/ProgramacionCompetitivaUFPS/training-center-back"
# secrets de GitHub
gh secret set WIF_PROVIDER --body "projects/54498548428/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
gh secret set GCP_SERVICE_ACCOUNT --body "github-deployer@training-center-502916.iam.gserviceaccount.com"
```

   *Probar / desplegar:*
```bash
gh workflow run deploy-gke.yml --ref <rama>    # dispatch de prueba (imagen {rama}-{sha})
git tag v0.3.0 && git push origin v0.3.0       # release real (imagen v0.3.0)
```

   *Migraciones*: el CD las corre solo (no-op si no hay pendientes; corta el deploy si fallan).
   Para correrlas a mano contra una versión (PowerShell):
```powershell
$IMG = "us-east1-docker.pkg.dev/training-center-502916/training-center/backend:v0.3.0"
kubectl delete job migrate -n training-center --ignore-not-found
(Get-Content training-and-judge-center-backend/deploy/k8s/app/migrate-job.yaml) -replace '__IMAGE__', $IMG | kubectl apply -f -
kubectl wait --for=condition=complete job/migrate -n training-center --timeout=180s
```

   El `cd.yml` viejo (build a ghcr.io + deploy a VM de staging vía docker-compose, modelo pre-GKE)
   se eliminó. Nota: la federación son recursos de proyecto, sobreviven a la recreación del paso 3.

   *Cleanup policy del registro* (pendiente de aplicar — codificar en Terraform, o a mano así):
   los dispatch acumulan imágenes `{rama}-{sha}`; la política conserva releases `v*` para siempre
   + las 5 más recientes, y borra el resto tras 14 días. Guardar como `cleanup-policy.json`:
```json
[
  {"name": "keep-releases", "action": {"type": "Keep"}, "condition": {"tagState": "TAGGED", "tagPrefixes": ["v"]}},
  {"name": "keep-recent",   "action": {"type": "Keep"}, "mostRecentVersions": {"keepCount": 5}},
  {"name": "delete-old",    "action": {"type": "Delete"}, "condition": {"olderThan": "1209600s"}}
]
```
```bash
gcloud artifacts repositories set-cleanup-policies training-center --location=us-east1 --policy=cleanup-policy.json
```

**3. Terraform para la capa GCP** (`deploy/gcp/*.tf`): codifica los ~21 recursos de nube
   (APIs, cluster **con `datapath_provider = "ADVANCED_DATAPATH"`**, node pools, Artifact Registry +
   cleanup policy, bucket, IP estática, WIF, IAM aditivo). State remoto en un bucket GCS aparte.
   Justificación: el trial expira → el entorno tendrá que recrearse; con IaC son minutos y
   `terraform destroy` demuele limpio. El .tf es transcripción de experiencia, no conjuro copiado.

   **Recreación completa del entorno** (capa 1 Terraform + capa 2 con `deploy/k8s/bootstrap.ps1`).
   Los recursos manuales NO están en el state, así que la recreación es *borrar lo manual → apply fresco*.

   **Recomendado**: correrla con la skill de Claude Code `/recreate-environment`
   (`.claude/skills/recreate-environment/`) — guía el proceso interactivamente: pregunta el alcance
   (ambas capas o solo la app), si es recreación en el mismo proyecto o migración a otra cuenta, y
   qué conservar (datos/imágenes/secrets) uno por uno; muestra cada comando y su salida, y confirma
   antes de cualquier paso destructivo. Lo de abajo es la referencia manual — y lo que hace
   `bootstrap.ps1` sin supervisión (la vía rápida para capa 2 cuando no hace falta ese control):
```bash
# 1. (si preservas datos) backup fresco y descargarlo a local ANTES de borrar el bucket
kubectl create job -n training-center --from=cronjob/postgres-backup backup-pre-recreate
gcloud storage cp gs://training-center-502916-uploads/backups/<archivo>.sql.gz .

# 2. borrar los recursos manuales (no están en el state de Terraform)
gcloud container clusters delete training-center --zone us-east1-b --quiet
gcloud storage rm --recursive gs://training-center-502916-uploads
gcloud artifacts repositories delete training-center --location=us-east1 --quiet
gcloud compute addresses delete training-center-ip --global --quiet
gcloud iam workload-identity-pools providers delete github-provider --location=global --workload-identity-pool=github-pool --quiet
gcloud iam workload-identity-pools delete github-pool --location=global --quiet
gcloud iam service-accounts delete github-deployer@training-center-502916.iam.gserviceaccount.com --quiet
# (las APIs habilitadas y los accesos humanos se quedan — no estorban)

# 3. crear todo con Terraform (incluye Dataplane V2)
cd deploy/gcp && terraform apply

# 4. conectar kubectl al cluster nuevo
gcloud container clusters get-credentials training-center --zone us-east1-b

# 5. re-subir las imágenes al registro nuevo (nació vacío). Manual, NO el CI/CD (su deploy
#    fallaría contra un cluster sin deployments). Backend: docker build + tag vX.Y.Z + push (Fase 1).
#    Lenguajes: scripts/build-judge-images.sh + tag + push de las 3 (Prerreq. 3 de la Fase 6).

# 6. bootstrap de la app (capa 2) — todo en orden, con esperas, en un comando. Los secrets NO
#    salen al azar: la 1a corrida escribe deploy/k8s/secrets.env (internos ya generados y
#    visibles; ADMIN_PASSWORD/SMTP_PASSWORD en blanco) y se detiene; tú completas esos dos, y la
#    2a corrida crea los secrets desde ese archivo y lo borra. Al final también aplica el
#    NetworkPolicy (punto 4 de las mejoras, más abajo) — ya no es un paso manual aparte.
powershell -ExecutionPolicy Bypass -File deploy/k8s/bootstrap.ps1

# 7. DNS: apuntar el registro A de api. a la nueva IP (terraform output ingress_ip); esperar el cert
# 8. (si preservas) restaurar: gunzip -c <archivo>.sql.gz | kubectl exec -i postgres-0 -n training-center -- psql -U postgres -d training_center
```
   Los secrets de GitHub (WIF_PROVIDER, GCP_SERVICE_ACCOUNT) NO cambian: Terraform recrea el provider
   y la SA con los mismos nombres → el CI/CD sigue funcionando. Lo único nuevo es la IP (paso 7).

**4. NetworkPolicy** (`deploy/k8s/network/policies.yaml`) — ya no es un paso manual aparte: el
   `bootstrap.ps1` la aplica automáticamente como último paso (y la skill `/recreate-environment`
   la incluye en su paso 9). Solo se hace cumplir con el enforcer de Dataplane V2 (punto 3); sin
   él, GKE acepta las políticas pero NO las hace cumplir (letrero sin guardia) — comprobar con
   `gcloud container clusters describe ... --format="value(networkConfig.datapathProvider)"` →
   debe decir `ADVANCED_DATAPATH`.

   Diseño (pass 1, **solo ingress**): `default-deny-ingress` en el namespace + allows de los
   flujos reales — rangos del LB de Google (`130.211.0.0/22`, `35.191.0.0/16`) → `api:8080`;
   mismo namespace → `postgres:5432`, `redis:6379`, `rabbitmq:5672`; namespace `keda` →
   `rabbitmq:5672` (así lee la profundidad de la cola). El origen de los datastores es "mismo
   namespace", no por-app: el mayor riesgo real —el `judge-worker`, que corre código no
   confiable— ya es cliente legítimo de postgres/rabbitmq, así que ir más estricto por-app no lo
   frena y sí añade fragilidad (los Jobs migrate/seed/backup no tienen un label estable propio).
   Egress se deja abierto a propósito (fase 2 deliberada, no aplicada): cerrarlo es frágil (DNS,
   GCS, SMTP, metadata, pull de imágenes del judge) y el riesgo que de verdad importa —que una
   submission exfiltre datos— ya está resuelto en otra capa: los contenedores de submission
   corren con `NetworkMode: "none"` (`internal/adapter/judge/pool/pool.go`), sin red en absoluto.

   **Probarlo** tras cada aplicación: `kubectl get pods -n training-center` sano, KEDA sigue
   leyendo la cola, el api responde por el Ingress — DPv2 descarta el tráfico no permitido en
   silencio, así que una regla de más se nota como un fallo "misterioso", no como un error claro.

---

## Orden de dependencias

```
Fase 0 → 1 → 2 → 3 → 4 → 5
                      ↘ 6 → 7
                 (5 y 6 son independientes entre sí; 8 es transversal)
```

Hito mínimo demostrable: **Fase 4** (API funcional en el cluster).
Hito de tesis completo: **Fase 7** (judging autoescalado por cola).

---

## Chuleta — apagar/encender/demoler (para no olvidar y ahorrar crédito)

Todos los comandos van desde **cmd** (por los gotchas de Windows de la Fase 0). Ajustar zona/cluster si cambian.

### Al TERMINAR una sesión (apagar cómputo, conservar todo lo demás)

```bash
# 1. apagar los nodos del pool default (Postgres, Redis, RabbitMQ, API, KEDA)
#    --node-pool es OBLIGATORIO: con judge-pool también existiendo, gcloud no adivina cuál
gcloud container clusters resize training-center --zone us-east1-b --node-pool default-pool --num-nodes 0 --quiet

# 2. verificar que no queda cómputo facturando (lista vacía = OK)
gcloud compute instances list
```

El `judge-worker` lo gestiona KEDA: con la cola vacía ya está en 0 (y su nodo recogido por el
cluster autoscaler, que vive en el control plane), así que
no hay que escalarlo a mano. Si quisieras forzarlo a 0 aunque haya mensajes en cola (p. ej. para
un apagado tajante): `kubectl annotate scaledobject judge-worker -n training-center autoscaling.keda.sh/paused-replicas="0" --overwrite` (quitar con el sufijo `-` al volver).

Sobrevive apagado (cuesta centavos): control plane + config en etcd, discos PVC (datos de
Postgres/RabbitMQ), imágenes del registro, la IP estática y el certificado, el Load Balancer
del Ingress (~US$0.60/día — se deja vivo para no re-emitir el certificado cada día).

### Al VOLVER (encender)

```bash
# 1. reencender los nodos del pool default (--node-pool obligatorio)
gcloud container clusters resize training-center --zone us-east1-b --node-pool default-pool --num-nodes 2 --quiet

# 2. esperar a que todo esté Ready
kubectl get pods -w
```

El `judge-worker` NO se reenciende a mano: KEDA lo levanta solo cuando llegue la primera
submission (0→1, y el judge-pool comprará su nodo, ~2 min). Si pausaste KEDA al apagar,
quitar la anotación: `kubectl annotate scaledobject judge-worker -n training-center autoscaling.keda.sh/paused-replicas-`.

Si `kubectl` falla con "Una directiva de Control de aplicaciones bloqueó este archivo", re-aplicar
el wrapper de auth de la Fase 0 (`kubectl config set-credentials ... --exec-command=...gke-auth.cmd`).

### Si se acaba el crédito / se acaba el trial (demoler todo)

El trial se pausa solo al agotar el crédito; para recrear el entorno en otra cuenta o proyecto,
o para no dejar nada facturando, demoler en orden inverso a la creación:

```bash
kubectl delete namespace training-center                 # borra todos los workloads + PVCs (¡datos!)
gcloud container clusters delete training-center --zone us-east1-b --quiet
gcloud compute addresses delete training-center-ip --global --quiet
gcloud artifacts repositories delete training-center --location=us-east1 --quiet
gcloud storage rm --recursive gs://training-center-502916-uploads
```

Recrear desde cero: seguir este roadmap de la Fase 0 en adelante (o, si se implementó la mejora
de Terraform de la Fase 8, `terraform apply`).
