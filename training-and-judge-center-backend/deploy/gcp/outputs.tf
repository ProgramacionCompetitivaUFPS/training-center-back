# Valores que necesitas tras el apply para el bootstrap (DNS + secrets de GitHub).

output "ingress_ip" {
  description = "IP estática del Ingress — apuntar aquí el registro A de api.<dominio>"
  value       = google_compute_global_address.ingress.address
}

output "wif_provider" {
  description = "Nombre del provider WIF — valor del secret WIF_PROVIDER de GitHub"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "github_deployer_email" {
  description = "Email de la SA deployer — valor del secret GCP_SERVICE_ACCOUNT de GitHub"
  value       = google_service_account.github_deployer.email
}
