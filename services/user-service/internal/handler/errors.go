package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"marketplace/user-service/internal/api/generated"
	"marketplace/user-service/internal/service"
)

type httpError struct {
	Status  int
	Code    generated.ErrorCode
	Message string
	Details map[string]any
}

func (e *httpError) Error() string {
	return e.Message
}

func newHTTPError(status int, code generated.ErrorCode, message string, details map[string]any) *httpError {
	return &httpError{Status: status, Code: code, Message: message, Details: details}
}

func (s *Server) writeResponseError(w http.ResponseWriter, _ *http.Request, err error) {
	responseErr := mapHTTPError(err)
	if responseErr.Status >= http.StatusInternalServerError {
		s.logger.Error("request processing failed", "error", err.Error())
	}
	writeHTTPError(w, responseErr)
}

func mapHTTPError(err error) *httpError {
	var handlerErr *httpError
	if errors.As(err, &handlerErr) {
		return handlerErr
	}

	var serviceErr *service.Error
	if !errors.As(err, &serviceErr) {
		return internalHTTPError()
	}

	switch serviceErr.Code {
	case service.CodeValidation:
		return newHTTPError(http.StatusBadRequest, generated.VALIDATIONERROR, serviceErr.Message, serviceErr.Details)
	case service.CodeInvalidCredentials:
		return newHTTPError(http.StatusUnauthorized, generated.INVALIDCREDENTIALS, serviceErr.Message, serviceErr.Details)
	case service.CodeRefreshTokenInvalid:
		return newHTTPError(http.StatusUnauthorized, generated.REFRESHTOKENINVALID, serviceErr.Message, serviceErr.Details)
	case service.CodeEmailAlreadyExists:
		return newHTTPError(http.StatusConflict, generated.EMAILALREADYEXISTS, serviceErr.Message, serviceErr.Details)
	case service.CodeInternal:
		return internalHTTPError()
	default:
		return internalHTTPError()
	}
}

func internalHTTPError() *httpError {
	return newHTTPError(http.StatusInternalServerError, generated.INTERNALERROR, "Internal server error", nil)
}

func writeHTTPError(w http.ResponseWriter, err *httpError) {
	response := generated.ErrorResponse{ErrorCode: err.Code, Message: err.Message}
	if err.Details != nil {
		response.Details = &err.Details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(response)
}
