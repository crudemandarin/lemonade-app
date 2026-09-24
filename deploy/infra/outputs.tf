output "web_url" {
  value = google_cloud_run_v2_service.web.uri
}

output "api_url" {
  value = google_cloud_run_v2_service.api.uri
}

output "registry" {
  value = local.registry
}

output "sql_instance" {
  value = google_sql_database_instance.db.name
}

output "sql_connection_name" {
  value = google_sql_database_instance.db.connection_name
}

output "dns_records" {
  description = "DNS records to create at your DNS provider (DNS only / not proxied)."
  value = {
    for service, mapping in google_cloud_run_domain_mapping.custom :
    mapping.name => [for r in mapping.status[0].resource_records : "${r.type} ${r.rrdata}"]
  }
}

output "github_workload_identity_provider" {
  value = one(google_iam_workload_identity_pool_provider.github[*].name)
}

output "github_service_account" {
  value = one(google_service_account.deployer[*].email)
}
