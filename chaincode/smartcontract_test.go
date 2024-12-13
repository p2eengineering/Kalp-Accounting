package chaincode_test

import (
	"fmt"
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/mocks"
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
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

func TestInitLedger(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}

	giniContract := chaincode.SmartContract{}

	err := giniContract.InitLedger(transactionContext)

	require.NoError(t, err)
}

func TestInitialize(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}

	giniContract := chaincode.SmartContract{}

	transactionContext.GetUserIDReturns(constants.KalpFoundationAddress, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)
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
	transactionContext.DelStateWithKYCStub = func(s string) error {
		delete(worldState, s)
		return nil
	}
	// ****************END define helper functions*********************

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve
	// admin approves userM 100
	ok, err = giniContract.Approve(transactionContext, userM, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom
	// userM calls transferFrom to transfer 100 tokens from admin to userC
	transactionContext.GetUserIDReturns(userM, nil)
	// adding some balance to userM
	// TODO use transfer in future
	amountToAdd, _ := big.NewInt(0).SetString(constants.InitialGasFees, 10)
	internal.AddUtxo(transactionContext, userM, amountToAdd)

	ok, err = giniContract.TransferFrom(transactionContext, admin, userC, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Checking balances for admin, userM, userC
	// Test case 3: Check Balance

	balance, err := giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "400", balance)

	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "100", balance)

	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, "400", balance)
}
