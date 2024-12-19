package chaincode

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
	"math/big"
	"net/http"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

type SmartContract struct {
	kalpsdk.Contract
}

func (s *SmartContract) Initialize(ctx kalpsdk.TransactionContextInterface, name string, symbol string, vestingContract string) (bool, error) {
	logger.Log.Infoln("Initializing smart contract... with arguments", name, symbol, vestingContract)

	// checking if signer is kalp foundation else return error
	if signerKalp, err := helper.IsSignerKalpFoundation(ctx); err != nil {
		return false, err
	} else if !signerKalp {
		return false, ginierr.New("Only Kalp Foundation can initialize the contract", http.StatusUnauthorized)
	}

	// checking if contract is already initialized
	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		logger.Log.Errorf("Error in GetState %s: %v", constants.NameKey, e)
		return false, ginierr.ErrFailedToGetKey(constants.NameKey)
	} else if bytes != nil {
		return false, ginierr.New(fmt.Sprintf("cannot initialize again,%s already set: %s", constants.NameKey, string(bytes)), http.StatusBadRequest)
	}
	if bytes, e := ctx.GetState(constants.SymbolKey); e != nil {
		logger.Log.Errorf("Error in GetState %s: %v", constants.SymbolKey, e)
		return false, ginierr.ErrFailedToGetKey(constants.SymbolKey)
	} else if bytes != nil {
		return false, ginierr.New(fmt.Sprintf("cannot initialize again,%s already set: %s", constants.SymbolKey, string(bytes)), http.StatusBadRequest)
	}

	if !helper.IsValidAddress(vestingContract) {
		return false, ginierr.ErrIncorrectAddress(vestingContract)
	}

	// Checking if kalp foundation & gateway admin are KYC'd
	if kyced, e := ctx.GetKYC(constants.KalpFoundationAddress); e != nil {
		err := ginierr.NewWithInternalError(e, "Error fetching KYC status of foundation", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Foundation is not KYC'd", http.StatusBadRequest)
	}
	if kyced, e := ctx.GetKYC(constants.KalpGateWayAdminAddress); e != nil {
		err := ginierr.NewWithInternalError(e, "Error fetching KYC status of Gateway Admin", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Gateway Admin is not KYC'd", http.StatusBadRequest)
	}

	// intializing roles for kalp foundation & gateway admin
	if _, err := internal.InitializeRoles(ctx, constants.KalpFoundationAddress, constants.KalpFoundationRole); err != nil {
		return false, err
	}
	if _, err := internal.InitializeRoles(ctx, constants.KalpGateWayAdminAddress, constants.KalpGateWayAdminRole); err != nil {
		return false, err
	}

	// minting initial tokens
	if err := internal.Mint(ctx, []string{constants.KalpFoundationAddress, vestingContract}, []string{constants.InitialFoundationBalance, constants.InitialVestingContractBalance}); err != nil {
		return false, err
	}

	// storing name, symbol and initial gas fees
	if e := ctx.PutStateWithoutKYC(constants.NameKey, []byte(name)); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to set token name", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.SymbolKey, []byte(symbol)); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to set symbol", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.GasFeesKey, []byte(constants.InitialGasFees)); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to set gas fees", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.VestingContractKey, []byte(vestingContract)); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to set vesting Contract", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if err := s.SetBridgeContract(ctx, constants.InitialBridgeContractAddress); err != nil {
		return false, err
	}
	logger.Log.Infoln("Initialize Invoked complete")
	return true, nil
}

func (s *SmartContract) Allow(ctx kalpsdk.TransactionContextInterface, address string) error {

	logger.Log.Infof("Allow invoked for address: %s", address)

	if signerKalp, err := helper.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can Allow", http.StatusUnauthorized)
	}

	if denied, err := internal.IsDenied(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error checking if address already allowed: %v", http.StatusInternalServerError, err)
	} else if !denied {
		return ginierr.ErrNotDenied(address)
	}

	if err := internal.AllowAddress(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error allowing address: %v", http.StatusInternalServerError, err)
	}
	return nil
}

