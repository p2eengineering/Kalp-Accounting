package internal_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/internal"
	"gini-contract/mocks"

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
		clientID := "x509::CN=" + constants.KalpFoundationAddress + ","
		b64ClientID := base64.StdEncoding.EncodeToString([]byte(clientID))
		mockClientIdentity.GetIDReturns(b64ClientID, nil)
		mockCtx.GetClientIdentityReturns(mockClientIdentity)
		mockCtx.GetUserIDReturns("random-address", nil)

		result, err := internal.IsSignerKalpFoundation(mockCtx)

		require.NoError(t, err)
		require.False(t, result)
		require.Equal(t, 1, mockCtx.GetUserIDCallCount())
	})

	t.Run("should return error when GetUserID fails", func(t *testing.T) {
		expectedError := errors.New("failed to get user ID")
		mockCtx.GetUserIDReturns("", expectedError)

		result, err := internal.IsSignerKalpFoundation(mockCtx)

		require.Error(t, err)
		require.False(t, result)
		require.Equal(t, http.StatusInternalServerError, http.StatusAccepted)
		require.Contains(t, err.Error(), "failed to get public address")
		require.Equal(t, 1, mockCtx.GetUserIDCallCount())
	})
}
