package chaincode

import (
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/logger"
	"math/big"
	"net/http"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func (s *SmartContract) GasFeesTransfer(ctx kalpsdk.TransactionContextInterface, gasFeesAccount string, amount string) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if signer != constants.KalpGateWayAdminAddress {
		err := ginierr.New("signer should be gateway admin for gas fees deduction", http.StatusUnauthorized)
		logger.Log.Error(err.FullError())
		return false, err
	}

	isValidAddress, err := helper.IsUserAddress(gasFeesAccount)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(gasFeesAccount)
	}

	amountInt, e := strconv.ParseUint(amount, 10, 64)
	if e != nil {
		err := ginierr.NewInternalError(e, "error parsing amount", http.StatusBadRequest)
		logger.Log.Error(err.FullError())
		return false, ginierr.ErrInvalidAmount(amount)
	}
	if amountInt == 0 {
		return false, ginierr.ErrInvalidAmount(amount)
	}
	if amountInt > constants.InitialGatewayMaxGasFeeInt {
		return false, ginierr.ErrInvalidAmount(amount)
	}

	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}
	if denied, err := internal.IsDenied(ctx, gasFeesAccount); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(gasFeesAccount)
	}

	calledContractAddress, err := internal.GetCalledContractAddress(ctx)
	if err != nil {
		return false, err
	}
	if calledContractAddress != s.GetName() {
		err := ginierr.New("GasFeesTransfer should not be called by other contracts", http.StatusUnauthorized)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if gasFeesAccount != constants.KalpFoundationAddress {
		if err = internal.RemoveUtxoForGasFees(ctx, gasFeesAccount, amount); err != nil {
			return false, err
		}
		if err = internal.AddUtxoForGasFees(ctx, constants.KalpFoundationAddress, amount); err != nil {
			return false, err
		}
		if err := events.EmitTransfer(ctx, gasFeesAccount, constants.KalpFoundationAddress, amount); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *SmartContract) MintByFaucetAdmin(ctx kalpsdk.TransactionContextInterface, address string, amount string) error {
	logger.Log.Infof("MintByFaucetAdmin---->")

	userId, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if userId != constants.TestnetFaucetAdmin {
		err := ginierr.New("signer should be faucet admin for mint", http.StatusUnauthorized)
		logger.Log.Error(err.FullError())
		return err
	}
	isValidAddress, err := helper.IsUserAddress(address)
	if err != nil {
		return err
	}
	if !isValidAddress {
		return ginierr.ErrInvalidAddress(address)
	}

	accAmount, su := big.NewInt(0).SetString(amount, 10)
	if !su {
		return ginierr.ErrConvertingAmountToBigInt(amount)
	}
	if accAmount.Cmp(big.NewInt(0)) <= 0 { // accAmount <= 0
		return ginierr.ErrInvalidAmount(amount)
	}

	// Mint tokens
	err = internal.AddUtxo(ctx, address, accAmount)
	if err != nil {
		return err
	}
	logger.Log.Infof("MintToken Amount By FaucetAdmin---->%v\n", amount)
	return nil

}

// Deprecated: GINI-2.4  KWALA admin management is now managed in Web2.
func (s *SmartContract) SetKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	err := ginierr.ErrDeprecatedFunction("SetKwalaAdmin")
	logger.Log.Error(err.FullError())
	return err
}

// Deprecated: GINI-2.4  KWALA admin management is now managed in Web2.
func (s *SmartContract) DeleteKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	err := ginierr.ErrDeprecatedFunction("DeleteKwalaAdmin")
	logger.Log.Error(err.FullError())
	return err
}

// Deprecated: GINI-3.1  disabled LAST, after KWALA->Web2 migration was confirmed complete.
// This was intentionally the LAST KWALA function disabled, since it was the fn used
// by the migration to burn/settle remaining KWALA gas-fee balances.
func (s *SmartContract) TransferGasFeesToFoundation(ctx kalpsdk.TransactionContextInterface, fromAddress string, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferGasFeesToFoundation")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.1  KWALA credit movement is now managed in Web2.
func (s *SmartContract) TransferKWALACredits(ctx kalpsdk.TransactionContextInterface, fromAddress string, toAddress string, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferKWALACredits")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.3  the Kalp<->Kwala bridge is now managed in Web2.
func (s *SmartContract) TransferKalpToKwala(ctx kalpsdk.TransactionContextInterface, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferKalpToKwala")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.3  the Kalp<->Kwala bridge is now managed in Web2.
func (s *SmartContract) TransferFromAnyKalpToKwala(ctx kalpsdk.TransactionContextInterface, to, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferFromAnyKalpToKwala")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.2  KWALA minting is now managed in Web2.
func (s *SmartContract) MintToKwalaAccountOnBehalfOfKawalaAdmin(ctx kalpsdk.TransactionContextInterface, to, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("MintToKwalaAccountOnBehalfOfKawalaAdmin")
	logger.Log.Error(err.FullError())
	return false, err
}
