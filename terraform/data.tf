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

data "aws_lb" "backend" {
  count = var.backend_url == "" ? 1 : 0
  arn   = tolist(data.aws_lbs.backend.arns)[0]
}

locals {
  backend_url = var.backend_url != "" ? var.backend_url : "http://${data.aws_lb.backend[0].dns_name}"
}
