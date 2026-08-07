# Registro de imágenes Docker + la cleanup policy (releases eternas, 5 recientes, resto a 14 días).
resource "google_artifact_registry_repository" "backend" {
  project       = var.project_id
  location      = var.region
  repository_id = "training-center"
  format        = "DOCKER"

  cleanup_policies {
    id     = "keep-releases"
    action = "KEEP"
    condition {
      tag_state    = "TAGGED"
      tag_prefixes = ["v"]
    }
  }

  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }

  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      older_than = "1209600s" # 14 días
    }
  }

  depends_on = [google_project_service.apis]
}
