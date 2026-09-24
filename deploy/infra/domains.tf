# Custom domains via Cloud Run domain mappings (free; Google issues the TLS
# certificates). Point each domain at ghs.googlehosted.com with a DNS-only
# CNAME; see the dns_records output.
locals {
  domain_mappings = {
    for service, domain in {
      web = var.web_domain
      api = var.api_domain
    } : service => domain if domain != ""
  }
}

resource "google_cloud_run_domain_mapping" "custom" {
  for_each = local.domain_mappings

  name     = each.value
  location = var.region

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = each.key == "web" ? google_cloud_run_v2_service.web.name : google_cloud_run_v2_service.api.name
  }
}
