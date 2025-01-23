package chaincode_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/models"
	"gini-contract/mocks"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
	"github.com/stretchr/testify/require"
)

//go:generate counterfeiter -o mocks/transaction.go -fake-name TransactionContext . transactionContext
type transactionContext interface {
	kalpsdk.TransactionContextInterface
}

//go:generate counterfeiter -o mocks/chaincodestub.go -fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	kalpsdk.ChaincodeStubInterface
}

//go:generate counterfeiter -o mocks/statequeryiterator.go -fake-name StateQueryIterator . stateQueryIterator
type stateQueryIterator interface {
	kalpsdk.StateQueryIteratorInterface
}

//go:generate counterfeiter -o mocks/clientidentity.go -fake-name ClientIdentity . clientIdentity
type clientIdentity interface {
	cid.ClientIdentity
}

func SetUserID(transactionContext *mocks.TransactionContext, userID string) {
	completeId := fmt.Sprintf("x509::CN=%s,O=Organization,L=City,ST=State,C=Country", userID)

	// Base64 encode the complete ID
	b64ID := base64.StdEncoding.EncodeToString([]byte(completeId))

	clientIdentity := &mocks.ClientIdentity{}
	clientIdentity.GetIDReturns(b64ID, nil)
	transactionContext.GetClientIdentityReturns(clientIdentity)
}

func TestInitialize(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}
	// ****************START define helper functions*********************
	worldState := map[string][]byte{}
	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
		key := "_" + s1 + "_"
		for _, s := range s2 {
			key += s + "_"
		}
		return key, nil
	}
	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
		worldState[s] = b
		return nil
	}
	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
		var docType string
		var account string

		// finding doc type
		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
		match := re.FindStringSubmatch(s)

		if len(match) > 1 {
			docType = match[1]
		}

		// finding account
		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
		match = re.FindStringSubmatch(s)

		if len(match) > 1 {
			account = match[1]
		}

		iteratorData := struct {
			index int
			data  []queryresult.KV
		}{}
		for key, val := range worldState {
			if strings.Contains(key, docType) && strings.Contains(key, account) {
				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
			}
		}
		iterator := &mocks.StateQueryIterator{}
		iterator.HasNextStub = func() bool {
			return iteratorData.index < len(iteratorData.data)
		}
		iterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorData.index < len(iteratorData.data) {
				iteratorData.index++
				return &iteratorData.data[iteratorData.index-1], nil
			}
			return nil, fmt.Errorf("iterator out of bounds")
		}
		return iterator, nil
	}
	transactionContext.GetStateStub = func(s string) ([]byte, error) {
		data, found := worldState[s]
		if found {
			return data, nil
		}
		return nil, nil
	}
	transactionContext.DelStateWithoutKYCStub = func(s string) error {
		delete(worldState, s)
		return nil
	}
	transactionContext.GetTxIDStub = func() string {
		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
		length := 10
		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return string(result)
	}
	// ****************END define helper functions*********************

	SetUserID(transactionContext, constants.KalpFoundationAddress)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, constants.KalpFoundationAddress)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)
}

