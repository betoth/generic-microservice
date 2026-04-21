package sqs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

type dlqError struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Details string `json:"details"`
}

type dlqMessage struct {
	Error   dlqError        `json:"error"`
	Message json.RawMessage `json:"message"`
}

type DLQPublisher struct {
	api      sqsAPI
	queueURL string
}

func NewDLQPublisher(api *awssqs.Client, queueURL string) *DLQPublisher {
	return &DLQPublisher{api: api, queueURL: queueURL}
}

func (p *DLQPublisher) Publish(ctx context.Context, errorID, code, details, originalMessage string) error {
	msg := dlqMessage{
		Error: dlqError{
			ID:      errorID,
			Code:    code,
			Details: details,
		},
		Message: json.RawMessage(originalMessage),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = p.api.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}
