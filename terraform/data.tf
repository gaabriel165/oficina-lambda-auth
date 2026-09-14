data "aws_ssm_parameter" "jwt_secret" {
  name            = local.jwt_secret_parameter_name
  with_decryption = true
}

data "aws_ssm_parameter" "database_url" {
  name            = local.database_url_parameter_name
  with_decryption = true
}

data "aws_lbs" "backend" {
  tags = {
    "kubernetes.io/service-name" = var.backend_service_tag
  }
}

locals {
  discovered_backend_arns = tolist(data.aws_lbs.backend.arns)
  discover_backend        = var.backend_url == "" && length(local.discovered_backend_arns) > 0
}

data "aws_lb" "backend" {
  count = local.discover_backend ? 1 : 0
  arn   = local.discovered_backend_arns[0]
}

locals {
  backend_url = var.backend_url != "" ? var.backend_url : (
    local.discover_backend ? "http://${data.aws_lb.backend[0].dns_name}" : "http://backend-not-found.invalid"
  )
}

check "backend_load_balancer_present" {
  assert {
    condition     = var.backend_url != "" || length(local.discovered_backend_arns) > 0
    error_message = "No load balancer tagged kubernetes.io/service-name=${var.backend_service_tag} was found. Deploy the application Service first or set backend_url."
  }
}