func TestCase1(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	// ****************START define helper functions*********************
	worldState := map[string][]byte{}
	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
		key := "_" + s1 + "_"
		for _, s := range s2 {
			key += s + "_"
		}
		return key, nil
	}
	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
		worldState[s] = b
		return nil
	}
	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
		var docType string
		var account string

		// finding doc type
		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
		match := re.FindStringSubmatch(s)

		if len(match) > 1 {
			docType = match[1]
		}

		// finding account
		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
		match = re.FindStringSubmatch(s)

		if len(match) > 1 {
			account = match[1]
		}

		iteratorData := struct {
			index int
			data  []queryresult.KV
		}{}
		for key, val := range worldState {
			if strings.Contains(key, docType) && strings.Contains(key, account) {
				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
			}
		}
		iterator := &mocks.StateQueryIterator{}
		iterator.HasNextStub = func() bool {
			return iteratorData.index < len(iteratorData.data)
		}
		iterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorData.index < len(iteratorData.data) {
				iteratorData.index++
				return &iteratorData.data[iteratorData.index-1], nil
			}
			return nil, fmt.Errorf("iterator out of bounds")
		}
		return iterator, nil
	}
	transactionContext.GetStateStub = func(s string) ([]byte, error) {
		data, found := worldState[s]
		if found {
			return data, nil
		}
		return nil, nil
	}
	transactionContext.DelStateWithoutKYCStub = func(s string) error {
		delete(worldState, s)
		return nil
	}
	transactionContext.GetStateByPartialCompositeKeyStub = func(prefix string, attributes []string) (kalpsdk.StateQueryIteratorInterface, error) {
		// Define the mock data to simulate the world state
		mockWorldState := map[string][]byte{
			"ID~UserRoleMap_0b87970433b22494faff1cc7a819e71bddc7880c_UserRoleMap": []byte(`{"Id": "0b87970433b22494faff1cc7a819e71bddc7880c", "Role": "KalpGateWayAdminRole"}`),
			"ID~UserRoleMap_user2_UserRoleMap":                                    []byte(`{"Id": "user2", "Role": "KalpGateWayAdminRole"}`),
		}

		// Filter keys that match the prefix and attributes
		filteredData := []queryresult.KV{}
		for key, value := range mockWorldState {
			if strings.HasPrefix(key, prefix) && strings.Contains(key, attributes[0]) && strings.Contains(key, attributes[1]) {
				filteredData = append(filteredData, queryresult.KV{Key: key, Value: value})
			}
		}

		// Mock iterator
		mockIterator := &mocks.StateQueryIterator{}
		iteratorIndex := 0

		// Define HasNext and Next methods
		mockIterator.HasNextStub = func() bool {
			return iteratorIndex < len(filteredData)
		}
		mockIterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorIndex < len(filteredData) {
				item := &filteredData[iteratorIndex]
				iteratorIndex++
				return item, nil
			}
			return nil, fmt.Errorf("no more items")
		}
		mockIterator.CloseStub = func() error {
			// No-op for closing the iterator in this mock
			return nil
		}

		return mockIterator, nil
	}
	transactionContext.GetTxIDStub = func() string {
		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
		length := 10
		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return string(result)
	}

	transactionContext.GetQueryResultStub = func(queryString string) (kalpsdk.StateQueryIteratorInterface, error) {
		// Simulated mock data based on the query string
		mockWorldState := []map[string]interface{}{
			{"amount": "10000", "account": "klp-abc101-cc", "docType": constants.UTXO},
		}

		// Filter the mock world state based on the queryString if necessary.
		// For simplicity, assuming all records match the query string.
		filteredData := make([]*queryresult.KV, len(mockWorldState))
		for i, record := range mockWorldState {
			recordBytes, err := json.Marshal(record)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal record: %v", err)
			}
			filteredData[i] = &queryresult.KV{
				Key:   "klp-abc101-cc",
				Value: recordBytes,
			}
		}

		// Mock iterator
		mockIterator := &mocks.StateQueryIterator{}
		iteratorIndex := 0

		// Define HasNext and Next methods for the iterator
		mockIterator.HasNextStub = func() bool {
			return iteratorIndex < len(filteredData)
		}
		mockIterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorIndex < len(filteredData) {
				item := filteredData[iteratorIndex]
				iteratorIndex++
				return item, nil
			}
			return nil, fmt.Errorf("no more items")
		}
		mockIterator.CloseStub = func() error {
			// No operation needed for closing the mock iterator
			return nil
		}

		return mockIterator, nil
	}

	// ****************END define helper functions*********************

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"

	// Initialize
	SetUserID(transactionContext, admin)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))
	transactionContext.PutStateWithoutKYC(constants.VestingContractKey, []byte("klp-abc100-cc"))
	transactionContext.PutStateWithoutKYC(constants.BridgeContractKey, []byte("klp-abc101-cc"))
	transactionContext.PutStateWithoutKYC("_denyList_0b87970433b22494faff1cc7a819e71bddc7880c_", []byte("false"))
	// Mock the TransactionContext
	transactionContext.GetSignedProposalStub = func() (*peer.SignedProposal, error) {
		mockChannelHeader := "klp-abc101-cc"
		mockHeader := &common.Header{
			ChannelHeader: []byte(mockChannelHeader),
		}
		mockHeaderBytes, err := proto.Marshal(mockHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal header: %v", err)
		}
		mockPayload := &common.Payload{
			Header: mockHeader,
			Data:   []byte("mockData"),
		}
		mockPayloadBytes, err := proto.Marshal(mockPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
		mockProposal := &peer.Proposal{
			Header:  mockHeaderBytes,
			Payload: mockPayloadBytes,
		}
		mockProposalBytes, err := proto.Marshal(mockProposal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal proposal: %v", err)
		}
		mockSignedProposal := &peer.SignedProposal{
			ProposalBytes: mockProposalBytes,
		}

		return mockSignedProposal, nil
	}

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func TestCase2(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}
	giniContract.Contract.Name = "klp-abc101-cc"
	giniContract.Logger = kalpsdk.NewLogger()
	// _, err := kalpsdk.NewChaincode(&chaincode.SmartContract{Contract: contract})
	// ****************START define helper functions*********************
	worldState := map[string][]byte{}
	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
		key := "_" + s1 + "_"
		for _, s := range s2 {
			key += s + "_"
		}
		return key, nil
	}
	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
		worldState[s] = b
		return nil
	}
	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
		var docType string
		var account string

		// finding doc type
		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
		match := re.FindStringSubmatch(s)

		if len(match) > 1 {
			docType = match[1]
		}

		// finding account
		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
		match = re.FindStringSubmatch(s)

		if len(match) > 1 {
			account = match[1]
		}

		iteratorData := struct {
			index int
			data  []queryresult.KV
		}{}
		for key, val := range worldState {
			if strings.Contains(key, docType) && strings.Contains(key, account) {
				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
			}
		}
		iterator := &mocks.StateQueryIterator{}
		iterator.HasNextStub = func() bool {
			return iteratorData.index < len(iteratorData.data)
		}
		iterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorData.index < len(iteratorData.data) {
				iteratorData.index++
				return &iteratorData.data[iteratorData.index-1], nil
			}
			return nil, fmt.Errorf("iterator out of bounds")
		}
		return iterator, nil
	}
	transactionContext.GetStateStub = func(s string) ([]byte, error) {
		data, found := worldState[s]
		if found {
			return data, nil
		}
		return nil, nil
	}
	transactionContext.DelStateWithoutKYCStub = func(s string) error {
		delete(worldState, s)
		return nil
	}
	transactionContext.GetStateByPartialCompositeKeyStub = func(prefix string, attributes []string) (kalpsdk.StateQueryIteratorInterface, error) {
		// Define the mock data to simulate the world state
		mockWorldState := map[string][]byte{
			"ID~UserRoleMap_0b87970433b22494faff1cc7a819e71bddc7880c_UserRoleMap": []byte(`{"Id": "0b87970433b22494faff1cc7a819e71bddc7880c", "Role": "KalpGateWayAdminRole"}`),
			"ID~UserRoleMap_user2_UserRoleMap":                                    []byte(`{"Id": "user2", "Role": "KalpGateWayAdminRole"}`),
		}

		// Filter keys that match the prefix and attributes
		filteredData := []queryresult.KV{}
		for key, value := range mockWorldState {
			if strings.HasPrefix(key, prefix) && strings.Contains(key, attributes[0]) && strings.Contains(key, attributes[1]) {
				filteredData = append(filteredData, queryresult.KV{Key: key, Value: value})
			}
		}

		// Mock iterator
		mockIterator := &mocks.StateQueryIterator{}
		iteratorIndex := 0

		// Define HasNext and Next methods
		mockIterator.HasNextStub = func() bool {
			return iteratorIndex < len(filteredData)
		}
		mockIterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorIndex < len(filteredData) {
				item := &filteredData[iteratorIndex]
				iteratorIndex++
				return item, nil
			}
			return nil, fmt.Errorf("no more items")
		}
		mockIterator.CloseStub = func() error {
			// No-op for closing the iterator in this mock
			return nil
		}

		return mockIterator, nil
	}
	transactionContext.GetTxIDStub = func() string {
		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
		length := 10
		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return string(result)
	}
	transactionContext.GetQueryResultStub = func(queryString string) (kalpsdk.StateQueryIteratorInterface, error) {
		// Simulated mock data based on the query string
		mockWorldState := []map[string]interface{}{
			{"amount": "10000", "account": "klp-abc101-cc", "docType": constants.UTXO},
		}

		// Filter the mock world state based on the queryString if necessary.
		// For simplicity, assuming all records match the query string.
		filteredData := make([]*queryresult.KV, len(mockWorldState))
		for i, record := range mockWorldState {
			recordBytes, err := json.Marshal(record)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal record: %v", err)
			}
			filteredData[i] = &queryresult.KV{
				Key:   "klp-abc101-cc",
				Value: recordBytes,
			}
		}

		// Mock iterator
		mockIterator := &mocks.StateQueryIterator{}
		iteratorIndex := 0

		// Define HasNext and Next methods for the iterator
		mockIterator.HasNextStub = func() bool {
			return iteratorIndex < len(filteredData)
		}
		mockIterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorIndex < len(filteredData) {
				item := filteredData[iteratorIndex]
				iteratorIndex++
				return item, nil
			}
			return nil, fmt.Errorf("no more items")
		}
		mockIterator.CloseStub = func() error {
			// No operation needed for closing the mock iterator
			return nil
		}

		return mockIterator, nil
	}

	// ****************END define helper functions*********************

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	SetUserID(transactionContext, admin)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)
	approval := models.Allow{
		Owner:   "abc",
		Amount:  "10000000000",
		DocType: "abcd",
		Spender: "spen",
	}
	approvalBytes, err := json.Marshal(approval)
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))
	transactionContext.PutStateWithoutKYC(constants.VestingContractKey, []byte("klp-abc100-cc"))
	transactionContext.PutStateWithoutKYC(constants.BridgeContractKey, []byte("klp-abc101-cc"))
	transactionContext.PutStateWithoutKYC("_denyList_0b87970433b22494faff1cc7a819e71bddc7880c_", []byte("false"))
	transactionContext.PutStateWithoutKYC("_Approval_35581086b9b262a62f5d2d1603d901d9375777b8_klp-abc101-cc_", approvalBytes)
	// Mock the TransactionContext
	transactionContext.GetSignedProposalStub = func() (*peer.SignedProposal, error) {
		mockChannelHeader := "klp-abc101-cc"
		mockHeader := &common.Header{
			ChannelHeader: []byte(mockChannelHeader),
		}
		mockHeaderBytes, err := proto.Marshal(mockHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal header: %v", err)
		}
		mockPayload := &common.Payload{
			Header: mockHeader,
			Data:   []byte("mockData"),
		}
		mockPayloadBytes, err := proto.Marshal(mockPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
		mockProposal := &peer.Proposal{
			Header:  mockHeaderBytes,
			Payload: mockPayloadBytes,
		}
		mockProposalBytes, err := proto.Marshal(mockProposal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal proposal: %v", err)
		}
		mockSignedProposal := &peer.SignedProposal{
			ProposalBytes: mockProposalBytes,
		}

		return mockSignedProposal, nil
	}

	// Approve: userG approves userM to spend 100 units
	SetUserID(transactionContext, userG)
	ok, err = giniContract.Approve(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	SetUserID(transactionContext, userM)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

}

// func TestCase2(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
// 	require.ErrorContains(t, err, "insufficient allowance")
// 	require.Equal(t, false, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "1000", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3000", balance)

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "100", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("4100", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase3(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, userM, "80")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "80")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "999", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3080", balance) // 3000 + 80 = 3080

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "20", balance) // 2000 - 80 = 1920

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("4099", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase4(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "80")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, userM, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
// 	require.ErrorContains(t, err, "insufficient balance in sender's account for amount")
// 	require.Equal(t, false, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "1000", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3000", balance)

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "80", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("4080", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase5(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1111 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1111"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, userM, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
// 	require.ErrorContains(t, err, "insufficient balance in signer's account for gas fees")
// 	require.Equal(t, false, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "1000", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3000", balance)

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "100", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("4100", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase7(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1000 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1000"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "500")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, admin, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, admin)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "1000", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "600", balance) // 500 + 100 = 600

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "0", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("1600", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase8(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, userM, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "999", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3000", balance)

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "0", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("3999", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase9(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
// 	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 1 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userG, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userG approves userM to spend 100 units
// 	SetUserID(transactionContext, userG)
// 	ok, err = giniContract.Approve(transactionContext, userM, "200")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 100 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom: userM transfers 200 units from userG to userC
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "200")
// 	require.ErrorContains(t, err, "insufficient balance in sender's account for amount")
// 	require.Equal(t, false, ok)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "999", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "3000", balance)

// 	// Check userG balance (should reflect the deduction of 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, userG)
// 	require.NoError(t, err)
// 	require.Equal(t, "0", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("3999", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

// func TestCase10(t *testing.T) {
// 	t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
// 	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 10 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("10"))

// 	// Admin transfers everything except 100 wei
// 	ok, err = giniContract.Transfer(transactionContext, userC, "11199999999999999999999800")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	ok, err = giniContract.Transfer(transactionContext, userM, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve
// 	// admin approves userM 100
// 	ok, err = giniContract.Approve(transactionContext, userM, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// TransferFrom
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.TransferFrom(transactionContext, admin, userC, "100")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Checking balances for admin, userM, userC
// 	// Test case 3: Check Balance

// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "90", balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, userC)
// 	require.NoError(t, err)
// 	require.Equal(t, "11199999999999999999999900", balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, "10", balance)
// }

// func TestCase11(t *testing.T) {
// 	// t.Parallel()
// 	transactionContext := &mocks.TransactionContext{}
// 	giniContract := chaincode.SmartContract{}

// 	// ****************START define helper functions*********************
// 	worldState := map[string][]byte{}
// 	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
// 		key := "_" + s1 + "_"
// 		for _, s := range s2 {
// 			key += s + "_"
// 		}
// 		return key, nil
// 	}
// 	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
// 		worldState[s] = b
// 		return nil
// 	}
// 	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
// 		var docType string
// 		var account string

// 		// finding doc type
// 		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
// 		match := re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			docType = match[1]
// 		}

// 		// finding account
// 		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
// 		match = re.FindStringSubmatch(s)

// 		if len(match) > 1 {
// 			account = match[1]
// 		}

// 		iteratorData := struct {
// 			index int
// 			data  []queryresult.KV
// 		}{}
// 		for key, val := range worldState {
// 			if strings.Contains(key, docType) && strings.Contains(key, account) {
// 				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
// 			}
// 		}
// 		iterator := &mocks.StateQueryIterator{}
// 		iterator.HasNextStub = func() bool {
// 			return iteratorData.index < len(iteratorData.data)
// 		}
// 		iterator.NextStub = func() (*queryresult.KV, error) {
// 			if iteratorData.index < len(iteratorData.data) {
// 				iteratorData.index++
// 				return &iteratorData.data[iteratorData.index-1], nil
// 			}
// 			return nil, fmt.Errorf("iterator out of bounds")
// 		}
// 		return iterator, nil
// 	}
// 	transactionContext.GetStateStub = func(s string) ([]byte, error) {
// 		data, found := worldState[s]
// 		if found {
// 			return data, nil
// 		}
// 		return nil, nil
// 	}
// 	transactionContext.DelStateWithoutKYCStub = func(s string) error {
// 		delete(worldState, s)
// 		return nil
// 	}
// 	transactionContext.GetTxIDStub = func() string {
// 		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
// 		length := 10
// 		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
// 		result := make([]byte, length)
// 		for i := range result {
// 			result[i] = charset[rand.Intn(len(charset))]
// 		}
// 		return string(result)
// 	}
// 	transactionContext.InvokeChaincodeStub = func(s1 string, b [][]byte, s2 string) response.Response {
// 		if s1 == constants.InitialBridgeContractAddress && string(b[0]) == "BridgeToken" {
// 			signer, _ := transactionContext.GetUserID()

// 			giniContract.TransferFrom(transactionContext, signer, constants.InitialBridgeContractAddress, string(b[1]))
// 			return response.Response{
// 				Response: peer.Response{
// 					Status:  http.StatusOK,
// 					Payload: []byte("true"),
// 				},
// 			}
// 		}
// 		return response.Response{
// 			Response: peer.Response{
// 				Status:  http.StatusBadRequest,
// 				Payload: []byte("false"),
// 			},
// 		}

// 	}

// 	// ****************END define helper functions*********************

// 	// define users
// 	admin := constants.KalpFoundationAddress
// 	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"

// 	// Initialize
// 	SetUserID(transactionContext, admin)
// 	transactionContext.GetKYCReturns(true, nil)

// 	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	balance, err := giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialFoundationBalance, balance)

// 	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
// 	require.NoError(t, err)
// 	require.Equal(t, constants.InitialVestingContractBalance, balance)

// 	// Updating gas fess to 10 Wei
// 	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("10"))

// 	// Admin recharges userM, userG, and userC

// 	ok, err = giniContract.Transfer(transactionContext, userM, "110") // 100 + 10 gas fees

// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	// Approve: userM approves bridge contract to spend 100 units
// 	SetUserID(transactionContext, userM)
// 	ok, err = giniContract.Approve(transactionContext, constants.InitialBridgeContractAddress, "100")
// 	require.NoError(t, err)
// 	require.Equal(t, true, ok)

// 	//
// 	output := transactionContext.InvokeChaincode(constants.InitialBridgeContractAddress, [][]byte{[]byte("BridgeToken"), []byte("100")}, "kalptantra")
// 	b, _ := strconv.ParseBool(string(output.Payload))
// 	require.Equal(t, true, b)

// 	// Verify balances after transfer
// 	// Check userM balance
// 	balance, err = giniContract.BalanceOf(transactionContext, userM)
// 	require.NoError(t, err)
// 	require.Equal(t, "0", balance)

// 	// Check userC balance (should reflect the additional 100 units)
// 	balance, err = giniContract.BalanceOf(transactionContext, constants.InitialBridgeContractAddress)
// 	require.NoError(t, err)
// 	require.Equal(t, "100", balance)

// 	// Check admin balance (unchanged in this scenario)
// 	balance, err = giniContract.BalanceOf(transactionContext, admin)
// 	require.NoError(t, err)
// 	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
// 	userBalanceSum, _ := new(big.Int).SetString("100", 10)
// 	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
// }

func SetWorldState(transactionContext *mocks.TransactionContext) {
	worldState := map[string][]byte{}
	transactionContext.CreateCompositeKeyStub = func(s1 string, s2 []string) (string, error) {
		key := "_" + s1 + "_"
		for _, s := range s2 {
			key += s + "_"
		}
		return key, nil
	}
	transactionContext.PutStateWithoutKYCStub = func(s string, b []byte) error {
		worldState[s] = b
		return nil
	}
	transactionContext.GetQueryResultStub = func(s string) (kalpsdk.StateQueryIteratorInterface, error) {
		var docType string
		var account string

		// finding doc type
		re := regexp.MustCompile(`"docType"\s*:\s*"([^"]+)"`)
		match := re.FindStringSubmatch(s)

		if len(match) > 1 {
			docType = match[1]
		}

		// finding account
		re = regexp.MustCompile(`"account"\s*:\s*"([^"]+)"`)
		match = re.FindStringSubmatch(s)

		if len(match) > 1 {
			account = match[1]
		}

		iteratorData := struct {
			index int
			data  []queryresult.KV
		}{}
		for key, val := range worldState {
			if strings.Contains(key, docType) && strings.Contains(key, account) {
				iteratorData.data = append(iteratorData.data, queryresult.KV{Key: key, Value: val})
			}
		}
		iterator := &mocks.StateQueryIterator{}
		iterator.HasNextStub = func() bool {
			return iteratorData.index < len(iteratorData.data)
		}
		iterator.NextStub = func() (*queryresult.KV, error) {
			if iteratorData.index < len(iteratorData.data) {
				iteratorData.index++
				return &iteratorData.data[iteratorData.index-1], nil
			}
			return nil, fmt.Errorf("iterator out of bounds")
		}
		return iterator, nil
	}
	transactionContext.GetStateStub = func(s string) ([]byte, error) {
		data, found := worldState[s]
		if found {
			return data, nil
		}
		return nil, nil
	}
	transactionContext.DelStateWithoutKYCStub = func(s string) error {
		delete(worldState, s)
		return nil
	}
	transactionContext.GetTxIDStub = func() string {
		const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
		length := 10
		rand.Seed(time.Now().UnixNano()) // Seed the random number generator
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return string(result)
	}
}

/*
	func TestTransfer_SenderIsContract_Invalid_TC_1(t *testing.T) {
		t.Parallel()

		// Arrange - Setup the transaction context, contract, and world state
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		admin := constants.KalpFoundationAddress

		SetWorldState(transactionContext)

		transactionContext.GetUserIDReturns(admin, nil)
		transactionContext.GetKYCReturns(true, nil)
		ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

		gas, err := giniContract.GetGasFees(transactionContext)
		require.NoError(t, err)
		require.Equal(t, "1000000000000000", gas)

		sender := "klp-Contract1-cc"
		recipient := "User2"
		amount := "100"

		// Setup transaction context to return Contract1 as the signer
		transactionContext.GetUserIDReturns(sender, nil)

		// Act - Attempt the transfer
		ok, err = giniContract.Transfer(transactionContext, recipient, amount)

		// Assert - Expect an error and no successful transfer
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "signer cannot be a contract",
			})
	}

	func TestTransfer_InvalidSenderAddress(t *testing.T) {
		t.Parallel()
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}

		SetWorldState(transactionContext)

		// Arrange
		sender := "Invalid"
		recipient := "User2"
		amount := "100"
		admin := constants.KalpFoundationAddress

		transactionContext.GetUserIDReturns(admin, nil)
		transactionContext.GetKYCReturns(true, nil)
		ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

		// Setup transaction context to return an invalid sender address
		transactionContext.GetUserIDReturns(sender, nil)
		gas, err := giniContract.GetGasFees(transactionContext)
		require.NoError(t, err)
		require.Equal(t, "1000000000000000", gas)

		// Act
		ok, err = giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid sender address",
			})
	}

	func TestTransfer_InvalidRecipientAddress(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "User1"
		recipient := "Invalid"
		amount := "100"

		// Setup transaction context to return a valid sender
		transactionContext.GetUserIDReturns(sender, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid recipient address",
			})
	}

	func TestTransfer_SenderIsContract_Invalid_TC2(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "Contract1"
		recipient := "User2"
		amount := "100"

		// Setup transaction context to return Contract1 as the signer
		transactionContext.GetUserIDReturns(sender, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "signer cannot be a contract",
			})
	}

	func TestTransfer_SenderAndRecipientAreContracts(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "Contract1"
		recipient := "Contract2"
		amount := "100"

		// Setup transaction context to return Contract1 as the signer
		transactionContext.GetUserIDReturns(sender, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "both sender and recipient cannot be contracts",
			})
	}

	func TestTransfer_AmountIsNegative(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "User1"
		recipient := "User2"
		amount := "-100"

		// Setup transaction context to return User1 as the signer
		transactionContext.GetUserIDReturns(sender, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusBadRequest,
				Message:    "amount cannot be negative",
			})
	}

	func TestTransfer_SignerIsNotKYCed(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "User1"
		recipient := "User2"
		amount := "100"

		// Setup transaction context to simulate non-KYCed user
		transactionContext.GetUserIDReturns(sender, nil)
		transactionContext.GetKYCReturns(false, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusForbidden,
				Message:    "signer is not KYCed",
			})
	}

	func TestTransfer_SenderIsNotKYCed(t *testing.T) {
		t.Parallel()

		// Arrange
		transactionContext := &mocks.TransactionContext{}
		giniContract := chaincode.SmartContract{}
		sender := "User1"
		recipient := "User2"
		amount := "100"

		// Setup transaction context to simulate non-KYCed sender
		transactionContext.GetUserIDReturns(sender, nil)
		transactionContext.GetKYCReturns(false, nil)

		// Act
		ok, err := giniContract.Transfer(transactionContext, recipient, amount)

		// Assert
		require.Error(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, err,
			&ginierr.CustomError{
				StatusCode: http.StatusForbidden,
				Message:    "sender is not KYCed",
			})
	}
*/
func TestName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		worldState    map[string][]byte
		getStateErr   error
		expectedName  string
		expectedError error
	}{
		{
			testName: "Success - Get token name",
			worldState: map[string][]byte{
				constants.NameKey: []byte("GINI Token"),
			},
			getStateErr:   nil,
			expectedName:  "GINI Token",
			expectedError: nil,
		},
		{
			testName:      "Failure - GetState error",
			worldState:    map[string][]byte{},
			getStateErr:   errors.New("get state error"),
			expectedName:  "",
			expectedError: ginierr.ErrFailedToGetKey(constants.NameKey),
		},
		{
			testName:      "Failure - Name not initialized",
			worldState:    map[string][]byte{},
			getStateErr:   nil,
			expectedName:  "",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := chaincode.SmartContract{}

			// Setup GetState stub
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.getStateErr != nil {
					return nil, tt.getStateErr
				}
				return tt.worldState[key], nil
			}

			// Execute test
			name, err := giniContract.Name(transactionContext)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedName, name)
			}
		})
	}
}

func TestDecimals(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	// Execute test
	decimals := giniContract.Decimals(transactionContext)

	// Assert
	require.Equal(t, uint8(18), decimals, "Decimals should return 18")
}

func TestGetGasFees(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		worldState    map[string][]byte
		getStateErr   error
		expectedFees  string
		expectedError error
	}{
		{
			testName: "Success - Get gas fees",
			worldState: map[string][]byte{
				constants.GasFeesKey: []byte("1000000000000000"),
			},
			getStateErr:   nil,
			expectedFees:  "1000000000000000",
			expectedError: nil,
		},
		{
			testName:      "Failure - GetState error",
			worldState:    map[string][]byte{},
			getStateErr:   errors.New("failed to get state"),
			expectedFees:  "",
			expectedError: fmt.Errorf("failed to get Gas Fee: failed to get state"),
		},
		{
			testName:      "Failure - Gas fee not set",
			worldState:    map[string][]byte{},
			getStateErr:   nil,
			expectedFees:  "",
			expectedError: fmt.Errorf("gas fee not set"),
		},
		{
			testName: "Success - Zero gas fees",
			worldState: map[string][]byte{
				constants.GasFeesKey: []byte("0"),
			},
			getStateErr:   nil,
			expectedFees:  "0",
			expectedError: nil,
		},
		{
			testName: "Success - Large gas fees",
			worldState: map[string][]byte{
				constants.GasFeesKey: []byte("999999999999999999999999999999"),
			},
			getStateErr:   nil,
			expectedFees:  "999999999999999999999999999999",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := chaincode.SmartContract{}

			// Setup GetState stub
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.getStateErr != nil {
					return nil, tt.getStateErr
				}
				return tt.worldState[key], nil
			}

			// Execute test
			fees, err := giniContract.GetGasFees(transactionContext)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
				require.Equal(t, tt.expectedFees, fees)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedFees, fees)
			}
		})
	}
}