func (s *SmartContract) Deny(ctx kalpsdk.TransactionContextInterface, address string) error {

	logger.Log.Infof("Deny invoked for address: %s", address)

	if signerKalp, err := helper.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can Deny", http.StatusUnauthorized)
	}
	if address == constants.KalpFoundationAddress {
		return ginierr.New("admin cannot be denied", http.StatusBadRequest)
	}
	if denied, err := internal.IsDenied(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error checking if address already denied: %v", http.StatusInternalServerError, err)
	} else if denied {
		return ginierr.ErrAlreadyDenied(address)
	}
	if err := internal.DenyAddress(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error denying address: %v", http.StatusInternalServerError, err)
	}
	return nil
}

func (s *SmartContract) Name(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.NameKey)
	if err != nil {
		return "", ginierr.ErrFailedToGetKey(constants.NameKey)
	}
	return string(bytes), nil
}

func (s *SmartContract) Symbol(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.SymbolKey)
	if err != nil {
		return "", ginierr.ErrFailedToGetKey(constants.SymbolKey)
	}
	return string(bytes), nil
}

func (s *SmartContract) Decimals(ctx kalpsdk.TransactionContextInterface) uint8 {
	return 18
}

func (s *SmartContract) GetGasFees(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.GasFeesKey)
	if err != nil {
		return "", fmt.Errorf("failed to get Gas Fee: %v", err)
	}
	if bytes == nil {
		return "", fmt.Errorf("gas fee not set")
	}
	return string(bytes), nil
}

func (s *SmartContract) SetGasFees(ctx kalpsdk.TransactionContextInterface, gasFees string) error {

	operator, err := helper.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	userRole, err := internal.GetUserRoles(ctx, operator)
	if err != nil {
		logger.Log.Infof("error checking operator's role: %v", err)
		return fmt.Errorf("error checking operator's role: %v", err)
	}
	logger.Log.Infof("useRole: %s\n", userRole)
	// if userRole != constants.GasFeesAdminRole {
	// 	return fmt.Errorf("error with status code %v, error: only gas fees admin is allowed to update gas fees", http.StatusInternalServerError)
	// }
	err = ctx.PutStateWithoutKYC(constants.GasFeesKey, []byte(gasFees))
	if err != nil {
		return fmt.Errorf("failed to set gasfees: %v", err)
	}
	return nil
}

// func (s *SmartContract) Burn(ctx kalpsdk.TransactionContextInterface, address string) (Response, error) {
// 	//check if contract has been intilized first
//
// 	logger.Log.Infof("RemoveFunds---->%s", env)
// 	initialized, err := CheckInitialized(ctx)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v,error: failed to check if contract is already initialized: %v", http.StatusInternalServerError, err)

// 	}
// 	if !initialized {
// 		return Response{
// 			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v,error: contract options need to be set before calling any function, call Initialize() to initialize contract", http.StatusInternalServerError)
// 	}

// 	err = InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("payment admin role check failed in Brun request: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error: payment admin role check failed in Brun request: %v", http.StatusInternalServerError, err)
// 	}

// 	operator, err := utils.GetUserId(ctx)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to get client id: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusBadRequest,
// 		}, fmt.Errorf("error with status code %v, error:failed to get client id: %v", http.StatusBadRequest, err)
// 	}

// 	amount, err := s.BalanceOf(ctx, address)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to get balance : %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusBadRequest,
// 		}, fmt.Errorf("error with status code %v, error:failed to get balance : %v", http.StatusBadRequest, err)
// 	}
// 	accAmount, su := big.NewInt(0).SetString(amount, 10)
// 	if !su {
// 		return Response{
// 			Message:    fmt.Sprintf("can't convert amount to big int %s", amount),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusConflict,
// 		}, fmt.Errorf("error with status code %v,can't convert amount to big int %s", http.StatusConflict, amount)
// 	}
// 	err = utils.RemoveUtxo(ctx, address, false, accAmount)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("Remove balance in burn has error: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error:Remove balance in burn has error: %v", http.StatusBadRequest, err)
// 	}

// 	if err := EmitTransferSingle(ctx, TransferSingle{Operator: operator, From: address, To: "0x0", Value: accAmount}); err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("unable to remove funds: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error:unable to remove funds: %v", http.StatusBadRequest, err)
// 	}

// 	funcName, _ := ctx.GetFunctionAndParameters()
// 	response := map[string]interface{}{
// 		"txId":            ctx.GetTxID(),
// 		"txFcn":           funcName,
// 		"txType":          "Invoke",
// 		"transactionData": address,
// 	}
// 	return Response{
// 		Message:    "Funds removed successfully",
// 		Success:    true,
// 		Status:     "Success",
// 		StatusCode: http.StatusCreated,
// 		Response:   response,
// 	}, nil
// }

