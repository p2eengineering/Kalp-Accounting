package ginierr

import (
	"fmt"
	"net/http"
)

type CustomError struct {
	statusCode  int
	message     string
	internalErr error
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("%s, status code:%d", e.message, e.statusCode)
}

func (e *CustomError) FullError() string {
	return fmt.Sprintf("%s, status code:%d, internal err: %v", e.message, e.statusCode, e.internalErr)
}

func New(message string, statusCode int) *CustomError {
	return &CustomError{
		statusCode: statusCode,
		message:    message,
	}
}

func NewWithInternalError(err error, message string, statusCode int) *CustomError {
	return &CustomError{
		statusCode:  statusCode,
		message:     message,
		internalErr: err,
	}
}

var (
	ErrFailedToGetClientID     = New("failed to get public address", http.StatusInternalServerError)
	ErrOnlyFoundationHasAccess = New("only kalp foundation has access to perform this action", http.StatusUnauthorized)
	ErrMinitingTokens          = New("error while minting tokens", http.StatusInternalServerError)
)

func ErrFailedToEmitEvent(event string) *CustomError {
	return New(fmt.Sprintf("failed to emit event: %s", event), http.StatusInternalServerError)
}

func ErrInvalidAmount(amount string) *CustomError {
	return New(fmt.Sprintf("invalid amount passed: %s", amount), http.StatusBadRequest)
}

func ErrInvalidAddress(address string) *CustomError {
	return New(fmt.Sprintf("address: %s is not valid", address), http.StatusBadRequest)
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

func ErrConvertingAmountToBigInt(number string) *CustomError {
	return New(fmt.Sprintf("failed to covert amount %s to big int", number), http.StatusInternalServerError)
}

func ErrAlreadyDenied(address string) *CustomError {
	return New(fmt.Sprintf("AlreadyDenied : address : %s ", address), http.StatusConflict)
}

func ErrNotDenied(address string) *CustomError {
	return New(fmt.Sprintf("NotDenied : address : %s ", address), http.StatusConflict)
}

func ErrDeniedAddress(address string) *CustomError {
	return New(fmt.Sprintf("DeniedAddress : address : %s ", address), http.StatusForbidden)
}

func ErrInvalidUserAddress(address string) *CustomError {
	return New(fmt.Sprintf("Invalid user address : %s ", address), http.StatusBadRequest)
}

func ErrFailedToGetKey(key string) *CustomError {
	return New(fmt.Sprintf("FailedToGetKey : %s", key), http.StatusInternalServerError)
}
