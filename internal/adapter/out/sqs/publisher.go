package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsAPI interface {
	SendMessage(ctx context.Context, params *awssqs.SendMessageInput, optFns ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error)
}

type Publisher struct {
	api      sqsAPI
	queueURL string
}

func NewPublisher(api *awssqs.Client, queueURL string) *Publisher {
	return &Publisher{api: api, queueURL: queueURL}
}

func (p *Publisher) Publish(ctx context.Context, body string) error {
	_, err := p.api.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(body),
	})
	return err
}
