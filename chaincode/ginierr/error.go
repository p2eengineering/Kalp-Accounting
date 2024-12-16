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

func NewWithInternalError(err error, message string, statusCode int) *CustomError {
	return &CustomError{
		StatusCode:  statusCode,
		Message:     message,
		internalErr: err,
	}
}

var (
	ErrFailedToGetClientID     = New("failed to get public address", http.StatusInternalServerError)
	ErrOnlyFoundationHasAccess = New("only kalp foundation has access to perform this action", http.StatusUnauthorized)
	ErrFailedToGetName         = New("failed to get name", http.StatusInternalServerError)
	ErrFailedToGetSymbol       = New("failed to get symbol", http.StatusInternalServerError)
	ErrInitializingRoles       = New("error while initializing roles", http.StatusInternalServerError)
	ErrMinitingTokens          = New("error while minting tokens", http.StatusInternalServerError)
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
	return NewWithInternalError(e, "failed to delete data from world state", http.StatusInternalServerError)
}

func ErrFailedToPutState(e error) *CustomError {
	return NewWithInternalError(e, "failed to put data in world state", http.StatusInternalServerError)
}

func ErrFailedToGetState(e error) *CustomError {
	return NewWithInternalError(e, "failed to get data from world state", http.StatusInternalServerError)
}

func ErrCreatingCompositeKey(e error) *CustomError {
	return NewWithInternalError(e, "failed to create the composite key", http.StatusInternalServerError)
}

func ErrFailedToSetEvent(e error, event string) *CustomError {
	return NewWithInternalError(e, "failed to set event: "+event, http.StatusInternalServerError)
}

func ErrConvertingStringToBigInt(number string) *CustomError {
	return New(fmt.Sprintf("failed to covert number %s to big int", number), http.StatusInternalServerError)
}

func AlreadyDenied(address string) *CustomError {
	return New(fmt.Sprintf("AlreadyDenied : address : %s ", address), http.StatusConflict)
}

func NotDenied(address string) *CustomError {
	return New(fmt.Sprintf("NotDenied : address : %s ", address), http.StatusConflict)
}

func DeniedAddress(address string) *CustomError {
	return New(fmt.Sprintf("DeniedAddress : address : %s ", address), http.StatusForbidden)
}
