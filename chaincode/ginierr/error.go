package ginierr

import (
	"fmt"
	"net/http"
)

type CustomError struct {
	StatusCode  int
	Message     string
	internalErr error
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("%s, status code:%d", e.Message, e.StatusCode)
}

func (e *CustomError) FullError() string {
	return fmt.Sprintf("%s, status code:%d, internal err: %v", e.Message, e.StatusCode, e.internalErr)
}

func New(message string, statusCode int) *CustomError {
	return &CustomError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func NewWithError(err error, message string, statusCode int) *CustomError {
	return &CustomError{
		StatusCode:  statusCode,
		Message:     message,
		internalErr: err,
	}
}

var (
	// ErrFailedToGetClientID = NewGiniError(http.StatusInternalServerError, "failed to get client id")
	ErrFailedToGetClientID     = New("failed to get client id", http.StatusInternalServerError)
	ErrOnlyFoundationHasAccess = New("only kalp foundation has access to perform this action", http.StatusUnauthorized)
	ErrFailedToGetName         = New("failed to get state name", http.StatusInternalServerError)
	ErrFailedToGetSymbol       = New("failed to get state symbol", http.StatusInternalServerError)
	ErrInitializingRoles       = New("error while initializing roles", http.StatusInternalServerError)
	ErrMinitingTokens          = New("error while minting", http.StatusInternalServerError)
	ErrAlreadyAllowed          = New("address already allowed", http.StatusBadRequest)
	ErrAlreadyDenied           = New("address already denied", http.StatusBadRequest)
	ErrFailedToEmitEvent       = New("failed to emit event", http.StatusInternalServerError)
	ErrInvalidAddress          = New("invalid address", http.StatusBadRequest)
)

func ErrInvalidAmount(amount string) *CustomError {
	return New(fmt.Sprintf("invalid amount passed: %s", amount), http.StatusBadRequest)
}

func ErrIncorrectAddress(user string) *CustomError {
	return New(fmt.Sprintf("%s address is not valid", user), http.StatusBadRequest)
}

func ErrFailedToDeleteState(e error) *CustomError {
	return NewWithError(e, "failed to delete data from world state while removing UTXO", http.StatusInternalServerError)
}

func ErrFailedToPutState(e error) *CustomError {
	return NewWithError(e, "failed to put data in world state while removing UTXO", http.StatusInternalServerError)
}
