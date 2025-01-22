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

func TestRemoveUtxo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		account     string
		amount      interface{}
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:    "Success - Remove exact UTXO amount",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(1000),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Setup existing UTXO
				utxo := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxoJSON, _ := json.Marshal(utxo)
				utxoKey := "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid_"
				worldState[utxoKey] = utxoJSON

				// Setup query iterator
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(&queryresult.KV{
					Key:   utxoKey,
					Value: utxoJSON,
				}, nil)
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: false,
		},
		{
			name:    "Success - Remove partial UTXO amount",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(500),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Setup existing UTXO with larger amount
				utxo := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxoJSON, _ := json.Marshal(utxo)
				utxoKey := "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid_"
				worldState[utxoKey] = utxoJSON

				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(&queryresult.KV{
					Key:   utxoKey,
					Value: utxoJSON,
				}, nil)
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: false,
		},
		{
			name:    "Success - Remove from multiple UTXOs",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(1500),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Setup two UTXOs
				utxo1 := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxo2 := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxoJSON1, _ := json.Marshal(utxo1)
				utxoJSON2, _ := json.Marshal(utxo2)

				worldState["_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_"] = utxoJSON1
				worldState["_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid2_"] = utxoJSON2

				results := []queryresult.KV{
					{Key: "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_", Value: utxoJSON1},
					{Key: "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid2_", Value: utxoJSON2},
				}

				currentIndex := 0
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextCalls(func() bool {
					return currentIndex < len(results)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if currentIndex < len(results) {
						result := &results[currentIndex]
						currentIndex++
						return result, nil
					}
					return nil, nil
				})
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - Insufficient balance",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(2000),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Setup UTXO with less amount
				utxo := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxoJSON, _ := json.Marshal(utxo)
				utxoKey := "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid_"
				worldState[utxoKey] = utxoJSON

				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(&queryresult.KV{
					Key:   utxoKey,
					Value: utxoJSON,
				}, nil)
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - Query error",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(1000),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.GetQueryResultReturns(nil, errors.New("query error"))
			},
			shouldError: true,
		},
		{
			name:    "Failure - Iterator.Next error",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(1000),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(nil, errors.New("iterator error"))
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: true,
		},
		{
			name:    "Failure - Invalid UTXO JSON",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  "1000",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(&queryresult.KV{
					Key:   "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid_",
					Value: []byte("invalid json"),
				}, nil)
				ctx.GetQueryResultReturns(iterator, nil)
			},
			shouldError: true,
		},
		{
			name:    "Failure - DelState error",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(1000),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				utxo := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxoJSON, _ := json.Marshal(utxo)
				utxoKey := "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid_"
				worldState[utxoKey] = utxoJSON

				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(&queryresult.KV{
					Key:   utxoKey,
					Value: utxoJSON,
				}, nil)
				ctx.GetQueryResultReturns(iterator, nil)
				ctx.DelStateWithoutKYCReturns(errors.New("delete error"))
			},
			shouldError: true,
		},
		{
			name:    "Success - Zero amount",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:  big.NewInt(0),
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
			},
			shouldError: false,
		},
		{
			name:        "Failure - Negative amount",
			account:     "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:      big.NewInt(-1000),
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

			err := internal.RemoveUtxo(ctx, tt.account, tt.amount)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.name == "Success - Remove partial UTXO amount" {
					// Verify remaining amount
					for key, value := range worldState {
						if strings.HasPrefix(key, "_UTXO_") {
							var utxo models.Utxo
							json.Unmarshal(value, &utxo)
							remainingAmount, _ := new(big.Int).SetString(utxo.Amount, 10)
							require.Equal(t, "500", remainingAmount.String())
						}
					}
				}

				if tt.name == "Success - Remove from multiple UTXOs" {
					// Verify correct UTXOs were removed/updated
					remainingUTXOs := 0
					for key := range worldState {
						if strings.HasPrefix(key, "_UTXO_") {
							remainingUTXOs++
						}
					}
					require.Equal(t, 1, remainingUTXOs)
				}
			}
		})
	}
}
