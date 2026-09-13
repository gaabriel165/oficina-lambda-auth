data "terraform_remote_state" "k8s" {
  backend = "s3"

  config = {
    bucket = var.state_bucket
    key    = "k8s/terraform.tfstate"
    region = var.region
  }
}

data "terraform_remote_state" "db" {
  backend = "s3"

  config = {
    bucket = var.state_bucket
    key    = "db/terraform.tfstate"
    region = var.region
  }
}

locals {
  vpc_id                      = data.terraform_remote_state.k8s.outputs.vpc_id
  private_subnet_ids          = data.terraform_remote_state.k8s.outputs.private_subnet_ids
  jwt_secret_parameter_name   = data.terraform_remote_state.k8s.outputs.jwt_secret_parameter_name
  db_client_security_group_id = data.terraform_remote_state.db.outputs.db_client_security_group_id
  database_url_parameter_name = data.terraform_remote_state.db.outputs.database_url_parameter_name
}