// TestGetGasFees_AfterInitialization tests the gas fees value after contract initialization
func TestGetGasFees_AfterInitialization(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}
	worldState := map[string][]byte{}

	// Setup stubs
	transactionContext.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
		worldState[key] = value
		return nil
	}
	transactionContext.GetKYCReturns(true, nil)

	// Initialize contract
	SetUserID(transactionContext, constants.KalpFoundationAddress)
	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.True(t, ok)

	// Get gas fees after initialization
	fees, err := giniContract.GetGasFees(transactionContext)
	require.NoError(t, err)
	require.Equal(t, constants.InitialGasFees, fees,
		"Gas fees should match initial value after initialization")
}

// TestGetGasFees_AfterSetGasFees tests getting gas fees after setting a new value
func TestGetGasFees_AfterSetGasFees(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}
	worldState := map[string][]byte{}

	// Setup stubs
	transactionContext.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
		worldState[key] = value
		return nil
	}
	transactionContext.GetKYCReturns(true, nil)

	// Initialize contract
	SetUserID(transactionContext, constants.KalpFoundationAddress)
	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.True(t, ok)

	// Set new gas fees
	newGasFees := "2000000000000000"
	err = giniContract.SetGasFees(transactionContext, newGasFees)
	require.NoError(t, err)

	// Get gas fees after setting new value
	fees, err := giniContract.GetGasFees(transactionContext)
	require.NoError(t, err)
	require.Equal(t, newGasFees, fees,
		"Gas fees should match newly set value")
}

