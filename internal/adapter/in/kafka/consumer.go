package kafka

import (
	"context"
	"errors"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

// Run reads messages in a loop and calls handler for each one.
// The Kafka offset is committed only after handler returns nil.
// If handler returns a *domain.BusinessError the offset is committed (already handled).
// Any other error leaves the offset uncommitted and terminates the loop.
func (c *Consumer) Run(ctx context.Context, handler func(key, value string) error) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		err = handler(string(msg.Key), string(msg.Value))
		if err != nil {
			var bizErr *domain.BusinessError
			if errors.As(err, &bizErr) {
				// Business error: already handled, commit and continue
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					return commitErr
				}
				continue
			}
			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
