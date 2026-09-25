data "google_project" "this" {}

resource "google_project_service" "services" {
  for_each = toset([
    "artifactregistry.googleapis.com",
    "billingbudgets.googleapis.com",
    "cloudbuild.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "sqladmin.googleapis.com",
    "sts.googleapis.com",
  ])

  service = each.value

  # Leaving APIs enabled costs nothing and avoids dependency errors on re-bootstrap.
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "app" {
  location      = var.region
  repository_id = "app"
  format        = "DOCKER"

  depends_on = [google_project_service.services]
}

# Cloud Build runs as the default compute service account. It needs to push
# images, write logs and read the uploaded source.
resource "google_project_iam_member" "cloudbuild" {
  for_each = toset([
    "roles/artifactregistry.writer",
    "roles/logging.logWriter",
    "roles/storage.objectViewer",
  ])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${local.compute_sa}"

  depends_on = [google_project_service.services]
}

resource "google_service_account" "api" {
  account_id   = "api-runtime"
  display_name = "Cloud Run api"

  depends_on = [google_project_service.services]
}

resource "google_service_account" "web" {
  account_id   = "web-runtime"
  display_name = "Cloud Run web"

  depends_on = [google_project_service.services]
}

resource "google_project_iam_member" "api_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.api.email}"
}
