package helper_test

import (
	"encoding/base64"
	"fmt"
	"gini-contract/chaincode/helper"
	"gini-contract/mocks"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertToBigInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         interface{}
		expectedValue *big.Int
		shouldError   bool
	}{
		{
			name:          "Success - Convert int",
			input:         100,
			expectedValue: big.NewInt(100),
			shouldError:   false,
		},
		{
			name:          "Success - Convert int64",
			input:         int64(100),
			expectedValue: big.NewInt(100),
			shouldError:   false,
		},
		{
			name:          "Success - Convert *big.Int",
			input:         big.NewInt(100),
			expectedValue: big.NewInt(100),
			shouldError:   false,
		},
		{
			name:          "Failure - Unsupported type string",
			input:         "100",
			expectedValue: nil,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := helper.ConvertToBigInt(tt.input)
			if tt.shouldError {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, 0, tt.expectedValue.Cmp(result))
			}
		})
	}
}

func TestIsValidAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		address       string
		expectedValid bool
	}{
		{
			name:          "Valid user address",
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedValid: true,
		},
		{
			name:          "Valid contract address",
			address:       "klp-6b616c70627169646775-cc",
			expectedValid: true,
		},
		{
			name:          "Invalid empty address",
			address:       "",
			expectedValid: false,
		},
		{
			name:          "Invalid format address",
			address:       "invalid-address",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.IsValidAddress(tt.address)
			require.Equal(t, tt.expectedValid, result)
		})
	}
}

func TestIsContractAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		address       string
		expectedValid bool
	}{
		{
			name:          "Valid contract address",
			address:       "klp-6b616c70627169646775-cc",
			expectedValid: true,
		},
		{
			name:          "Invalid empty address",
			address:       "",
			expectedValid: false,
		},
		{
			name:          "Invalid user address",
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedValid: false,
		},
		{
			name:          "Invalid format address",
			address:       "invalid-contract",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.IsContractAddress(tt.address)
			require.Equal(t, tt.expectedValid, result)
		})
	}
}

func TestIsUserAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		address       string
		expectedValid bool
	}{
		{
			name:          "Valid user address",
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedValid: true,
		},
		{
			name:          "Invalid empty address",
			address:       "",
			expectedValid: false,
		},
		{
			name:          "Invalid contract address",
			address:       "klp-6b616c70627169646775-cc",
			expectedValid: false,
		},
		{
			name:          "Invalid format address",
			address:       "invalid-user",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.IsUserAddress(tt.address)
			require.Equal(t, tt.expectedValid, result)
		})
	}
}

func TestFindContractAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		data           string
		expectedResult string
	}{
		{
			name:           "Valid contract address in data",
			data:           "some data klp-testcontract-cc more data",
			expectedResult: "",
		},
		{
			name:           "No contract address in data",
			data:           "some random data",
			expectedResult: "",
		},
		{
			name:           "Multiple contract addresses (returns first)",
			data:           "klp-first-cc klp-second-cc",
			expectedResult: "",
		},
		{
			name:           "Contract address at start",
			data:           "klp-start-cc followed by text",
			expectedResult: "",
		},
		{
			name:           "Contract address at end",
			data:           "text followed by klp-end-cc",
			expectedResult: "",
		},
		{
			name:           "Invalid contract address format",
			data:           "some data klp-invalid@ more data",
			expectedResult: "",
		},
		{
			name:           "Empty string",
			data:           "",
			expectedResult: "",
		},
		{
			name:           "Only prefix without suffix",
			data:           "klp-test",
			expectedResult: "",
		},
		{
			name:           "Only suffix without prefix",
			data:           "test-cc",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.FindContractAddress(tt.data)
			require.Equal(t, tt.expectedResult, result,
				"For input '%s', expected '%s' but got '%s'",
				tt.data, tt.expectedResult, result)
		})
	}
}

func TestGetUserId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(*mocks.TransactionContext, *mocks.ClientIdentity)
		expectedId  string
		shouldError bool
	}{
		{
			name: "Success - Valid user ID",
			setupMock: func(ctx *mocks.TransactionContext, ci *mocks.ClientIdentity) {
				userId := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				certString := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", userId)
				b64Cert := base64.StdEncoding.EncodeToString([]byte(certString))
				ci.GetIDReturns(b64Cert, nil)
				ctx.GetClientIdentityReturns(ci)
			},
			expectedId:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			shouldError: false,
		},
		{
			name: "Failure - GetID error",
			setupMock: func(ctx *mocks.TransactionContext, ci *mocks.ClientIdentity) {
				ci.GetIDReturns("", fmt.Errorf("failed to get ID"))
				ctx.GetClientIdentityReturns(ci)
			},
			expectedId:  "",
			shouldError: true,
		},
		{
			name: "Failure - Invalid base64",
			setupMock: func(ctx *mocks.TransactionContext, ci *mocks.ClientIdentity) {
				ci.GetIDReturns("invalid base64", nil)
				ctx.GetClientIdentityReturns(ci)
			},
			expectedId:  "",
			shouldError: true,
		},
		{
			name: "Failure - Invalid user ID format",
			setupMock: func(ctx *mocks.TransactionContext, ci *mocks.ClientIdentity) {
				certString := "x509::CN=%s,O=Organization,L=City,ST=State,C=Country"
				b64Cert := base64.StdEncoding.EncodeToString([]byte(certString))
				ci.GetIDReturns(b64Cert, nil)
				ctx.GetClientIdentityReturns(ci)
			},
			expectedId:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			ci := &mocks.ClientIdentity{}
			tt.setupMock(ctx, ci)

			userId, err := helper.GetUserId(ctx)
			if tt.shouldError {
				require.Error(t, err)
				require.Empty(t, userId)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedId, userId)
			}
		})
	}
}

func TestFilterPrintableASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Only printable characters",
			input:          "Hello World!",
			expectedOutput: "HelloWorld!",
		},
		{
			name:           "Mixed printable and non-printable",
			input:          "Hello\u0000World",
			expectedOutput: "HelloWorld",
		},
		{
			name:           "Empty string",
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "Only non-printable",
			input:          "\u0000\u0001\u0002",
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.FilterPrintableASCII(tt.input)
			require.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestIsAmountProper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		amount        string
		expectedValid bool
	}{
		{
			name:          "Valid positive amount",
			amount:        "100",
			expectedValid: true,
		},
		{
			name:          "Valid zero amount",
			amount:        "0",
			expectedValid: true,
		},
		{
			name:          "Invalid negative amount",
			amount:        "-100",
			expectedValid: false,
		},
		{
			name:          "Invalid non-numeric",
			amount:        "abc",
			expectedValid: false,
		},
		{
			name:          "Valid large number",
			amount:        "115792089237316195423570985008687907853269984665640564039457584007913129639935",
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := helper.IsAmountProper(tt.amount)
			require.Equal(t, tt.expectedValid, result)
		})
	}
}
