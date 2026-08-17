package events

import (
	"context"
	"log/slog"
	"time"
)

// OutboxRelay polls outbox for pending events and publishes them to Kafka.
type OutboxRelay struct {
	outbox   *OutboxStore
	producer *Producer
}

func NewOutboxRelay(outbox *OutboxStore, producer *Producer) *OutboxRelay {
	return &OutboxRelay{
		outbox:   outbox,
		producer: producer,
	}
}

// Run polls every interval until ctx is cancelled.
func (r *OutboxRelay) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "[outbox] relay started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.processBatch(ctx); err != nil {
				slog.ErrorContext(ctx, "[outbox] batch failed", "err", err)
			}
		}
	}
}

func (r *OutboxRelay) processBatch(ctx context.Context) error {
	entries, err := r.outbox.GetPending(ctx, 100)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	for _, entry := range entries {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := r.producer.Publish(pubCtx, entry.Event)
		cancel()

		if err != nil {
			slog.ErrorContext(ctx, "[outbox] publish failed, will retry", "id", entry.ID, "err", err)
			_ = r.outbox.IncrementRetries(ctx, entry.ID)
			continue
		}

		if err := r.outbox.MarkProcessed(ctx, entry.ID); err != nil {
			slog.ErrorContext(ctx, "[outbox] mark processed failed", "id", entry.ID, "err", err)
		}
	}

	return nil
}