// func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, address string, amount string) (bool, error) {

// 	logger.Log.Info("Transfer---->")
// 	address = strings.Trim(address, " ")
// 	if address == "" {
// 		return false, fmt.Errorf("invalid input address")
// 	}

// 	sender, err := helper.GetUserId(ctx)
// 	if err != nil {
// 		return false, fmt.Errorf("error in getting user id: %v", err)
// 	}
// 	userRole, err := internal.GetUserRoles(ctx, sender)
// 	if err != nil {
// 		logger.Log.Infof("error checking user's role: %v", err)
// 		return false, fmt.Errorf("error checking user's role:: %v", err)
// 	}
// 	if len(address) != 40 && userRole != constants.KalpGateWayAdminRole {
// 		return false, fmt.Errorf("address must be 40 characters long")
// 	}
// 	if strings.ContainsAny(address, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") && userRole != constants.KalpGateWayAdminRole {
// 		return false, fmt.Errorf("invalid address")
// 	}
// 	gasFees, err := s.GetGasFees(ctx)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to get gas gee: %v", err)
// 	}
// 	gasFeesAmount, su := big.NewInt(0).SetString(gasFees, 10)
// 	if !su {
// 		return false, fmt.Errorf("gasfee can't be converted to big int")
// 	}
// 	validateAmount, su := big.NewInt(0).SetString(amount, 10)
// 	if !su {
// 		logger.Log.Infof("Amount can't be converted to string")
// 		return false, fmt.Errorf("error with status code %v, invalid Amount %v", http.StatusBadRequest, amount)
// 	}
// 	if validateAmount.Cmp(big.NewInt(0)) == -1 || validateAmount.Cmp(big.NewInt(0)) == 0 { // <= 0 {
// 		return false, fmt.Errorf("error with status code %v, invalid Amount %v", http.StatusBadRequest, amount)
// 	}
// 	logger.Log.Infof("useRole: %s\n", userRole)
// 	// Covers below 2 scenarios where gateway deducts gas fees and transfers to kalp foundation:
// 	// 1. when Dapp/users sends non-GINI transactions via gateway
// 	// 2. when HandleBridgeToken from bridge contract is called by Bridge Admin
// 	if userRole == constants.KalpGateWayAdminRole {
// 		var send models.Sender
// 		errs := json.Unmarshal([]byte(address), &send)
// 		if errs != nil {
// 			logger.Log.Info("internal error: error in parsing sender data")
// 			return false, fmt.Errorf("internal error: error in parsing sender data")
// 		}
// 		if len(send.Sender) != 40 {
// 			return false, fmt.Errorf("address must be 40 characters long")
// 		}
// 		if strings.ContainsAny(send.Sender, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") {
// 			return false, fmt.Errorf("invalid address")
// 		}
// 		if send.Sender != constants.KalpFoundationAddress {
// 			gRemoveAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")

// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}
// 			err = internal.RemoveUtxo(ctx, send.Sender, gRemoveAmount)
// 			if err != nil {
// 				logger.Log.Infof("transfer remove err: %v", err)
// 				return false, fmt.Errorf("transfer remove err: %v", err)
// 			}
// 			gAddAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")

// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}
// 			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gAddAmount)
// 			if err != nil {
// 				logger.Log.Infof("err: %v\n", err)
// 				return false, fmt.Errorf("transfer add err: %v", err)
// 			}
// 			logger.Log.Infof("foundation transfer : %s\n", userRole)
// 		}
// 	} else if b, err := internal.IsCallerKalpBridge(ctx, constants.BridgeContractAddress); b && err == nil {
// 		// In this scenario transfer function is invoked fron Withdraw token funtion from bridge contract address
// 		logger.Log.Infof("sender address changed to Bridge contract addres: \n", constants.BridgeContractAddress)
// 		// In this scenario sender is kalp foundation is bridgeing from WithdrawToken Function,
// 		// will credit amount to kalp foundation and remove amount from sender without gas fees
// 		if sender == constants.KalpFoundationAddress {
// 			sender = constants.BridgeContractAddress
// 			subAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")
// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}

