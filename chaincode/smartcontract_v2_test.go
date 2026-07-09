package chaincode_test

import (
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/mocks"
	"net/http"
	"strconv"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

// GINI-3.1 — TransferGasFeesToFoundation is disabled LAST, after migration completion
// was confirmed. Migration is confirmed complete on Web2, so this fn must now reject too.

func TestTransferGasFeesToFoundation_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.TransferGasFeesToFoundation(transactionContext, "kwl-someaddress-cc", "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("TransferGasFeesToFoundation").Error(), err.Error())
	requireGone(t, err)
}

func TestGasFeesTransfer(t *testing.T) {
	t.Parallel()

	setSignedProposal := func(ctx *mocks.TransactionContext, contractName string) {
		ctx.GetSignedProposalStub = func() (*peer.SignedProposal, error) {
			mockHeader := &common.Header{ChannelHeader: []byte(contractName)}
			mockHeaderBytes, err := proto.Marshal(mockHeader)
			if err != nil {
				return nil, err
			}
			mockPayload := &common.Payload{Header: mockHeader, Data: []byte("mockData")}
			mockPayloadBytes, err := proto.Marshal(mockPayload)
			if err != nil {
				return nil, err
			}
			mockProposal := &peer.Proposal{Header: mockHeaderBytes, Payload: mockPayloadBytes}
			mockProposalBytes, err := proto.Marshal(mockProposal)
			if err != nil {
				return nil, err
			}
			return &peer.SignedProposal{ProposalBytes: mockProposalBytes}, nil
		}
	}

	tests := []struct {
		testName       string
		gasFeesAccount string
		amount         string
		setupContext   func(*mocks.TransactionContext, *chaincode.SmartContract)
		expectedBool   bool
		expectedErr    error
	}{
		{
			testName:       "Error - signer is not gateway admin",
			gasFeesAccount: constants.KalpFoundationAddress,
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			expectedBool: false,
			expectedErr:  ginierr.New("signer should be gateway admin for gas fees deduction", http.StatusUnauthorized),
		},
		{
			testName:       "Error - invalid gas fees account address",
			gasFeesAccount: "not-a-valid-address",
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrInvalidAddress("not-a-valid-address"),
		},
		{
			testName:       "Error - amount is not a valid number",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "abc",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrInvalidAmount("abc"),
		},
		{
			testName:       "Error - amount is zero",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "0",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrInvalidAmount("0"),
		},
		{
			testName:       "Error - amount exceeds max gateway gas fee",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "100000000000000001",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrInvalidAmount("100000000000000001"),
		},
		{
			testName:       "Error - signer is denied",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
				ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
					key := "_" + prefix + "_"
					for _, a := range attrs {
						key += a + "_"
					}
					return key, nil
				}
				ctx.GetStateReturnsOnCall(0, []byte("true"), nil)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrDeniedAddress(constants.KalpGateWayAdminAddress),
		},
		{
			testName:       "Error - gas fees account is denied",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
				ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
					key := "_" + prefix + "_"
					for _, a := range attrs {
						key += a + "_"
					}
					return key, nil
				}
				ctx.GetStateReturnsOnCall(0, []byte("false"), nil)
				ctx.GetStateReturnsOnCall(1, []byte("true"), nil)
			},
			expectedBool: false,
			expectedErr:  ginierr.ErrDeniedAddress("16f8ff33ef05bb24fb9a30fa79e700f57a496184"),
		},
		{
			testName:       "Error - called by a different contract",
			gasFeesAccount: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
				ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
					key := "_" + prefix + "_"
					for _, a := range attrs {
						key += a + "_"
					}
					return key, nil
				}
				ctx.GetStateReturnsOnCall(0, []byte("false"), nil)
				ctx.GetStateReturnsOnCall(1, []byte("false"), nil)
				contract.Contract.Name = "klp-gini-cc"
				setSignedProposal(ctx, "klp-someothercontract-cc")
			},
			expectedBool: false,
			expectedErr:  ginierr.New("GasFeesTransfer should not be called by other contracts", http.StatusUnauthorized),
		},
		{
			testName:       "Success - gas fees account is the foundation address (no-op transfer)",
			gasFeesAccount: constants.KalpFoundationAddress,
			amount:         "100",
			setupContext: func(ctx *mocks.TransactionContext, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpGateWayAdminAddress)
				ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
					key := "_" + prefix + "_"
					for _, a := range attrs {
						key += a + "_"
					}
					return key, nil
				}
				ctx.GetStateReturnsOnCall(0, []byte("false"), nil)
				ctx.GetStateReturnsOnCall(1, []byte("false"), nil)
				contract.Contract.Name = "klp-gini-cc"
				setSignedProposal(ctx, "klp-gini-cc")
			},
			expectedBool: true,
			expectedErr:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			tt.setupContext(transactionContext, giniContract)

			ok, err := giniContract.GasFeesTransfer(transactionContext, tt.gasFeesAccount, tt.amount)

			require.Equal(t, tt.expectedBool, ok)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMintByFaucetAdmin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName     string
		address      string
		amount       string
		setupContext func(*mocks.TransactionContext)
		expectedErr  error
	}{
		{
			testName: "Error - signer is not the faucet admin",
			address:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:   "100",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			expectedErr: ginierr.New("signer should be faucet admin for mint", http.StatusUnauthorized),
		},
		{
			testName: "Error - invalid recipient address",
			address:  "not-a-valid-address",
			amount:   "100",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, constants.TestnetFaucetAdmin)
			},
			expectedErr: ginierr.ErrInvalidAddress("not-a-valid-address"),
		},
		{
			testName: "Error - amount cannot be converted to big int",
			address:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:   "abc",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, constants.TestnetFaucetAdmin)
			},
			expectedErr: ginierr.ErrConvertingAmountToBigInt("abc"),
		},
		{
			testName: "Error - amount is zero",
			address:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:   "0",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, constants.TestnetFaucetAdmin)
			},
			expectedErr: ginierr.ErrInvalidAmount("0"),
		},
		{
			testName: "Error - amount is negative",
			address:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:   "-100",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, constants.TestnetFaucetAdmin)
			},
			expectedErr: ginierr.ErrInvalidAmount("-100"),
		},
		{
			testName: "Success - mint by faucet admin",
			address:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			amount:   "100",
			setupContext: func(ctx *mocks.TransactionContext) {
				SetUserID(ctx, constants.TestnetFaucetAdmin)
				ctx.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
					key := "_" + prefix + "_"
					for _, a := range attrs {
						key += a + "_"
					}
					return key, nil
				}
				ctx.PutStateWithoutKYCStub = func(key string, value []byte) error {
					return nil
				}
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			tt.setupContext(transactionContext)

			err := giniContract.MintByFaucetAdmin(transactionContext, tt.address, tt.amount)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// GINI-2.1 — credit-movement functions must reject calls with a deprecation error.

func TestTransferKWALACredits_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.TransferKWALACredits(transactionContext, "kwl-from-cc", "kwl-to-cc", "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("TransferKWALACredits").Error(), err.Error())
	requireGone(t, err)
}

// GINI-2.2 — mint functions must reject calls with a deprecation error.

func TestMintToKwalaAccountOnBehalfOfKawalaAdmin_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.MintToKwalaAccountOnBehalfOfKawalaAdmin(transactionContext, "klp-someaddress-cc", "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("MintToKwalaAccountOnBehalfOfKawalaAdmin").Error(), err.Error())
	requireGone(t, err)
}

// GINI-2.3 — Kalp<->Kwala bridge functions must reject calls with a deprecation error.

func TestTransferKalpToKwala_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.TransferKalpToKwala(transactionContext, "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("TransferKalpToKwala").Error(), err.Error())
	requireGone(t, err)
}

func TestTransferFromAnyKalpToKwala_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.TransferFromAnyKalpToKwala(transactionContext, "kwl-to-cc", "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("TransferFromAnyKalpToKwala").Error(), err.Error())
	requireGone(t, err)
}

// GINI-2.4 — Kwala admin management functions must reject calls with a deprecation error.

func TestSetKwalaAdmin_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	err := giniContract.SetKwalaAdmin(transactionContext, "klp-someuser-cc")

	require.Error(t, err)
	require.Equal(t, ginierr.ErrDeprecatedFunction("SetKwalaAdmin").Error(), err.Error())
	requireGone(t, err)
}

func TestDeleteKwalaAdmin_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	err := giniContract.DeleteKwalaAdmin(transactionContext, "klp-someuser-cc")

	require.Error(t, err)
	require.Equal(t, ginierr.ErrDeprecatedFunction("DeleteKwalaAdmin").Error(), err.Error())
	requireGone(t, err)
}

// requireGone asserts the deprecation error carries the expected HTTP status (410 Gone)
// by checking the FullError string, since CustomError does not expose the status code directly.
func requireGone(t *testing.T, err error) {
	t.Helper()
	require.Contains(t, err.Error(), "status code:"+strconv.Itoa(http.StatusGone))
}
