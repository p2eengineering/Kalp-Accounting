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

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
	"golang.org/x/exp/slices"
)

type SmartContract struct {
	kalpsdk.Contract
}

func (s *SmartContract) Initialize(ctx kalpsdk.TransactionContextInterface, name string, symbol string, vestingContractAddress string) (bool, error) {
	logger.Log.Infoln("Initializing smart contract... with arguments", name, symbol, vestingContractAddress)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return false, err
	} else if !signerKalp {
		return false, ginierr.New("Only Kalp Foundation can initialize the contract", http.StatusUnauthorized)
	}

	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		return false, ginierr.ErrFailedToGetKey(constants.NameKey)
	} else if bytes != nil {
		return false, ginierr.New(fmt.Sprintf("cannot initialize again,%s already set: %s", constants.NameKey, string(bytes)), http.StatusBadRequest)
	}
	if bytes, e := ctx.GetState(constants.SymbolKey); e != nil {
		return false, ginierr.ErrFailedToGetKey(constants.SymbolKey)
	} else if bytes != nil {
		return false, ginierr.New(fmt.Sprintf("cannot initialize again,%s already set: %s", constants.SymbolKey, string(bytes)), http.StatusBadRequest)
	}

	isContract, err := helper.IsContractAddress(vestingContractAddress)
	if err != nil {
		return false, err
	}
	if !isContract {
		return false, ginierr.ErrInvalidContractAddress(vestingContractAddress)
	}

	if kyced, e := ctx.GetKYC(constants.KalpFoundationAddress); e != nil {
		err := ginierr.NewInternalError(e, "Error fetching KYC status of foundation", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Foundation is not KYC'd", http.StatusBadRequest)
	}
	if kyced, e := ctx.GetKYC(constants.KalpGateWayAdminAddress); e != nil {
		err := ginierr.NewInternalError(e, "Error fetching KYC status of Gateway Admin", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	} else if !kyced {
		return false, ginierr.New("Gateway Admin is not KYC'd", http.StatusBadRequest)
	}

	if _, err := internal.InitializeRoles(ctx, constants.KalpGateWayAdminAddress, constants.KalpGateWayAdminRole); err != nil {
		return false, err
	}

	if err := internal.Mint(ctx, []string{constants.KalpFoundationAddress, vestingContractAddress}, []string{constants.InitialFoundationBalance, constants.InitialVestingContractBalance}); err != nil {
		return false, err
	}

	if e := ctx.PutStateWithoutKYC(constants.NameKey, []byte(name)); e != nil {
		err := ginierr.NewInternalError(e, "failed to set token name: "+name, http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.SymbolKey, []byte(symbol)); e != nil {
		err := ginierr.NewInternalError(e, "failed to set symbol: "+symbol, http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.GasFeesKey, []byte(constants.InitialGasFees)); e != nil {
		err := ginierr.NewInternalError(e, "failed to set gas fees: "+constants.InitialGasFees, http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(constants.VestingContractKey, []byte(vestingContractAddress)); e != nil {
		err := ginierr.NewInternalError(e, "failed to set vesting Contract: "+vestingContractAddress, http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if err := s.SetGatewayMaxFee(ctx, constants.InitialGatewayMaxGasFee); err != nil {
		return false, err
	}
	if err := s.SetBridgeContract(ctx, constants.InitialBridgeContractAddress); err != nil {
		return false, err
	}
	logger.Log.Infoln("Initialize Invoked complete")
	return true, nil
}

func (s *SmartContract) SetUserRoles(ctx kalpsdk.TransactionContextInterface, data string) error {
	logger.Log.Info("SetUserRoles........", data)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the roles", http.StatusUnauthorized)
	}

	var userRole models.UserRole
	errs := json.Unmarshal([]byte(data), &userRole)
	if errs != nil {
		return fmt.Errorf("failed to parse data: %v", errs)
	}

	if userRole.Id == "" {
		return fmt.Errorf("user Id can not be null")
	}

	isUser, err := helper.IsUserAddress(userRole.Id)
	if err != nil {
		return err
	}

	if !isUser {
		return ginierr.ErrInvalidUserAddress(userRole.Id)
	}

	if userRole.Role == "" {
		return fmt.Errorf("role can not be null")
	}

	ValidRoles := []string{constants.KalpGateWayAdminRole}
	if !slices.Contains(ValidRoles, userRole.Role) {
		return fmt.Errorf("invalid input role")
	}

	if kyced, e := ctx.GetKYC(userRole.Id); e != nil {
		err := ginierr.NewInternalError(e, "Error fetching KYC status of user for creating Gateway admin", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	} else if !kyced {
		return ginierr.New("User is not KYC'd", http.StatusBadRequest)
	}

	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userRole.Id, constants.KalpGateWayAdminRole})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	usrRoleJSON, err := json.Marshal(userRole)
	if err != nil {
		return fmt.Errorf("unable to Marshal userRole struct : %v", err)
	}

	if e := ctx.PutStateWithoutKYC(key, usrRoleJSON); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role struct: %v", e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	return nil
}

func (s *SmartContract) DeleteUserRoles(ctx kalpsdk.TransactionContextInterface, userID string) error {
	logger.Log.Info("DeleteUserRoles........", userID)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the roles", http.StatusUnauthorized)
	}

	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userID, constants.KalpGateWayAdminRole})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	existingRoleBytes, e := ctx.GetState(key)
	if e != nil || existingRoleBytes == nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("user role not found for userID %s", userID), http.StatusNotFound)
		logger.Log.Errorf(err.FullError())
		return err
	}

	var userRole models.UserRole
	if err := json.Unmarshal(existingRoleBytes, &userRole); err != nil {
		return fmt.Errorf("failed to unmarshal user role: %v", err)
	}

	if userRole.Role == constants.KalpFoundationRole {
		return fmt.Errorf("foundation role cannot be deleted")
	}

	if e := ctx.DelStateWithoutKYC(key); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to delete user role struct: %v", e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	return nil
}

