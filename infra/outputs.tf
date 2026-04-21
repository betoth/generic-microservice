output "s3_bucket_name" {
  description = "S3 bucket name"
  value       = aws_s3_bucket.entries.bucket
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = aws_sqs_queue.events.url
}

output "sqs_input_queue_url" {
  description = "SQS input queue URL"
  value       = aws_sqs_queue.input.url
}

output "sqs_dlq_url" {
  description = "SQS DLQ URL"
  value       = aws_sqs_queue.dlq.url
}