// 			err = internal.RemoveUtxo(ctx, sender, subAmount)
// 			if err != nil {
// 				logger.Log.Infof("transfer remove err: %v", err)
// 				return false, fmt.Errorf("transfer remove err: %v", err)
// 			}
// 			addAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")
// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}
// 			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, addAmount)
// 			if err != nil {
// 				logger.Log.Infof("err: %v\n", err)
// 				return false, fmt.Errorf("transfer add err: %v", err)
// 			}
// 			logger.Log.Infof("bridge transfer to foundation : %s\n", constants.KalpFoundationAddress)
// 		} else {
// 			// In this scenario sender is Kalp Bridge we will credit gas fees to kalp foundation and remove amount from bridge contract
// 			// address. Reciver will recieve amount after gas fees deduction
// 			sender = constants.BridgeContractAddress
// 			removeAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")
// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}
// 			if removeAmount.Cmp(gasFeesAmount) == -1 || removeAmount.Cmp(gasFeesAmount) == 0 {
// 				return false, fmt.Errorf("error with status code %v, error:bridge amount can not be less than equal to gas fee", http.StatusBadRequest)
// 			}
// 			err = internal.RemoveUtxo(ctx, sender, removeAmount)
// 			if err != nil {
// 				logger.Log.Infof("transfer remove err: %v", err)
// 				return false, fmt.Errorf("transfer remove err: %v", err)
// 			}
// 			addAmount, su := big.NewInt(0).SetString(amount, 10)
// 			if !su {
// 				logger.Log.Infof("amount can't be converted to string ")
// 				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 			}

// 			bridgedAmount := addAmount.Sub(addAmount, gasFeesAmount)
// 			logger.Log.Infof("bridgedAmount :%v", bridgedAmount)
// 			err = internal.AddUtxo(ctx, address, bridgedAmount)
// 			if err != nil {
// 				logger.Log.Infof("err: %v\n", err)
// 				return false, fmt.Errorf("transfer add err: %v", err)
// 			}
// 			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFeesAmount)
// 			if err != nil {
// 				logger.Log.Infof("err: %v\n", err)
// 				return false, fmt.Errorf("transfer add err: %v", err)
// 			}
// 			logger.Log.Infof("bridge transfer to normal user : %s\n", userRole)
// 		}
// 	} else if sender == constants.KalpFoundationAddress && address == constants.KalpFoundationAddress {
// 		//In this scenario sender is kalp foundation and address is the kalp foundation so no addition or removal is required
// 		logger.Log.Infof("foundation transfer to foundation : %s address:%s\n", sender, address)

// 	} else if sender == constants.KalpFoundationAddress {
// 		//In this scenario sender is kalp foundation and address is the reciver so no gas fees deduction in code
// 		subAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("amount can't be converted to string ")
// 			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 		}
// 		err := internal.RemoveUtxo(ctx, sender, subAmount)
// 		if err != nil {
// 			logger.Log.Infof("transfer remove err: %v", err)
// 			return false, fmt.Errorf("transfer remove err: %v", err)
// 		}
// 		addAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("amount can't be converted to string ")
// 			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 		}
// 		err = internal.AddUtxo(ctx, address, addAmount)
// 		if err != nil {
// 			logger.Log.Infof("err: %v\n", err)
// 			return false, fmt.Errorf("transfer add err: %v", err)
// 		}
// 		logger.Log.Infof("foundation transfer to user : %s\n", userRole)

