# Valores del proyecto. Con defaults para que `terraform plan/apply` funcione sin -var,
# pero parametrizados para poder recrear en otro proyecto/región (el punto de la IaC).

variable "project_id" {
  description = "ID del proyecto GCP"
  type        = string
  default     = "training-center-502916"
}

variable "project_number" {
  description = "Número del proyecto GCP (para los principals de Workload Identity)"
  type        = string
  default     = "54498548428"
}

variable "region" {
  description = "Región de los recursos regionales (registro, bucket, node pools)"
  type        = string
  default     = "us-east1"
}

variable "zone" {
  description = "Zona del cluster — zonal exime la cuota de gestión (free tier)"
  type        = string
  default     = "us-east1-b"
}

variable "github_repo" {
  description = "Repo de GitHub autorizado en la federación WIF (owner/repo)"
  type        = string
  default     = "ProgramacionCompetitivaUFPS/training-center-back"
}