func TestSetGasFees(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		gasFees       string
		expectedError error
	}{
		{
			testName: "Success - Set gas fees by foundation",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			gasFees:       "2000000000000000",
			expectedError: nil,
		},
		{
			testName: "Failure - Non-foundation user attempts to set gas fees",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// First initialize with foundation
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Then switch to non-foundation user
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			gasFees:       "2000000000000000",
			expectedError: ginierr.New("Only Kalp Foundation can set the gas fees", http.StatusUnauthorized),
		},
		{
			testName: "Failure - PutState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				ctx.PutStateWithoutKYCReturns(errors.New("failed to put state"))
			},
			gasFees:       "2000000000000000",
			expectedError: ginierr.ErrFailedToPutState(errors.New("failed to put state")),
		},
		{
			testName: "Success - Set zero gas fees",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			gasFees:       "0",
			expectedError: nil,
		},
		{
			testName: "Success - Set very large gas fees",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			gasFees:       "999999999999999999999999999999",
			expectedError: nil,
		},
		{
			testName: "Failure - GetUserID error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns("", errors.New("failed to get ID"))
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			gasFees:       "2000000000000000",
			expectedError: ginierr.ErrFailedToGetPublicAddress,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				var kvs []queryresult.KV
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				index := 0
				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.SetGasFees(transactionContext, tt.gasFees)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)

				// Verify gas fees were updated correctly
				currentFees, err := giniContract.GetGasFees(transactionContext)
				require.NoError(t, err)
				require.Equal(t, tt.gasFees, currentFees,
					"Gas fees should have been updated to new value")
			}
		})
	}
}

