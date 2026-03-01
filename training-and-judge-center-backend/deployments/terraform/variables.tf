variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "training-center"
}

variable "machine_type" {
  description = "GKE node machine type"
  type        = string
  default     = "e2-micro"
}

variable "node_count" {
  description = "Number of nodes in the cluster"
  type        = number
  default     = 1
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "training_center"
}

variable "db_host" {
  description = "Database host"
  type        = string
  default     = "localhost"
}

variable "db_port" {
  description = "Database port"
  type        = string
  default     = "5432"
}

variable "db_user" {
  description = "Database user"
  type        = string
  default     = "postgres"
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
  default     = "postgres123"
}

variable "container_image" {
  description = "Container image URL (e.g., gcr.io/PROJECT_ID/training-center-api:latest)"
  type        = string
  default     = "gcr.io/your-project-id/training-center-api:latest"
}
