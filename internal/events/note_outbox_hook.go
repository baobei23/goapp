package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// NoteOutboxHook returns a usernotes.SaveHook that appends a note.created event
// to the outbox within the note's transaction. Signature matches usernotes.SaveHook.
func NoteOutboxHook(outbox *OutboxStore) func(ctx context.Context, tx pgx.Tx, noteID, userID string) error {
	return func(ctx context.Context, tx pgx.Tx, noteID, userID string) error {
		return outbox.AppendTx(ctx, tx, Event{
			EventType: EventTypeNoteCreated,
			UserID:    userID,
			EntityID:  noteID,
			Timestamp: time.Now(),
		})
	}
}
