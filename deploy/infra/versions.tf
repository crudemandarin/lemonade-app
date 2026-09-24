terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # The bucket is passed at init time: terraform init -backend-config=bucket=<bucket>
  backend "gcs" {
    prefix = "lemonade"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region

  default_labels = local.labels
}
