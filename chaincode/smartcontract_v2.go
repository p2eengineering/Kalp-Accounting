package chaincode

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
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

func (s *SmartContract) SetKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	logger.Log.Info("SetKwalaAdmin........", userID)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the roles", http.StatusUnauthorized)
	}

	if userID == "" {
		return fmt.Errorf("user Id can not be null")
	}

	isValidAddress, err := helper.IsUserAddress(userID)
	if err != nil {
		return err
	}
	if !isValidAddress {
		return ginierr.ErrInvalidAddress(userID)
	}

	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userID, constants.KwalaAdminRole})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	kwalaAdminRole := models.UserRole{
		Id:   userID,
		Role: constants.KwalaAdminRole,
	}

	kwalaAdminRoleJSON, err := json.Marshal(kwalaAdminRole)
	if err != nil {
		return fmt.Errorf("unable to Marshal kwalaAdminRole struct : %v", err)
	}

	if e := ctx.PutStateWithoutKYC(key, kwalaAdminRoleJSON); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role struct: %v", e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	return nil
}

func (s *SmartContract) DeleteKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	logger.Log.Info("DeleteKwalaAdmin........", userID)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can delete the roles", http.StatusUnauthorized)
	}

	if userID == "" {
		return fmt.Errorf("user Id can not be null")
	}

	isValidAddress, err := helper.IsUserAddress(userID)
	if err != nil {
		return err
	}
	if !isValidAddress {
		return ginierr.ErrInvalidAddress(userID)
	}

	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userID, constants.KwalaAdminRole})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	existingRoleBytes, e := ctx.GetState(key)
	if e != nil || existingRoleBytes == nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("kwala admin role not found for userID %s", userID), http.StatusNotFound)
		logger.Log.Errorf(err.FullError())
		return err
	}

	var userRole models.UserRole
	if err := json.Unmarshal(existingRoleBytes, &userRole); err != nil {
		return fmt.Errorf("failed to unmarshal user role: %v", err)
	}

	if e := ctx.DelStateWithoutKYC(key); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to delete kwala admin role: %v", e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	return nil
}

// TransferGasFeesToFoundation transfers gas fees from a specified address to the foundation
func (s *SmartContract) TransferGasFeesToFoundation(ctx kalpsdk.TransactionContextInterface, fromAddress string, amount string) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	// Only kwala admin can transfer gas fees from any kwala address
	isKwalaAdmin, err := internal.IsKwalaAdminAddress(ctx, signer)
	if err != nil {
		return false, err
	}
	if !isKwalaAdmin {
		err := ginierr.New("signer should be kwala admin for gas fees transfer", http.StatusUnauthorized)
		logger.Log.Error(err.FullError())
		return false, err
	}

	isValidAddress, err := helper.IsKwalaAccountAddress(fromAddress)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(fromAddress)
	}

	amountBigInt, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return false, ginierr.ErrConvertingAmountToBigInt(amount)
	}
	if amountBigInt.Cmp(big.NewInt(0)) <= 0 {
		return false, ginierr.ErrInvalidAmount(amount)
	}

	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}
	if denied, err := internal.IsKwalaDenied(ctx, fromAddress); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(fromAddress)
	}

	if fromAddress != constants.KalpFoundationAddress {
		if err = internal.RemoveUtxo(ctx, fromAddress, amountBigInt); err != nil {
			return false, err
		}
		if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, amountBigInt); err != nil {
			return false, err
		}
		if err := events.EmitTransferGasFromKwalaAccountToFoundation(ctx, fromAddress, amount); err != nil {
			return false, err
		}
	}
	return true, nil
}

// TransferKalpToKwala transfers funds from Kalp account to Kwala account
func (s *SmartContract) TransferKalpToKwala(ctx kalpsdk.TransactionContextInterface, amount string) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	isValidAddress, err := helper.IsUserAddress(signer)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(signer)
	}

	amountBigInt, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return false, ginierr.ErrConvertingAmountToBigInt(amount)
	}
	if amountBigInt.Cmp(big.NewInt(0)) <= 0 {
		return false, ginierr.ErrInvalidAmount(amount)
	}

	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}
	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}

	kwalaAccountAddress := fmt.Sprintf("%s-%s-%s", constants.KwalaAccountPrefix, signer, constants.KwalaAccountSuffix)

	// Validate the constructed kwala account address
	isValidKwalaAddress, err := helper.IsKwalaAccountAddress(kwalaAccountAddress)
	if err != nil {
		return false, err
	}
	if !isValidKwalaAddress {
		return false, ginierr.ErrInvalidAddress(kwalaAccountAddress)
	}

	// Check if kwala account address is denied
	if denied, err := internal.IsDenied(ctx, kwalaAccountAddress); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(kwalaAccountAddress)
	}

	// Transfer from Kalp account to Kwala account
	if err = internal.RemoveUtxo(ctx, signer, amountBigInt); err != nil {
		return false, err
	}
	if err = internal.AddUtxo(ctx, kwalaAccountAddress, amountBigInt); err != nil {
		return false, err
	}
	if err := events.EmitTransferKalpToKwala(ctx, signer, kwalaAccountAddress, amount); err != nil {
		return false, err
	}

	return true, nil
}
