# GCP reserves a deleted instance's name for about a week, so a random suffix
# lets teardown.sh followed by bootstrap.sh work right away.
resource "random_id" "db_suffix" {
  byte_length = 3
}

resource "google_sql_database_instance" "db" {
  name             = "lemonade-db-${random_id.db_suffix.hex}"
  database_version = "POSTGRES_16"
  region           = var.region

  deletion_protection = var.deletion_protection

  settings {
    # "ENTERPRISE" is Cloud SQL's standard (cheapest) edition, available to every
    # account, not an enterprise billing plan. Postgres 16 may otherwise default
    # to Enterprise Plus, which is pricier and doesn't offer db-f1-micro.
    edition           = "ENTERPRISE"
    tier              = var.db_tier
    availability_type = "ZONAL"
    disk_size         = 10
    disk_autoresize   = false

    deletion_protection_enabled = var.deletion_protection
    user_labels                 = local.labels

    # Public IP with no authorized networks: only reachable through the Cloud SQL connector.
    ip_configuration {
      ipv4_enabled = true
    }

    backup_configuration {
      enabled = true
    }
  }

  depends_on = [google_project_service.services]
}

resource "google_sql_database" "app" {
  name     = "sample"
  instance = google_sql_database_instance.db.name
}

resource "random_password" "db" {
  length  = 32
  special = false
}

resource "google_sql_user" "app" {
  name     = "app"
  instance = google_sql_database_instance.db.name
  password = random_password.db.result
}

resource "google_secret_manager_secret" "db_password" {
  secret_id = "db-password"

  replication {
    auto {}
  }

  depends_on = [google_project_service.services]
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db.result
}

resource "google_secret_manager_secret_iam_member" "api_db_password" {
  secret_id = google_secret_manager_secret.db_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api.email}"
}
