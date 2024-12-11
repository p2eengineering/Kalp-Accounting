package chaincode

import (
	//Standard Libs

	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/internal"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
	"math/big"
	"net/http"
	"strings"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

type SmartContract struct {
	kalpsdk.Contract
}

func (s *SmartContract) InitLedger(ctx kalpsdk.TransactionContextInterface) error {

	logger.Log.Infof("InitLedger invoked...")
	return nil
}

// Initialize initializes smart contract
func (s *SmartContract) Initialize(ctx kalpsdk.TransactionContextInterface, name string, symbol string, vestingContract string) (bool, error) {
	//check contract name & symbol are not already set, client is not authorized to change them once intitialized

	logger.Log.Infoln("Initializing smart contract")

	// checking if signer is kalp foundation else return error
	if signer, e := ctx.GetUserID(); e != nil {
		logger.Log.Errorf("Error getting user ID: %v", e)
		return false, ginierr.ErrFailedToGetClientID
	} else if signer != constants.KalpFoundationAddress {
		return false, ginierr.ErrOnlyFoundationHasAccess
	}

	// checking if contract is already initialized
	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		logger.Log.Errorf("Error in GetState %s: %v", constants.NameKey, e)
		return false, ginierr.ErrFailedToGetName
	} else if bytes != nil {
		logger.Log.Errorf("Found name key: %v, cannot initialize again", string(bytes))
		return false, ginierr.New("contract already initialized", http.StatusBadRequest)
	}

	if !internal.IsValidAddress(vestingContract) {
		return false, ginierr.ErrIncorrectAddress
	}

	// Checking if kalp foundation & gateway admin are KYC'd
	if kyced, e := ctx.GetKYC(constants.KalpFoundationAddress); e != nil {
		err := ginierr.NewWithError(e, "Error fetching KYC status of foundation", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Foundation is not KYC'd", http.StatusBadRequest)
	}
	if kyced, e := ctx.GetKYC(constants.KalpGateWayAdminAddress); e != nil {
		err := ginierr.NewWithError(e, "Error fetching KYC status of Gateway Admin", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Gateway Admin is not KYC'd", http.StatusBadRequest)
	}

	// intializing roles for kalp foundation & gateway admin
	if _, err := internal.InitializeRoles(ctx, constants.KalpFoundationAddress, constants.KalpFoundationRole); err != nil {
		logger.Log.Errorf("error in initializing roles: %v\n", err)
		return false, ginierr.ErrInitializingRoles
	}
	if _, err := internal.InitializeRoles(ctx, constants.KalpGateWayAdminAddress, constants.KalpGateWayAdminRole); err != nil {
		logger.Log.Errorf("error in initializing roles: %v\n", err)
		return false, ginierr.ErrInitializingRoles
	}

	// minting initial tokens
	if err := internal.Mint(ctx, constants.KalpFoundationAddress, constants.InitialFoundationBalance); err != nil {
		logger.Log.Errorf("error with status code %v,error in minting: %v\n", http.StatusInternalServerError, err)
		return false, ginierr.ErrMinitingTokens
	}
	if err := internal.Mint(ctx, vestingContract, constants.InitialVestingContractBalance); err != nil {
		logger.Log.Errorf("error with status code %v,error in minting: %v\n", http.StatusInternalServerError, err)
		return false, ginierr.ErrMinitingTokens
	}

	// storing name, symbol and initial gas fees
	if e := ctx.PutStateWithoutKYC(constants.NameKey, []byte(name)); e != nil {
		err := ginierr.NewWithError(e, "failed to set token name", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.SymbolKey, []byte(symbol)); e != nil {
		err := ginierr.NewWithError(e, "failed to set symbol", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.GasFeesKey, []byte(constants.InitialGasFees)); e != nil {
		err := ginierr.NewWithError(e, "failed to set gas fees", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	logger.Log.Infoln("Initializing complete")
	return true, nil
}

func (s *SmartContract) Allow(ctx kalpsdk.TransactionContextInterface, address string) error {

	logger.Log.Infof("Allow invoked for address: %s", address)

	signer, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	if signer != constants.KalpFoundationAddress {
		return ginierr.ErrOnlyFoundationHasAccess
	}

	if denied, err := internal.IsDenied(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error checking if address already allowed: %v", http.StatusInternalServerError, err)
	} else if !denied {
		return ginierr.ErrAlreadyAllowed
	}

	if err := internal.AllowAddress(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error allowing address: %v", http.StatusInternalServerError, err)
	}
	return nil
}

func (s *SmartContract) Deny(ctx kalpsdk.TransactionContextInterface, address string) error {

	logger.Log.Infof("Deny invoked for address: %s", address)

	signer, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	if signer != constants.KalpFoundationAddress {
		return ginierr.ErrOnlyFoundationHasAccess
	}
	// TODO: Ask if FoundationAdmin or GatewayAdmin can be denied
	if denied, err := internal.IsDenied(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error checking if address already denied: %v", http.StatusInternalServerError, err)
	} else if denied {
		return ginierr.ErrAlreadyDenied
	}
	if err := internal.DenyAddress(ctx, address); err != nil {
		return fmt.Errorf("error with status code %v, error denying address: %v", http.StatusInternalServerError, err)
	}
	return nil
}

func (s *SmartContract) Name(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.NameKey)
	if err != nil {
		return "", ginierr.ErrFailedToGetName
	}
	return string(bytes), nil
}

func (s *SmartContract) Symbol(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.SymbolKey)
	if err != nil {
		return "", ginierr.ErrFailedToGetSymbol
	}
	return string(bytes), nil
}

func (s *SmartContract) Decimals(ctx kalpsdk.TransactionContextInterface) uint8 {
	return 18
}

func (s *SmartContract) GetGasFees(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.GasFeesKey)
	if err != nil {
		// return "", fmt.Errorf("failed to get Name: %v", err)
		fmt.Printf("failed to get Gas Fee: %v", err)
		return "", fmt.Errorf("failed to get Gas Fee: %v", err)
	}
	if bytes == nil {
		return "", fmt.Errorf("gas fee not set")
	}
	return string(bytes), nil
}

func (s *SmartContract) SetGasFees(ctx kalpsdk.TransactionContextInterface, gasFees string) error {

	operator, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	userRole, err := internal.GetUserRoles(ctx, operator)
	if err != nil {
		logger.Log.Infof("error checking operator's role: %v", err)
		return fmt.Errorf("error checking operator's role: %v", err)
	}
	logger.Log.Infof("useRole: %s\n", userRole)
	if userRole != constants.GasFeesAdminRole {
		return fmt.Errorf("error with status code %v, error: only gas fees admin is allowed to update gas fees", http.StatusInternalServerError)
	}
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

func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, address string, amount string) (bool, error) {

	logger.Log.Info("Transfer---->")
	address = strings.Trim(address, " ")
	if address == "" {
		return false, fmt.Errorf("invalid input address")
	}

	sender, err := ctx.GetUserID()
	if err != nil {
		return false, fmt.Errorf("error in getting user id: %v", err)
	}
	userRole, err := internal.GetUserRoles(ctx, sender)
	if err != nil {
		logger.Log.Infof("error checking user's role: %v", err)
		return false, fmt.Errorf("error checking user's role:: %v", err)
	}
	if len(address) != 40 && userRole != constants.KalpGateWayAdminRole {
		return false, fmt.Errorf("address must be 40 characters long")
	}
	if strings.ContainsAny(address, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") && userRole != constants.KalpGateWayAdminRole {
		return false, fmt.Errorf("invalid address")
	}
	gasFees, err := s.GetGasFees(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get gas gee: %v", err)
	}
	gasFeesAmount, su := big.NewInt(0).SetString(gasFees, 10)
	if !su {
		return false, fmt.Errorf("gasfee can't be converted to big int")
	}
	validateAmount, su := big.NewInt(0).SetString(amount, 10)
	if !su {
		logger.Log.Infof("Amount can't be converted to string")
		return false, fmt.Errorf("error with status code %v, invalid Amount %v", http.StatusBadRequest, amount)
	}
	if validateAmount.Cmp(big.NewInt(0)) == -1 || validateAmount.Cmp(big.NewInt(0)) == 0 { // <= 0 {
		return false, fmt.Errorf("error with status code %v, invalid Amount %v", http.StatusBadRequest, amount)
	}
	logger.Log.Infof("useRole: %s\n", userRole)
	// Covers below 2 scenarios where gateway deducts gas fees and transfers to kalp foundation:
	// 1. when Dapp/users sends non-GINI transactions via gateway
	// 2. when HandleBridgeToken from bridge contract is called by Bridge Admin
	if userRole == constants.KalpGateWayAdminRole {
		var send models.Sender
		errs := json.Unmarshal([]byte(address), &send)
		if errs != nil {
			logger.Log.Info("internal error: error in parsing sender data")
			return false, fmt.Errorf("internal error: error in parsing sender data")
		}
		if len(send.Sender) != 40 {
			return false, fmt.Errorf("address must be 40 characters long")
		}
		if strings.ContainsAny(send.Sender, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") {
			return false, fmt.Errorf("invalid address")
		}
		if send.Sender != constants.KalpFoundationAddress {
			gRemoveAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")

				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}
			err = internal.RemoveUtxo(ctx, send.Sender, gRemoveAmount)
			if err != nil {
				logger.Log.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}
			gAddAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")

				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}
			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gAddAmount)
			if err != nil {
				logger.Log.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Log.Infof("foundation transfer : %s\n", userRole)
		}
	} else if b, err := internal.IsCallerKalpBridge(ctx, constants.BridgeContractAddress); b && err == nil {
		// In this scenario transfer function is invoked fron Withdraw token funtion from bridge contract address
		logger.Log.Infof("sender address changed to Bridge contract addres: \n", constants.BridgeContractAddress)
		// In this scenario sender is kalp foundation is bridgeing from WithdrawToken Function,
		// will credit amount to kalp foundation and remove amount from sender without gas fees
		if sender == constants.KalpFoundationAddress {
			sender = constants.BridgeContractAddress
			subAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}

			err = internal.RemoveUtxo(ctx, sender, subAmount)
			if err != nil {
				logger.Log.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}
			addAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}
			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, addAmount)
			if err != nil {
				logger.Log.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Log.Infof("bridge transfer to foundation : %s\n", constants.KalpFoundationAddress)
		} else {
			// In this scenario sender is Kalp Bridge we will credit gas fees to kalp foundation and remove amount from bridge contract
			// address. Reciver will recieve amount after gas fees deduction
			sender = constants.BridgeContractAddress
			removeAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}
			if removeAmount.Cmp(gasFeesAmount) == -1 || removeAmount.Cmp(gasFeesAmount) == 0 {
				return false, fmt.Errorf("error with status code %v, error:bridge amount can not be less than equal to gas fee", http.StatusBadRequest)
			}
			err = internal.RemoveUtxo(ctx, sender, removeAmount)
			if err != nil {
				logger.Log.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}
			addAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Log.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}

			bridgedAmount := addAmount.Sub(addAmount, gasFeesAmount)
			logger.Log.Infof("bridgedAmount :%v", bridgedAmount)
			err = internal.AddUtxo(ctx, address, bridgedAmount)
			if err != nil {
				logger.Log.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFeesAmount)
			if err != nil {
				logger.Log.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Log.Infof("bridge transfer to normal user : %s\n", userRole)
		}
	} else if sender == constants.KalpFoundationAddress && address == constants.KalpFoundationAddress {
		//In this scenario sender is kalp foundation and address is the kalp foundation so no addition or removal is required
		logger.Log.Infof("foundation transfer to foundation : %s address:%s\n", sender, address)

	} else if sender == constants.KalpFoundationAddress {
		//In this scenario sender is kalp foundation and address is the reciver so no gas fees deduction in code
		subAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		err := internal.RemoveUtxo(ctx, sender, subAmount)
		if err != nil {
			logger.Log.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("transfer remove err: %v", err)
		}
		addAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		err = internal.AddUtxo(ctx, address, addAmount)
		if err != nil {
			logger.Log.Infof("err: %v\n", err)
			return false, fmt.Errorf("transfer add err: %v", err)
		}
		logger.Log.Infof("foundation transfer to user : %s\n", userRole)

	} else if address == constants.KalpFoundationAddress {
		//In this scenario sender is normal user and address is the kap foundation so gas fees+amount will be credited to kalp foundation
		removeAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("removeAmount can't be converted to string ")
			return false, fmt.Errorf("removeAmount can't be converted to string: %v ", err)
		}
		err := internal.RemoveUtxo(ctx, sender, removeAmount)
		if err != nil {
			logger.Log.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("transfer remove err: %v", err)
		}
		addAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		err = internal.AddUtxo(ctx, address, addAmount)
		if err != nil {
			logger.Log.Infof("err: %v\n", err)
			return false, fmt.Errorf("transfer add err: %v", err)
		}
		logger.Log.Infof("foundation transfer to user : %s\n", userRole)
	} else {
		//This is normal scenario where amount will be deducted from sender and amount-gas fess will credited to address and gas fees will be credited to kalp foundation
		logger.Log.Infof("operator-->", sender)
		logger.Log.Info("transfer transferAmount")
		if sender == address {
			return false, fmt.Errorf("transfer to self not alllowed")
		}
		transferAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("Amount can't be converted to string")
			return false, fmt.Errorf("error with status code %v,Amount can't be converted to string", http.StatusConflict)
		}
		if transferAmount.Cmp(gasFeesAmount) == -1 || transferAmount.Cmp(gasFeesAmount) == 0 {
			return false, fmt.Errorf("error with status code %v, error:transfer amount can not be less than equal to gas fee", http.StatusBadRequest)
		}
		logger.Log.Infof("transferAmount %v\n", transferAmount)
		logger.Log.Infof("gasFeesAmount %v\n", gasFeesAmount)
		// Withdraw the funds from the sender address
		err = internal.RemoveUtxo(ctx, sender, transferAmount)
		if err != nil {
			logger.Log.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("error with status code %v, error:error while reducing balance %v", http.StatusBadRequest, err)
		}
		addAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Log.Infof("transfer Amount can't be converted to string ")
			return false, fmt.Errorf("error with status code %v,transaction %v already accounted", http.StatusConflict, transferAmount)
		}
		logger.Log.Infof("Add amount %v\n", addAmount)
		addAmounts := addAmount.Sub(addAmount, gasFeesAmount)
		// Deposit the fund to the recipient address
		err = internal.AddUtxo(ctx, address, addAmounts)
		if err != nil {
			logger.Log.Infof("err: %v\n", err)
			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
		}
		logger.Log.Infof("gasFeesAmount %v\n", gasFeesAmount)
		err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFeesAmount)
		if err != nil {
			logger.Log.Infof("err: %v\n", err)
			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
		}
	}
	transferSingleEvent := models.TransferSingle{Operator: sender, From: sender, To: address, Value: amount}
	if err := internal.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		logger.Log.Infof("err: %v\n", err)
		return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
	}
	return true, nil

}