// TestSetGasFees_FullLifecycle tests the complete lifecycle of setting gas fees
func TestSetGasFees_FullLifecycle(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}
	worldState := map[string][]byte{}

	// Setup stubs
	transactionContext.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
		worldState[key] = value
		return nil
	}
	transactionContext.GetKYCReturns(true, nil)

	// Initialize contract
	SetUserID(transactionContext, constants.KalpFoundationAddress)
	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.True(t, ok)

	// Verify initial gas fees
	initialFees, err := giniContract.GetGasFees(transactionContext)
	require.NoError(t, err)
	require.Equal(t, constants.InitialGasFees, initialFees)

	// Set new gas fees
	newGasFees := "2000000000000000"
	err = giniContract.SetGasFees(transactionContext, newGasFees)
	require.NoError(t, err)

	// Verify updated gas fees
	currentFees, err := giniContract.GetGasFees(transactionContext)
	require.NoError(t, err)
	require.Equal(t, newGasFees, currentFees)

	// Try to set gas fees with non-foundation user
	SetUserID(transactionContext, "0b87970433b22494faff1cc7a819e71bddc788bv")
	err = giniContract.SetGasFees(transactionContext, "3000000000000000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get public address, status code:500")

	// Verify gas fees remained unchanged
	currentFees, err = giniContract.GetGasFees(transactionContext)
	require.NoError(t, err)
	require.Equal(t, newGasFees, currentFees)
}

func TestTotalSupply(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := &chaincode.SmartContract{}

	// Execute test
	totalSupply, err := giniContract.TotalSupply(transactionContext)

	// Assert
	require.NoError(t, err)
	require.Equal(t, constants.TotalSupply, totalSupply,
		"Total supply should match the constant value")
}
func TestSetUserRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		roleData      string
		expectedError error
	}{
		{
			testName: "Success - Set Gateway Admin role",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			roleData:      `{"user":"16f8ff33ef05bb24fb9a30fa79e700f57a496184","role":"KalpGatewayAdmin"}`,
			expectedError: nil,
		},
		{
			testName: "Failure - Non-foundation user",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				ctx.GetKYCReturns(true, nil)
			},
			roleData:      `{"user":"2da4c4908a393a387b728206b18388bc529fa8d7","role":"KalpGatewayAdmin"}`,
			expectedError: ginierr.New("Only Kalp Foundation can set the roles", http.StatusUnauthorized),
		},
		{
			testName: "Failure - Empty user ID",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
			},
			roleData:      `{"user":"","role":"KalpGatewayAdmin"}`,
			expectedError: fmt.Errorf("user Id can not be null"),
		},
		{
			testName: "Failure - Empty role name",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
			},
			roleData:      `{"user":"2da4c4908a393a387b728206b18388bc529fa8d7","role":""}`,
			expectedError: fmt.Errorf("role can not be null"),
		},
		{
			testName: "Failure - Invalid role data JSON",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			roleData:      `invalid-json`,
			expectedError: fmt.Errorf("failed to parse data"),
		},
		{
			testName: "Failure - Missing user ID",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			roleData:      `{"user":"","role":"KalpGatewayAdmin"}`,
			expectedError: fmt.Errorf("user Id can not be null"),
		},
		{
			testName: "Failure - Invalid user address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			roleData:      `{"user":"invalid-address","role":"KalpGatewayAdmin"}`,
			expectedError: ginierr.ErrInvalidUserAddress("invalid-address"),
		},
	}
	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.SetUserRoles(transactionContext, tt.roleData)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)

				// Verify role was set correctly
				var userRole models.UserRole
				json.Unmarshal([]byte(tt.roleData), &userRole)
				roleKey, _ := transactionContext.CreateCompositeKey(constants.UserRolePrefix,
					[]string{userRole.Id, constants.UserRoleMap})

				storedRoleBytes := worldState[roleKey]
				require.NotNil(t, storedRoleBytes)

				var storedRole models.UserRole
				json.Unmarshal(storedRoleBytes, &storedRole)
				require.Equal(t, userRole.Id, storedRole.Id)
				require.Equal(t, userRole.Role, storedRole.Role)
			}
		})
	}
}

func TestDeleteUserRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		userID        string
		expectedError error
	}{
		{
			testName: "Success - Delete user role",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Set up a role to delete
				err = contract.SetUserRoles(ctx, `{"user":"16f8ff33ef05bb24fb9a30fa79e700f57a496184","role":"KalpGatewayAdmin"}`)
				require.NoError(t, err)
			},
			userID:        "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: nil,
		},
		{
			testName: "Failure - Non-foundation user",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				ctx.GetKYCReturns(true, nil)
			},
			userID:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedError: ginierr.New("Only Kalp Foundation can set the roles", http.StatusUnauthorized),
		},
		{
			testName: "Failure - Role not found",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			userID:        "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: ginierr.NewInternalError(nil, "user role not found for userID 16f8ff33ef05bb24fb9a30fa79e700f57a496184", http.StatusNotFound),
		},
		{
			testName: "Failure - Cannot delete foundation role",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			userID:        constants.KalpFoundationAddress,
			expectedError: fmt.Errorf("foundation role cannot be deleted"),
		},
	}
	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
				worldState[key] = value
				return nil
			}
			transactionContext.DelStateWithoutKYCStub = func(key string) error {
				delete(worldState, key)
				return nil
			}
			transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
				key := "_" + prefix + "_"
				for _, attr := range attrs {
					key += attr + "_"
				}
				return key, nil
			}
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.DeleteUserRoles(transactionContext, tt.userID)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err)

				// Verify role was deleted
				roleKey, _ := transactionContext.CreateCompositeKey(constants.UserRolePrefix,
					[]string{tt.userID, constants.UserRoleMap})

				_, exists := worldState[roleKey]
				require.False(t, exists, "Role should have been deleted")
			}
		})
	}
}

func TestAllow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		address       string
		expectedError error
	}{
		{
			testName: "Success - Allow previously denied address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Deny the address first
				err = contract.Deny(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				require.NoError(t, err)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: nil,
		},
		{
			testName: "Failure - Address not previously denied",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: ginierr.ErrNotDenied("16f8ff33ef05bb24fb9a30fa79e700f57a496184"),
		},
		{
			testName: "Success - Allow after multiple operations",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Deny -> Allow -> Deny sequence
				err = contract.Deny(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				require.NoError(t, err)
				err = contract.Allow(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				require.NoError(t, err)
				err = contract.Deny(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				require.NoError(t, err)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
				worldState[key] = value
				return nil
			}
			transactionContext.DelStateWithoutKYCStub = func(key string) error {
				delete(worldState, key)
				return nil
			}
			transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
				key := "_" + prefix + "_"
				for _, attr := range attrs {
					key += attr + "_"
				}
				return key, nil
			}
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.Allow(transactionContext, tt.address)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())

				if tt.testName != "Failure - Address not previously denied" {
					// Verify denied status hasn't changed for error cases
					deniedKey, _ := transactionContext.CreateCompositeKey(constants.Denied, []string{tt.address})
					deniedStatus := worldState[deniedKey]
					require.NotNil(t, deniedStatus, "Denied status should not have changed")
				}
			} else {
				require.NoError(t, err)

				// Verify address is no longer denied
				deniedKey, _ := transactionContext.CreateCompositeKey(constants.Denied, []string{tt.address})
				deniedStatus := worldState[deniedKey]
				require.Nil(t, deniedStatus, "Address should no longer be denied")
			}
		})
	}
}

// TestAllow_FullLifecycle tests the complete lifecycle of denying and allowing an address
func TestAllow_FullLifecycle(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := &chaincode.SmartContract{}
	worldState := map[string][]byte{}

	// Setup stubs
	transactionContext.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
		worldState[key] = value
		return nil
	}
	transactionContext.DelStateWithoutKYCStub = func(key string) error {
		delete(worldState, key)
		return nil
	}
	transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
		key := "_" + prefix + "_"
		for _, attr := range attrs {
			key += attr + "_"
		}
		return key, nil
	}

	// Setup identity and KYC
	SetUserID(transactionContext, constants.KalpFoundationAddress)
	transactionContext.GetKYCReturns(true, nil)

	// Initialize contract
	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627169646775-cc")
	require.NoError(t, err)
	require.True(t, ok)

	testAddress := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"

	// Test full lifecycle
	// 1. Verify initial state
	deniedKey, _ := transactionContext.CreateCompositeKey(constants.Denied, []string{testAddress})
	require.Nil(t, worldState[deniedKey], "Address should not be denied initially")

	// 2. Deny the address
	err = giniContract.Deny(transactionContext, testAddress)
	require.NoError(t, err)

	// 3. Allow the address
	err = giniContract.Allow(transactionContext, testAddress)
	require.NoError(t, err)
	require.Nil(t, worldState[deniedKey], "Address should not be denied after allowing")

	// 4. Try to allow again (should fail)
	err = giniContract.Allow(transactionContext, testAddress)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NotDenied")

	// 5. Deny again
	err = giniContract.Deny(transactionContext, testAddress)
	require.NoError(t, err)

	// 6. Allow again
	err = giniContract.Allow(transactionContext, testAddress)
	require.NoError(t, err)
	require.Nil(t, worldState[deniedKey], "Address should not be denied after allowing again")
}

