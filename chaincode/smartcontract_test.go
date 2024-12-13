package chaincode_test

import (
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/mocks"
	"testing"

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
