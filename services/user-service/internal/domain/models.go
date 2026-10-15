package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser   Role = "USER"
	RoleSeller Role = "SELLER"
	RoleAdmin  Role = "ADMIN"
)

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleSeller || r == RoleAdmin
}

func (r Role) Registerable() bool {
	return r == RoleUser || r == RoleSeller
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32
}