func TestDeny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		address       string
		expectedError error
	}{
		{
			testName: "Success - Deny new address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: nil,
		},
		{
			testName: "Failure - Non-foundation user attempts to deny",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// Initialize with foundation first
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Switch to non-foundation user
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			address:       "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedError: ginierr.New("Only Kalp Foundation can Deny", http.StatusUnauthorized),
		},
		{
			testName: "Failure - Attempt to deny foundation address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			address:       constants.KalpFoundationAddress,
			expectedError: ginierr.New("admin cannot be denied", http.StatusBadRequest),
		},
		{
			testName: "Failure - Address already denied",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Deny the address first
				err = contract.Deny(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				require.NoError(t, err)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: ginierr.ErrAlreadyDenied("16f8ff33ef05bb24fb9a30fa79e700f57a496184"),
		},
		{
			testName: "Failure - GetUserID error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns("", errors.New("failed to get ID"))
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			address:       "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			expectedError: ginierr.ErrFailedToGetPublicAddress,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
				worldState[key] = value
				return nil
			}
			transactionContext.DelStateWithoutKYCStub = func(key string) error {
				delete(worldState, key)
				return nil
			}
			transactionContext.CreateCompositeKeyStub = func(prefix string, attrs []string) (string, error) {
				key := "_" + prefix + "_"
				for _, attr := range attrs {
					key += attr + "_"
				}
				return key, nil
			}
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.Deny(transactionContext, tt.address)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())

				if tt.testName != "Success - Deny new address" {
					// Verify denied status hasn't changed for error cases
					deniedKey, _ := transactionContext.CreateCompositeKey(constants.Denied, []string{tt.address})
					if worldState[deniedKey] != nil {
						require.Equal(t, []byte("true"), worldState[deniedKey],
							"Denied status should not have changed")
					}
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetVestingContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName         string
		setupContext     func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		expectedContract string
		expectedError    error
	}{
		{
			testName: "Success - Get vesting contract after initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract with vesting contract address
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			expectedContract: "klp-6b616c70627169646775-cc",
			expectedError:    nil,
		},
		{
			testName: "Failure - GetState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				ctx.GetStateReturns(nil, errors.New("failed to get state"))
			},
			expectedContract: "",
			expectedError:    ginierr.ErrFailedToGetState(errors.New("failed to get state")),
		},
		{
			testName: "Success - Empty state before initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// No initialization, empty state
			},
			expectedContract: "",
			expectedError:    nil,
		},
		{
			testName: "Success - Get after multiple operations",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize with first address
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			expectedContract: "klp-6b616c70627169646775-cc",
			expectedError:    nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.testName == "Failure - GetState error" {
					return nil, errors.New("failed to get state")
				}
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			contract, err := giniContract.GetVestingContract(transactionContext)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
				require.Equal(t, tt.expectedContract, contract)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedContract, contract)
			}
		})
	}
}

// TestGetVestingContract_StateConsistency tests the consistency of the vesting contract address across operations
func TestGetVestingContract_StateConsistency(t *testing.T) {
	t.Parallel()

	// Setup
	transactionContext := &mocks.TransactionContext{}
	giniContract := &chaincode.SmartContract{}
	worldState := map[string][]byte{}

	// Setup stubs
	transactionContext.GetStateStub = func(key string) ([]byte, error) {
		return worldState[key], nil
	}
	transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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

	// Setup identity and KYC
	SetUserID(transactionContext, constants.KalpFoundationAddress)
	transactionContext.GetKYCReturns(true, nil)

	// 1. Check initial state
	contract, err := giniContract.GetVestingContract(transactionContext)
	require.NoError(t, err)
	require.Empty(t, contract, "Vesting contract should be empty before initialization")

	// 2. Initialize contract
	vestingContract := "klp-6b616c70627169646775-cc"
	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", vestingContract)
	require.NoError(t, err)
	require.True(t, ok)

	// 3. Verify vesting contract after initialization
	contract, err = giniContract.GetVestingContract(transactionContext)
	require.NoError(t, err)
	require.Equal(t, vestingContract, contract,
		"Vesting contract should match initialized value")

	// 4. Verify persistence
	contract, err = giniContract.GetVestingContract(transactionContext)
	require.NoError(t, err)
	require.Equal(t, vestingContract, contract,
		"Vesting contract should persist across calls")
}
func TestSetBridgeContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		bridgeAddr    string
		expectedError error
	}{
		{
			testName: "Success - Set bridge contract by foundation",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			bridgeAddr:    "klp-newbridge-cc",
			expectedError: nil,
		},
		{
			testName: "Failure - Non-foundation user attempts to set bridge contract",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// Initialize with foundation first
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)

				// Switch to non-foundation user
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			bridgeAddr:    "klp-newbridge-cc",
			expectedError: ginierr.New("Only Kalp Foundation can set the bridge contract", http.StatusUnauthorized),
		},
		{
			testName: "Failure - GetUserID error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				clientIdentity := &mocks.ClientIdentity{}
				clientIdentity.GetIDReturns("", errors.New("failed to get ID"))
				ctx.GetClientIdentityReturns(clientIdentity)
			},
			bridgeAddr:    "klp-newbridge-cc",
			expectedError: ginierr.ErrFailedToGetPublicAddress,
		},
		{
			testName: "Success - Update existing bridge contract",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				// Set initial bridge contract
				err = contract.SetBridgeContract(ctx, "klp-oldbridge-cc")
				require.NoError(t, err)
			},
			bridgeAddr:    "klp-newbridge-cc",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			err := giniContract.SetBridgeContract(transactionContext, tt.bridgeAddr)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())

				// If error occurred, verify bridge contract hasn't changed
				if tt.testName != "Success - Set bridge contract by foundation" {
					currentBridge, _ := giniContract.GetBridgeContract(transactionContext)
					require.NotEqual(t, tt.bridgeAddr, currentBridge,
						"Bridge contract should not have changed due to error")
				}
			} else {
				require.NoError(t, err)

				// Verify bridge contract was updated correctly
				currentBridge, err := giniContract.GetBridgeContract(transactionContext)
				require.NoError(t, err)
				require.Equal(t, tt.bridgeAddr, currentBridge,
					"Bridge contract should have been updated to new value")
			}
		})
	}
}

func TestGetBridgeContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName         string
		setupContext     func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		expectedContract string
		expectedError    error
	}{
		{
			testName: "Success - Get bridge contract after initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			expectedContract: constants.InitialBridgeContractAddress,
			expectedError:    nil,
		},
		{
			testName: "Success - Get bridge contract after update",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				// Update bridge contract
				err = contract.SetBridgeContract(ctx, "klp-newbridge-cc")
				require.NoError(t, err)
			},
			expectedContract: "klp-newbridge-cc",
			expectedError:    nil,
		},
		{
			testName: "Failure - GetState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				ctx.GetStateReturns(nil, errors.New("failed to get state"))
			},
			expectedContract: "",
			expectedError:    ginierr.ErrFailedToGetState(errors.New("failed to get state")),
		},
		{
			testName: "Success - Empty state before initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// No initialization, empty state
			},
			expectedContract: "",
			expectedError:    nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.testName == "Failure - GetState error" {
					return nil, errors.New("failed to get state")
				}
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			contract, err := giniContract.GetBridgeContract(transactionContext)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
				require.Equal(t, tt.expectedContract, contract)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedContract, contract)
			}
		})
	}
}

func TestSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName       string
		setupContext   func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		expectedSymbol string
		expectedError  error
	}{
		{
			testName: "Success - Get symbol after initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			expectedSymbol: "GINI",
			expectedError:  nil,
		},
		{
			testName: "Failure - GetState error",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				ctx.GetStateReturns(nil, errors.New("failed to get state"))
			},
			expectedSymbol: "",
			expectedError:  ginierr.ErrFailedToGetKey(constants.SymbolKey),
		},
		{
			testName: "Success - Empty state before initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				// No initialization, empty state
			},
			expectedSymbol: "",
			expectedError:  nil,
		},
		{
			testName: "Success - Custom symbol",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				// Initialize contract with custom symbol
				ok, err := contract.Initialize(ctx, "Kalp Token", "KLP", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			expectedSymbol: "KLP",
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				if tt.testName == "Failure - GetState error" {
					return nil, errors.New("failed to get state")
				}
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			symbol, err := giniContract.Symbol(transactionContext)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
				require.Equal(t, tt.expectedSymbol, symbol)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedSymbol, symbol)
			}
		})
	}
}

