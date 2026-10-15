package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"marketplace/user-service/internal/domain"
)

var (
	ErrNotFound      = errors.New("repository: not found")
	ErrAlreadyExists = errors.New("repository: already exists")
)

type Repository interface {
	Ping(context.Context) error
	CreateUser(context.Context, string, string, domain.Role) (domain.User, error)
	GetUserByEmail(context.Context, string) (domain.User, error)
	EnsureAdmin(context.Context, string, string) (domain.User, bool, error)
	CreateRefreshToken(context.Context, domain.RefreshToken) error
	RotateRefreshToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time, domain.RefreshToken) error
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}
