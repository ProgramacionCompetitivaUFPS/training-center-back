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
     300 pesos (~US$0.09) y las alertas se disparan el primer día (lección aprendida: 2026-07-20).
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

**Notas de la instalación real (2026-07-19):**
- PowerShell bloqueaba `gcloud.ps1` por execution policy → se trabaja desde **cmd** (usa `gcloud.cmd`, sin políticas).
- WDAC (Device Guard) bloqueaba el `kubectl` de Chocolatey, que ganaba en el PATH → se desinstalaron `minikube` + `kubernetes-cli` de Chocolatey. `kubectl` resuelve ahora a la copia de Docker Desktop (v1.34.1); la de Cloud SDK (v1.35) queda de respaldo.
- **Smart App Control bloqueó `gke-gcloud-auth-plugin.exe`** (2026-07-20; el día anterior funcionaba —
  la reputación en la nube cambió; el binario de Google NO está firmado). SAC no admite excepciones
  puntuales. Solución: wrapper `C:\Users\Ryzen 7\.kube\gke-auth.cmd` que obtiene el token con
  `gcloud auth print-access-token` y emite el JSON `ExecCredential`; el kubeconfig apunta a él:
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

   **⚠️ Gotcha (mordió el 2026-07-20)**: en proyectos GCP nuevos, la cuenta de servicio de cómputo
   por defecto nace SIN permisos → los nodos reciben `403 Forbidden` al hacer pull del Artifact
   Registry (`ImagePullBackOff` en todos los pods con imagen propia). Arreglo — lectura sobre el
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
4. **Jobs** (`deploy/k8s/app/jobs.yaml`) — migrate y seed, misma imagen, distinto `command`.
   - K8s no tiene `depends_on`: ambos arrancan a la vez; si seed le gana a migrate, falla y el
     Job reintenta (`backoffLimit: 2`) hasta converger. Un pod `Error` + uno `Completed` es normal.
   - Job `Failed` (reintentos agotados) = lápida: `kubectl delete job seed` + re-apply.
   - El seed es un **upsert de contraseña**: si el admin existe, la actualiza (`admin password updated`).
   - Cambiar la clave del admin después: patch del secret + re-correr seed. El patch sin peleas
     de comillas: `Set-Content` (PowerShell) del JSON a un archivo + `kubectl patch secret
     app-secrets --type merge --patch-file ...` + borrar el archivo.
5. **API** (`deploy/k8s/app/api.yaml`): Deployment (1 réplica) + Service. Claves:
   `serviceAccountName: backend` (activa Workload Identity), probes `httpGet /ping`,
   sin `command` (usa el ENTRYPOINT `/bin/api`).

```bash
kubectl apply -f deploy/k8s/app/configmap.yaml
kubectl apply -f deploy/k8s/app/jobs.yaml
kubectl wait --for=condition=complete job/migrate job/seed --timeout=180s
kubectl logs job/migrate && kubectl logs job/seed
kubectl apply -f deploy/k8s/app/api.yaml
kubectl port-forward svc/api 8080:8080
# navegador: http://localhost:8080/ping  y  /swagger/index.html → POST /auth/login con el admin
```

Verificado 2026-07-20: login del admin → 200 con token JWT. **Pendiente de verificar**: subida
de archivos a GCS vía Workload Identity (se probará con la primera creación de problema real).

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

2. **Dominio**: `trainingjudgecenter.com` (Hostinger, comprado 2026-07-20). Registro `A`:
   `api` → la IP reservada (panel DNS de Hostinger). Reparto: `api.` → GKE; apex y `www.` → Vercel (frontend).
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

**Estado final: una submission enviada por la API termina con veredicto real.**

**El problema**: en compose, el worker monta el socket Docker del host. En GKE los nodos
usan containerd — **no existe socket Docker en el nodo**. La solución que preserva el diseño
de `RUNNER_ARCHITECTURE.md` (pool de contenedores por pod, el pod no sabe que K8s existe):

- **Sidecar Docker-in-Docker**: el pod del worker lleva dos contenedores:
  1. `worker` (la imagen del backend, `command: ["/bin/worker"]`)
  2. `dind` (`docker:27-dind`, `securityContext: {privileged: true}`) — un daemon Docker
     completo y privado dentro del pod. Los contenedores de lenguaje del pool viven ahí.
- El worker apunta al daemon con `DOCKER_HOST=tcp://localhost:2375` (mismo pod = misma red).
  Verificar que el cliente Docker del worker se construye con `client.FromEnv` para que respete `DOCKER_HOST`.
- **Downward API** para `POD_MEMORY_LIMIT`, tal como prevé `RUNNER_ARCHITECTURE.md` §3:

