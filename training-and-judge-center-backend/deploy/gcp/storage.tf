# Bucket de archivos de problemas y submissions (el nombre lleva el project_id porque es global).
resource "google_storage_bucket" "uploads" {
  project                     = var.project_id
  name                        = "${var.project_id}-uploads"
  location                    = var.region
  uniform_bucket_level_access = true

  depends_on = [google_project_service.apis]
}
