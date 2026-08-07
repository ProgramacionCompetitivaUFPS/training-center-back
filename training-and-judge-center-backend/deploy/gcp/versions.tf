terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }

  # Estado remoto en GCS: compartido entre colaboradores, con locking y versionado.
  # ⚠️ El bucket se crea A MANO antes del primer `terraform init` (bootstrap chicken-and-egg):
  #   gcloud storage buckets create gs://training-center-502916-tfstate --location=us-east1 --uniform-bucket-level-access
  #   gcloud storage buckets update gs://training-center-502916-tfstate --versioning
  backend "gcs" {
    bucket = "training-center-502916-tfstate"
    prefix = "gke"
  }
}
