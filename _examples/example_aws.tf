
resource "aws_iam_role" "reactor" {
  name = "redshift-serverless-usage-limit-reporter"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_policy" "reactor" {
  name   = "redshift-serverless-usage-limit-reporter"
  path   = "/"
  policy = data.aws_iam_policy_document.reactor.json
}

resource "aws_cloudwatch_log_group" "reactor" {
  name              = "/aws/lambda/redshift-serverless-usage-limit-reporter"
  retention_in_days = 7
}

resource "aws_iam_role_policy_attachment" "reactor" {
  role       = aws_iam_role.reactor.name
  policy_arn = aws_iam_policy.reactor.arn
}

data "aws_iam_policy_document" "reactor" {
  statement {
    actions = [
      "ssm:GetParameter*",
      "ssm:DescribeParameters",
      "ssm:List*",
    ]
    resources = ["*"]
  }
  statement {
    actions = [
      "logs:GetLog*",
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
      "logs:GetQueryResults",
      "logs:StartQuery",
      "logs:StopQuery",
    ]
    resources = ["*"]
  }
  statement {
    actions = [
      "redshift-serverless:GetUsageLimit",
    ]
    resources = ["*"]
  }
}

resource "aws_sns_topic" "default" {
  name = "redshift-serverless-usage-limit-reporter"
}

resource "aws_sns_topic_subscription" "default" {
  topic_arn = aws_sns_topic.default.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_alias.reactor.arn
}

data "archive_file" "reactor_dummy" {
  type        = "zip"
  output_path = "${path.module}/reactor_dummy.zip"
  source {
    content  = "reactor_dummy"
    filename = "bootstrap"
  }
  depends_on = [
    null_resource.reactor_dummy
  ]
}

resource "null_resource" "reactor_dummy" {}

resource "aws_lambda_function" "reactor" {
  lifecycle {
    ignore_changes = all
  }

  function_name = "redshift-serverless-usage-limit-reporter"
  role          = aws_iam_role.reactor.arn
  architectures = ["arm64"]
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  filename      = data.archive_file.reactor_dummy.output_path
}

resource "aws_lambda_alias" "reactor" {
  lifecycle {
    ignore_changes = all
  }
  name             = "current"
  function_name    = aws_lambda_function.reactor.arn
  function_version = aws_lambda_function.reactor.version
}

resource "aws_lambda_permission" "default" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_alias.reactor.function_name
  qualifier     = aws_lambda_alias.reactor.name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.default.arn
}

resource "aws_ssm_parameter" "slack_bot_token" {
  name        = "/redshift-serverless-usage-limit-reporter/SLACK_BOT_TOKEN"
  description = "Slack bot token for redshift-serverless-usage-limit-reporter"
  type        = "SecureString"
  value       = local.slack_bot_token
}

resource "aws_ssm_parameter" "slack_channel" {
  name        = "/redshift-serverless-usage-limit-reporter/SLACK_CHANNEL"
  description = "SLACK_CHANNEL for redshift-serverless-usage-limit-reporter"
  type        = "String"
  value       = local.slack_channel
}
