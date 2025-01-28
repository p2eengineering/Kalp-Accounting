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

func NewInternalError(err error, message string, statusCode int) *CustomError {
	return &CustomError{
		statusCode:  statusCode,
		message:     message,
		internalErr: err,
	}
}

var (
	ErrFailedToGetPublicAddress = New("failed to get public address", http.StatusInternalServerError)
	ErrOnlyFoundationHasAccess  = New("only kalp foundation has access to perform this action", http.StatusUnauthorized)
	ErrMintingTokens            = New("error while minting tokens", http.StatusInternalServerError)
)

func ErrEmptyAddress() *CustomError {
	return New("address cannot be empty", http.StatusBadRequest)
}

func ErrRegexValidationFailed(field string, err error) *CustomError {
	return New(fmt.Sprintf("failed to validate %s due to regex error: %v", field, err), http.StatusInternalServerError)
}

func ErrAddressValidationFailed(address string, reason string) *CustomError {
	return New(fmt.Sprintf("address validation failed for address: %s, reason: %s", address, reason), http.StatusBadRequest)
}

func ErrFailedToEmitEvent(event string) *CustomError {
	return New(fmt.Sprintf("failed to emit event: %s", event), http.StatusInternalServerError)
}

func ErrInvalidAmount(amount string) *CustomError {
	return New(fmt.Sprintf("invalid amount passed: %s", amount), http.StatusBadRequest)
}

func ErrInvalidAddress(address string) *CustomError {
	return New(fmt.Sprintf("address: %s is not valid", address), http.StatusBadRequest)
}

func ErrInvalidContractAddress(address string) *CustomError {
	return New(fmt.Sprintf("contract address: %s is not valid", address), http.StatusBadRequest)
}

func ErrFailedToPutState(e error) *CustomError {
	return NewInternalError(e, "failed to put data", http.StatusInternalServerError)
}

func ErrFailedToGetState(e error) *CustomError {
	return NewInternalError(e, "failed to get data", http.StatusInternalServerError)
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
	return New(fmt.Sprintf("FailedToGetKey for %s", key), http.StatusInternalServerError)
}

func ErrInsufficientAllowance() *CustomError {
	return New(fmt.Sprintf("The account does not have sufficient allowance"), http.StatusInternalServerError)
}