```yaml
env:
  - name: POD_MEMORY_LIMIT
    valueFrom:
      resourceFieldRef:
        containerName: dind        # los contenedores de lenguaje viven en dind
        resource: limits.memory
```

- `resources.limits` explícitos en ambos contenedores — el semáforo de CPU y la contabilidad
  de memoria del pool se derivan de ahí; sin limits el diseño no funciona.
- Las imágenes de lenguaje se descargan dentro de dind en el primer uso (cold start).
  Opcional: initContainer que haga `docker pull` de las imágenes del `judge_config.yaml` al arrancar el pod.
- Si el pool queda degradado el worker hace `os.Exit` → K8s lo reinicia solo (`restartPolicy: Always`). El diseño ya cuenta con esto.

Dimensionamiento inicial: 1 réplica con limits ~2 CPU / 4-6 GiB (ajustar según `judge_config.yaml`).
Si el nodo `e2-standard-2` queda corto al correr todo junto, subir el node pool a `e2-standard-4`
(4 vCPU / 16 GB, ~$98/mes el nodo) y compensar apagando el cluster fuera de horas de trabajo.

Verificación end-to-end: submission por API → mensaje en RabbitMQ → worker compila y ejecuta → veredicto en la DB.

---

## Fase 7 — Autoscaling por profundidad de cola (KEDA)

**Estado final: los pods del worker escalan solos cuando la cola de RabbitMQ crece.**

El HPA nativo de K8s solo ve CPU/memoria; para escalar por profundidad de cola
(el diseño de `RUNNER_ARCHITECTURE.md` §2) se usa **KEDA**:

1. Instalar KEDA (manifests oficiales o Helm chart oficial de kedacore).
2. `ScaledObject` sobre el Deployment del worker con trigger `rabbitmq`:
   `queueLength` por réplica según la fórmula del doc (`maxConcurrent × threshold`),
   `minReplicaCount: 1`, `maxReplicaCount: 3`.
3. Autoscaling de nodos solo donde aporta: crear un **node pool dedicado** `judge-pool`
   (p. ej. `e2-standard-4`) con `--enable-autoscaling --min-nodes 0 --max-nodes 2` y un
   taint que solo los pods del worker toleran. El pool `default` (infra + API) queda de
   tamaño fijo → la rutina `resize 0/2` sigue siendo determinista.
   Cadena completa: cola crece → KEDA sube réplicas del worker → pods `Pending` →
   autoscaler crea nodo en `judge-pool` → cola vacía → KEDA baja réplicas →
   nodo vacío → autoscaler lo elimina (~10 min de gracia).

Verificación: inyectar N submissions de golpe y observar `kubectl get pods -w` + la cola en el management UI de RabbitMQ.

---

## Fase 8 — Operación y cierre

**Estado final: sabes qué gasta, dónde ver logs, y cómo apagar/encender todo.**

- **Logs**: `stdout` de los pods llega solo a Cloud Logging (Console → Logging → filtrar por namespace). `slog` en JSON se indexa por campos.
- **Monitoreo**: Console → Kubernetes Engine → Workloads muestra CPU/memoria por pod sin configurar nada.
- **Gasto**: Billing → Reports, agrupado por SKU, cada pocos días. Comparar contra la tabla de esta guía.
- **Rutina diaria**: resize a 0 al terminar, resize a 2 al volver (Fase 2). Es la diferencia entre 90 días de crédito y 30.
- **Mejoras opcionales** (si sobra tiempo de tesis):
  - CI/CD: GitHub Actions → build + push + `kubectl set image` (auth con Workload Identity Federation, sin keys).
  - Backup de Postgres: CronJob con `pg_dump` al bucket.
  - `NetworkPolicy` para aislar el namespace.
  - **Terraform para la capa GCP** (`deploy/gcp/main.tf`): codificar los ~8 recursos de nube
    (proyecto/APIs, cluster, node pools, Artifact Registry, bucket, IP estática, workload identity).
    Justificación: el trial expira → el entorno tendrá que recrearse; con IaC son minutos y cero
    omisiones, más `terraform destroy` para demoler limpio. Hacerlo AL FINAL, con el sistema ya
    entendido: el .tf debe ser transcripción de experiencia, no conjuro copiado. Los manifests
    de `deploy/k8s/` ya son la IaC de la capa cluster; esto completa la capa 1.

---

## Orden de dependencias

```
Fase 0 → 1 → 2 → 3 → 4 → 5
                      ↘ 6 → 7
                 (5 y 6 son independientes entre sí; 8 es transversal)
```

Hito mínimo demostrable: **Fase 4** (API funcional en el cluster).
Hito de tesis completo: **Fase 7** (judging autoescalado por cola).
