package chaincode_test

import (
	"gini-contract/chaincode"
	"gini-contract/chaincode/ginierr"
	"gini-contract/mocks"
	"net/http"
	"strconv"
	"testing"

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

// GINI-2.1 — credit-movement functions must reject calls with a deprecation error.

func TestGasFeesTransfer_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	ok, err := giniContract.GasFeesTransfer(transactionContext, "klp-someaddress-cc", "100")

	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, ginierr.ErrDeprecatedFunction("GasFeesTransfer").Error(), err.Error())
	requireGone(t, err)
}

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

func TestMintByFaucetAdmin_Deprecated(t *testing.T) {
	t.Parallel()
	transactionContext := &mocks.TransactionContext{}
	giniContract := chaincode.SmartContract{}

	err := giniContract.MintByFaucetAdmin(transactionContext, "klp-someaddress-cc", "100")

	require.Error(t, err)
	require.Equal(t, ginierr.ErrDeprecatedFunction("MintByFaucetAdmin").Error(), err.Error())
	requireGone(t, err)
}

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
