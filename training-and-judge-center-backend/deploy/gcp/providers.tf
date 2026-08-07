# El "driver" que traduce recursos de Terraform a llamadas de la API de GCP.
# Usa las credenciales de gcloud (Application Default Credentials) de quien corra terraform.
provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}
