# Bootstrap de la CAPA 2 (la app) sobre un cluster GKE recién creado por Terraform.
# Aplica los manifests en orden, resuelve la última imagen v* del registro, instala KEDA
# y crea los secrets. Idempotente: se puede re-correr sin daño.
#
# Secrets: la primera corrida escribe la plantilla secrets.env (DB/JWT/RabbitMQ/
# ROTATION_CACHE_ENCRYPTION_KEY ya generados y visibles; ADMIN y SMTP en blanco) y se
# detiene. Complétala y vuelve a correr: el script crea los secrets desde ese archivo y
# luego lo borra (es texto plano).
#
# Prerrequisitos (capa 1, aparte):
#   1. terraform apply                                              (cluster + infra de nube)
#   2. gcloud container clusters get-credentials training-center --zone us-east1-b
#   3. Si el registro nacio vacio (recreacion/migracion): backend YA empujado (imagen v*) Y
#      las 4 imagenes del sandbox del judge (scripts/build-judge-images.sh + tag + push), en
#      las versiones que declara judge/images-configmap.yaml. Si faltan, este script corre
#      entero sin error, pero el judge-worker queda en crash-loop silencioso
#      (prepull-language-images: "manifest unknown") hasta que las subas.
#
# Uso (desde cmd, para saltar la execution policy de PowerShell):
#   powershell -ExecutionPolicy Bypass -File deploy\k8s\bootstrap.ps1

$ErrorActionPreference = "Stop"
$NS  = "training-center"
$REG = "us-east1-docker.pkg.dev/training-center-502916/training-center/backend"
$K8S = $PSScriptRoot

