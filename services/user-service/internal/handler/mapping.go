package handler

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"marketplace/user-service/internal/api/generated"
	"marketplace/user-service/internal/domain"
)

func userResponse(user domain.User) generated.UserResponse {
	id := openapi_types.UUID(user.ID)
	return generated.UserResponse{
		Id:        &id,
		Email:     openapi_types.Email(user.Email),
		Role:      generated.UserRole(user.Role),
		CreatedAt: &user.CreatedAt,
		UpdatedAt: &user.UpdatedAt,
	}
}

func tokensResponse(pair domain.TokenPair) generated.AuthTokensResponse {
	tokenType := generated.Bearer
	return generated.AuthTokensResponse{
		AccessToken:  &pair.AccessToken,
		RefreshToken: &pair.RefreshToken,
		TokenType:    &tokenType,
		ExpiresIn:    &pair.ExpiresIn,
	}
}
