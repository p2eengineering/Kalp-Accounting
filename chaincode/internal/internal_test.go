package internal_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/internal"
	"gini-contract/mocks"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

func TestIsSignerKalpFoundation(t *testing.T) {
	mockCtx := new(mocks.TransactionContext)
	mockClientIdentity := new(mocks.ClientIdentity)

	t.Run("should return true when signer is KalpFoundationAddress", func(t *testing.T) {
		clientID := "x509::CN=" + constants.KalpFoundationAddress + ","
		b64ClientID := base64.StdEncoding.EncodeToString([]byte(clientID))
		mockClientIdentity.GetIDReturns(b64ClientID, nil)
		mockCtx.GetClientIdentityReturns(mockClientIdentity)

		result, err := internal.IsSignerKalpFoundation(mockCtx)

		require.NoError(t, err)
		require.True(t, result)
		require.Equal(t, 1, mockClientIdentity.GetIDCallCount())
	})

	t.Run("should return false when signer is not KalpFoundationAddress", func(t *testing.T) {
		// Mock GetUserID to return a different address
		nonFoundationAddress := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
		clientID := fmt.Sprintf("x509::CN=%s::C=US,ST=North Carolina,O=Hyperledger,OU=client,E=user@gmail.com", nonFoundationAddress)
		b64ClientID := base64.StdEncoding.EncodeToString([]byte(clientID))
		mockClientIdentity.GetIDReturns(b64ClientID, nil)
		mockCtx.GetClientIdentityReturns(mockClientIdentity)

		result, _ := internal.IsSignerKalpFoundation(mockCtx)
		require.False(t, result)

	})

	t.Run("should return error when GetUserID fails", func(t *testing.T) {
		expectedError := errors.New("failed to get user ID")
		mockCtx.GetUserIDReturns("", expectedError)

		result, err := internal.IsSignerKalpFoundation(mockCtx)

		require.Error(t, err)
		require.False(t, result)
		require.Contains(t, err.Error(), "failed to get public address")
	})
}

