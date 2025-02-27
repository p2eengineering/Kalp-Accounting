package chaincode

import (
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/logger"
	"net/http"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func (s *SmartContract) GasFeesTransferSimple(ctx kalpsdk.TransactionContextInterface, gasFeesAccount string, amount string) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if signer != constants.KalpGateWayAdminAddress {
		err := ginierr.New("signer should be gateway admin for gas fees deduction", http.StatusBadRequest)
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
		logger.Log.Error("error parsing amount ", amount, e)
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
	// TODO: discuss if we need to check if admin is denied
	// if denied, err := internal.IsDenied(ctx, constants.KalpFoundationAddress); err != nil {
	// 	return false, err
	// } else if denied {
	// 	return false, ginierr.ErrDeniedAddress(constants.KalpFoundationAddress)
	// }

	calledContractAddress, err := internal.GetCalledContractAddress(ctx)
	if err != nil {
		return false, err
	}
	if calledContractAddress != s.GetName() {
		err := ginierr.New("GasFeesTransferSimple should not be called by other contracts", http.StatusBadRequest)
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

func (s *SmartContract) GasFeesTransferComplex(ctx kalpsdk.TransactionContextInterface, gasFeesAccount string, amount string) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	isGatewayAdmin, err := internal.IsGatewayAdminAddress(ctx, signer)
	if err != nil {
		return false, err
	}
	if !isGatewayAdmin {
		return false, fmt.Errorf("signer is not gateway admin : %s", signer)
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
		logger.Log.Error("error parsing amount ", amount, e)
		return false, ginierr.ErrInvalidAmount(amount)
	}
	if amountInt == 0 {
		return false, ginierr.ErrInvalidAmount(amount)
	}
	var strGatewayMaxGasFee string
	if strGatewayMaxGasFee, err = s.GetGatewayMaxFee(ctx); err != nil {
		return false, err
	}
	gatewayMaxFeeInt, e := strconv.ParseUint(strGatewayMaxGasFee, 10, 64)
	if e != nil {
		logger.Log.Error("error parsing gas fees amount", e, strGatewayMaxGasFee)
		return false, fmt.Errorf("error converting gateway max fees: %v to uint: %v", strGatewayMaxGasFee, err)
	}
	if amountInt > gatewayMaxFeeInt {
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
	// TODO: discuss if we need to check if admin is denied
	// if denied, err := internal.IsDenied(ctx, constants.KalpFoundationAddress); err != nil {
	// 	return false, err
	// } else if denied {
	// 	return false, ginierr.ErrDeniedAddress(constants.KalpFoundationAddress)
	// }

	calledContractAddress, err := internal.GetCalledContractAddress(ctx)
	if err != nil {
		return false, err
	}
	if calledContractAddress != s.GetName() {
		err := ginierr.New("GasFeesTransferSimple should not be called by other contracts", http.StatusBadRequest)
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
