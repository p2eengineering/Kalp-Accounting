package helper

import (
	"encoding/base64"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/logger"
	"math/big"
	"net/http"
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

func IsValidAddress(address string) bool {
	return IsUserAddress(address) || IsContractAddress(address)
}

func IsContractAddress(address string) bool {
	// Example validation logic (you can modify this to fit your use case)
	if address == "" {
		return false
	}
	// Assuming contract addresses should start with "0x" and have 42 characters
	isValid, _ := regexp.MatchString(`^klp-[a-fA-F0-9]+-cc$`, address)
	return isValid
}

func IsUserAddress(address string) bool {
	// Example validation logic (you can modify this to fit your use case)
	if address == "" {
		return false
	}
	// Assuming user addresses have the same structure as contract addresses
	isValid, _ := regexp.MatchString(`^[0-9a-fA-F]{40}$`, address)
	return isValid
}

func FindContractAddress(data string) string {
	// Define the regex pattern
	pattern := `^klp-[a-fA-F0-9]+-cc`

	// Compile the regex
	re := regexp.MustCompile(pattern)

	// Find the first match in the byte slice
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
		if char >= 33 && char <= 127 { // Printable ASCII characters are in the range 33-127
			result = append(result, char)
		}
	}
	return string(result)
}

func IsSignerKalpFoundation(ctx kalpsdk.TransactionContextInterface) (bool, error) {
	signer, e := GetUserId(ctx)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to get client id", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if signer != constants.KalpFoundationAddress {
		return false, nil
	}
	return true, nil
}

func IsAmountProper(amount string) bool {
	// Parse the amount as a big.Int
	bigAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		// Return false if amount cannot be converted to big.Int
		return false
	}

	// Check if the amount is less than 0
	return bigAmount.Sign() >= 0
}
