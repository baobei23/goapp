package users

import (
	"context"
	"fmt"
	"strings"

	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgstore struct {
	pqdriver  *pgxpool.Pool
	tableName string
}

func (ps *pgstore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, full_name, email, password
		FROM %s
		WHERE email = $1`,
		ps.tableName,
	)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := new(User)

	row := ps.pqdriver.QueryRow(ctx, query, email)
	err := row.Scan(&user.ID, &user.FullName, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserEmailNotFound, email)
		}
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}

	return user, nil
}

func (ps *pgstore) SaveUser(ctx context.Context, user *User) (string, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, full_name, email, password)
		VALUES (gen_random_uuid(), $1, $2, $3) RETURNING id`,
		ps.tableName,
	)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := ps.pqdriver.QueryRow(ctx, query,
		user.FullName,
		user.Email,
		user.Password,
	).Scan(&user.ID)

	if err != nil {
		if strings.Contains(err.Error(), "violates unique constraint \"users_email_key\"") {
			return "", fmt.Errorf("%w: %s", ErrUserEmailAlreadyExists, user.Email)
		}
		return "", fmt.Errorf("failed storing user info: %w", err)
	}

	return user.ID, nil
}

func (ps *pgstore) BulkSaveUser(ctx context.Context, users []User) error {
	rows := make([][]any, 0, len(users))

	for _, user := range users {
		rows = append(rows, []any{
			user.ID,
			user.FullName,
			user.Email,
			user.Password,
		})
	}

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	inserted, err := ps.pqdriver.CopyFrom(
		ctx,
		pgx.Identifier{ps.tableName},
		[]string{"id", "full_name", "email", "password"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed inserting users: %w", err)
	}

	ulen := int64(len(users))
	if inserted != ulen {
		return fmt.Errorf(
			"failed inserting %d out of %d users",
			ulen-inserted,
			ulen,
		)
	}

	return nil
}

func (ps *pgstore) UpdatePassword(ctx context.Context, email string, newPassword []byte) error {
	query := fmt.Sprintf(`UPDATE %s SET password = $1 WHERE email = $2`, ps.tableName)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := ps.pqdriver.Exec(ctx, query, newPassword, email)
	if err != nil {
		return fmt.Errorf("failed updating password: %w", err)
	}

	return nil
}

func NewPostgresStore(pqdriver *pgxpool.Pool, tablename string) *pgstore {
	return &pgstore{
		pqdriver:  pqdriver,
		tableName: tablename,
	}
}
