package chaincode_test

import (
	"fmt"
	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/mocks"
	"math/big"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
	"github.com/p2eengineering/kalp-sdk-public/response"
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "999", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3100", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "0", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("4099", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase2(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.ErrorContains(t, err, "insufficient allowance")
	require.Equal(t, false, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "1000", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3000", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "100", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("4100", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase3(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "80")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "80")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "999", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3080", balance) // 3000 + 80 = 3080

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "20", balance) // 2000 - 80 = 1920

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("4099", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase4(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "80")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.ErrorContains(t, err, "insufficient balance in sender's account for amount")
	require.Equal(t, false, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "1000", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3000", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "80", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("4080", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase5(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1111 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1111"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.ErrorContains(t, err, "insufficient balance in signer's account for gas fees")
	require.Equal(t, false, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "1000", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3000", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "100", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("4100", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase7(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1000 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1000"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "500")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, admin, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(admin, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, userC, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "1000", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "600", balance) // 500 + 100 = 600

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "0", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("1600", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase8(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "999", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3000", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "0", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("3999", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase9(t *testing.T) {
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

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"
	userC := "2da4c4908a393a387b728206b18388bc529fa8d7"
	userG := "35581086b9b262a62f5d2d1603d901d9375777b8"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 1 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("1"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "1000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userG, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userC, "3000")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userG approves userM to spend 100 units
	transactionContext.GetUserIDReturns(userG, nil)
	ok, err = giniContract.Approve(transactionContext, userM, "200")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 100 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom: userM transfers 200 units from userG to userC
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, userG, admin, "200")
	require.ErrorContains(t, err, "insufficient balance in sender's account for amount")
	require.Equal(t, false, ok)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "999", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "3000", balance)

	// Check userG balance (should reflect the deduction of 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, userG)
	require.NoError(t, err)
	require.Equal(t, "0", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("3999", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}

func TestCase10(t *testing.T) {
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

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 10 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("10"))

	// Admin transfers everything except 100 wei
	ok, err = giniContract.Transfer(transactionContext, userC, "11199999999999999999999800")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	ok, err = giniContract.Transfer(transactionContext, userM, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve
	// admin approves userM 100
	ok, err = giniContract.Approve(transactionContext, userM, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// TransferFrom
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.TransferFrom(transactionContext, admin, userC, "100")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Checking balances for admin, userM, userC
	// Test case 3: Check Balance

	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "90", balance)

	balance, err = giniContract.BalanceOf(transactionContext, userC)
	require.NoError(t, err)
	require.Equal(t, "11199999999999999999999900", balance)

	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, "10", balance)
}

func TestCase11(t *testing.T) {
	// t.Parallel()
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
	transactionContext.InvokeChaincodeStub = func(s1 string, b [][]byte, s2 string) response.Response {
		if s1 == constants.BridgeContractAddress && string(b[0]) == "BridgeToken" {
			signer, _ := transactionContext.GetUserID()

			giniContract.TransferFrom(transactionContext, signer, constants.BridgeContractAddress, string(b[1]))
			return response.Response{
				Response: peer.Response{
					Status:  http.StatusOK,
					Payload: []byte("true"),
				},
			}
		}
		return response.Response{
			Response: peer.Response{
				Status:  http.StatusBadRequest,
				Payload: []byte("false"),
			},
		}

	}
	internal.GetCallingContractAddress = func(ctx kalpsdk.TransactionContextInterface) (string, error) {
		return constants.BridgeContractAddress, nil
	}

	// ****************END define helper functions*********************

	// define users
	admin := constants.KalpFoundationAddress
	userM := "16f8ff33ef05bb24fb9a30fa79e700f57a496184"

	// Initialize
	transactionContext.GetUserIDReturns(admin, nil)
	transactionContext.GetKYCReturns(true, nil)

	ok, err := giniContract.Initialize(transactionContext, "GINI", "GINI", "klp-6b616c70627269646775-cc")

	require.NoError(t, err)
	require.Equal(t, true, ok)

	balance, err := giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	require.Equal(t, constants.InitialFoundationBalance, balance)

	balance, err = giniContract.BalanceOf(transactionContext, "klp-6b616c70627269646775-cc")
	require.NoError(t, err)
	require.Equal(t, constants.InitialVestingContractBalance, balance)

	// Updating gas fess to 10 Wei
	transactionContext.PutStateWithoutKYC(constants.GasFeesKey, []byte("10"))

	// Admin recharges userM, userG, and userC

	ok, err = giniContract.Transfer(transactionContext, userM, "110") // 100 + 10 gas fees

	require.NoError(t, err)
	require.Equal(t, true, ok)

	// Approve: userM approves bridge contract to spend 100 units
	transactionContext.GetUserIDReturns(userM, nil)
	ok, err = giniContract.Approve(transactionContext, constants.BridgeContractAddress, "100")
	require.NoError(t, err)
	require.Equal(t, true, ok)

	//
	output := transactionContext.InvokeChaincode(constants.BridgeContractAddress, [][]byte{[]byte("BridgeToken"), []byte("100")}, "kalptantra")
	b, _ := strconv.ParseBool(string(output.Payload))
	require.Equal(t, true, b)

	// Verify balances after transfer
	// Check userM balance
	balance, err = giniContract.BalanceOf(transactionContext, userM)
	require.NoError(t, err)
	require.Equal(t, "0", balance)

	// Check userC balance (should reflect the additional 100 units)
	balance, err = giniContract.BalanceOf(transactionContext, constants.BridgeContractAddress)
	require.NoError(t, err)
	require.Equal(t, "100", balance)

	// Check admin balance (unchanged in this scenario)
	balance, err = giniContract.BalanceOf(transactionContext, admin)
	require.NoError(t, err)
	totalSupply, _ := new(big.Int).SetString(constants.InitialFoundationBalance, 10)
	userBalanceSum, _ := new(big.Int).SetString("100", 10)
	require.Equal(t, new(big.Int).Sub(totalSupply, userBalanceSum).String(), balance)
}
