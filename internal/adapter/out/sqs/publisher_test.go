package sqs

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

type fakeSQS struct {
	sentBody string
	err      error
}

func (f *fakeSQS) SendMessage(_ context.Context, params *awssqs.SendMessageInput, _ ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sentBody = aws.ToString(params.MessageBody)
	return &awssqs.SendMessageOutput{}, nil
}

func TestPublish_Success(t *testing.T) {
	fake := &fakeSQS{}
	p := &Publisher{api: fake, queueURL: "http://localhost:4566/queue/test"}

	if err := p.Publish(context.Background(), `{"business_key":"abc"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.sentBody != `{"business_key":"abc"}` {
		t.Errorf("unexpected body: %s", fake.sentBody)
	}
}

func TestPublish_Error(t *testing.T) {
	fake := &fakeSQS{err: errors.New("sqs unavailable")}
	p := &Publisher{api: fake, queueURL: "http://localhost:4566/queue/test"}

	if err := p.Publish(context.Background(), "body"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