// 	} else if address == constants.KalpFoundationAddress {
// 		//In this scenario sender is normal user and address is the kap foundation so gas fees+amount will be credited to kalp foundation
// 		removeAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("removeAmount can't be converted to string ")
// 			return false, fmt.Errorf("removeAmount can't be converted to string: %v ", err)
// 		}
// 		err := internal.RemoveUtxo(ctx, sender, removeAmount)
// 		if err != nil {
// 			logger.Log.Infof("transfer remove err: %v", err)
// 			return false, fmt.Errorf("transfer remove err: %v", err)
// 		}
// 		addAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("amount can't be converted to string ")
// 			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
// 		}
// 		err = internal.AddUtxo(ctx, address, addAmount)
// 		if err != nil {
// 			logger.Log.Infof("err: %v\n", err)
// 			return false, fmt.Errorf("transfer add err: %v", err)
// 		}
// 		logger.Log.Infof("foundation transfer to user : %s\n", userRole)
// 	} else {
// 		//This is normal scenario where amount will be deducted from sender and amount-gas fess will credited to address and gas fees will be credited to kalp foundation
// 		logger.Log.Infof("operator-->", sender)
// 		logger.Log.Info("transfer transferAmount")
// 		if sender == address {
// 			return false, fmt.Errorf("transfer to self not alllowed")
// 		}
// 		transferAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("Amount can't be converted to string")
// 			return false, fmt.Errorf("error with status code %v,Amount can't be converted to string", http.StatusConflict)
// 		}
// 		if transferAmount.Cmp(gasFeesAmount) == -1 || transferAmount.Cmp(gasFeesAmount) == 0 {
// 			return false, fmt.Errorf("error with status code %v, error:transfer amount can not be less than equal to gas fee", http.StatusBadRequest)
// 		}
// 		logger.Log.Infof("transferAmount %v\n", transferAmount)
// 		logger.Log.Infof("gasFeesAmount %v\n", gasFeesAmount)
// 		// Withdraw the funds from the sender address
// 		err = internal.RemoveUtxo(ctx, sender, transferAmount)
// 		if err != nil {
// 			logger.Log.Infof("transfer remove err: %v", err)
// 			return false, fmt.Errorf("error with status code %v, error:error while reducing balance %v", http.StatusBadRequest, err)
// 		}
// 		addAmount, su := big.NewInt(0).SetString(amount, 10)
// 		if !su {
// 			logger.Log.Infof("transfer Amount can't be converted to string ")
// 			return false, fmt.Errorf("error with status code %v,transaction %v already accounted", http.StatusConflict, transferAmount)
// 		}
// 		logger.Log.Infof("Add amount %v\n", addAmount)
// 		addAmounts := addAmount.Sub(addAmount, gasFeesAmount)
// 		// Deposit the fund to the recipient address
// 		err = internal.AddUtxo(ctx, address, addAmounts)
// 		if err != nil {
// 			logger.Log.Infof("err: %v\n", err)
// 			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
// 		}
// 		logger.Log.Infof("gasFeesAmount %v\n", gasFeesAmount)
// 		err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFeesAmount)
// 		if err != nil {
// 			logger.Log.Infof("err: %v\n", err)
// 			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
// 		}
// 	}
// 	transferSingleEvent := models.TransferSingle{Operator: sender, From: sender, To: address, Value: amount}
// 	if err := internal.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
// 		logger.Log.Infof("err: %v\n", err)
// 		return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
// 	}
// 	return true, nil

// }

func (s *SmartContract) BalanceOf(ctx kalpsdk.TransactionContextInterface, account string) (string, error) {

	if !helper.IsValidAddress(account) {
		return "0", ginierr.ErrIncorrectAddress(account)
	}
	amt, err := internal.GetTotalUTXO(ctx, account)
	if err != nil {
		return "0", fmt.Errorf("error fetching balance: %v", err)
	}
	logger.Log.Infof("total balance:%v account:%s\n", amt, account)

	return amt, nil
}

func (s *SmartContract) balance(ctx kalpsdk.TransactionContextInterface, account string) (*big.Int, error) {
	if balanceStr, err := s.BalanceOf(ctx, account); err != nil {
		return big.NewInt(0), err
	} else if senderBalance, ok := big.NewInt(0).SetString(balanceStr, 10); !ok {
		err := ginierr.ErrConvertingAmountToBigInt(balanceStr)
		logger.Log.Error(err.FullError())
		return big.NewInt(0), err
	} else {
		return senderBalance, nil
	}
}

func (s *SmartContract) Approve(ctx kalpsdk.TransactionContextInterface, spender string, amount string) (bool, error) {
	logger.Log.Infoln("Approve invoked.... with arguments", spender, amount)
	if err := models.SetAllowance(ctx, spender, amount); err != nil {
		logger.Log.Infof("error unable to approve funds: %v\n", err)
		return false, err
	}
	return true, nil
}

