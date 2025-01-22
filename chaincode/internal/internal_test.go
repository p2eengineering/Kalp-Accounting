package internal_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/models"
	"gini-contract/mocks"
	"math/big"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"
)

// Helper function to setup the test context
func setupTestContext() (*mocks.TransactionContext, map[string][]byte) {
	ctx := &mocks.TransactionContext{}
	worldState := make(map[string][]byte)

	// Setup basic stubs
	ctx.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	ctx.PutStateWithoutKYCStub = func(key string, value []byte) error {
		worldState[key] = value
		return nil
	}
	ctx.DelStateWithoutKYCStub = func(key string) error {
		delete(worldState, key)
		return nil
	}
	ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
		key := "_" + prefix + "_"
		for _, attr := range attrs {
			key += attr + "_"
		}
		return key, nil
	}

	return ctx, worldState
}

func setupStateQueryIterator(worldState map[string][]byte) func(string, []string) (*mocks.StateQueryIterator, error) {
	return func(objectType string, keys []string) (*mocks.StateQueryIterator, error) {
		iterator := &mocks.StateQueryIterator{}
		var kvs []queryresult.KV

		prefix := "_" + objectType + "_"
		if len(keys) > 0 {
			prefix += keys[0] + "_"
		}

		for key, value := range worldState {
			if strings.HasPrefix(key, prefix) {
				kvs = append(kvs, queryresult.KV{
					Key:   key,
					Value: value,
				})
			}
		}

		iterator.HasNextReturns(len(kvs) > 0)
		iterator.NextReturns(&kvs[0], nil)
		return iterator, nil
	}
}

func TestIsSignerKalpFoundation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func(*mocks.TransactionContext)
		expectedResult bool
		shouldError    bool
	}{
		{
			name: "Success - Is Kalp Foundation",
			setupMock: func(ctx *mocks.TransactionContext) {
				clientID := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", constants.KalpFoundationAddress)
				b64ClientID := base64.StdEncoding.EncodeToString([]byte(clientID))
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns(b64ClientID, nil)
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			expectedResult: true,
			shouldError:    false,
		},
		{
			name: "Success - Not Kalp Foundation",
			setupMock: func(ctx *mocks.TransactionContext) {
				clientID := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				b64ClientID := base64.StdEncoding.EncodeToString([]byte(clientID))
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns(b64ClientID, nil)
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			expectedResult: false,
			shouldError:    false,
		},
		{
			name: "Failure - GetID error",
			setupMock: func(ctx *mocks.TransactionContext) {
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns("", errors.New("failed to get ID"))
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			expectedResult: false,
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			tt.setupMock(ctx)

			result, err := internal.IsSignerKalpFoundation(ctx)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestDenyAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		address     string
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:    "Success - Deny address",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - CreateCompositeKey error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.CreateCompositeKeyReturns("", errors.New("composite key error"))
			},
			shouldError: true,
		},
		{
			name:    "Failure - PutState error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.PutStateWithoutKYCReturns(errors.New("put state error"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, worldState := setupTestContext()
			if tt.setupMock != nil {
				tt.setupMock(ctx, worldState)
			}

			err := internal.DenyAddress(ctx, tt.address)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify state change
				key, _ := ctx.CreateCompositeKey(constants.DenyListKey, []string{tt.address})
				value := worldState[key]
				require.Equal(t, []byte("true"), value)
			}
		})
	}
}

func TestAllowAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		address     string
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:    "Success - Allow address",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - CreateCompositeKey error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.CreateCompositeKeyReturns("", errors.New("composite key error"))
			},
			shouldError: true,
		},
		{
			name:    "Failure - PutState error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.PutStateWithoutKYCReturns(errors.New("put state error"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, worldState := setupTestContext()
			if tt.setupMock != nil {
				tt.setupMock(ctx, worldState)
			}

			err := internal.AllowAddress(ctx, tt.address)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify state change
				key, _ := ctx.CreateCompositeKey(constants.DenyListKey, []string{tt.address})
				value := worldState[key]
				require.Equal(t, []byte("false"), value)
			}
		})
	}
}

func TestIsDenied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		address        string
		setupMock      func(*mocks.TransactionContext, map[string][]byte)
		expectedResult bool
		shouldError    bool
	}{
		{
			name:    "Success - Address is denied",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				key, _ := ctx.CreateCompositeKey(constants.DenyListKey, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184"})
				worldState[key] = []byte("true")
			},
			expectedResult: true,
			shouldError:    false,
		},
		{
			name:    "Success - Address is not denied",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// No entry in worldState means not denied
			},
			expectedResult: false,
			shouldError:    false,
		},
		{
			name:    "Success - Address is explicitly not denied",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				key, _ := ctx.CreateCompositeKey(constants.DenyListKey, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184"})
				worldState[key] = []byte("false")
			},
			expectedResult: false,
			shouldError:    false,
		},
		{
			name:    "Failure - GetState error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.GetStateReturns(nil, errors.New("get state error"))
			},
			expectedResult: false,
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, worldState := setupTestContext()
			if tt.setupMock != nil {
				tt.setupMock(ctx, worldState)
			}

			result, err := internal.IsDenied(ctx, tt.address)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestMint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		addresses   []string
		amounts     []string
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:      "Success - Mint to multiple addresses",
			addresses: []string{"addr1", "addr2"},
			amounts:   []string{"1000", "2000"},
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.SetEventReturns(nil)
			},
			shouldError: true,
		},
		{
			name:        "Failure - Invalid amount",
			addresses:   []string{"addr1", "addr2"},
			amounts:     []string{"invalid", "2000"},
			shouldError: true,
		},
		{
			name:        "Failure - Invalid address",
			addresses:   []string{"invalid", "addr2"},
			amounts:     []string{"1000", "2000"},
			shouldError: true,
		},
		{
			name:      "Failure - Already initialized",
			addresses: []string{"addr1", "addr2"},
			amounts:   []string{"1000", "2000"},
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				worldState[constants.NameKey] = []byte("Already Set")
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, worldState := setupTestContext()
			if tt.setupMock != nil {
				tt.setupMock(ctx, worldState)
			}

			err := internal.Mint(ctx, tt.addresses, tt.amounts)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify UTXO creation
				for i, addr := range tt.addresses {
					utxoKey, _ := ctx.CreateCompositeKey(constants.UTXO, []string{addr, ctx.GetTxID()})
					var utxo models.Utxo
					json.Unmarshal(worldState[utxoKey], &utxo)
					require.Equal(t, tt.amounts[i], utxo.Amount)
				}
			}
		})
	}
}

func TestAddUtxo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		account     string
		amount      interface{}
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:        "Success - Add UTXO with int amount",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      1000,
			shouldError: false,
		},
		{
			name:        "Success - Add UTXO with big.Int amount",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      big.NewInt(1000),
			shouldError: false,
		},
		{
			name:        "Success - Add UTXO with zero amount",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      0,
			shouldError: false,
		},
		{
			name:        "Failure - Negative amount",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      -1000,
			shouldError: true,
		},
		{
			name:        "Failure - Invalid amount type",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      "invalid",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, worldState := setupTestContext()
			if tt.setupMock != nil {
				tt.setupMock(ctx, worldState)
			}

			err := internal.AddUtxo(ctx, tt.account, tt.amount)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
