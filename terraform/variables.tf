variable "region" {
  description = "AWS region to provision into"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name prefix applied to all resources"
  type        = string
  default     = "oficina-api"
}

variable "state_bucket" {
  description = "S3 bucket holding the Terraform state of the other repositories"
  type        = string
  default     = "oficina-api-terraform-state-728750563430"
}

variable "backend_urls" {
  description = "Base URL of each microservice on Kubernetes, keyed by os, billing and execution. Leave a key out to discover the NLB created by its Service"
  type        = map(string)
  default     = {}
}

variable "backend_service_tags" {
  description = "Value of the kubernetes.io/service-name tag (namespace/name) on the NLB of each microservice"
  type        = map(string)
  default = {
    os        = "oficina-api/oficina-api"
    billing   = "oficina-billing/oficina-billing-service"
    execution = "oficina-execution/oficina-execution-service"
  }
}

variable "token_ttl_seconds" {
  description = "Lifetime of the JWT issued after CPF authentication"
  type        = number
  default     = 3600
}

variable "log_retention_days" {
  description = "CloudWatch Logs retention for Lambda and API Gateway logs"
  type        = number
  default     = 7
}

variable "authorizer_cache_ttl_seconds" {
  description = "How long API Gateway caches an authorizer decision for the same Authorization header"
  type        = number
  default     = 300
}