func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, recipient string, value string) (bool, error) {
	logger.Log.Info("Transfer operation initiated")

	signer, err := helper.GetUserId(ctx)
	if err != nil {
		err := ginierr.NewWithInternalError(err, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	sender := signer

	var gasFees *big.Int
	if gasFeesString, err := s.GetGasFees(ctx); err != nil {
		return false, err
	} else if val, ok := big.NewInt(0).SetString(gasFeesString, 10); !ok {
		return false, ginierr.New("invalid gas fees found:"+gasFeesString, http.StatusInternalServerError)
	} else {
		gasFees = val
	}

	gatewayAdmin := internal.GetGatewayAdminAddress(ctx)

	if signer == gatewayAdmin {
		var gasDeductionAccount models.Account
		err := json.Unmarshal([]byte(recipient), &gasDeductionAccount)
		if err == nil {
			sender = gasDeductionAccount.Recipient
			gasFees = big.NewInt(0)
		} else {
			return false, fmt.Errorf("failed to unmarshal recipient: %v", err)
		}
	}

	// Input validation
	if helper.IsContractAddress(signer) {
		return false, ginierr.New("signer cannot be a contract", http.StatusBadRequest)
	}
	if helper.IsContractAddress(sender) && helper.IsContractAddress(recipient) {
		return false, ginierr.New("both sender and recipient cannot be contracts", http.StatusBadRequest)
	}
	if !helper.IsValidAddress(sender) {
		return false, ginierr.ErrIncorrectAddress("sender")
	} else if !helper.IsValidAddress(recipient) {
		return false, ginierr.ErrIncorrectAddress("recipient")
	} else if !internal.IsAmountProper(value) {
		return false, ginierr.ErrInvalidAmount(value)
	}

	if kycSender, e := ctx.GetKYC(sender); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for sender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	} else if !kycSender {
		err := ginierr.New("sender is not KYCed", http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if kycSigner, e := ctx.GetKYC(signer); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	} else if !kycSigner {
		err := ginierr.New("signer is not KYCed", http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

	amount, ok := big.NewInt(0).SetString(value, 10)
	if !ok || amount.Cmp(big.NewInt(0)) != 1 {
		return false, ginierr.ErrInvalidAmount(value)
	}

	senderBalance, err := s.balance(ctx, sender)
	if err != nil {
		return false, err
	}

	signerBalance, err := s.balance(ctx, signer)
	if err != nil {
		return false, err
	}

	if sender == signer {
		if senderBalance.Cmp(new(big.Int).Add(amount, gasFees)) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount + gas fees", http.StatusBadRequest)
		}
	} else {
		if senderBalance.Cmp(amount) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount", http.StatusBadRequest)
		}
		if signerBalance.Cmp(gasFees) < 0 {
			return false, ginierr.New("insufficient balance in signer's account for gas fees", http.StatusBadRequest)
		}
	}

	if signer == sender && signer == recipient {
		if signer != constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, gasFees); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
				return false, err
			}
		}
	} else if signer == sender && signer != recipient {
		if signer == constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
				return false, err
			}
		} else {
			if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	} else {
		if signer == gatewayAdmin {
			if sender != constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, amount); err != nil {
					return false, err
				}
			}
		} else {
			if signer == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
			} else if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	}

	eventPayload := map[string]interface{}{
		"from":  sender,
		"to":    recipient,
		"value": amount.String(),
	}
	eventBytes, _ := json.Marshal(eventPayload)
	_ = ctx.SetEvent("Transfer", eventBytes)

	return true, nil
}

