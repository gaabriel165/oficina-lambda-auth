data "terraform_remote_state" "k8s" {
  backend = "s3"

  config = {
    bucket = var.state_bucket
    key    = "k8s/terraform.tfstate"
    region = var.region
  }
}

locals {
  jwt_secret_parameter_name          = data.terraform_remote_state.k8s.outputs.jwt_secret_parameter_name
  internal_api_secret_parameter_name = data.terraform_remote_state.k8s.outputs.internal_api_secret_parameter_name
}
