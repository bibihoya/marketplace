package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"marketplace/commerce-service/internal/api/generated"
	"marketplace/commerce-service/internal/auth"
	"marketplace/commerce-service/internal/service"
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
	return &httpError{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
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

	var authErr *auth.Error
	if errors.As(err, &authErr) {
		return newHTTPError(
			http.StatusUnauthorized,
			generated.ErrorCode(authErr.Code),
			authErr.Message,
			nil,
		)
	}

	var serviceErr *service.Error
	if errors.As(err, &serviceErr) {
		status, ok := serviceErrorStatus(serviceErr.Code)
		if !ok {
			return internalHTTPError()
		}

		return newHTTPError(
			status,
			generated.ErrorCode(serviceErr.Code),
			serviceErr.Message,
			serviceErr.Details,
		)
	}

	return internalHTTPError()
}

func serviceErrorStatus(code service.ErrorCode) (int, bool) {
	switch code {
	case service.CodeValidation:
		return http.StatusBadRequest, true
	case service.CodeAccessDenied, service.CodeOrderOwnershipViolation:
		return http.StatusForbidden, true
	case service.CodeProductNotFound, service.CodeOrderNotFound:
		return http.StatusNotFound, true
	case service.CodeProductInactive,
		service.CodeInsufficientStock,
		service.CodeOrderHasActive,
		service.CodeInvalidStateTransition,
		service.CodePromoAlreadyExists:
		return http.StatusConflict, true
	case service.CodePromoInvalid, service.CodePromoMinAmount:
		return http.StatusUnprocessableEntity, true
	case service.CodeOrderLimitExceeded:
		return http.StatusTooManyRequests, true
	case service.CodeInternal:
		return http.StatusInternalServerError, true
	default:
		return 0, false
	}
}

func internalHTTPError() *httpError {
	return newHTTPError(
		http.StatusInternalServerError,
		generated.INTERNALERROR,
		"Internal server error",
		nil,
	)
}

func writeAuthError(w http.ResponseWriter, err *auth.Error) {
	writeHTTPError(w, mapHTTPError(err))
}

func writeHTTPError(w http.ResponseWriter, err *httpError) {
	response := generated.ErrorResponse{
		ErrorCode: err.Code,
		Message:   err.Message,
	}
	if err.Details != nil {
		response.Details = &err.Details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(response)
}
