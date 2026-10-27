# IP estática global para el Load Balancer del Ingress (el LB anycast de Google la usa).
resource "google_compute_global_address" "ingress" {
  project = var.project_id
  name    = "training-center-ip"

  depends_on = [google_project_service.apis]
}
