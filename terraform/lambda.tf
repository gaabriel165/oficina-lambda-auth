locals {
  auth_cpf_function_name   = "${var.project_name}-auth-cpf"
  authorizer_function_name = "${var.project_name}-authorizer"
  auth_cpf_package         = "${path.module}/../build/auth-cpf.zip"
  authorizer_package       = "${path.module}/../build/authorizer.zip"
}

resource "aws_cloudwatch_log_group" "auth_cpf" {
  name              = "/aws/lambda/${local.auth_cpf_function_name}"
  retention_in_days = var.log_retention_days
}

resource "aws_cloudwatch_log_group" "authorizer" {
  name              = "/aws/lambda/${local.authorizer_function_name}"
  retention_in_days = var.log_retention_days
}

resource "aws_lambda_function" "auth_cpf" {
  function_name    = local.auth_cpf_function_name
  description      = "Validates a customer CPF through the service order microservice and issues a JWT"
  role             = aws_iam_role.lambda_execution.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = local.auth_cpf_package
  source_code_hash = filebase64sha256(local.auth_cpf_package)
  memory_size      = 128
  timeout          = 10

  environment {
    variables = {
      CUSTOMER_API_URL    = local.backend_urls["os"]
      INTERNAL_API_SECRET = data.aws_ssm_parameter.internal_api_secret.value
      JWT_SECRET          = data.aws_ssm_parameter.jwt_secret.value
      TOKEN_TTL_SECONDS   = tostring(var.token_ttl_seconds)
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.auth_cpf,
    aws_iam_role_policy_attachment.lambda_basic_execution,
  ]
}

resource "aws_lambda_function" "authorizer" {
  function_name    = local.authorizer_function_name
  description      = "Validates the Bearer JWT before API Gateway forwards protected routes"
  role             = aws_iam_role.lambda_execution.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = local.authorizer_package
  source_code_hash = filebase64sha256(local.authorizer_package)
  memory_size      = 128
  timeout          = 10

  environment {
    variables = {
      JWT_SECRET = data.aws_ssm_parameter.jwt_secret.value
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.authorizer,
    aws_iam_role_policy_attachment.lambda_basic_execution,
  ]
}
