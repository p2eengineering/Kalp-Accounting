package helper

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func CustomBigIntConvertor(value interface{}) (*big.Int, error) {
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
	isValid, _ := regexp.MatchString(`^klp.*cc$`, address)
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
	pattern := `^klp-.*?-cc`

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