func TestGetCalledContractAddress(t *testing.T) {
	mockCtx := new(mocks.TransactionContext)

	t.Run("Success - Get contract address from valid proposal", func(t *testing.T) {
		// Create valid signed proposal with contract address
		channelHeader := &common.ChannelHeader{
			Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
			ChannelId: "mychannel",
			Timestamp: ptypes.TimestampNow(),
			TxId:      "mock-tx-id",
		}
		channelHeaderBytes, err := proto.Marshal(channelHeader)
		require.NoError(t, err)

		payload := &common.Payload{
			Header: &common.Header{
				ChannelHeader: channelHeaderBytes,
			},
		}
		payloadBytes, err := proto.Marshal(payload)
		require.NoError(t, err)

		proposal := &peer.Proposal{
			Header:  []byte("header"),
			Payload: payloadBytes,
		}
		proposalBytes, err := proto.Marshal(proposal)
		require.NoError(t, err)

		signedProposal := &peer.SignedProposal{
			ProposalBytes: proposalBytes,
			Signature:     []byte("signature"),
		}

		mockCtx.GetSignedProposalReturns(signedProposal, nil)

		_, err = internal.GetCalledContractAddress(mockCtx)

	})

	t.Run("Failure - Nil signed proposal", func(t *testing.T) {
		mockCtx.GetSignedProposalReturns(nil, nil)

		contractAddr, err := internal.GetCalledContractAddress(mockCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not retrieve signed proposal")
		require.Empty(t, contractAddr)
	})

	t.Run("Failure - Invalid proposal bytes", func(t *testing.T) {
		signedProposal := &peer.SignedProposal{
			ProposalBytes: []byte("invalid-proposal-bytes"),
			Signature:     []byte("mock-signature"),
		}
		mockCtx.GetSignedProposalReturns(signedProposal, nil)

		contractAddr, err := internal.GetCalledContractAddress(mockCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error in parsing signed proposal")
		require.Empty(t, contractAddr)
	})

	t.Run("Failure - Invalid payload", func(t *testing.T) {
		proposal := &peer.Proposal{
			Header:  []byte("header"),
			Payload: []byte("invalid-payload"),
		}
		proposalBytes, err := proto.Marshal(proposal)
		require.NoError(t, err)

		signedProposal := &peer.SignedProposal{
			ProposalBytes: proposalBytes,
			Signature:     []byte("signature"),
		}
		mockCtx.GetSignedProposalReturns(signedProposal, nil)

		contractAddr, err := internal.GetCalledContractAddress(mockCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error in parsing payload")
		require.Empty(t, contractAddr)
	})

	t.Run("Failure - Contract address not found", func(t *testing.T) {
		// Create proposal without contract address
		channelHeader := &common.ChannelHeader{
			Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
			ChannelId: "mychannel",
			Timestamp: ptypes.TimestampNow(),
			TxId:      "mock-tx-id",
		}
		channelHeaderBytes, err := proto.Marshal(channelHeader)
		require.NoError(t, err)

		payload := &common.Payload{
			Header: &common.Header{
				ChannelHeader: channelHeaderBytes,
			},
		}
		payloadBytes, err := proto.Marshal(payload)
		require.NoError(t, err)

		proposal := &peer.Proposal{
			Header:  []byte("header"),
			Payload: payloadBytes,
		}
		proposalBytes, err := proto.Marshal(proposal)
		require.NoError(t, err)

		signedProposal := &peer.SignedProposal{
			ProposalBytes: proposalBytes,
			Signature:     []byte("signature"),
		}
		mockCtx.GetSignedProposalReturns(signedProposal, nil)

		contractAddr, err := internal.GetCalledContractAddress(mockCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "contract address not found")
		require.Empty(t, contractAddr)
	})
}

// Helper function to create a mock signed proposal
func createMockSignedProposal(contractAddr string) (*peer.SignedProposal, error) {
	channelHeader := &common.ChannelHeader{
		Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
		ChannelId: "mychannel",
		Timestamp: ptypes.TimestampNow(),
		TxId:      "mock-tx-id",
	}
	channelHeaderBytes, err := proto.Marshal(channelHeader)
	if err != nil {
		return nil, err
	}

	payload := &common.Payload{
		Header: &common.Header{
			ChannelHeader: channelHeaderBytes,
		},
		Data: []byte(contractAddr),
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	proposal := &peer.Proposal{
		Header:  []byte("header"),
		Payload: payloadBytes,
	}
	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		return nil, err
	}

	return &peer.SignedProposal{
		ProposalBytes: proposalBytes,
		Signature:     []byte("signature"),
	}, nil
}
func SetUserID(transactionContext *mocks.TransactionContext, userID string) {
	completeId := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", userID)

	// Base64 encode the complete ID
	b64ID := base64.StdEncoding.EncodeToString([]byte(completeId))

	clientIdentity := &mocks.ClientIdentity{}
	clientIdentity.GetIDReturns(b64ID, nil)
	transactionContext.GetClientIdentityReturns(clientIdentity)
}

// func TestGetGatewayAdminAddress(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		testName       string
// 		setupContext   func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
// 		expectedAdmins []string
// 		expectedError  error
// 	}{
// 		{
// 			testName: "Success - Get single gateway admin",
// 			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
// 				SetUserID(ctx, constants.KalpFoundationAddress)
// 				ctx.GetKYCReturns(true, nil)
// 				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
// 				require.NoError(t, err)
// 				require.True(t, ok)
// 				err = contract.SetUserRoles(ctx, `{"user":"16f8ff33ef05bb24fb9a30fa79e700f57a496184","role":"KalpGatewayAdmin"}`)
// 				require.NoError(t, err)
// 			},
// 			expectedAdmins: []string{"16f8ff33ef05bb24fb9a30fa79e700f57a496184"},
// 			expectedError:  nil,
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt // capture range variable
// 		t.Run(tt.testName, func(t *testing.T) {
// 			t.Parallel()

// 			// Setup
// 			transactionContext := &mocks.TransactionContext{}
// 			worldState := map[string][]byte{}

// 			// Setup stubs
// 			transactionContext.GetStateStub = func(key string) ([]byte, error) {
// 				return worldState[key], nil
// 			}

// 			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
// 				if tt.testName == "Failure - GetStateByPartialCompositeKey error" {
// 					return nil, errors.New("failed to get state")
// 				}

// 				iterator := &mocks.StateQueryIterator{}
// 				var kvs []queryresult.KV

// 				prefix := "_" + objectType + "_"
// 				for key, value := range worldState {
// 					if strings.HasPrefix(key, prefix) {
// 						kvs = append(kvs, queryresult.KV{
// 							Key:   key,
// 							Value: value,
// 						})
// 					}
// 				}

// 				index := 0
// 				iterator.HasNextCalls(func() bool {
// 					return index < len(kvs)
// 				})
// 				iterator.NextCalls(func() (*queryresult.KV, error) {
// 					if tt.testName == "Failure - Iterator.Next error" {
// 						return nil, errors.New("iterator error")
// 					}
// 					if index < len(kvs) {
// 						kv := kvs[index]
// 						index++
// 						return &kv, nil
// 					}
// 					return nil, nil
// 				})
// 				iterator.CloseCalls(func() error {
// 					return nil
// 				})
// 				return iterator, nil
// 			}

// 			// Execute test
// 			admins, err := internal.GetGatewayAdminAddress(transactionContext)

// 			// Assert results
// 			if tt.expectedError != nil {
// 				require.Error(t, err)
// 				require.Contains(t, err.Error(), tt.expectedError.Error())
// 				require.Nil(t, admins)
// 			} else {
// 				require.NoError(t, err)
// 				require.Equal(t, len(tt.expectedAdmins), len(admins))
// 				// Check each admin is in the expected list
// 				for _, expectedAdmin := range tt.expectedAdmins {
// 					require.Contains(t, admins, expectedAdmin)
// 				}
// 			}
// 		})
// 	}
// }
