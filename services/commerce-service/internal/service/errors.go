package service

import "errors"

type ErrorCode string

const (
	CodeAccessDenied            ErrorCode = "ACCESS_DENIED"
	CodeInsufficientStock       ErrorCode = "INSUFFICIENT_STOCK"
	CodeInternal                ErrorCode = "INTERNAL_ERROR"
	CodeInvalidStateTransition  ErrorCode = "INVALID_STATE_TRANSITION"
	CodeOrderHasActive          ErrorCode = "ORDER_HAS_ACTIVE"
	CodeOrderLimitExceeded      ErrorCode = "ORDER_LIMIT_EXCEEDED"
	CodeOrderNotFound           ErrorCode = "ORDER_NOT_FOUND"
	CodeOrderOwnershipViolation ErrorCode = "ORDER_OWNERSHIP_VIOLATION"
	CodeProductInactive         ErrorCode = "PRODUCT_INACTIVE"
	CodeProductNotFound         ErrorCode = "PRODUCT_NOT_FOUND"
	CodePromoAlreadyExists      ErrorCode = "PROMO_CODE_ALREADY_EXISTS"
	CodePromoInvalid            ErrorCode = "PROMO_CODE_INVALID"
	CodePromoMinAmount          ErrorCode = "PROMO_CODE_MIN_AMOUNT"
	CodeValidation              ErrorCode = "VALIDATION_ERROR"
)

// Error describes a failure in a commerce use case. It intentionally does not
// contain an HTTP status: translating business failures to transport semantics
// is the handler's responsibility.
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

	return &Error{
		Code:    CodeInternal,
		Message: "Internal server error",
		Cause:   err,
	}
}

func productNotFound() error {
	return newError(CodeProductNotFound, "Product not found", nil)
}

func orderNotFound() error {
	return newError(CodeOrderNotFound, "Order not found", nil)
}

func validationError(message string, details map[string]any) error {
	return newError(CodeValidation, message, details)
}
