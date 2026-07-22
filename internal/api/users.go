package api

import (
	"context"
	"time"

	"github.com/baobei23/goapp/internal/users"
)

// Register is the API to create/signup a new user
func (a *API) Register(ctx context.Context, u *users.User) (*users.User, error) {
	u, err := a.users.Register(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (a *API) Login(ctx context.Context, email, password string) (*users.User, error) {
	return a.users.Login(ctx, email, password)
}

// ReadUserByID is the API to read an existing user by their ID
func (a *API) ReadUserByID(ctx context.Context, id string) (*users.User, error) {
	u, err := a.users.ReadByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (a *API) ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error {
	return a.users.ChangePassword(ctx, id, oldPassword, newPassword)
}

func (a *API) AsyncRegisters(ctx context.Context, users []users.User) error {
	return a.users.AsyncRegisters(ctx, users)
}

func (a *API) SaveRefreshToken(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	return a.users.SaveRefreshToken(ctx, jti, userID, expiresAt)
}

func (a *API) CheckRefreshToken(ctx context.Context, jti string) (bool, error) {
	return a.users.CheckRefreshToken(ctx, jti)
}

func (a *API) RevokeRefreshToken(ctx context.Context, jti string) error {
	return a.users.RevokeRefreshToken(ctx, jti)
}
