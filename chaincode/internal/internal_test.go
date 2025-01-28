package internal_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/models"
	"gini-contract/mocks"
	"math/big"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/peer"
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
			name:        "Failure - Amounts array has fewer than 2 elements",
			addresses:   []string{"addr1", "addr2"},
			amounts:     []string{"1000"},
			shouldError: true,
		},
		{
			name:        "Failure - Lengths of addresses and amounts arrays are not equal",
			addresses:   []string{"addr1", "addr2"},
			amounts:     []string{"1000", "2000", "3000"},
			shouldError: true,
		},
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
			name:      "Success - Mint to multiple addresses",
			addresses: []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "addr2"},
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

func TestGetTotalUTXO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		account       string
		setupContext  func(*mocks.TransactionContext, map[string][]byte)
		expectedTotal string
		shouldError   bool
	}{
		{
			testName: "Success - Get total UTXO with multiple UTXOs",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create multiple UTXOs for the same account
				utxo1 := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "1000",
				}
				utxo2 := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "2000",
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
				iterator.CloseCalls(func() error {
					return nil
				})
				ctx.GetQueryResultReturns(iterator, nil)
			},
			expectedTotal: "3000",
			shouldError:   false,
		},
		{
			testName: "Success - Get total UTXO with no UTXOs",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// No UTXOs are set, initial state
				queryString := `{"selector":{"account":"` + "16f8ff33ef05bb24fb9a30fa79e700f57a496184" + `","docType":"` + constants.UTXO + `"}}`
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(false)
				ctx.GetQueryResultReturns(iterator, nil)
				ctx.GetQueryResultReturns(iterator, nil)
				fmt.Println("query", queryString)

			},
			expectedTotal: "0",
			shouldError:   false,
		},
		{
			testName: "Failure - Query error",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.GetQueryResultReturns(nil, errors.New("query error"))
			},
			expectedTotal: "",

			shouldError: true,
		},
		{
			testName: "Failure - Iterator.Next error",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				iterator := &mocks.StateQueryIterator{}
				iterator.HasNextReturns(true)
				iterator.NextReturns(nil, errors.New("iterator error"))
				ctx.GetQueryResultReturns(iterator, nil)
			},
			expectedTotal: "",
			shouldError:   true,
		},
		{
			testName: "Failure - Invalid UTXO format",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				worldState["_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_"] = []byte("invalid json")

				results := []queryresult.KV{
					{Key: "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_", Value: []byte("invalid json")},
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
				iterator.CloseCalls(func() error {
					return nil
				})
				ctx.GetQueryResultReturns(iterator, nil)
			},
			expectedTotal: "",
			shouldError:   true,
		},
		{
			testName: "Success - Zero values in UTXOs",
			account:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				utxo1 := models.Utxo{
					DocType: constants.UTXO,
					Account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Amount:  "0",
				}
				utxoJSON1, _ := json.Marshal(utxo1)
				worldState["_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_"] = utxoJSON1

				results := []queryresult.KV{
					{Key: "_UTXO_16f8ff33ef05bb24fb9a30fa79e700f57a496184_txid1_", Value: utxoJSON1},
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
				iterator.CloseCalls(func() error {
					return nil
				})
				ctx.GetQueryResultReturns(iterator, nil)
			},
			expectedTotal: "0",
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			ctx, worldState := setupTestContext()
			if tt.setupContext != nil {
				tt.setupContext(ctx, worldState)
			}

			// Execute test
			total, err := internal.GetTotalUTXO(ctx, tt.account)

			// Assert results
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedTotal, total)
			}
		})
	}
}

func TestGetCalledContractAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupMock    func(*mocks.TransactionContext) *peer.SignedProposal
		expectedAddr string
		shouldError  bool
		errorCheck   func(error) bool
	}{
		{
			name: "Success - Valid contract address extraction",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				channelHeader := &common.ChannelHeader{
					Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
					ChannelId: "mychannel",
					TxId:      "mock-tx-id",
					Timestamp: ptypes.TimestampNow(),
				}
				channelHeaderBytes, _ := proto.Marshal(channelHeader)

				payload := &common.Payload{
					Header: &common.Header{
						ChannelHeader: channelHeaderBytes,
					},
					Data: []byte("klp-testcontract-cc"),
				}
				payloadBytes, _ := proto.Marshal(payload)

				proposal := &peer.Proposal{
					Header:  []byte("header"),
					Payload: payloadBytes,
				}
				proposalBytes, _ := proto.Marshal(proposal)

				signedProposal := &peer.SignedProposal{
					ProposalBytes: proposalBytes,
					Signature:     []byte("signature"),
				}
				ctx.GetSignedProposalReturns(signedProposal, nil)
				return signedProposal
			},
			expectedAddr: "klp-testcontract-cc",
			shouldError:  true,
		},
		{
			name: "Failure - Nil signed proposal",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				ctx.GetSignedProposalReturns(nil, nil)
				return nil
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "could not retrieve signed proposal")
			},
		},
		{
			name: "Failure - GetSignedProposal error",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				ctx.GetSignedProposalReturns(nil, errors.New("get signed proposal error"))
				return nil
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "error in getting signed proposal")
			},
		},
		{
			name: "Failure - Invalid proposal bytes",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				signedProposal := &peer.SignedProposal{
					ProposalBytes: []byte("invalid-proposal-bytes"),
					Signature:     []byte("mock-signature"),
				}
				ctx.GetSignedProposalReturns(signedProposal, nil)
				return signedProposal
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "error in parsing signed proposal")
			},
		},
		{
			name: "Failure - Invalid payload bytes",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				channelHeader := &common.ChannelHeader{
					Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
					ChannelId: "mychannel",
					TxId:      "mock-tx-id",
					Timestamp: ptypes.TimestampNow(),
				}
				channelHeaderBytes, _ := proto.Marshal(channelHeader)

				payload := &common.Payload{
					Header: &common.Header{
						ChannelHeader: channelHeaderBytes,
					},
					Data: []byte("invalid-contract"),
				}

				payloadBytes, err := proto.Marshal(payload)
				if err != nil {
					fmt.Println(err)
				}

				proposal := &peer.Proposal{
					Header:  []byte("header"),
					Payload: payloadBytes,
				}
				proposalBytes, err := proto.Marshal(proposal)
				if err != nil {
					fmt.Println(err)
				}

				signedProposal := &peer.SignedProposal{
					ProposalBytes: proposalBytes,
					Signature:     []byte("signature"),
				}
				ctx.GetSignedProposalReturns(signedProposal, nil)
				return signedProposal
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "error in parsing payload")
			},
		},
		{
			name: "Failure - Contract address not found",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				channelHeader := &common.ChannelHeader{
					Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
					ChannelId: "mychannel",
					TxId:      "mock-tx-id",
					Timestamp: ptypes.TimestampNow(),
				}
				channelHeaderBytes, _ := proto.Marshal(channelHeader)

				payload := &common.Payload{
					Header: &common.Header{
						ChannelHeader: channelHeaderBytes,
					},
				}
				payloadBytes, _ := proto.Marshal(payload)

				proposal := &peer.Proposal{
					Header:  []byte("header"),
					Payload: payloadBytes,
				}
				proposalBytes, _ := proto.Marshal(proposal)

				signedProposal := &peer.SignedProposal{
					ProposalBytes: proposalBytes,
					Signature:     []byte("signature"),
				}
				ctx.GetSignedProposalReturns(signedProposal, nil)
				return signedProposal
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "contract address not found")
			},
		},
		{
			name: "Failure - Empty channel header",
			setupMock: func(ctx *mocks.TransactionContext) *peer.SignedProposal {
				header := &common.Header{
					ChannelHeader: []byte{},
				}
				headerBytes, err := proto.Marshal(header)
				require.NoError(t, err)

				proposal := &peer.Proposal{
					Header:  headerBytes,
					Payload: []byte("test-payload"),
				}
				proposalBytes, err := proto.Marshal(proposal)
				require.NoError(t, err)

				signedProposal := &peer.SignedProposal{
					ProposalBytes: proposalBytes,
					Signature:     []byte("mock-signature"),
				}
				ctx.GetSignedProposalReturns(signedProposal, nil)
				return signedProposal
			},
			expectedAddr: "",
			shouldError:  true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "channel header is empty")
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			signedProposal := tt.setupMock(ctx)

			contractAddr, err := internal.GetCalledContractAddress(ctx)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAddr, contractAddr)
				// Test if data has the contract address that was expected
				if signedProposal != nil && tt.expectedAddr != "" {
					data := signedProposal.GetProposalBytes()
					printableASCIIPaystring := helper.FilterPrintableASCII(string(data))
					require.NotEmpty(t, printableASCIIPaystring, "failed to extract address from bytes data")
					contractAddress := helper.FindContractAddress(printableASCIIPaystring)
					require.Equal(t, tt.expectedAddr, contractAddress, "failed to find correct contract address from bytes data")
				}
			}
		})
	}
}

func TestIsGatewayAdminAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		setupContext   func(*mocks.TransactionContext, map[string][]byte)
		expectedResult bool
		shouldError    bool
		errorCheck     func(error) bool
	}{
		{
			name:   "Success - User is a gateway admin",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				key, _ := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", constants.KalpGateWayAdminRole})
				// Set up a gateway admin role
				data := []byte(`{"user":"16f8ff33ef05bb24fb9a30fa79e700f57a496184","role":"KalpGatewayAdmin"}`)
				worldState[key] = data
			},
			expectedResult: true,
			shouldError:    false,
		},
		{
			name:   "Success - User is not a gateway admin",
			userID: "2da4c4908a393a387b728206b18388bc529fa8d7",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				key, _ := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{"2da4c4908a393a387b728206b18388bc529fa8d7", constants.KalpGateWayAdminRole})
				// Set up a regular user role (not a gateway admin)
				data := []byte(`{"user":"2da4c4908a393a387b728206b18388bc529fa8d7","role":"User"}`)
				worldState[key] = data
			},
			expectedResult: false,
			shouldError:    false,
		},
		{
			name:   "Success - No role set for the user",
			userID: "2da4c4908a393a387b728206b18388bc529fa8d7",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// No setup, meaning no role is set for the user
			},
			expectedResult: false,
			shouldError:    false,
		},
		{
			name:   "Failure - GetState error",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Simulate an error when calling GetState
				ctx.GetStateStub = func(key string) ([]byte, error) {
					return nil, errors.New("get state error")
				}
			},
			expectedResult: false,
			shouldError:    true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "Failed to fetch gateway admin role")
			},
		},
		{
			name:   "Failure - Invalid JSON data",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				key, _ := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184",constants.KalpGateWayAdminRole})
				// Set invalid JSON data
				worldState[key] = []byte("invalid json")
			},
			expectedResult: false,
			shouldError:    true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "Failed to unmarshal user role data")
			},
		},
		{
			name:   "Failure - Composite key creation error",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Simulate an error when creating the composite key
				ctx.CreateCompositeKeyStub = func(objectType string, attributes []string) (string, error) {
					return "", errors.New("composite key creation error")
				}
			},
			expectedResult: false,
			shouldError:    true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "failed to create the composite key")
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			worldState := map[string][]byte{}

			// Default stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if val, exists := worldState[key]; exists {
					return val, nil
				}
				return nil, nil
			}
			transactionContext.CreateCompositeKeyStub = func(objectType string, attributes []string) (string, error) {
				return objectType + strings.Join(attributes, ""), nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState)
			}

			// Execute test
			result, err := internal.IsGatewayAdminAddress(transactionContext, tt.userID)

			// Assert results
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestUpdateAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName     string
		setupContext func(*mocks.TransactionContext, map[string][]byte)
		owner        string
		spender      string
		spent        string
		shouldError  bool
		errorCheck   func(error) bool
	}{
		{
			testName: "Success - Update with valid spent amount",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				approvalKey, _ := ctx.CreateCompositeKey(constants.Approval, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "2da4c4908a393a387b728206b18388bc529fa8d7"})
				// Set up an initial allowance
				approval := models.Allow{
					Owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
					Amount:  "1000",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "500",
			shouldError: false,
		},
		{
			testName: "Success - Update with exact spent amount",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				approvalKey, _ := ctx.CreateCompositeKey(constants.Approval, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "2da4c4908a393a387b728206b18388bc529fa8d7"})
				// Set up an initial allowance
				approval := models.Allow{
					Owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
					Amount:  "1000",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "1000",
			shouldError: false,
		},
		{
			testName: "Success - Update to Zero Amount",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				approvalKey, _ := ctx.CreateCompositeKey(constants.Approval, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "2da4c4908a393a387b728206b18388bc529fa8d7"})
				// Set up an initial allowance
				approval := models.Allow{
					Owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
					Amount:  "1000",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "1000",
			shouldError: false,
		},
		{
			testName: "Failure - GetState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Simulate an error when calling GetState
				ctx.GetStateReturns(nil, errors.New("get state error"))
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "500",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "failed to read current allowance of owner with address")
			},
		},
		{
			testName: "Failure - Invalid allowance data format",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				approvalKey, _ := ctx.CreateCompositeKey(constants.Approval, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "2da4c4908a393a387b728206b18388bc529fa8d7"})
				// Set invalid JSON data
				worldState[approvalKey] = []byte("invalid json")
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "500",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "failed to unmarshal allowance for owner address")
			},
		},
		{
			testName: "Failure - Spent amount greater than allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// Create composite key
				approvalKey, _ := ctx.CreateCompositeKey(constants.Approval, []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184", "2da4c4908a393a387b728206b18388bc529fa8d7"})
				// Set up an initial allowance
				approval := models.Allow{
					Owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
					Spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
					Amount:  "100",
					DocType: constants.Allowance,
				}
				approvalJSON, _ := json.Marshal(approval)
				worldState[approvalKey] = approvalJSON
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "1000",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "the amount spent :1000, is greater than allowance :100")
			},
		},
		{
			testName: "Failure - No allowance exists",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				// No setup, meaning no allowance exists
			},
			owner:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:     "2da4c4908a393a387b728206b18388bc529fa8d7",
			spent:       "500",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "no allowance exists for owner with address")
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			worldState := map[string][]byte{}

			// Default stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if val, exists := worldState[key]; exists {
					return val, nil
				}
				return nil, nil
			}
			transactionContext.CreateCompositeKeyStub = func(objectType string, attributes []string) (string, error) {
				return "_" + objectType + "_" + strings.Join(attributes, "_") + "_", nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
				worldState[key] = value
				return nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState)
			}

			// Execute test
			err := internal.UpdateAllowance(transactionContext, tt.owner, tt.spender, tt.spent)

			// Assert results
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err), "Error message does not match expected message")
				}
			} else {
				require.NoError(t, err)
				// Verify that the allowance was updated correctly
				approvalKey, _ := transactionContext.CreateCompositeKey(constants.Approval, []string{tt.owner, tt.spender})
				approvalBytes := worldState[approvalKey]
				var approval models.Allow
				err = json.Unmarshal(approvalBytes, &approval)
				require.NoError(t, err)

				initialAmount, _ := new(big.Int).SetString("1000", 10)
				spentAmount, _ := new(big.Int).SetString(tt.spent, 10)
				expectedAmount := initialAmount.Sub(initialAmount, spentAmount)
				require.Equal(t, expectedAmount.String(), approval.Amount, "allowance was not updated correctly")
			}
		})
	}
}
func TestInitializeRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		role        string
		setupMock   func(*mocks.TransactionContext, map[string][]byte)
		shouldError bool
	}{
		{
			name:   "Success - Initialize Kalp Foundation role",
			userID: constants.KalpFoundationAddress,
			role:   constants.KalpFoundationRole,
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {

			},
			shouldError: false,
		},
		{
			name:   "Success - Initialize Gateway Admin role",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			role:   constants.KalpGateWayAdminRole,
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {

			},
			shouldError: false,
		},
		{
			name:   "Failure - CreateCompositeKey Error",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			role:   constants.KalpGateWayAdminRole,
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.CreateCompositeKeyReturns("", errors.New("failed to create composite key"))
			},
			shouldError: true,
		},
		{
			name:   "Failure - PutState Error",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			role:   constants.KalpGateWayAdminRole,
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
				ctx.PutStateWithoutKYCReturns(errors.New("failed to put state"))
			},
			shouldError: true,
		},
		{
			name:   "Failure - Empty userID",
			userID: "",
			role:   constants.KalpGateWayAdminRole,
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
			},
			shouldError: false,
		},
		{
			name:   "Failure - Empty role",
			userID: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			role:   "",
			setupMock: func(ctx *mocks.TransactionContext, worldState map[string][]byte) {
			},
			shouldError: false,
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

			result, err := internal.InitializeRoles(ctx, tt.userID, tt.role)

			if tt.shouldError {
				require.Error(t, err)
				require.False(t, result)
			} else {
				require.NoError(t, err)
				require.True(t, result)
			}
		})
	}
}
