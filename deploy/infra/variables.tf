variable "project_id" {
  description = "Google Cloud project to deploy into."
  type        = string
  default     = "lemonade-app-dev"
}

variable "region" {
  description = "Region for Cloud Run, Cloud SQL and Artifact Registry."
  type        = string
  default     = "us-central1"
}

variable "db_tier" {
  description = "Cloud SQL machine tier."
  type        = string
  default     = "db-f1-micro"
}

variable "deletion_protection" {
  description = "Protect the Cloud SQL instance from deletion. teardown.sh sets this to false before destroying."
  type        = bool
  default     = true
}

variable "image_tag" {
  description = "Image tag used only when the Cloud Run services are first created. Later images are set by deploy.sh; Terraform ignores them."
  type        = string
  default     = "latest"
}

variable "billing_account" {
  description = "Billing account ID for the budget alert (e.g. 015E56-C42150-9C05C1). Detected by the scripts. Empty to skip the budget."
  type        = string
  default     = ""
}

variable "budget_amount" {
  description = "Monthly budget in the billing account's currency. Alerts email billing admins at 50%, 90% and 100% of actual spend, and 100% of forecast."
  type        = number
  default     = 20
}

variable "github_repo" {
  description = "GitHub repo (owner/name) allowed to deploy from its main branch. Detected by the scripts from the git remote. Empty to skip GitHub Actions access."
  type        = string
  default     = ""
}

locals {
  labels = {
    app = "lemonade"
  }

  registry = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.app.repository_id}"
}

variable "web_domain" {
  description = "Custom domain for the web service. Must be under a domain verified in Google Search Console. Empty to disable."
  type        = string
  default     = ""
}

variable "api_domain" {
  description = "Custom domain for the api service. Empty to disable."
  type        = string
  default     = ""
}

variable "firebase_api_key" {
  description = "Web API key of the Firebase web app (Firebase console, Project settings, Your apps). Public, not a secret; see deploy/README.md, Sign in with Google."
  type        = string
  default     = ""
}

variable "firebase_auth_domain" {
  description = "Optional override of Firebase authDomain. Empty (default) uses the web host, which needs its /__/auth/handler added to the OAuth client (deploy/README.md). Set to <project>.firebaseapp.com to skip that."
  type        = string
  default     = ""
}

variable "firebase_app_id" {
  description = "App ID of the Firebase web app (looks like 1:123456789:web:abc123)."
  type        = string
  default     = ""
}