function New-Pass { -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 40 | ForEach-Object { [char]$_ }) }
# ROTATION_CACHE_ENCRYPTION_KEY debe ser base64 de exactamente 32 bytes crudos (lo exige
# NewRedisRotationCache) generados con un RNG criptográfico — New-Pass no sirve (no es base64
# estructurado) y Get-Random tampoco (no es criptográficamente seguro, no apto para una llave
# AES-256-GCM real).
function New-Base64Key {
  $bytes = [byte[]]::new(32)
  [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
  [Convert]::ToBase64String($bytes)
}
function Apply-Tpl($file) { (Get-Content "$K8S/$file") -replace '__IMAGE__', $script:IMG | kubectl apply -f - }

# 0. Resolver la última imagen v* del registro (nada quemado)
$LATEST = gcloud artifacts docker images list $REG --include-tags --filter="tags~^v" --sort-by="~UPDATE_TIME" --format="value(tags)" --limit=1 2>$null
if (-not $LATEST) { throw "No hay imagen v* en el registro. Corre el CI/CD (tag vX.Y.Z) primero." }
$script:IMG = "${REG}:${LATEST}"
Write-Host "==> Imagen a desplegar: $script:IMG" -ForegroundColor Cyan

# 1. Namespace
kubectl apply -f "$K8S/namespace.yaml"

# 2. Secrets (solo si no existen). Se leen de secrets.env que TÚ completas — nada al azar.
if (-not (kubectl get secret app-secrets -n $NS --ignore-not-found 2>$null)) {
  $envFile = "$K8S/secrets.env"

  # Primera corrida: deja la plantilla (internos ya generados y visibles) y se detiene.
  if (-not (Test-Path $envFile)) {
    @(
      "DB_PASSWORD=$(New-Pass)"
      "JWT_SECRET=$(New-Pass)"
      "RABBITMQ_PASSWORD=$(New-Pass)"
      "ROTATION_CACHE_ENCRYPTION_KEY=$(New-Base64Key)"
      "ADMIN_PASSWORD="
      "SMTP_PASSWORD="
    ) | Set-Content -Encoding ascii $envFile
    throw "Escribí la plantilla $envFile con DB/JWT/RabbitMQ/ROTATION_CACHE_ENCRYPTION_KEY ya generados. Completa ADMIN_PASSWORD y SMTP_PASSWORD (y cambia los internos si quieres) y vuelve a correr."
  }

  # Segunda corrida: parsea, valida que no queden vacíos, y crea los secrets.
  $secrets = @{}
  Get-Content $envFile | Where-Object { $_ -match '^\s*[^#].*=' } | ForEach-Object {
    $k, $v = $_ -split '=', 2
    $secrets[$k.Trim()] = $v
  }
  foreach ($k in 'DB_PASSWORD', 'JWT_SECRET', 'RABBITMQ_PASSWORD', 'ROTATION_CACHE_ENCRYPTION_KEY', 'ADMIN_PASSWORD', 'SMTP_PASSWORD') {
    if ([string]::IsNullOrWhiteSpace($secrets[$k])) { throw "Falta $k en $envFile. Complétalo y vuelve a correr." }
  }
  # Falla rápido acá, no varios kubectl apply / crash-loops después: ROTATION_CACHE_ENCRYPTION_KEY
  # tiene que decodificar a exactamente 32 bytes crudos, o el pod de la API nunca arranca.
  try {
    $keyBytes = [Convert]::FromBase64String($secrets['ROTATION_CACHE_ENCRYPTION_KEY'])
  } catch {
    throw "ROTATION_CACHE_ENCRYPTION_KEY en $envFile no es base64 válido."
  }
  if ($keyBytes.Length -ne 32) {
    throw "ROTATION_CACHE_ENCRYPTION_KEY en $envFile decodifica a $($keyBytes.Length) bytes, se necesitan exactamente 32."
  }

  Write-Host "==> Creando secrets desde $envFile" -ForegroundColor Cyan
  kubectl create secret generic app-secrets -n $NS --from-env-file=$envFile
  kubectl create secret generic keda-rabbitmq -n $NS `
    --from-literal=host="amqp://judge:$($secrets['RABBITMQ_PASSWORD'])@rabbitmq.$NS.svc.cluster.local:5672/"

  Remove-Item $envFile
  Write-Host "==> secrets.env borrado (era texto plano)" -ForegroundColor Cyan
}

# 3. Infraestructura (Postgres, Redis, RabbitMQ) y esperar a que esté lista
Write-Host "==> Infraestructura" -ForegroundColor Cyan
kubectl apply -f "$K8S/infra/postgres.yaml" -f "$K8S/infra/redis.yaml" -f "$K8S/infra/rabbitmq.yaml"
kubectl rollout status statefulset/postgres -n $NS --timeout=180s
kubectl rollout status statefulset/rabbitmq -n $NS --timeout=180s

# 4. App: serviceaccount + config + migraciones + seed + API
Write-Host "==> App" -ForegroundColor Cyan
kubectl apply -f "$K8S/app/serviceaccount.yaml" -f "$K8S/app/configmap.yaml"
kubectl delete job migrate seed -n $NS --ignore-not-found
Apply-Tpl "app/migrate-job.yaml"
Apply-Tpl "app/seed-job.yaml"
kubectl wait --for=condition=complete job/migrate job/seed -n $NS --timeout=180s
Apply-Tpl "app/api.yaml"

# 5. KEDA (operador) + worker del judge + ScaledObject (la cola que escala sola)
Write-Host "==> KEDA + judge" -ForegroundColor Cyan
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.20.1/keda-2.20.1.yaml
kubectl wait --for=condition=available deployment/keda-operator -n keda --timeout=180s
kubectl apply -f "$K8S/judge/images-configmap.yaml"
Apply-Tpl "judge/worker.yaml"
kubectl apply -f "$K8S/judge/keda.yaml"

# 6. Ingress + backup programado + limpieza de refresh tokens
Write-Host "==> Ingress + backup + cleanup-sessions" -ForegroundColor Cyan
kubectl apply -f "$K8S/ingress/ingress.yaml" -f "$K8S/infra/backup-cronjob.yaml"
Apply-Tpl "app/cleanup-sessions-cronjob.yaml"

# 7. NetworkPolicy (endurecimiento: default-deny ingress + allows de los flujos reales).
#    Solo se aplican de verdad en un cluster con Dataplane V2.
Write-Host "==> NetworkPolicy" -ForegroundColor Cyan
kubectl apply -f "$K8S/network/policies.yaml"

Write-Host "`n==> Bootstrap completo. Verifica: kubectl get pods -n $NS" -ForegroundColor Green
