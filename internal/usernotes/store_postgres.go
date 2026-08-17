package usernotes

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var QueryTimeoutDuration = 5 * time.Second

// SaveHook runs inside the SaveNote transaction, after the note is inserted.
// Used for the transactional outbox: atomic note + event write.
type SaveHook func(ctx context.Context, tx pgx.Tx, noteID, userID string) error

type pgstore struct {
	pqdriver *pgxpool.Pool
	onSave   SaveHook
}

func (ps *pgstore) GetNoteByID(ctx context.Context, userID string, noteID string) (*Note, error) {
	query := `
		SELECT title, content, created_at, updated_at
		FROM user_notes
		WHERE id = $1 AND user_id = $2`

	usernote := &Note{
		ID:     noteID,
		UserID: userID,
	}

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := ps.pqdriver.QueryRow(
		ctx, query, noteID, userID,
	).Scan(
		&usernote.Title,
		&usernote.Content,
		&usernote.CreatedAt,
		&usernote.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed getting user note: %w", err)
	}

	return usernote, nil
}

func (ps *pgstore) SaveNote(ctx context.Context, note *Note) (string, error) {
	query := `
		INSERT INTO user_notes (id, title, content, user_id)
		VALUES (gen_random_uuid(), $1, $2, $3) RETURNING id`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tx, err := ps.pqdriver.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed beginning tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var noteID string
	err = tx.QueryRow(ctx, query,
		note.Title,
		note.Content,
		note.UserID,
	).Scan(&noteID)
	if err != nil {
		return "", fmt.Errorf("failed storing note: %w", err)
	}

	// Outbox: event written in the same tx as the note. Atomic — both or neither.
	if ps.onSave != nil {
		if err := ps.onSave(ctx, tx, noteID, note.UserID); err != nil {
			return "", fmt.Errorf("save hook failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed committing note: %w", err)
	}

	return noteID, nil
}

func NewPostgresStore(pqdriver *pgxpool.Pool, onSave SaveHook) store {
	return &pgstore{
		pqdriver: pqdriver,
		onSave:   onSave,
	}
}
