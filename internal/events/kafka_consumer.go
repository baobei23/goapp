package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// Handler persists a consumed event. store_postgres implements it.
type Handler interface {
	Record(ctx context.Context, event Event) error
}

type Consumer struct {
	reader  *kafka.Reader
	handler Handler
}

func NewConsumer(brokers []string, topic, groupID string, handler Handler) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		handler: handler,
	}
}

// Run blocks until ctx is cancelled, consuming events and handing them to handler.
func (c *Consumer) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "[kafka] consumer started")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return c.reader.Close()
			}
			slog.ErrorContext(ctx, "[kafka] fetch failed", "err", err)
			continue
		}

		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			slog.ErrorContext(ctx, "[kafka] bad event, skipping", "err", err)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		if err := c.handler.Record(ctx, event); err != nil {
			slog.ErrorContext(ctx, "[kafka] record failed, will retry", "err", err)
			continue // don't commit: re-delivered on next fetch
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.ErrorContext(ctx, "[kafka] commit failed", "err", err)
		}
	}
}
