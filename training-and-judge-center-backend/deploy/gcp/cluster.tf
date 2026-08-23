# El cluster GKE. Node pools se gestionan aparte (patrón estándar): se crea con un pool
# temporal mínimo que se elimina de inmediato, y los pools reales son recursos separados.
resource "google_container_cluster" "primary" {
  name     = "training-center"
  project  = var.project_id
  location = var.zone # zonal → free tier de la cuota de gestión

  remove_default_node_pool = true
  initial_node_count       = 1

  # Dataplane V2 (eBPF/Cilium): el enforcer NATIVO de NetworkPolicy (desbloquea la mejora #4).
  # Solo se puede activar al CREAR el cluster — por eso hay que recrear.
  datapath_provider = "ADVANCED_DATAPATH"

  # Requerido por DPv2 (VPC-native); rangos auto-asignados.
  ip_allocation_policy {}

  # Workload Identity: la KSA backend se autentica ante GCS sin llaves.
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Este es un entorno de tesis que recreamos con Terraform: permitir `destroy`.
  deletion_protection = false

  depends_on = [google_project_service.apis]
}

# Pool default: infraestructura + API. Tamaño fijo (2), sin autoscaling → apagado determinista.
resource "google_container_node_pool" "default" {
  name     = "default-pool"
  project  = var.project_id
  cluster  = google_container_cluster.primary.name
  location = google_container_cluster.primary.location

  node_count = 2

  node_config {
    machine_type = "e2-standard-2"
    disk_size_gb = 30
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    # Necesario para que los pods usen Workload Identity vía el metadata server.
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }
}

# Pool judge: worker del juez. Autoscala 0..2 + taint que solo el worker tolera.
resource "google_container_node_pool" "judge" {
  name     = "judge-pool"
  project  = var.project_id
  cluster  = google_container_cluster.primary.name
  location = google_container_cluster.primary.location

  autoscaling {
    min_node_count = 0
    max_node_count = 2
  }

  node_config {
    machine_type = "e2-standard-8"
    disk_size_gb = 50
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    taint {
      key    = "judge"
      value  = "true"
      effect = "NO_SCHEDULE"
    }
  }
}
