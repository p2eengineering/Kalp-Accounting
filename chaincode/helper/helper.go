package helper

import (
	"encoding/base64"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/muditp2e/kalp-sdk-public/kalpsdk"
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

func IsValidAddress(address string) bool {
	return IsUserAddress(address) || IsContractAddress(address)
}

func IsContractAddress(address string) bool {

	if address == "" {
		return false
	}

	isValid, _ := regexp.MatchString(constants.IsContractAddressRegex, address)
	return isValid
}

func IsUserAddress(address string) bool {

	if address == "" {
		return false
	}

	isValid, _ := regexp.MatchString(constants.UserAddressRegex, address)
	return isValid
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
	if !IsUserAddress(userId) {
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
