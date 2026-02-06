package api

import (
	"context"

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

// ReadUserByEmail is the API to read an existing user by their email
func (a *API) ReadUserByEmail(ctx context.Context, email string) (*users.User, error) {
	u, err := a.users.ReadByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (a *API) AsyncRegisters(ctx context.Context, users []users.User) error {
	return a.users.AsyncRegisters(ctx, users)
}
