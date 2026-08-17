package api

import (
	"context"
	"time"

	"github.com/baobei23/goapp/internal/events"
	"github.com/baobei23/goapp/internal/usernotes"
	"github.com/baobei23/goapp/internal/users"
)

// Server has all the methods required to run the server
type Server interface {
	Register(ctx context.Context, user *users.User) (*users.User, error)
	Login(ctx context.Context, email, password string) (*users.User, error)
	ReadUserByID(ctx context.Context, id string) (*users.User, error)
	ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error
	RegisterNote(ctx context.Context, un *usernotes.Note) (*usernotes.Note, error)
	ReadUserNote(ctx context.Context, userID string, noteID string) (*usernotes.Note, error)
	SaveRefreshToken(ctx context.Context, jti, userID string, expiresAt time.Time) error
	CheckRefreshToken(ctx context.Context, jti string) (bool, error)
	RevokeRefreshToken(ctx context.Context, jti string) error
	GetUserActivity(ctx context.Context, userID string, limit int) ([]events.Activity, error)
}

// Subscriber has all the methods required to run the subscriber
type Subscriber interface {
	AsyncRegisters(ctcx context.Context, users []users.User) error
}

type API struct {
	users  *users.Users
	unotes *usernotes.UserNotes
	astore activityStore
}

type activityStore interface {
	GetUserActivity(ctx context.Context, userID string, limit int) ([]events.Activity, error)
}

func New(us *users.Users, un *usernotes.UserNotes, as activityStore) *API {
	return &API{
		users:  us,
		unotes: un,
		astore: as,
	}
}

func NewServer(us *users.Users, un *usernotes.UserNotes, as activityStore) Server {
	return New(us, un, as)
}

func NewSubscriber(us *users.Users) Subscriber {
	return New(us, nil, nil)
}
