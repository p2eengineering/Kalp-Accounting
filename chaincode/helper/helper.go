package helper

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"
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
	return IsHexAddress(address) || IsContractAddress(address)
}

func IsContractAddress(address string) bool {
	// Check if the string starts with "klp-" and ends with "-cc"
	if strings.HasPrefix(address, "klp-") && strings.HasSuffix(address, "-cc") {
		return true
	}
	return false
}

func IsHexAddress(address string) bool {
	// Check if the string is at least 40 characters hexadecimal
	if len(address) >= 40 && isHexadecimal(address) {
		return true
	}
	return false
}

// Helper function to check if a string is hexadecimal
func isHexadecimal(input string) bool {
	_, err := hex.DecodeString(input)
	return err == nil
}

func FindContractAddress(data []byte) string {
	// Define the regex pattern
	pattern := `kpl-.*?-cc`

	// Compile the regex
	re := regexp.MustCompile(pattern)

	// Find the first match in the byte slice
	matches := re.Find(data)

	// Return the matched string or an empty string if no match is found
	if matches != nil {
		return string(matches)
	}
	return ""
}
