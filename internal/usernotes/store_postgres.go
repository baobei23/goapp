package usernotes

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naughtygopher/errors"
)

var QueryTimeoutDuration = 5 * time.Second

type pgstore struct {
	pqdriver  *pgxpool.Pool
	tableName string
}

func (ps *pgstore) GetNoteByID(ctx context.Context, userID string, noteID string) (*Note, error) {
	query := fmt.Sprintf(`
		SELECT title, content, created_at, updated_at
		FROM %s
		WHERE id = $1 AND user_id = $2`,
		ps.tableName,
	)

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
		return nil, errors.Wrap(err, "failed getting user note")
	}

	return usernote, nil
}

func (ps *pgstore) SaveNote(ctx context.Context, note *Note) (string, error) {
	noteID := ps.newNoteID()

	query := fmt.Sprintf(`
		INSERT INTO %s (id, title, content, user_id)
		VALUES ($1, $2, $3, $4)`,
		ps.tableName,
	)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := ps.pqdriver.Exec(ctx, query,
		noteID,
		note.Title,
		note.Content,
		note.UserID,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed storing note")
	}

	return noteID, nil
}

func (ps *pgstore) newNoteID() string {
	return uuid.New().String()
}

func NewPostgresStore(pqdriver *pgxpool.Pool, tableName string) store {
	return &pgstore{
		pqdriver:  pqdriver,
		tableName: tableName,
	}
}