func (s *SmartContract) TransferFrom(ctx kalpsdk.TransactionContextInterface, sender string, recipient string, amount string) (bool, error) {
	logger.Log.Infoln("TransferFrom invoked.... with arguments", sender, recipient, amount)

	var spender, signer string
	var e error

	if signer, e = helper.GetUserId(ctx); e != nil {
		err := ginierr.NewWithInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	// Determine if the call is from a contract
	calledContractAddress, err := internal.GetCalledContractAddress(ctx)
	logger.Log.Debug("calledContractAddress => ", calledContractAddress)
	if err != nil {
		return false, ginierr.New("called contract address not found", http.StatusBadRequest)
	}

	vestingContract, err := s.GetVestingContract(ctx)
	if err != nil {
		return false, err
	}
	bridgeContract, err := s.GetBridgeContract(ctx)
	if err != nil {
		return false, err
	}

	if calledContractAddress != s.GetName() {
		if calledContractAddress == bridgeContract || calledContractAddress == vestingContract {
			spender = calledContractAddress
		} else {
			err := ginierr.New("The called contract is neither bridge contract nor vesting contract", http.StatusBadRequest)
			logger.Log.Error(err.FullError())
			return false, err
		}
	} else {
		spender = signer
	}

	// Input validation
	if helper.IsContractAddress(signer) {
		return false, ginierr.New("signer cannot be a contract", http.StatusBadRequest)
	}
	if !helper.IsUserAddress(signer) {
		return false, ginierr.ErrInvalidUserAddress(signer)
	}
	if helper.IsContractAddress(sender) && helper.IsContractAddress(recipient) {
		return false, ginierr.New("both sender and recipient cannot be contracts", http.StatusBadRequest)
	}
	if !helper.IsValidAddress(spender) {
		return false, ginierr.ErrIncorrectAddress(spender)
	}
	if !helper.IsValidAddress(sender) {
		return false, ginierr.ErrIncorrectAddress(sender)
	}
	if !helper.IsValidAddress(recipient) {
		return false, ginierr.ErrIncorrectAddress(recipient)
	}
	if !internal.IsAmountProper(amount) {
		return false, ginierr.ErrInvalidAmount(amount)
	}
	// Parse the amt
	amt, _ := big.NewInt(0).SetString(amount, 10)

	// check if signer, spender, sender and recipient are not denied
	if denied, err := internal.IsDenied(ctx, sender); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(sender)
	}
	if denied, err := internal.IsDenied(ctx, recipient); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(recipient)
	}
	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}
	if denied, err := internal.IsDenied(ctx, spender); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(spender)
	}

	// Ensure KYC compliance
	var kycSender, kycSpender, kycSigner bool
	if kycSender, e = ctx.GetKYC(sender); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for sender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if kycSpender, e = ctx.GetKYC(spender); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for spender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if kycSigner, e = ctx.GetKYC(signer); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if !(kycSender || kycSpender || kycSigner) {
		err := ginierr.New("sender, spender AND signer is not KYCed", http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

	// Get balances
	senderBalance, err := s.balance(ctx, sender)
	if err != nil {
		return false, err
	}
	signerBalance, err := s.balance(ctx, signer)
	if err != nil {
		return false, err
	}
	allowanceStr, err := models.GetAllowance(ctx, sender, spender)
	if err != nil {
		return false, err
	}
	allowance, ok := big.NewInt(0).SetString(allowanceStr, 10)
	if !ok {
		err := ginierr.ErrConvertingAmountToBigInt(allowanceStr)
		logger.Log.Error(err.FullError())
		return false, err
	}

	var gasFees *big.Int
	if gasFeesString, err := s.GetGasFees(ctx); err != nil {
		return false, err
	} else if val, ok := big.NewInt(0).SetString(gasFeesString, 10); !ok {
		return false, ginierr.New("invalid gas fees found:"+gasFeesString, http.StatusInternalServerError)
	} else {
		gasFees = val
	}

	// Balance checks
	if signer == sender {
		if signerBalance.Cmp(new(big.Int).Add(amt, gasFees)) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount + gas fees", http.StatusBadRequest)
		}
	} else {
		if senderBalance.Cmp(amt) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount", http.StatusBadRequest)
		}
		if signerBalance.Cmp(gasFees) < 0 {
			return false, ginierr.New("insufficient balance in signer's account for gas fees", http.StatusBadRequest)
		}
	}

	// Allowance check
	if allowance.Cmp(amt) < 0 {
		return false, ginierr.New("insufficient allowance for spender's account for the sender", http.StatusForbidden)
	}

	// Transfer logic with UTXO updates
	if signer == sender && signer == recipient {
		if signer != constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
				return false, err
			}
		}
	} else if signer == sender && signer != recipient {
		if signer == constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
				return false, err
			}
		} else {
			if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, new(big.Int).Add(amt, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amt, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Add(amt, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}

	} else if signer != sender && signer == recipient {
		if signer == constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
				return false, err
			}
		} else if sender == constants.KalpFoundationAddress {
			if amt.Cmp(gasFees) > 0 {
				if err = internal.AddUtxo(ctx, signer, new(big.Int).Sub(amt, gasFees)); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Sub(amt, gasFees)); err != nil {
					return false, err
				}
			} else if amt.Cmp(gasFees) < 0 {
				if err = internal.RemoveUtxo(ctx, signer, new(big.Int).Sub(gasFees, amt)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, sender, new(big.Int).Sub(gasFees, amt)); err != nil {
					return false, err
				}
			}
		} else {
			if amt.Cmp(gasFees) > 0 {
				if err = internal.AddUtxo(ctx, signer, new(big.Int).Sub(amt, gasFees)); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			} else if amt.Cmp(gasFees) == 0 {
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, signer, new(big.Int).Sub(gasFees, amt)); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}

	} else if signer != sender && signer != recipient {
		if sender == recipient {
			if sender == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		} else {
			if signer == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
					return false, err
				}
			} else if sender == constants.KalpFoundationAddress {
				if amt.Cmp(gasFees) > 0 {
					if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
						return false, err
					}
					if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Sub(amt, gasFees)); err != nil {
						return false, err
					}
					if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
						return false, err
					}
				} else if amt.Cmp(gasFees) == 0 {
					if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
						return false, err
					}
					if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
						return false, err
					}
				} else {
					if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
						return false, err
					}
					if err = internal.AddUtxo(ctx, sender, new(big.Int).Sub(gasFees, amt)); err != nil {
						return false, err
					}
					if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
						return false, err
					}
				}
			} else if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amt, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	}

	// Update allowance
	err = internal.UpdateAllowance(ctx, sender, spender, amt.String())
	if err != nil {
		return false, err
	}
	logger.Log.Info("TransferFrom Invoked complete... transferred ", amt, " tokens from: ", sender, " to: ", recipient, " spender: ", spender)

	// Emit transfer event
	eventPayload := map[string]interface{}{
		"from":  sender,
		"to":    recipient,
		"value": amt.String(),
	}
	eventBytes, _ := json.Marshal(eventPayload)
	_ = ctx.SetEvent("Transfer", eventBytes)

	return true, nil
}

