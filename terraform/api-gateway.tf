resource "aws_apigatewayv2_api" "this" {
  name          = "${var.project_name}-gateway"
  description   = "Single entry point: CPF authentication on Lambda and routing to the microservices on Kubernetes"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_headers = ["*"]
    allow_methods = ["*"]
  }
}

resource "aws_cloudwatch_log_group" "api_gateway" {
  name              = "/aws/apigateway/${var.project_name}-gateway"
  retention_in_days = var.log_retention_days
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.this.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gateway.arn
    format = jsonencode({
      requestId          = "$context.requestId"
      ip                 = "$context.identity.sourceIp"
      requestTime        = "$context.requestTime"
      httpMethod         = "$context.httpMethod"
      routeKey           = "$context.routeKey"
      path               = "$context.path"
      status             = "$context.status"
      integrationStatus  = "$context.integrationStatus"
      integrationLatency = "$context.integrationLatency"
      responseLatency    = "$context.responseLatency"
      authorizerError    = "$context.authorizer.error"
      errorMessage       = "$context.error.message"
    })
  }
}

resource "aws_apigatewayv2_integration" "auth_cpf" {
  api_id                 = aws_apigatewayv2_api.this.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.auth_cpf.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "backend" {
  for_each = local.backend_urls

  api_id             = aws_apigatewayv2_api.this.id
  integration_type   = "HTTP_PROXY"
  integration_method = "ANY"
  integration_uri    = each.value

  request_parameters = {
    "overwrite:path" = "$request.path"
  }
}

resource "aws_apigatewayv2_authorizer" "jwt" {
  api_id                            = aws_apigatewayv2_api.this.id
  name                              = "${var.project_name}-jwt-authorizer"
  authorizer_type                   = "REQUEST"
  authorizer_uri                    = aws_lambda_function.authorizer.invoke_arn
  authorizer_payload_format_version = "2.0"
  enable_simple_responses           = true
  identity_sources                  = ["$request.header.Authorization"]
  authorizer_result_ttl_in_seconds  = var.authorizer_cache_ttl_seconds
}

resource "aws_apigatewayv2_route" "auth_cpf" {
  api_id    = aws_apigatewayv2_api.this.id
  route_key = "POST /auth/cpf"
  target    = "integrations/${aws_apigatewayv2_integration.auth_cpf.id}"
}

locals {
  public_routes = {
    "GET /health"                            = "os"
    "GET /swagger/{proxy+}"                  = "os"
    "POST /api/v1/auth/login"                = "os"
    "POST /api/v1/auth/register"             = "os"
    "GET /api/v1/service-orders/{id}/status" = "os"
    "POST /api/v1/webhooks/mercadopago"      = "billing"
  }

  protected_routes = {
    "ANY /api/v1/{proxy+}"             = "os"
    "ANY /api/v1/budgets"              = "billing"
    "ANY /api/v1/budgets/{proxy+}"     = "billing"
    "ANY /api/v1/payments/{proxy+}"    = "billing"
    "ANY /api/v1/parts"                = "execution"
    "ANY /api/v1/parts/{proxy+}"       = "execution"
    "ANY /api/v1/services"             = "execution"
    "ANY /api/v1/services/{proxy+}"    = "execution"
    "ANY /api/v1/work-orders"          = "execution"
    "ANY /api/v1/work-orders/{proxy+}" = "execution"
  }
}

resource "aws_apigatewayv2_route" "public_backend" {
  for_each = local.public_routes

  api_id    = aws_apigatewayv2_api.this.id
  route_key = each.key
  target    = "integrations/${aws_apigatewayv2_integration.backend[each.value].id}"
}

resource "aws_apigatewayv2_route" "protected_backend" {
  for_each = local.protected_routes

  api_id             = aws_apigatewayv2_api.this.id
  route_key          = each.key
  target             = "integrations/${aws_apigatewayv2_integration.backend[each.value].id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.jwt.id
}

resource "aws_ssm_parameter" "api_gateway_url" {
  name        = "/${var.project_name}/api_gateway_url"
  description = "Public URL of the gateway, used by the billing microservice to build the payment webhook address"
  type        = "String"
  value       = trimsuffix(aws_apigatewayv2_stage.default.invoke_url, "/")
}

resource "aws_lambda_permission" "auth_cpf_invoke" {
  statement_id  = "AllowAPIGatewayInvokeAuthCPF"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.auth_cpf.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.this.execution_arn}/*/*"
}

resource "aws_lambda_permission" "authorizer_invoke" {
  statement_id  = "AllowAPIGatewayInvokeAuthorizer"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.authorizer.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.this.execution_arn}/authorizers/*"
}