func (s *SmartContract) Allow(ctx kalpsdk.TransactionContextInterface, address string) error {

	logger.Log.Infof("Allow invoked for address: %s", address)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
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

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
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
	logger.Log.Infoln("SetGasFees... with arguments", gasFees)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the gas fees", http.StatusUnauthorized)
	}
	if e := ctx.PutStateWithoutKYC(constants.GasFeesKey, []byte(gasFees)); e != nil {
		err := ginierr.ErrFailedToPutState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func (s *SmartContract) BalanceOf(ctx kalpsdk.TransactionContextInterface, account string) (string, error) {
	logger.Log.Infoln("BalanceOf... with arguments", account)

	isValidAddress, err := helper.IsValidAddress(account)
	if err != nil {
		return "0", err
	}
	if !isValidAddress {
		return "0", ginierr.ErrInvalidAddress(account)
	}
	amt, err := internal.GetTotalUTXO(ctx, account)
	if err != nil {
		return "0", fmt.Errorf("error fetching balance: %v", err)
	}
	logger.Log.Infof("total balance:%v account:%s\n", amt, account)

	return amt, nil
}

func (s *SmartContract) balance(ctx kalpsdk.TransactionContextInterface, account string) (*big.Int, error) {
	logger.Log.Infoln("balance... with arguments", account)

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
	signer, err := helper.GetUserId(ctx)
	if err != nil {
		return false, ginierr.ErrFailedToGetPublicAddress
	}
	if err := events.EmitApproval(ctx, signer, spender, amount); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, recipient string, amount string) (bool, error) {
	logger.Log.Info("Transfer operation initiated", recipient, amount)

	foundationAddress, err := internal.GetFoundationAddress(ctx)
	if err != nil {
		err := ginierr.NewInternalError(err, "error getting foundationAddress", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	signer, err := helper.GetUserId(ctx)
	if err != nil {
		err := ginierr.NewInternalError(err, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	isGatewayAdmin, err := internal.IsGatewayAdminAddress(ctx, signer)
	if err != nil {
		return false, err
	}

	sender := signer
	var gasFees, actualAmount *big.Int
	if gasFeesString, err := s.GetGasFees(ctx); err != nil {
		return false, err
	} else if val, ok := big.NewInt(0).SetString(gasFeesString, 10); !ok {
		return false, ginierr.New("invalid gas fees found:"+gasFeesString, http.StatusInternalServerError)
	} else {
		gasFees = val
	}

	amountInInt, ok := big.NewInt(0).SetString(amount, 10)
	if !ok || amountInInt.Cmp(big.NewInt(0)) != 1 {
		return false, ginierr.ErrInvalidAmount(amount)
	}

	isValidAddress, err := helper.IsValidAddress(recipient)
	if err != nil {
		return false, err
	}
	if isValidAddress {
		if amountInInt.Cmp(gasFees) < 0 {
			return false, ginierr.ErrInvalidAmount(amount)
		}

	} else if isGatewayAdmin {
		var gasDeductionAccount models.Sender
		err := json.Unmarshal([]byte(recipient), &gasDeductionAccount)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal recipient: %v", err)
		}

		isUser, err := helper.IsUserAddress(gasDeductionAccount.Sender)
		if err != nil {
			return false, err
		}

		if !isUser {
			return false, ginierr.ErrInvalidAddress(gasDeductionAccount.Sender)
		}

		sender = gasDeductionAccount.Sender
		recipient = foundationAddress
		gasFees = big.NewInt(0)

		var strGatewayMaxGasFee string
		if strGatewayMaxGasFee, err = s.GetGatewayMaxFee(ctx); err != nil {
			return false, err
		}
		gatewayMaxGasFee, ok := big.NewInt(0).SetString(strGatewayMaxGasFee, 10)
		if !ok || gatewayMaxGasFee.Cmp(big.NewInt(0)) < 0 {
			return false, ginierr.ErrInvalidAmount(strGatewayMaxGasFee)
		}

		if !helper.IsAmountProper(amount) || amountInInt.Cmp(gatewayMaxGasFee) > 0 {
			return false, ginierr.ErrInvalidAmount(amount)
		}
	} else {
		return false, ginierr.ErrInvalidAddress(recipient)
	}

	actualAmount = new(big.Int).Sub(amountInInt, gasFees)
	logger.Log.Info("actualAmount => ", actualAmount)

	var e error

	vestingContract, err := s.GetVestingContract(ctx)
	if err != nil {
		return false, err
	}
	bridgeContract, err := s.GetBridgeContract(ctx)
	if err != nil {
		return false, err
	}

	calledContractAddress, err := internal.GetCalledContractAddress(ctx)
	logger.Log.Info("calledContractAddress => ", calledContractAddress, err, s.GetName(), vestingContract, bridgeContract)
	if err != nil {
		return false, err
	}

	if calledContractAddress != s.GetName() {
		if calledContractAddress != bridgeContract && calledContractAddress != vestingContract {
			err := ginierr.New("The called contract is not bridge contract or vesting contract", http.StatusForbidden)
			logger.Log.Error(err.FullError())
			return false, err
		}
		sender = calledContractAddress
	}

	isContract, err := helper.IsContractAddress(signer)
	if err != nil {
		return false, err
	}

	isUser, err := helper.IsUserAddress(signer)
	if err != nil {
		return false, err
	}

	if isContract {
		return false, ginierr.New("signer cannot be a contract", http.StatusBadRequest)
	} else if !isUser {
		return false, ginierr.ErrInvalidAddress(signer)
	}

	isSenderContract, err := helper.IsContractAddress(sender)
	if err != nil {
		return false, err
	}

	isRecipientContract, err := helper.IsContractAddress(recipient)
	if err != nil {
		return false, err
	}

	if isSenderContract && isRecipientContract {
		return false, ginierr.New("both sender and recipient cannot be contracts", http.StatusBadRequest)
	}
	isValidAddress, err = helper.IsValidAddress(sender)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(sender)
	}

	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.ErrDeniedAddress(signer)
	}

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

	var kycSender, kycSigner bool
	if kycSender, e = ctx.GetKYC(sender); e != nil {
		err := ginierr.NewInternalError(e, "error fetching KYC for sender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if kycSigner, e = ctx.GetKYC(signer); e != nil {
		err := ginierr.NewInternalError(e, "error fetching KYC for signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if !(kycSender || kycSigner) {
		err := ginierr.New(fmt.Sprintf("IsSender kyced: %v, IsSigner kyced: %v", kycSender, kycSigner), http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

	senderBalance, err := s.balance(ctx, sender)
	if err != nil {
		return false, err
	}

	if senderBalance.Cmp(amountInInt) < 0 {
		return false, ginierr.New("insufficient balance in sender's account for amount", http.StatusBadRequest)
	}

	if sender == signer && sender == recipient {
		if sender != foundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, gasFees); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
				return false, err
			}
		}
	} else if sender == signer && sender != recipient {
		if sender == foundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, actualAmount); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, actualAmount); err != nil {
				return false, err
			}
		} else {
			if recipient == foundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amountInInt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amountInInt); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, sender, amountInInt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, actualAmount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	} else {
		if isGatewayAdmin {
			if sender != foundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, actualAmount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, actualAmount); err != nil {
					return false, err
				}
			}
		} else {
			if err = internal.RemoveUtxo(ctx, sender, amountInInt); err != nil {
				return false, err
			}

			if recipient == foundationAddress {
				if err = internal.AddUtxo(ctx, recipient, amountInInt); err != nil {
					return false, err
				}
			} else {
				if err = internal.AddUtxo(ctx, recipient, actualAmount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	}

	if err := events.EmitTransfer(ctx, sender, recipient, amount); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SmartContract) TransferFrom(ctx kalpsdk.TransactionContextInterface, sender string, recipient string, amount string) (bool, error) {
	logger.Log.Infoln("TransferFrom invoked.... with arguments", sender, recipient, amount)

	var signer string
	var e error

	foundationAddress, err := internal.GetFoundationAddress(ctx)
	if err != nil {
		err := ginierr.NewInternalError(err, "error getting foundationAddress", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if signer, e = helper.GetUserId(ctx); e != nil {
		err := ginierr.NewInternalError(e, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	isContract, err := helper.IsContractAddress(signer)
	if err != nil {
		return false, err
	}
	if isContract {
		return false, ginierr.New("signer cannot be a contract", http.StatusBadRequest)
	}
	isUser, err := helper.IsUserAddress(signer)
	if err != nil {
		return false, err
	}

	if !isUser {
		return false, ginierr.ErrInvalidUserAddress(signer)
	}
	isSenderContract, err := helper.IsContractAddress(sender)
	if err != nil {
		return false, err
	}

	isRecipientContract, err := helper.IsContractAddress(recipient)
	if err != nil {
		return false, err
	}

	if isSenderContract && isRecipientContract {
		return false, ginierr.New("both sender and recipient cannot be contracts", http.StatusBadRequest)
	}
	isValidAddress, err := helper.IsValidAddress(sender)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(sender)
	}
	isValidAddress, err = helper.IsValidAddress(recipient)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(recipient)
	}
	if !helper.IsAmountProper(amount) {
		return false, ginierr.ErrInvalidAmount(amount)
	}

	amt, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		return false, ginierr.ErrInvalidAmount(amount)
	}

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

	spender := signer
	if calledContractAddress != s.GetName() {
		if calledContractAddress == bridgeContract || calledContractAddress == vestingContract {
			spender = calledContractAddress
		} else {
			err := ginierr.New("The called contract is neither bridge contract nor vesting contract", http.StatusBadRequest)
			logger.Log.Error(err.FullError())
			return false, err
		}
	}
	isValidAddress, err = helper.IsValidAddress(spender)
	if err != nil {
		return false, err
	}
	if !isValidAddress {
		return false, ginierr.ErrInvalidAddress(spender)
	}

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

	var kycSender, kycSpender, kycSigner bool
	if kycSender, e = ctx.GetKYC(sender); e != nil {
		err := ginierr.NewInternalError(e, "error fetching KYC for sender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if kycSpender, e = ctx.GetKYC(spender); e != nil {
		err := ginierr.NewInternalError(e, "error fetching KYC for spender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if kycSigner, e = ctx.GetKYC(signer); e != nil {
		err := ginierr.NewInternalError(e, "error fetching KYC for signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	if !(kycSender || kycSpender || kycSigner) {
		err := ginierr.New("None of the sender, spender, or signer is KYC'd", http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

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

	if allowance.Cmp(amt) < 0 {
		return false, ginierr.New(fmt.Sprintf("insufficient allowance for spender's account %s for the sender %s", spender, sender), http.StatusForbidden)
	}
	if spender == bridgeContract || spender == vestingContract {
		if signer != sender {
			return false, ginierr.New("If bridge or vesting contract is the spender then , sender and signer should be same", http.StatusBadRequest)
		}
	}
	if signer == sender && signer == recipient {
		if signer != foundationAddress {
			if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
				return false, err
			}
		}
	} else if signer == sender && signer != recipient {
		if signer == foundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
				return false, err
			}
		} else {
			if recipient == foundationAddress {
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
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}

	} else if signer != sender && signer == recipient {
		if signer == foundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
				return false, err
			}
		} else if sender == foundationAddress {
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
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			} else if amt.Cmp(gasFees) == 0 {
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, signer, new(big.Int).Sub(gasFees, amt)); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}

	} else if signer != sender && signer != recipient {
		if sender == recipient {
			if signer != foundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		} else {
			if signer == foundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amt); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amt); err != nil {
					return false, err
				}
			} else if sender == foundationAddress {
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
			} else if recipient == foundationAddress {
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
				if err = internal.AddUtxo(ctx, foundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	}

	err = internal.UpdateAllowance(ctx, sender, spender, amt.String())
	if err != nil {
		return false, err
	}
	logger.Log.Info("TransferFrom Invoked complete... transferred ", amt, " tokens from: ", sender, " to: ", recipient, " spender: ", spender)

	if err := events.EmitTransfer(ctx, sender, recipient, amount); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SmartContract) Allowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string) (string, error) {
	logger.Log.Infoln("Allowance... with arguments", owner, spender)

	allowance, err := models.GetAllowance(ctx, owner, spender)
	if err != nil {
		return "", fmt.Errorf("error code is %v: failed to get allowance: %v", http.StatusBadRequest, err)
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
	logger.Log.Infoln("SetBridgeContract... with arguments", contract)

	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the bridge contract", http.StatusUnauthorized)
	}
	e := ctx.PutStateWithoutKYC(constants.BridgeContractKey, []byte(contract))
	if e != nil {
		err := ginierr.ErrFailedToPutState(e)
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

func (s *SmartContract) IncreaseAllowance(ctx kalpsdk.TransactionContextInterface, spender string, delta string) (bool, error) {
	logger.Log.Infoln("IncreaseAllowance invoked with spender:", spender, "delta:", delta)

	signer, err := helper.GetUserId(ctx)
	if err != nil {
		return false, ginierr.ErrFailedToGetPublicAddress
	}

	current, err := models.GetAllowance(ctx, signer, spender)
	if err != nil {
		logger.Log.Infof("Error fetching current allowance: %v", err)
		return false, fmt.Errorf("failed to fetch allowance: %w", err)
	}

	// Validate delta using ginierr
	deltaInt := new(big.Int)
	if _, ok := deltaInt.SetString(delta, 10); !ok || deltaInt.Sign() < 0 {
		return false, ginierr.ErrInvalidAmount(delta) // This produces "invalid amount passed: ..."
	}

	currentInt := new(big.Int)
	if _, ok := currentInt.SetString(current, 10); !ok {
		return false, fmt.Errorf("invalid current allowance: %s", current)
	}

	newAmount := new(big.Int).Add(currentInt, deltaInt)
	if err := models.SetAllowance(ctx, spender, newAmount.String()); err != nil {
		return false, fmt.Errorf("failed to set allowance: %w", err)
	}

	if err := events.EmitApproval(ctx, signer, spender, newAmount.String()); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SmartContract) DecreaseAllowance(ctx kalpsdk.TransactionContextInterface, spender string, delta string) (bool, error) {

	logger.Log.Infoln("DecreaseAllowance invoked with spender:", spender, "delta:", delta)

	signer, err := helper.GetUserId(ctx)

	if err != nil {

		return false, ginierr.ErrFailedToGetPublicAddress

	}

	// Fetch current allowance

	current, err := models.GetAllowance(ctx, signer, spender)

	if err != nil {

		logger.Log.Infof("Error fetching current allowance: %v", err)

		return false, fmt.Errorf("failed to fetch allowance: %w", err)

	}

	// Validate delta

	deltaInt := new(big.Int)

	if _, ok := deltaInt.SetString(delta, 10); !ok || deltaInt.Sign() < 0 {

		return false, ginierr.ErrInvalidAmount(delta)

	}

	currentInt := new(big.Int)

	if _, ok := currentInt.SetString(current, 10); !ok {

		return false, fmt.Errorf("invalid current allowance: %s", current)

	}

	// Check for underflow

	if currentInt.Cmp(deltaInt) < 0 {

		return false, ginierr.ErrInsufficientAllowance()

	}

	// Calculate new allowance

	newAmount := new(big.Int).Sub(currentInt, deltaInt)

	// Update allowance

	if err := models.SetAllowance(ctx, spender, newAmount.String()); err != nil {

		logger.Log.Infof("Error updating allowance: %v", err)

		return false, fmt.Errorf("failed to set allowance: %w", err)

	}

	// Emit Approval event

	if err := events.EmitApproval(ctx, signer, spender, newAmount.String()); err != nil {

		return false, err

	}

	return true, nil

}

func (s *SmartContract) SetGatewayMaxFee(ctx kalpsdk.TransactionContextInterface, gatewayMaxFee string) error {
	logger.Log.Infoln("SetGatewayMaxFee... with arguments", gatewayMaxFee)

	// Validate numeric format
	feeInt := new(big.Int)
	if _, ok := feeInt.SetString(gatewayMaxFee, 10); !ok {
		return ginierr.ErrInvalidAmount(gatewayMaxFee)
	}

	// Validate non-negative
	if feeInt.Sign() < 0 {
		return ginierr.ErrInvalidAmount(gatewayMaxFee)
	}

	// Authorization check
	if signerKalp, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !signerKalp {
		return ginierr.New("Only Kalp Foundation can set the gatewayMaxFee", http.StatusUnauthorized)
	}

	// State update
	if e := ctx.PutStateWithoutKYC(constants.GatewayMaxFee, []byte(gatewayMaxFee)); e != nil {
		err := ginierr.ErrFailedToPutState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func (s *SmartContract) GetGatewayMaxFee(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(constants.GatewayMaxFee)
	if err != nil {
		return "", fmt.Errorf("failed to get gatewayMaxFee: %v", err)
	}
	if bytes == nil {
		return "", fmt.Errorf("gatewayMaxFee not set")
	}
	return string(bytes), nil
}

func (s *SmartContract) SetFoundationRole(ctx kalpsdk.TransactionContextInterface, address string) error {
	logger.Log.Info("SetFoundationRole invoked for address:", address)

	// Verify current foundation is the caller
	if isFoundation, err := internal.IsSignerKalpFoundation(ctx); err != nil {
		return err
	} else if !isFoundation {
		return ginierr.New("Only current foundation can transfer role", http.StatusUnauthorized)
	}

	// Validate recipient address format
	isUser, err := helper.IsUserAddress(address)
	if err != nil {
		return err
	}

	if !isUser {
		return ginierr.ErrInvalidUserAddress(address)
	}

	if kyced, e := ctx.GetKYC(address); e != nil {
		err := ginierr.NewInternalError(e, "Error fetching KYC status of foundation", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	} else if !kyced {
		return ginierr.New("address is not KYC'd", http.StatusBadRequest)
	}

	// key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userRole.Id, constants.UserRoleMap})
	// if e != nil {
	// 	err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
	// 	logger.Log.Errorf(err.FullError())
	// 	return err
	// }

	foundationAddress, err := internal.GetFoundationAddress(ctx)
	if err != nil {
		logError := ginierr.NewInternalError(err,
			fmt.Sprintf("Failed to get kalp foundation address : %v", err),
			http.StatusInternalServerError)
		logger.Log.Error(logError.FullError())
		return logError
	}

	// Create foundation role composite key
	foundationKey, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{foundationAddress, constants.UserRoleMap})
	if err != nil {
		logError := ginierr.NewInternalError(err,
			fmt.Sprintf("Failed creating composite key for foundation role: %v", err),
			http.StatusInternalServerError)
		logger.Log.Error(logError.FullError())
		return logError
	}

	if e := ctx.DelStateWithoutKYC(foundationKey); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to delete foudation role struct: %v", e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	newFoundationKey, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{address, constants.UserRoleMap})
	if err != nil {
		logError := ginierr.NewInternalError(err,
			fmt.Sprintf("Failed creating composite key for new foundation role: %v", err),
			http.StatusInternalServerError)
		logger.Log.Error(logError.FullError())
		return logError
	}

	userRole := models.UserRole{
		Id:      address,
		Role:    constants.KalpFoundationRole,
		DocType: constants.UserRoleMap,
	}
	roleJson, e := json.Marshal(userRole)
	if err != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to marshal user role: %s , id: %s ", constants.KalpFoundationRole, address), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	if e := ctx.PutStateWithoutKYC(newFoundationKey, roleJson); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role: %s , id: %s ", constants.KalpFoundationRole, address), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	logger.Log.Info("Foundation role successfully transferred to:", address)
	return nil
}

func (s *SmartContract) GetFoundationRoleAddress(ctx kalpsdk.TransactionContextInterface) (string, error) {
	return internal.GetFoundationAddress(ctx)
}
