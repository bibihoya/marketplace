package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"marketplace/commerce-service/internal/api/generated"
	"marketplace/commerce-service/internal/auth"
)

// NewHandler connects the generated routes to authentication, OpenAPI validation,
// request logging and the handwritten application handlers.
func NewHandler(service CommerceService, verifier *auth.Verifier, logger *slog.Logger) (http.Handler, error) {
	server := &Server{service: service, logger: logger}
	strictHandler := generated.NewStrictHandlerWithOptions(
		server,
		nil,
		generated.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				writeHTTPError(w, newHTTPError(
					http.StatusBadRequest,
					generated.VALIDATIONERROR,
					"Request validation failed",
					map[string]any{"violations": []string{err.Error()}},
				))
			},
			ResponseErrorHandlerFunc: server.writeResponseError,
		},
	)

	spec, err := generated.GetSwagger()
	if err != nil {
		return nil, err
	}
	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		DoNotValidateServers: true,
		Options: openapi3filter.Options{
			MultiError: true,
			AuthenticationFunc: func(ctx context.Context, _ *openapi3filter.AuthenticationInput) error {
				if _, ok := auth.PrincipalFromContext(ctx); !ok {
					return errors.New("authenticated principal is missing")
				}
				return nil
			},
		},
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			status := opts.StatusCode
			code := generated.VALIDATIONERROR
			message := "Request validation failed"
			if status == http.StatusUnauthorized {
				code = generated.TOKENINVALID
				message = "Access token is invalid"
			}
			writeHTTPError(w, newHTTPError(status, code, message, map[string]any{
				"violations": []string{err.Error()},
			}))
		},
	})

	base := generated.Handler(strictHandler)
	handler := validator(base)
	handler = verifier.Middleware(writeAuthError)(handler)
	handler = loggingMiddleware(logger, handler)
	return handler, nil
}
