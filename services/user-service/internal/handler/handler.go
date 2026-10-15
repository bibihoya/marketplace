package handler

import (
	"context"
	"log/slog"
	"net/http"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"marketplace/user-service/internal/api/generated"
)

func NewHandler(service UserService, logger *slog.Logger) (http.Handler, error) {
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
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			writeHTTPError(w, newHTTPError(
				opts.StatusCode,
				generated.VALIDATIONERROR,
				"Request validation failed",
				map[string]any{"violations": []string{err.Error()}},
			))
		},
	})

	base := generated.Handler(strictHandler)
	return loggingMiddleware(logger, validator(base)), nil
}
