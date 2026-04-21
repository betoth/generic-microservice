package port

import "context"

type DLQPublisher interface {
	Publish(ctx context.Context, errorID, code, details, originalMessage string) error
}
