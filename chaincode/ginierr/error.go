// TODO: use GiniError instead of error
package ginierr

import (
	"fmt"
	"net/http"
)

// TODO: use GiniError instead of error
type GiniError struct {
	StatusCode  int
	Message     string
	internalErr error
}

func (e *GiniError) Error() string {
	return fmt.Sprintf("%s, status code:%d", e.Message, e.StatusCode)
}

func (e *GiniError) FullError() string {
	return fmt.Sprintf("%s, status code:%d, internal err: %v", e.Message, e.StatusCode, e.internalErr)
}

func NewGiniError(statusCode int, message string) *GiniError {
	return &GiniError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func NewGiniErrorWithError(err error, statusCode int, message string) *GiniError {
	return &GiniError{
		StatusCode:  statusCode,
		Message:     message,
		internalErr: err,
	}
}

var (
	// ErrFailedToGetClientID = NewGiniError(http.StatusInternalServerError, "failed to get client id")
	ErrFailedToGetClientID     = fmt.Errorf("error with status code %v, failed to get client id", http.StatusInternalServerError)
	ErrOnlyFoundationHasAccess = fmt.Errorf("error with status code %v, only kalp foundation has access to perform this action", http.StatusUnauthorized)
	ErrFailedToGetName         = fmt.Errorf("failed to get state name")
	ErrFailedToGetSymbol       = fmt.Errorf("failed to get state symbol")
	ErrInitializingRoles       = fmt.Errorf("error in initializing roles")
	ErrMinitingTokens          = fmt.Errorf("error with status code %v,error in minting", http.StatusInternalServerError)
	ErrAlreadyAllowed          = fmt.Errorf("error with status code %v, address already allowed", http.StatusBadRequest)
	ErrAlreadyDenied           = fmt.Errorf("error with status code %v, address already denied", http.StatusBadRequest)
	ErrFailedToEmitEvent       = fmt.Errorf("failed to emit event")
	ErrIncorrectAddress        = fmt.Errorf("incorrect address, please enter a valid address")
)
