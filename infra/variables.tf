variable "localstack_endpoint" {
  description = "LocalStack endpoint URL"
  type        = string
  default     = "http://localhost:4566"
}

variable "aws_access_key" {
  description = "AWS access key"
  type        = string
  sensitive   = true
}

variable "aws_secret_key" {
  description = "AWS secret key"
  type        = string
  sensitive   = true
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "bucket_name" {
  description = "S3 bucket name"
  type        = string
  default     = "generic-microservice"
}

variable "queue_name" {
  description = "SQS queue name"
  type        = string
  default     = "generic-microservice"
}

variable "input_queue_name" {
  description = "SQS input queue name"
  type        = string
  default     = "input-generic-microservice"
}

variable "dlq_queue_name" {
  description = "SQS DLQ queue name"
  type        = string
  default     = "dlq-generic-microservice"
}
