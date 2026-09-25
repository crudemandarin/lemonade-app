resource "google_cloud_run_v2_service" "api" {
  name                = "api"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  # deploy.sh sets the image after creation; don't let Terraform roll it back.
  # gcloud also adds an empty service-level scaling block on update.
  lifecycle {
    ignore_changes = [template[0].containers[0].image, client, client_version, scaling]
  }

  template {
    service_account = google_service_account.api.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.db.connection_name]
      }
    }

    containers {
      image = "${local.registry}/api:${var.image_tag}"

      ports {
        container_port = 8080
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }

      env {
        name  = "DB_HOST"
        value = "/cloudsql/${google_sql_database_instance.db.connection_name}"
      }
      env {
        name  = "DB_PORT"
        value = "5432"
      }
      env {
        name  = "DB_USERNAME"
        value = google_sql_user.app.name
      }
      env {
        name  = "DB_NAME"
        value = google_sql_database.app.name
      }
      env {
        name  = "TZ"
        value = "America/Denver"
      }
      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.db_password.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  depends_on = [
    google_project_iam_member.api_cloudsql,
    google_secret_manager_secret_iam_member.api_db_password,
    google_secret_manager_secret_version.db_password,
  ]
}

resource "google_cloud_run_v2_service" "web" {
  name                = "web"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  # deploy.sh sets the image after creation; don't let Terraform roll it back.
  # gcloud also adds an empty service-level scaling block on update.
  lifecycle {
    ignore_changes = [template[0].containers[0].image, client, client_version, scaling]
  }

  template {
    service_account = google_service_account.web.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = "${local.registry}/web:${var.image_tag}"

      ports {
        container_port = 80
      }

      env {
        name  = "API_UPSTREAM"
        value = google_cloud_run_v2_service.api.uri
      }
      # Cloud Run's metadata server resolves public DNS names.
      env {
        name  = "NGINX_RESOLVER"
        value = "169.254.169.254"
      }
    }
  }
}

# Both services are public. The browser only talks to web, which proxies /api to api.
resource "google_cloud_run_v2_service_iam_member" "public" {
  for_each = {
    api = google_cloud_run_v2_service.api.name
    web = google_cloud_run_v2_service.web.name
  }

  location = var.region
  name     = each.value
  role     = "roles/run.invoker"
  member   = "allUsers"
}
