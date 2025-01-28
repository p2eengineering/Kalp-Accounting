package helper

import (
	"encoding/base64"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/logger"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func ConvertToBigInt(value interface{}) (*big.Int, error) {
	switch v := value.(type) {
	case int:
		return big.NewInt(int64(v)), nil
	case int64:
		return big.NewInt(v), nil
	case *big.Int:
		return new(big.Int).Set(v), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", reflect.TypeOf(value))
	}

}

func IsValidAddress(address string) (bool, error) {
	if address == "" {
		return false, ginierr.ErrEmptyAddress()
	}

	isUser, err1 := IsUserAddress(address)
	isContract, err2 := IsContractAddress(address)
	if err1 != nil {
		return false, err1
	} else if err2 != nil {
		return false, err2
	}

	return isUser || isContract, nil
}

func IsContractAddress(address string) (bool, error) {
	if address == "" {
		return false, ginierr.ErrEmptyAddress()
	}

	// Validate against contract address regex
	isValid, err := regexp.MatchString(constants.IsContractAddressRegex, address)
	if err != nil {
		logger.Log.Errorf("Error validating contract address: %v", err)
		return false, ginierr.ErrRegexValidationFailed("contract address", err)
	}

	if !isValid {
		return false, nil
	}
	return true, nil
}

func IsUserAddress(address string) (bool, error) {
	if address == "" {
		return false, ginierr.ErrEmptyAddress()
	}

	// Validate against user address regex
	isValid, err := regexp.MatchString(constants.UserAddressRegex, address)
	if err != nil {
		logger.Log.Errorf("Error validating user address: %v", err)
		return false, ginierr.ErrRegexValidationFailed("user address", err)
	}

	if !isValid {
		return false, nil
	}
	return true, nil
}

func FindContractAddress(data string) string {
	re := regexp.MustCompile(constants.ContractAddressRegex)

	return re.FindString(data)
}

func GetUserId(sdk kalpsdk.TransactionContextInterface) (string, error) {
	b64ID, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to read clientID: %v", err)
	}

	decodeID, err := base64.StdEncoding.DecodeString(b64ID)
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode clientID: %v", err)
	}

	completeId := string(decodeID)
	userId := completeId[(strings.Index(completeId, "x509::CN=") + 9):strings.Index(completeId, ",")]
	isUser, err := IsUserAddress(userId)
	if err != nil {
		return "", fmt.Errorf("error validating user address: %w", err)
	}

	if !isUser {
		return "", ginierr.ErrInvalidUserAddress(userId)
	}
	return userId, nil
}

func FilterPrintableASCII(input string) string {
	var result []rune
	for _, char := range input {
		if char >= 33 && char <= 127 {
			result = append(result, char)
		}
	}
	return string(result)
}

func IsAmountProper(amount string) bool {

	bigAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {

		return false
	}

	return bigAmount.Sign() >= 0
}
