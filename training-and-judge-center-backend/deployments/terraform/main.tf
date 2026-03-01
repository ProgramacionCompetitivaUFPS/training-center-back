terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# GKE Cluster
resource "google_container_cluster" "primary" {
  name     = "${var.app_name}-cluster"
  location = var.region

  remove_default_node_pool = true
  initial_node_count       = 1

  network    = "default"
  subnetwork = "default"
}

resource "google_container_node_pool" "primary_nodes" {
  name       = "${var.app_name}-node-pool"
  location   = var.region
  cluster    = google_container_cluster.primary.name
  node_count = var.node_count

  node_config {
    machine_type = var.machine_type
    disk_size_gb = 10

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

# Kubernetes provider
provider "kubernetes" {
  host                   = "https://${google_container_cluster.primary.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(google_container_cluster.primary.master_auth[0].cluster_ca_certificate)
}

data "google_client_config" "default" {}

# Kubernetes Deployment
resource "kubernetes_deployment" "api" {
  depends_on = [google_container_node_pool.primary_nodes]

  metadata {
    name = "${var.app_name}-api"
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "${var.app_name}-api"
      }
    }

    template {
      metadata {
        labels = {
          app = "${var.app_name}-api"
        }
      }

      spec {
        container {
          name  = "api"
          image = var.container_image

          port {
            container_port = 8080
          }

          env {
            name  = "DB_HOST"
            value = var.db_host
          }
          env {
            name  = "DB_PORT"
            value = var.db_port
          }
          env {
            name  = "DB_USER"
            value = var.db_user
          }
          env {
            name  = "DB_PASSWORD"
            value = var.db_password
          }
          env {
            name  = "DB_NAME"
            value = var.db_name
          }
          env {
            name  = "PORT"
            value = "8080"
          }
        }
      }
    }
  }
}

# Kubernetes Service
resource "kubernetes_service" "api" {
  metadata {
    name = "${var.app_name}-api"
  }

  spec {
    selector = {
      app = "${var.app_name}-api"
    }

    port {
      port        = 80
      target_port = 8080
    }

    type = "LoadBalancer"
  }
}
