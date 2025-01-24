package models_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/models"
	"gini-contract/mocks"
	"testing"

	"github.com/stretchr/testify/require"
)

func SetUserID(transactionContext *mocks.TransactionContext, userID string) {
	completeId := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", userID)

	// Base64 encode the complete ID
	b64ID := base64.StdEncoding.EncodeToString([]byte(completeId))

	clientIdentity := &mocks.ClientIdentity{}
	clientIdentity.GetIDReturns(b64ID, nil)
	transactionContext.GetClientIdentityReturns(clientIdentity)
}

func TestSetAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte)
		spender       string
		amount        string
		expectedError error
	}{
		{
			testName: "Success - Set valid allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			spender:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:        "1000",
			expectedError: nil,
		},
		{
			testName: "Success - Update existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				SetUserID(ctx, owner)

				// Set existing allowance
				approvalKey := fmt.Sprintf("_Approval_%s_%s_", owner, "2da4c4908a393a387b728206b18388bc529fa8d7")
				approval := models.Allow{
					Owner:   owner,
					Spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
					Amount:  "500",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			spender:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:        "1000",
			expectedError: nil,
		},
		{
			testName: "Failure - GetUserId error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns("", errors.New("failed to get ID"))
				ctx.GetClientIdentityReturns(clientIdentity)
				ctx.GetUserIDReturns("", errors.New("failed to get ID"))
			},
			spender:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:        "1000",
			expectedError: ginierr.ErrFailedToGetPublicAddress,
		},
		{
			testName: "Failure - Invalid spender address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				SetUserID(ctx, owner)
			},
			spender:       "invalid-address",
			amount:        "1000",
			expectedError: ginierr.ErrInvalidAddress("invalid-address"),
		},
		{
			testName: "Failure - Invalid amount",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				SetUserID(ctx, owner)
			},
			spender:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:        "invalid-amount",
			expectedError: ginierr.ErrInvalidAmount("invalid-amount"),
		},
		{
			testName: "Failure - PutState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				SetUserID(ctx, owner)
				ctx.PutStateWithoutKYCReturns(errors.New("failed to put state"))
			},
			spender:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:        "1000",
			expectedError: fmt.Errorf("failed to update data of smart contract: failed to put state, status code:500"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
				if tt.testName == "Failure - PutState error" {
					return errors.New("failed to put state")
				}
				worldState[key] = value
				return nil
			}
			transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
				key := "_" + prefix + "_"
				for _, attr := range attrs {
					key += attr + "_"
				}
				return key, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState)
			}

			// Execute test
			err := models.SetAllowance(transactionContext, tt.spender, tt.amount)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName       string
		setupContext   func(*mocks.TransactionContext, map[string][]byte)
		owner          string
		spender        string
		expectedAmount string
		shouldError    bool
	}{
		{
			testName: "Success - Get existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				spender := "2da4c4908a393a387b728206b18388bc529fa8d7"
				// Set up existing allowance
				approvalKey := fmt.Sprintf("_Approval_%s_%s_", owner, spender)
				approval := models.Allow{
					Owner:   owner,
					Spender: spender,
					Amount:  "1000",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "1000",
			shouldError:    false,
		},
		{
			testName: "Success - No existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "0",
			shouldError:    false,
		},
		{
			testName: "Failure - GetState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.GetStateReturns(nil, errors.New("failed to get state"))
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "",
			shouldError:    true,
		},
		{
			testName: "Success - Empty amount in allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				spender := "2da4c4908a393a387b728206b18388bc529fa8d7"
				approvalKey := fmt.Sprintf("_Approval_%s_%s_", owner, spender)
				approval := models.Allow{
					Owner:   owner,
					Spender: spender,
					Amount:  "",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "0",
			shouldError:    false,
		},
		{
			testName: "Failure - Invalid allowance data format",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				owner := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
				spender := "2da4c4908a393a387b728206b18388bc529fa8d7"
				approvalKey := fmt.Sprintf("_Approval_%s_%s_", owner, spender)
				worldState[approvalKey] = []byte("invalid json")
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "",
			shouldError:    true,
		},
		{
			testName: "Failure - CreateCompositeKey error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.CreateCompositeKeyReturns("", errors.New("failed to create composite key"))
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.testName == "Failure - GetState error" {
					return nil, errors.New("failed to get state")
				}
				return worldState[key], nil
			}
			transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
				if tt.testName == "Failure - CreateCompositeKey error" {
					return "", errors.New("failed to create composite key")
				}
				key := "_" + prefix + "_"
				for _, attr := range attrs {
					key += attr + "_"
				}
				return key, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState)
			}

			// Execute test
			amount, err := models.GetAllowance(transactionContext, tt.owner, tt.spender)

			// Assert results
			if tt.shouldError {
				require.Error(t, err)
				require.Equal(t, tt.expectedAmount, amount)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAmount, amount)

				// If successful, verify the returned amount matches what's in state
				if amount != "0" {
					approvalKey, _ := transactionContext.CreateCompositeKey(constants.Approval,
						[]string{tt.owner, tt.spender})
					approvalBytes := worldState[approvalKey]
					var approval models.Allow
					err = json.Unmarshal(approvalBytes, &approval)
					require.NoError(t, err)
					require.Equal(t, amount, approval.Amount)
				}
			}
		})
	}
}