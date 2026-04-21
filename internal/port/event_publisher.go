package port

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, body string) error
}