func (s *SmartContract) BalanceOf(ctx kalpsdk.TransactionContextInterface, owner string) (string, error) {

	owner = strings.Trim(owner, " ")
	if owner == "" {
		return "0", fmt.Errorf("invalid input account is required")
	}
	if !internal.IsValidAddress(owner) {
		return "0", ginierr.ErrIncorrectAddress
	}
	amt, err := internal.GetTotalUTXO(ctx, owner)
	if err != nil {
		return "0", fmt.Errorf("error fetching balance: %v", err)
	}
	logger.Log.Infof("total balance%v owner:%s\n", amt, owner)

	return amt, nil
}

func (s *SmartContract) Approve(ctx kalpsdk.TransactionContextInterface, spender string, value string) (bool, error) {

	if err := internal.Approve(ctx, spender, value); err != nil {
		logger.Log.Infof("error unable to approve funds: %v\n", err)
		return false, err
	}
	return true, nil
}

func (s *SmartContract) TransferFrom(ctx kalpsdk.TransactionContextInterface, from string, to string, value string) (bool, error) {

	logger.Log.Info("TransferFrom---->")
	spender, err := ctx.GetUserID()
	if err != nil {
		return false, fmt.Errorf("error iin getting spender's id: %v", err)
	}
	err = internal.TransferUTXOFrom(ctx, []string{from}, []string{spender}, to, value, constants.UTXO)
	if err != nil {
		logger.Log.Infof("err: %v\n", err)
		return false, fmt.Errorf("error: unable to transfer funds: %v", err)
	}
	return true, nil
}

func (s *SmartContract) Allowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string) (string, error) {

	allowance, err := internal.Allowance(ctx, owner, spender)
	if err != nil {
		return "", fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err) //big.NewInt(0).String(), fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err)
	}
	return allowance, nil
}

func (s *SmartContract) TotalSupply(ctx kalpsdk.TransactionContextInterface) (string, error) {
	return constants.TotalSupply, nil
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
