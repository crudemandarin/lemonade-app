# Keyless deploys from GitHub Actions (Workload Identity Federation): GitHub's
# OIDC token for a job on var.github_repo's main branch is exchanged for the
# github-deployer service account. No service account keys exist.

locals {
  github_enabled = var.github_repo != ""
  compute_sa     = "${data.google_project.this.number}-compute@developer.gserviceaccount.com"
}

# Deleted pools stay reserved for 30 days, so a suffix lets teardown.sh
# followed by bootstrap.sh work right away.
resource "random_id" "wif_suffix" {
  count       = local.github_enabled ? 1 : 0
  byte_length = 2
}

resource "google_iam_workload_identity_pool" "github" {
  count = local.github_enabled ? 1 : 0

  workload_identity_pool_id = "github-${random_id.wif_suffix[0].hex}"
  display_name              = "GitHub Actions"

  depends_on = [google_project_service.services]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  count = local.github_enabled ? 1 : 0

  workload_identity_pool_id          = google_iam_workload_identity_pool.github[0].workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
  }
  # Only this repo's main branch can authenticate.
  attribute_condition = "assertion.repository == '${var.github_repo}' && assertion.ref == 'refs/heads/main'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "deployer" {
  count = local.github_enabled ? 1 : 0

  account_id   = "github-deployer"
  display_name = "GitHub Actions deployer"

  depends_on = [google_project_service.services]
}

resource "google_service_account_iam_member" "deployer_wif" {
  count = local.github_enabled ? 1 : 0

  service_account_id = google_service_account.deployer[0].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github[0].name}/attribute.repository/${var.github_repo}"
}

# What deploy.sh needs: submit builds (and stream their logs, which requires
# viewer), and update the Cloud Run services.
resource "google_project_iam_member" "deployer" {
  for_each = local.github_enabled ? toset([
    "roles/cloudbuild.builds.editor",
    "roles/run.developer",
    "roles/viewer",
  ]) : toset([])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.deployer[0].email}"
}

# Upload build source. gcloud creates this bucket on the first build, which
# bootstrap.sh runs before this apply.
resource "google_storage_bucket_iam_member" "deployer_build_source" {
  count = local.github_enabled ? 1 : 0

  bucket = "${var.project_id}_cloudbuild"
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.deployer[0].email}"
}

# Act as the service accounts that builds (compute default) and Cloud Run
# revisions (api-runtime, web-runtime) run as.
resource "google_service_account_iam_member" "deployer_act_as" {
  for_each = local.github_enabled ? {
    compute = "projects/${var.project_id}/serviceAccounts/${local.compute_sa}"
    api     = google_service_account.api.name
    web     = google_service_account.web.name
  } : {}

  service_account_id = each.value
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer[0].email}"
}
