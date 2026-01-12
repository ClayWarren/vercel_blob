package vercelblob

import (
	"fmt"
)

// Error will be the type of all errors raised by this crate.
type Error struct {
	Msg  string
	Code string
}

func (e Error) Error() string {
	return e.Msg
}

// All errors raised by this crate will be instances of Error
var (
	ErrNotAuthenticated = &Error{
		Msg:  "No authentication token. Expected environment variable BLOB_READ_WRITE_TOKEN to contain a token",
		Code: "not_authenticated",
	}

	ErrBadRequest = func(msg string) Error {
		return Error{
			Msg:  fmt.Sprintf("Invalid request: %s", msg),
			Code: "bad_request",
		}
	}

	ErrForbidden = &Error{
		Msg:  "Access denied, please provide a valid token for this resource",
		Code: "forbidden",
	}

	ErrStoreNotFound = &Error{
		Msg:  "The requested store does not exist",
		Code: "store_not_found",
	}

	ErrStoreSuspended = &Error{
		Msg:  "The requested store has been suspended",
		Code: "store_suspended",
	}

	ErrBlobNotFound = &Error{
		Msg:  "The requested blob does not exist",
		Code: "not_found",
	}
)

// NewUnknownError creates a new Error for an unknown error.
func NewUnknownError(statusCode int, message string) Error {
	return Error{
		Msg:  fmt.Sprintf("Unknown error, please visit https://vercel.com/help (%d): %s", statusCode, message),
		Code: "unknown_error",
	}
}

// NewInvalidInputError creates a new Error for an invalid input field.
func NewInvalidInputError(field string) Error {
	return Error{
		Msg:  fmt.Sprintf("%s is required", field),
		Code: "invalid_input",
	}
}
