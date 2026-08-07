# Habilita las APIs que el proyecto necesita. `for_each` crea un recurso por cada una.
resource "google_project_service" "apis" {
  for_each = toset([
    "container.googleapis.com",        # GKE
    "artifactregistry.googleapis.com", # registro de imágenes
    "storage.googleapis.com",          # GCS
    "compute.googleapis.com",          # VMs, IPs, discos
    "iamcredentials.googleapis.com",   # federación WIF (runtime)
    "sts.googleapis.com",              # intercambio de token WIF
  ])

  project = var.project_id
  service = each.value

  # No deshabilitar la API al hacer `terraform destroy` — otros recursos/proyectos podrían usarla.
  disable_on_destroy = false
}
