package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"marketplace/user-service/internal/domain"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type tokenClaims struct {
	Role      domain.Role `json:"role"`
	TokenType string      `json:"token_type"`
	jwt.RegisteredClaims
}

type parsedRefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Role      domain.Role
	TokenHash []byte
}

type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenManager(secret, issuer string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

func (m *TokenManager) NewPair(user domain.User) (domain.TokenPair, domain.RefreshToken, error) {
	now := m.now().UTC()
	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)
	refreshID := uuid.New()

	accessToken, err := m.sign(tokenClaims{
		Role:      user.Role,
		TokenType: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
		},
	})
	if err != nil {
		return domain.TokenPair{}, domain.RefreshToken{}, err
	}

	refreshToken, err := m.sign(tokenClaims{
		Role:      user.Role,
		TokenType: tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshID.String(),
			Subject:   user.ID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
		},
	})
	if err != nil {
		return domain.TokenPair{}, domain.RefreshToken{}, err
	}

	hash := sha256.Sum256([]byte(refreshToken))
	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int32(m.accessTTL.Seconds()),
	}, domain.RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: hash[:],
		ExpiresAt: refreshExpiresAt,
		CreatedAt: now,
	}, nil
}

func (m *TokenManager) ParseRefresh(raw string) (parsedRefreshToken, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected JWT signing method")
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil || !token.Valid || claims.TokenType != tokenTypeRefresh || !claims.Role.Valid() {
		return parsedRefreshToken{}, errors.New("refresh token is invalid")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return parsedRefreshToken{}, errors.New("refresh token subject is invalid")
	}
	refreshID, err := uuid.Parse(claims.ID)
	if err != nil {
		return parsedRefreshToken{}, errors.New("refresh token ID is invalid")
	}

	hash := sha256.Sum256([]byte(raw))
	return parsedRefreshToken{
		ID:        refreshID,
		UserID:    userID,
		Role:      claims.Role,
		TokenHash: hash[:],
	}, nil
}

func (m *TokenManager) sign(claims tokenClaims) (string, error) {
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}
