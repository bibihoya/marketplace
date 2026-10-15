package service

import "errors"

type ErrorCode string

const (
	CodeEmailAlreadyExists  ErrorCode = "EMAIL_ALREADY_EXISTS"
	CodeInternal            ErrorCode = "INTERNAL_ERROR"
	CodeInvalidCredentials  ErrorCode = "INVALID_CREDENTIALS"
	CodeRefreshTokenInvalid ErrorCode = "REFRESH_TOKEN_INVALID"
	CodeValidation          ErrorCode = "VALIDATION_ERROR"
)

type Error struct {
	Code    ErrorCode
	Message string
	Details map[string]any
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func newError(code ErrorCode, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Details: details}
}

func internalError(err error) error {
	var serviceErr *Error
	if errors.As(err, &serviceErr) {
		return err
	}
	return &Error{Code: CodeInternal, Message: "Internal server error", Cause: err}
}

func validationError(message string, details map[string]any) error {
	return newError(CodeValidation, message, details)
}
