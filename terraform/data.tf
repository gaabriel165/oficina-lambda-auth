data "aws_ssm_parameter" "jwt_secret" {
  name            = local.jwt_secret_parameter_name
  with_decryption = true
}

data "aws_ssm_parameter" "internal_api_secret" {
  name            = local.internal_api_secret_parameter_name
  with_decryption = true
}

data "aws_lbs" "backend" {
  for_each = var.backend_service_tags

  tags = {
    "kubernetes.io/service-name" = each.value
  }
}

locals {
  discovered_backend_arns = { for service, lbs in data.aws_lbs.backend : service => tolist(lbs.arns) }

  discoverable_backends = {
    for service, arns in local.discovered_backend_arns :
    service => arns[0] if lookup(var.backend_urls, service, "") == "" && length(arns) > 0
  }
}

data "aws_lb" "backend" {
  for_each = local.discoverable_backends

  arn = each.value
}

locals {
  backend_urls = {
    for service in keys(var.backend_service_tags) :
    service => lookup(var.backend_urls, service, "") != "" ? var.backend_urls[service] : (
      contains(keys(local.discoverable_backends), service) ? "http://${data.aws_lb.backend[service].dns_name}" : "http://backend-not-found.invalid"
    )
  }
}

check "backend_load_balancers_present" {
  assert {
    condition = alltrue([
      for service, arns in local.discovered_backend_arns :
      lookup(var.backend_urls, service, "") != "" || length(arns) > 0
    ])
    error_message = "A microservice has no load balancer tagged with its kubernetes.io/service-name. Deploy the three Services first or set backend_urls."
  }
}
