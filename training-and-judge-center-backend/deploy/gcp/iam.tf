# Permisos IAM. TODOS usan recursos *_iam_member (aditivos): agregan UN binding sin tocar
# el resto. NUNCA *_iam_policy o *_iam_binding (autoritativos) — borrarían otros bindings,
# incluidos los service agents de Google y los accesos de tus compañeros.

# El principal de Workload Identity directo de la KSA `backend` (namespace training-center).
locals {
  backend_ksa = "principal://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${var.project_id}.svc.id.goog/subject/ns/training-center/sa/backend"
  node_sa     = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
  deployer_sa = "serviceAccount:${google_service_account.github_deployer.email}"
}

# La KSA backend escribe/lee el bucket de uploads (sin llaves, vía Workload Identity).
resource "google_storage_bucket_iam_member" "backend_uploads" {
  bucket = google_storage_bucket.uploads.name
  role   = "roles/storage.objectAdmin"
  member = local.backend_ksa
}

# Lectura del registro: la SA de los nodos (kubelet hace el pull) y la KSA backend (dind).
resource "google_artifact_registry_repository_iam_member" "node_reader" {
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.backend.repository_id
  role       = "roles/artifactregistry.reader"
  member     = local.node_sa
}

resource "google_artifact_registry_repository_iam_member" "backend_reader" {
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.backend.repository_id
  role       = "roles/artifactregistry.reader"
  member     = local.backend_ksa
}

# github-deployer: escribir imágenes en el registro + desplegar en GKE.
resource "google_artifact_registry_repository_iam_member" "deployer_writer" {
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.backend.repository_id
  role       = "roles/artifactregistry.writer"
  member     = local.deployer_sa
}

resource "google_project_iam_member" "deployer_container" {
  project = var.project_id
  role    = "roles/container.developer"
  member  = local.deployer_sa
}

# NOTA: los accesos de colaboradores humanos (roles de proyecto para el equipo) NO se
# gestionan aquí, a propósito: son administración del proyecto, no infra de despliegue.
# Dejarlos fuera del state evita que un `terraform destroy` le quite el acceso a alguien.
# Se otorgan aparte con `gcloud projects add-iam-policy-binding`.
