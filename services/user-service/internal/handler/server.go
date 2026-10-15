package handler

import (
	"context"
	"log/slog"
	"net/http"

	"marketplace/user-service/internal/api/generated"
	"marketplace/user-service/internal/domain"
)

type UserService interface {
	Register(context.Context, string, string, domain.Role) (domain.User, error)
	Login(context.Context, string, string) (domain.TokenPair, error)
	Refresh(context.Context, string) (domain.TokenPair, error)
}

type Server struct {
	service UserService
	logger  *slog.Logger
}

var _ generated.StrictServerInterface = (*Server)(nil)

func (s *Server) GetHealth(_ context.Context, _ generated.GetHealthRequestObject) (generated.GetHealthResponseObject, error) {
	return generated.GetHealth200JSONResponse{
		Status:  generated.Ok,
		Service: "user-service",
	}, nil
}

func (s *Server) RegisterUser(ctx context.Context, request generated.RegisterUserRequestObject) (generated.RegisterUserResponseObject, error) {
	if request.Body == nil || request.Body.Password == nil {
		return nil, missingBody()
	}
	user, err := s.service.Register(
		ctx,
		string(request.Body.Email),
		*request.Body.Password,
		domain.Role(request.Body.Role),
	)
	if err != nil {
		return nil, err
	}
	return generated.RegisterUser201JSONResponse(userResponse(user)), nil
}

func (s *Server) LoginUser(ctx context.Context, request generated.LoginUserRequestObject) (generated.LoginUserResponseObject, error) {
	if request.Body == nil || request.Body.Password == nil {
		return nil, missingBody()
	}
	pair, err := s.service.Login(ctx, string(request.Body.Email), *request.Body.Password)
	if err != nil {
		return nil, err
	}
	return generated.LoginUser200JSONResponse(tokensResponse(pair)), nil
}

func (s *Server) RefreshAccessToken(ctx context.Context, request generated.RefreshAccessTokenRequestObject) (generated.RefreshAccessTokenResponseObject, error) {
	if request.Body == nil || request.Body.RefreshToken == nil {
		return nil, missingBody()
	}
	pair, err := s.service.Refresh(ctx, *request.Body.RefreshToken)
	if err != nil {
		return nil, err
	}
	return generated.RefreshAccessToken200JSONResponse(tokensResponse(pair)), nil
}

func missingBody() error {
	return newHTTPError(http.StatusBadRequest, generated.VALIDATIONERROR, "Request body is required", map[string]any{
		"body": "is required",
	})
}
