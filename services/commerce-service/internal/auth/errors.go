package auth

type ErrorCode string

const (
	CodeTokenExpired ErrorCode = "TOKEN_EXPIRED"
	CodeTokenInvalid ErrorCode = "TOKEN_INVALID"
)

// Error describes an authentication failure without tying it to an HTTP
// response. The handler decides how the failure is exposed by the API.
type Error struct {
	Code    ErrorCode
	Message string
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

func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}
