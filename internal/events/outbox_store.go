package events

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxStore struct {
	db *pgxpool.Pool
}

type OutboxEntry struct {
	ID        string
	Event     Event
	CreatedAt time.Time
	Retries   int
}

// AppendTx writes an event to the outbox within the caller's transaction.
func (os *OutboxStore) AppendTx(ctx context.Context, tx pgx.Tx, event Event) error {

	query := `
		INSERT INTO outbox (event_type, user_id, entity_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := tx.Exec(ctx, query,
		event.EventType,
		event.UserID,
		event.EntityID,
		event.Metadata,
		event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("outbox append: %w", err)
	}
	return nil
}

// GetPending fetches up to limit unprocessed events, oldest first.
// ponytail: no FOR UPDATE SKIP LOCKED — safe for a single relay instance.
// Add row locking if you run multiple relays to avoid double-publish.
func (os *OutboxStore) GetPending(ctx context.Context, limit int) ([]OutboxEntry, error) {

	query := `
		SELECT id, event_type, user_id, entity_id, metadata, created_at, retries
		FROM outbox
		WHERE processed_at IS NULL
		ORDER BY created_at
		LIMIT $1`
	rows, err := os.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var meta map[string]any
		if err := rows.Scan(&e.ID, &e.Event.EventType, &e.Event.UserID, &e.Event.EntityID, &meta, &e.CreatedAt, &e.Retries); err != nil {
			return nil, err
		}
		e.Event.Metadata = meta
		e.Event.Timestamp = e.CreatedAt
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// MarkProcessed marks an outbox entry as published.
func (os *OutboxStore) MarkProcessed(ctx context.Context, id string) error {
	_, err := os.db.Exec(ctx, `UPDATE outbox SET processed_at = now() WHERE id = $1`, id)
	return err
}

// IncrementRetries bumps retry count for a failed publish.
func (os *OutboxStore) IncrementRetries(ctx context.Context, id string) error {
	_, err := os.db.Exec(ctx, `UPDATE outbox SET retries = retries + 1 WHERE id = $1`, id)
	return err
}

func NewOutboxStore(db *pgxpool.Pool) *OutboxStore {
	return &OutboxStore{db: db}
}
