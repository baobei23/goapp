package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgHandler struct {
	db *pgxpool.Pool
}

// entityType maps an event type to the entity it concerns. Kept trivial on purpose.
func entityType(eventType string) string {
	switch eventType {
	case EventTypeNoteCreated, EventTypeNoteUpdated, EventTypeNoteDeleted:
		return "note"
	default:
		return "unknown"
	}
}

func (h *pgHandler) Record(ctx context.Context, event Event) error {
	meta, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = h.db.Exec(ctx, `
		INSERT INTO activity_log (event_type, user_id, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5)`,
		event.EventType,
		event.UserID,
		entityType(event.EventType),
		event.EntityID,
		meta,
	)
	if err != nil {
		return fmt.Errorf("insert activity_log: %w", err)
	}
	return nil
}

func NewPostgresHandler(db *pgxpool.Pool) Handler {
	return &pgHandler{db: db}
}
