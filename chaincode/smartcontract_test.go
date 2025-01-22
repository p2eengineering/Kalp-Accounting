package chaincode_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/models"
	"gini-contract/mocks"
	"math/rand"
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
