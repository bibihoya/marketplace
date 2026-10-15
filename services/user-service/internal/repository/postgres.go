package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"marketplace/user-service/internal/domain"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (r *Postgres) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Postgres) CreateUser(ctx context.Context, email, passwordHash string, role domain.Role) (domain.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, role::text, created_at, updated_at
	`, email, passwordHash, role))
	if isUniqueViolation(err) {
		return domain.User{}, ErrAlreadyExists
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *Postgres) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role::text, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *Postgres) EnsureAdmin(ctx context.Context, email, passwordHash string) (domain.User, bool, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'ADMIN')
		ON CONFLICT (email) DO NOTHING
		RETURNING id, email, password_hash, role::text, created_at, updated_at
	`, email, passwordHash))
	if err == nil {
		return user, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, fmt.Errorf("create bootstrap admin: %w", err)
	}

	user, err = r.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("get bootstrap admin: %w", err)
	}
	return user, false, nil
}

func (r *Postgres) CreateRefreshToken(ctx context.Context, token domain.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *Postgres) RotateRefreshToken(
	ctx context.Context,
	oldID uuid.UUID,
	userID uuid.UUID,
	oldHash []byte,
	now time.Time,
	replacement domain.RefreshToken,
) error {
	var insertedID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		WITH consumed AS (
			DELETE FROM refresh_tokens
			WHERE id = $1
			  AND user_id = $2
			  AND token_hash = $3
			  AND expires_at > $4
			RETURNING user_id
		)
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		SELECT $5, user_id, $6, $7, $8
		FROM consumed
		RETURNING id
	`, oldID, userID, oldHash, now, replacement.ID, replacement.TokenHash, replacement.ExpiresAt, replacement.CreatedAt).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("rotate refresh token: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (domain.User, error) {
	var user domain.User
	var role string
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.Role = domain.Role(role)
	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
