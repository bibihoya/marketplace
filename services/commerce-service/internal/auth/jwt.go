package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"marketplace/commerce-service/internal/domain"
)

type contextKey struct{}
type requestStateKey struct{}

type requestState struct {
	principal *domain.Principal
}

type Claims struct {
	Role      domain.Role `json:"role"`
	TokenType string      `json:"token_type"`
	jwt.RegisteredClaims
}

type Verifier struct {
	secret []byte
	issuer string
}

type ErrorWriter func(http.ResponseWriter, *Error)

func NewVerifier(secret, issuer string) *Verifier {
	return &Verifier{secret: []byte(secret), issuer: issuer}
}

func (v *Verifier) Middleware(writeError ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			principal, authErr := v.Authenticate(r.Header.Get("Authorization"))
			if authErr != nil {
				writeError(w, authErr)
				return
			}

			setPrincipalForLog(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func (v *Verifier) Authenticate(authorization string) (domain.Principal, *Error) {
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return domain.Principal{}, tokenError(CodeTokenInvalid, "Bearer token is missing or invalid")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		parts[1],
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected JWT signing method")
			}
			return v.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(v.issuer),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return domain.Principal{}, tokenError(CodeTokenExpired, "Access token has expired")
		}
		return domain.Principal{}, tokenError(CodeTokenInvalid, "Access token is invalid")
	}
	if !token.Valid || claims.TokenType != "access" || !claims.Role.Valid() {
		return domain.Principal{}, tokenError(CodeTokenInvalid, "Access token is invalid")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.Principal{}, tokenError(CodeTokenInvalid, "Access token subject is invalid")
	}

	return domain.Principal{UserID: userID, Role: claims.Role}, nil
}

func WithPrincipal(ctx context.Context, principal domain.Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (domain.Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(domain.Principal)
	return principal, ok
}

func WithRequestState(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestStateKey{}, &requestState{})
}

func PrincipalForLog(ctx context.Context) (domain.Principal, bool) {
	state, ok := ctx.Value(requestStateKey{}).(*requestState)
	if !ok || state.principal == nil {
		return domain.Principal{}, false
	}

	return *state.principal, true
}

func setPrincipalForLog(ctx context.Context, principal domain.Principal) {
	if state, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		state.principal = &principal
	}
}

func tokenError(code ErrorCode, message string) *Error {
	return NewError(code, message)
}
