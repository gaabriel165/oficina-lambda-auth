output "api_gateway_url" {
  description = "Public invoke URL of the API Gateway (single entry point)"
  value       = aws_apigatewayv2_stage.default.invoke_url
}

output "auth_cpf_function_name" {
  description = "Name of the CPF authentication Lambda"
  value       = aws_lambda_function.auth_cpf.function_name
}

output "authorizer_function_name" {
  description = "Name of the JWT authorizer Lambda"
  value       = aws_lambda_function.authorizer.function_name
}

output "backend_urls" {
  description = "URL of each microservice on Kubernetes that the gateway proxies to"
  value       = local.backend_urls
}
