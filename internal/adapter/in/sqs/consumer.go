package sqs

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/betoth/generic-microservice/internal/port"
	"github.com/google/uuid"
)

type sqsConsumerAPI interface {
	ReceiveMessage(ctx context.Context, params *awssqs.ReceiveMessageInput, optFns ...func(*awssqs.Options)) (*awssqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *awssqs.DeleteMessageInput, optFns ...func(*awssqs.Options)) (*awssqs.DeleteMessageOutput, error)
}

type Consumer struct {
	api      sqsConsumerAPI
	queueURL string
	dlq      port.DLQPublisher
}

func NewConsumer(api *awssqs.Client, queueURL string, dlq port.DLQPublisher) *Consumer {
	return &Consumer{
		api:      api,
		queueURL: queueURL,
		dlq:      dlq,
	}
}

// Run polls the SQS input queue in a loop and calls handler for each message.
// On success: deletes the message from the queue.
// On handler error: sends to DLQ, then deletes from queue if DLQ succeeded.
// If DLQ send fails: does NOT delete the message (it returns after visibility timeout).
func (c *Consumer) Run(ctx context.Context, handler func(ctx context.Context, body string) error) error {
	for {
		out, err := c.api.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}

		for _, msg := range out.Messages {
			body := aws.ToString(msg.Body)
			handlerErr := handler(ctx, body)

			if handlerErr != nil {
				errorID := uuid.New().String()
				code, details := errorCodeAndDetails(handlerErr)

				if dlqErr := c.dlq.Publish(ctx, errorID, code, details, body); dlqErr != nil {
					// DLQ failed — do not delete, message returns after visibility timeout
					continue
				}
			}

			// Delete from input queue (success path or after DLQ confirmed)
			_, _ = c.api.DeleteMessage(ctx, &awssqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			})
		}
	}
}

func errorCodeAndDetails(err error) (code, details string) {
	var bizErr *domain.BusinessError
	if errors.As(err, &bizErr) {
		return bizErr.Code, bizErr.Description
	}
	return "500", err.Error()
}