func TestApprove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName       string
		setupContext   func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		spender        string
		amount         string
		expectedResult bool
		expectedError  error
	}{
		{
			testName: "Success - Valid approval",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:         "1000",
			expectedResult: true,
			expectedError:  nil,
		},
		{
			testName: "Success - Update existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				// Set initial allowance
				ok, err = contract.Approve(ctx, "2da4c4908a393a387b728206b18388bc529fa8d7", "500")
				require.NoError(t, err)
				require.True(t, ok)
			},
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:         "1000",
			expectedResult: true,
			expectedError:  nil,
		},
		{
			testName: "Failure - Invalid spender address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				ctx.GetKYCReturns(true, nil)
			},
			spender:        "invalid-address",
			amount:         "1000",
			expectedResult: false,
			expectedError:  ginierr.ErrInvalidAddress("invalid-address"),
		},
		{
			testName: "Failure - Invalid amount",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				ctx.GetKYCReturns(true, nil)
			},
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			amount:         "invalid-amount",
			expectedResult: false,
			expectedError:  ginierr.ErrInvalidAmount("invalid-amount"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs (include all necessary stubs as before)
			setupTestStubs(transactionContext, worldState)

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			result, err := giniContract.Approve(transactionContext, tt.spender, tt.amount)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
				require.Equal(t, tt.expectedResult, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)

				// Verify allowance was set correctly
				owner, err := helper.GetUserId(transactionContext)
				require.NoError(t, err)
				allowance, err := giniContract.Allowance(transactionContext, owner, tt.spender)
				require.NoError(t, err)
				require.Equal(t, tt.amount, allowance)
			}
		})
	}
}

func TestAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName       string
		setupContext   func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		owner          string
		spender        string
		expectedAmount string
		expectedError  error
	}{
		{
			testName: "Success - Get existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				// Set allowance
				ok, err = contract.Approve(ctx, "2da4c4908a393a387b728206b18388bc529fa8d7", "1000")
				require.NoError(t, err)
				require.True(t, ok)
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "1000",
			expectedError:  nil,
		},
		{
			testName: "Success - No existing allowance",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)

				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
			},
			owner:          "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender:        "2da4c4908a393a387b728206b18388bc529fa8d7",
			expectedAmount: "0",
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs (include all necessary stubs as before)
			setupTestStubs(transactionContext, worldState)

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			amount, err := giniContract.Allowance(transactionContext, tt.owner, tt.spender)

			// Assert results
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError.Error())
				require.Equal(t, tt.expectedAmount, amount)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAmount, amount)
			}
		})
	}
}

// Helper function to setup common test stubs
func setupTestStubs(ctx *mocks.TransactionContext, worldState map[string][]byte) {
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
	ctx.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
		iterator := &mocks.StateQueryIterator{}
		var kvs []queryresult.KV

		prefix := "_" + objectType + "_"
		if len(keys) > 0 {
			prefix += keys[0] + "_"
		}

		index := 0
		for key, value := range worldState {
			if strings.HasPrefix(key, prefix) {
				kvs = append(kvs, queryresult.KV{
					Key:   key,
					Value: value,
				})
			}
		}

		iterator.HasNextCalls(func() bool {
			return index < len(kvs)
		})
		iterator.NextCalls(func() (*queryresult.KV, error) {
			if index < len(kvs) {
				kv := kvs[index]
				index++
				return &kv, nil
			}
			return nil, nil
		})
		return iterator, nil
	}
}

func TestInitialize_NegativeScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName      string
		setupContext  func(*mocks.TransactionContext, map[string][]byte, *chaincode.SmartContract)
		name          string
		symbol        string
		vestingAddr   string
		shouldError   bool
		expectedError error
	}{
		{
			testName: "Failure - Non-Foundation User Initialization",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "non-foundation-user")
				ctx.GetKYCReturns(true, nil)
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Non-foundation user",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, "16f8ff33ef05bb24fb9a30fa79e700f57a496184")
				ctx.GetKYCReturns(true, nil)
			},
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Reinitialization Attempt",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ok, err := contract.Initialize(ctx, "GINI", "GINI", "klp-6b616c70627169646775-cc")
				require.NoError(t, err)
				require.True(t, ok)
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Invalid Vesting Contract Address",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "invalid-address",
			shouldError: true,
		},
		{
			testName: "Failure - Foundation Not KYC'd",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(false, nil)
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Gateway Admin Not KYC'd",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturnsOnCall(0, true, nil)
				ctx.GetKYCReturnsOnCall(1, false, nil)
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Get State for NameKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.GetStateReturnsOnCall(0, nil, errors.New("failed to get state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Get State for SymbolKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.GetStateReturnsOnCall(1, nil, errors.New("failed to get state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Put State for NameKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(0, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Put State for SymbolKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(1, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Get kyc for KalpFoundationAddress",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(false, fmt.Errorf("Error fetching KYC status of Kalp Foundation %v", http.StatusInternalServerError))
				ctx.PutStateWithoutKYCReturnsOnCall(2, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Get kyc for KalpGateWayAdminAddress",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturnsOnCall(0, true, nil)
				ctx.GetKYCReturnsOnCall(1, false, fmt.Errorf("Error fetching KYC status of Gateway Admin %v", http.StatusInternalServerError))
				ctx.PutStateWithoutKYCReturnsOnCall(2, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Put State for GasFeesKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(2, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Failed to Put State for VestingContractKey",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(3, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - PutStateWithoutKYC fails for VestingContractKey test",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(1, errors.New("failed to put state"))
			},
			name:          "GINI",
			symbol:        "GINI",
			vestingAddr:   "klp-6b616c70627169646775-cc",
			expectedError: ginierr.NewInternalError(errors.New("failed to put state"), "failed to set vesting Contract: klp-6b616c70627169646775-cc", http.StatusInternalServerError),
		}, {
			testName: "Failure - Failed to Set Bridge Contract",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				ctx.PutStateWithoutKYCReturnsOnCall(4, errors.New("failed to put state"))
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
		{
			testName: "Failure - Symbol is empty",
			setupContext: func(ctx *mocks.TransactionContext, worldState map[string][]byte, contract *chaincode.SmartContract) {
				SetUserID(ctx, constants.KalpFoundationAddress)
				ctx.GetKYCReturns(true, nil)
				worldState[constants.SymbolKey] = []byte("GINI")
			},
			name:        "GINI",
			symbol:      "GINI",
			vestingAddr: "klp-6b616c70627169646775-cc",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			// Setup
			transactionContext := &mocks.TransactionContext{}
			giniContract := &chaincode.SmartContract{}
			worldState := map[string][]byte{}

			// Setup stubs
			transactionContext.GetStateStub = func(key string) ([]byte, error) {
				return worldState[key], nil
			}
			transactionContext.PutStateWithoutKYCStub = func(key string, value []byte) error {
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
			transactionContext.GetStateByPartialCompositeKeyStub = func(objectType string, keys []string) (kalpsdk.StateQueryIteratorInterface, error) {
				iterator := &mocks.StateQueryIterator{}
				var kvs []queryresult.KV

				prefix := "_" + objectType + "_"
				if len(keys) > 0 {
					prefix += keys[0] + "_"
				}

				index := 0
				for key, value := range worldState {
					if strings.HasPrefix(key, prefix) {
						kvs = append(kvs, queryresult.KV{
							Key:   key,
							Value: value,
						})
					}
				}

				iterator.HasNextCalls(func() bool {
					return index < len(kvs)
				})
				iterator.NextCalls(func() (*queryresult.KV, error) {
					if index < len(kvs) {
						kv := kvs[index]
						index++
						return &kv, nil
					}
					return nil, nil
				})
				return iterator, nil
			}

			// Apply test-specific context setup
			if tt.setupContext != nil {
				tt.setupContext(transactionContext, worldState, giniContract)
			}

			// Execute test
			ok, err := giniContract.Initialize(transactionContext, tt.name, tt.symbol, tt.vestingAddr)

			require.Error(t, err)
			require.False(t, ok)
		})
	}
}
