package users

import (
	"context"
	"fmt"
	"strings"
	"time"

	"errors"

	"github.com/baobei23/goapp/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgstore struct {
	q *db.Queries
}

func (ps *pgstore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	row, err := ps.q.GetUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserEmailNotFound, email)
		}
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}

	return &User{
		ID:       row.ID,
		FullName: row.FullName.String,
		Email:    row.Email,
		Password: row.Password,
	}, nil
}

func (ps *pgstore) GetUserByID(ctx context.Context, id string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	row, err := ps.q.GetUserByID(ctx, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, id)
		}
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}

	return &User{
		ID:       row.ID,
		FullName: row.FullName.String,
		Email:    row.Email,
		Password: row.Password,
	}, nil
}

func (ps *pgstore) SaveUser(ctx context.Context, user *User) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	id, err := ps.q.CreateUser(ctx, db.CreateUserParams{
		FullName: pgtype.Text{String: user.FullName, Valid: true},
		Email:    user.Email,
		Password: user.Password,
	})

	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return "", fmt.Errorf("%w: %s", ErrUserEmailAlreadyExists, user.Email)
		}
		return "", fmt.Errorf("failed storing user info: %w", err)
	}

	return id, nil

}

func (ps *pgstore) BulkSaveUser(ctx context.Context, users []User) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	params := make([]db.BulkCreateUsersParams, 0, len(users))
	for _, user := range users {
		params = append(params, db.BulkCreateUsersParams{
			ID:       user.ID,
			FullName: pgtype.Text{String: user.FullName, Valid: true},
			Email:    user.Email,
			Password: user.Password,
		})
	}

	inserted, err := ps.q.BulkCreateUsers(ctx, params)
	if err != nil {
		return fmt.Errorf("failed inserting users: %w", err)
	}

	if inserted != int64(len(users)) {
		return fmt.Errorf("failed inserting %d out of %d users", len(users)-int(inserted), len(users))
	}

	return nil
}

func (ps *pgstore) UpdatePassword(ctx context.Context, id string, newPassword []byte) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := ps.q.UpdatePassword(ctx, db.UpdatePasswordParams{
		Password: newPassword,
		ID:       id,
	})
	if err != nil {
		return fmt.Errorf("failed updating password: %w", err)
	}

	return nil
}

func (ps *pgstore) SaveRefreshToken(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := ps.q.SaveRefreshToken(ctx, db.SaveRefreshTokenParams{
		Jti:       jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return fmt.Errorf("failed saving refresh token: %w", err)
	}
	return nil
}

func (ps *pgstore) CheckRefreshToken(ctx context.Context, jti string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	exists, err := ps.q.CheckRefreshToken(ctx, jti)
	if err != nil {
		return false, fmt.Errorf("failed checking refresh token: %w", err)
	}
	return exists, nil
}

func (ps *pgstore) RevokeRefreshToken(ctx context.Context, jti string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := ps.q.RevokeRefreshToken(ctx, jti)
	if err != nil {
		return fmt.Errorf("failed revoking refresh token: %w", err)
	}
	return nil
}

func NewPostgresStore(pqdriver *pgxpool.Pool) *pgstore {
	return &pgstore{
		q: db.New(pqdriver),
	}
}
