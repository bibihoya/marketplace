package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/auth"
)

const maxLoggedRequestBody = 1 << 20

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseRecorder) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseRecorder) Write(body []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := uuid.NewString()
		w.Header().Set("X-Request-Id", requestID)

		ctx := auth.WithRequestState(r.Context())
		r = r.WithContext(ctx)
		loggedBody, responseErr := requestBodyForLog(r)
		if responseErr != nil {
			writeHTTPError(w, responseErr)
			logger.Info("api request",
				"request_id", requestID,
				"method", r.Method,
				"endpoint", r.URL.Path,
				"status_code", responseErr.Status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"user_id", nil,
				"timestamp", startedAt.UTC().Format(time.RFC3339Nano),
			)
			return
		}

		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if recorder.statusCode == 0 {
			recorder.statusCode = http.StatusOK
		}

		var userID any
		if principal, ok := auth.PrincipalForLog(ctx); ok {
			userID = principal.UserID.String()
		}
		attributes := []any{
			"request_id", requestID,
			"method", r.Method,
			"endpoint", r.URL.Path,
			"status_code", recorder.statusCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"user_id", userID,
			"timestamp", startedAt.UTC().Format(time.RFC3339Nano),
		}
		if loggedBody != nil {
			attributes = append(attributes, "request_body", loggedBody)
		}
		logger.Info("api request", attributes...)
	})
}

func requestBodyForLog(r *http.Request) (any, *httpError) {
	if !isMutating(r.Method) || r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxLoggedRequestBody+1))
	if err != nil {
		return nil, newHTTPError(http.StatusBadRequest, "VALIDATION_ERROR", "Request body cannot be read", nil)
	}
	if len(body) > maxLoggedRequestBody {
		return nil, newHTTPError(http.StatusBadRequest, "VALIDATION_ERROR", "Request body is too large", nil)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}

	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return string(body), nil
	}
	maskSensitive(value)
	return value, nil
}

func maskSensitive(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lowerKey := strings.ToLower(key)
			if lowerKey == "password" || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "secret") {
				typed[key] = "***"
				continue
			}
			maskSensitive(child)
		}
	case []any:
		for _, child := range typed {
			maskSensitive(child)
		}
	}
}

func isMutating(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}
