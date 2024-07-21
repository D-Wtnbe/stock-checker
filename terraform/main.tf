# Lambda関数用のIAMロール
resource "aws_iam_role" "lambda_role" {
  name = "buffalo_stock_checker_lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# CloudWatch Logsへのアクセス権限
resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Lambda関数
resource "aws_lambda_function" "stock_checker" {
  filename      = "function.zip" # ローカルのZIPファイルパス
  function_name = "buffalo_stock_checker"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["arm64"]

  environment {
    variables = {
      SLACK_WEBHOOK_URL = var.slack_webhook_url
    }
  }
}

# CloudWatch Logsグループ
resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.stock_checker.function_name}"
  retention_in_days = 14
}

# CloudWatch Eventsルール（定期実行用）
resource "aws_cloudwatch_event_rule" "every_sixty_minutes" {
  name                = "every-sixty-minutes"
  description         = "Fires every 60 minutes"
  schedule_expression = "rate(60 minutes)"
}

# CloudWatch EventsルールとLambda関数の紐付け
resource "aws_cloudwatch_event_target" "check_stock_every_fifteen_minutes" {
  rule      = aws_cloudwatch_event_rule.every_sixty_minutes.name
  target_id = "stock_checker"
  arn       = aws_lambda_function.stock_checker.arn
}

# Lambda関数の実行権限をCloudWatch Eventsに付与
resource "aws_lambda_permission" "allow_cloudwatch_to_call_check_stock" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.stock_checker.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every_sixty_minutes.arn
}

# 変数の定義
variable "slack_webhook_url" {
  description = "Slack Webhook URL for notifications"
  type        = string
}

# 出力
output "lambda_function_arn" {
  description = "The ARN of the Lambda Function"
  value       = aws_lambda_function.stock_checker.arn
}