func (s *SmartContract) Allowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string) (string, error) {

	allowance, err := models.GetAllowance(ctx, owner, spender)
	if err != nil {
		return "", fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err) //big.NewInt(0).String(), fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err)
	}
	return allowance, nil
}

func (s *SmartContract) TotalSupply(ctx kalpsdk.TransactionContextInterface) (string, error) {
	return constants.TotalSupply, nil
}

func (s *SmartContract) GetBridgeContract(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, e := ctx.GetState(constants.BridgeContractKey)
	if e != nil {
		err := ginierr.ErrFailedToGetState(e)
		logger.Log.Error(err.FullError())
		return "", err
	}
	return string(bytes), nil
}

func (s *SmartContract) SetBridgeContract(ctx kalpsdk.TransactionContextInterface, contract string) error {

	// checking if signer is kalp foundation else return error
	if signerKalp, err := helper.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the bridge contract", http.StatusUnauthorized)
	}
	e := ctx.PutStateWithoutKYC(constants.BridgeContractKey, []byte(contract))
	if e != nil {
		err := ginierr.ErrFailedToGetState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func (s *SmartContract) GetVestingContract(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, e := ctx.GetState(constants.VestingContractKey)
	if e != nil {
		err := ginierr.ErrFailedToGetState(e)
		logger.Log.Error(err.FullError())
		return "", err
	}
	return string(bytes), nil
}

// TODO: needs to be deleted just for testing purposes
// func (s *SmartContract) DeleteDocTypes(ctx kalpsdk.TransactionContextInterface, queryString string) (string, error) {

// 	logger := kalpsdk.NewLogger()

// 	// queryString := `{"selector":{"DocType":"` + docType + `","Id":"GINI"}}`

// 	logger.Infof("queryString: %s\n", queryString)
// 	resultsIterator, err := ctx.GetQueryResult(queryString)
// 	if err != nil {
// 		return "fail", fmt.Errorf("err:failed to fetch UTXO tokens for: %v", err)
// 	}
// 	if !resultsIterator.HasNext() {
// 		return "fail", fmt.Errorf("error with status code %v, err:no records to delete", http.StatusInternalServerError)

// 	}

// 	for resultsIterator.HasNext() {
// 		queryResult, err := resultsIterator.Next()
// 		if err != nil {
// 			return "fail", fmt.Errorf("error with status code %v, err:failed to fetch unlocked tokens: %v ", http.StatusInternalServerError, err)
// 		}

// 		logger.Infof("deleting %s\n", queryResult.Key)

// 		if err = ctx.DelStateWithoutKYC(queryResult.Key); err != nil {
// 			logger.Errorf("Error in deleting %s\n", err.Error())
// 		}
// 	}
// 	return "success", nil

// }
