package chaincode

import (
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"net/http"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func (s *SmartContract) GasFeesTransferSimple(ctx kalpsdk.TransactionContextInterface, gasFeesAccount string, amount string) (bool, error) {
	signer, err := helper.GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("internal error %d: error getting signer: %v", http.StatusBadRequest, err)
	}
	if signer != constants.KalpGateWayAdminAddress {
		return false, fmt.Errorf("signer is not gateway admin : %s", signer)
	}

	isValidAddress, err := helper.IsValidAddress(gasFeesAccount)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(gasFeesAccount)
	}

	amountInt, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return false, fmt.Errorf("error converting amount: %v to uint: %v", amount, err)
	}
	if amountInt <= 0 {
		return false, fmt.Errorf("gas fees amount cannot be less than equal to zero: %d", amountInt)
	}
	if amountInt > constants.InitialGatewayMaxGasFeeInt {
		return false, fmt.Errorf("gas fees amount exceeds the maximum limit: %d", amountInt)
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
		return false, fmt.Errorf("GasFeesTransferSimple should not be called by other contracts")
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
	signer, err := helper.GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("internal error %d: error getting signer: %v", http.StatusBadRequest, err)
	}
	isGatewayAdmin, err := internal.IsGatewayAdminAddress(ctx, signer)
	if err != nil {
		return false, err
	}
	if !isGatewayAdmin {
		return false, fmt.Errorf("signer is not gateway admin : %s", signer)
	}

	isValidAddress, err := helper.IsValidAddress(gasFeesAccount)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(gasFeesAccount)
	}

	amountInt, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return false, fmt.Errorf("error converting amount: %v to uint: %v", amount, err)
	}
	if amountInt <= 0 {
		return false, fmt.Errorf("gas fees amount cannot be less than equal to zero: %d", amountInt)
	}
	var strGatewayMaxGasFee string
	if strGatewayMaxGasFee, err = s.GetGatewayMaxFee(ctx); err != nil {
		return false, err
	}
	gatewayMaxFeeInt, err := strconv.ParseUint(strGatewayMaxGasFee, 10, 64)
	if err != nil {
		return false, fmt.Errorf("error converting gateway max fees: %v to uint: %v", strGatewayMaxGasFee, err)
	}
	if amountInt > gatewayMaxFeeInt {
		return false, fmt.Errorf("gas fees amount exceeds the maximum limit: %d", amountInt)
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
		return false, fmt.Errorf("GasFeesTransferComplex should not be called by other contracts")
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
