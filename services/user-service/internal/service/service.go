package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"marketplace/user-service/internal/domain"
	"marketplace/user-service/internal/repository"
)

type Service struct {
	repository        repository.Repository
	tokens            *TokenManager
	dummyPasswordHash []byte
}

func New(repository repository.Repository, tokens *TokenManager) (*Service, error) {
	dummyPasswordHash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("create dummy password hash: %w", err)
	}
	return &Service{
		repository:        repository,
		tokens:            tokens,
		dummyPasswordHash: dummyPasswordHash,
	}, nil
}

func (s *Service) Register(ctx context.Context, email, password string, role domain.Role) (domain.User, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return domain.User{}, err
	}
	if !role.Registerable() {
		return domain.User{}, validationError("Role is not available for registration", map[string]any{
			"role": "must be USER or SELLER",
		})
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, internalError(fmt.Errorf("hash password: %w", err))
	}

	user, err := s.repository.CreateUser(ctx, email, string(passwordHash), role)
	if repository.IsAlreadyExists(err) {
		return domain.User{}, newError(CodeEmailAlreadyExists, "Email is already registered", nil)
	}
	if err != nil {
		return domain.User{}, internalError(err)
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (domain.TokenPair, error) {
	email = normalizeEmail(email)
	user, err := s.repository.GetUserByEmail(ctx, email)
	if repository.IsNotFound(err) {
		_ = bcrypt.CompareHashAndPassword(s.dummyPasswordHash, []byte(password))
		return domain.TokenPair{}, invalidCredentials()
	}
	if err != nil {
		return domain.TokenPair{}, internalError(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return domain.TokenPair{}, invalidCredentials()
	}

	pair, refreshToken, err := s.tokens.NewPair(user)
	if err != nil {
		return domain.TokenPair{}, internalError(err)
	}
	if err := s.repository.CreateRefreshToken(ctx, refreshToken); err != nil {
		return domain.TokenPair{}, internalError(err)
	}
	return pair, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (domain.TokenPair, error) {
	parsed, err := s.tokens.ParseRefresh(rawRefreshToken)
	if err != nil {
		return domain.TokenPair{}, invalidRefreshToken()
	}

	user := domain.User{ID: parsed.UserID, Role: parsed.Role}
	pair, replacement, err := s.tokens.NewPair(user)
	if err != nil {
		return domain.TokenPair{}, internalError(err)
	}
	if err := s.repository.RotateRefreshToken(
		ctx,
		parsed.ID,
		parsed.UserID,
		parsed.TokenHash,
		replacement.CreatedAt,
		replacement,
	); repository.IsNotFound(err) {
		return domain.TokenPair{}, invalidRefreshToken()
	} else if err != nil {
		return domain.TokenPair{}, internalError(err)
	}
	return pair, nil
}

func (s *Service) EnsureAdmin(ctx context.Context, email, password string) (domain.User, bool, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return domain.User{}, false, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, false, internalError(fmt.Errorf("hash admin password: %w", err))
	}
	user, created, err := s.repository.EnsureAdmin(ctx, email, string(passwordHash))
	if err != nil {
		return domain.User{}, false, internalError(err)
	}
	if user.Role != domain.RoleAdmin {
		return domain.User{}, false, validationError("Bootstrap admin email belongs to a non-admin user", map[string]any{
			"email": email,
		})
	}
	return user, created, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password string) error {
	if email == "" || len(email) > 254 {
		return validationError("Email is invalid", map[string]any{"email": "must contain at most 254 characters"})
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return validationError("Email is invalid", map[string]any{"email": "must be a valid email address"})
	}
	if len(password) < 8 || len(password) > 72 {
		return validationError("Password is invalid", map[string]any{"password": "must contain between 8 and 72 characters"})
	}
	return nil
}

func invalidCredentials() error {
	return newError(CodeInvalidCredentials, "Email or password is invalid", nil)
}

func invalidRefreshToken() error {
	return newError(CodeRefreshTokenInvalid, "Refresh token is invalid", nil)
}
