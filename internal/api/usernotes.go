package api

import (
	"context"

	"github.com/baobei23/goapp/internal/usernotes"
)

func (a *API) RegisterNote(ctx context.Context, un *usernotes.Note) (*usernotes.Note, error) {
	return a.unotes.SaveNote(ctx, un)
}

func (a *API) ReadUserNote(ctx context.Context, userID string, noteID string) (*usernotes.Note, error) {
	return a.unotes.GetNoteByID(ctx, userID, noteID)
}
