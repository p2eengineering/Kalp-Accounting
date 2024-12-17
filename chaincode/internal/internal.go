package internal

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
	"math/big"
	"net/http"

	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
	"golang.org/x/exp/slices"
)

var GetCallingContractAddress = getCallingContractAddressHelper

// CheckCallerIsContract checks if the caller is a contract
func CheckCallerIsContract(ctx kalpsdk.TransactionContextInterface) bool {
	return true
}

// GetCallingContractAddress returns calling contract's address
func getCallingContractAddressHelper(ctx kalpsdk.TransactionContextInterface) (string, error) {
	signedProposal, e := ctx.GetSignedProposal()
	if signedProposal == nil {
		err := ginierr.New("could not retrieve proposal details", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	if e != nil {
		err := ginierr.NewWithInternalError(e, "error in getting signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	data := signedProposal.GetProposalBytes()
	if data == nil {
		err := ginierr.New("error in fetching signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	proposal := &peer.Proposal{}
	e = proto.Unmarshal(data, proposal)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "error in parsing signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	payload := &common.Payload{}
	e = proto.Unmarshal(proposal.Payload, payload)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "error in parsing payload", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	paystring := payload.GetHeader().GetChannelHeader()
	if len(paystring) == 0 {
		err := ginierr.New("channel header is empty", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	logger.Log.Debug("Calling contract address:", paystring)
	contractAddress := helper.FindContractAddress(paystring)
	if contractAddress == "" {
		err := ginierr.New("contract address not found", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	return contractAddress, nil
}

// GetGatewayAdminAddress returns calling gateway admin's address
func GetGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface) string {
	return constants.KalpGateWayAdminAddress
}

// GetKalpFoundationAdminAddress returns calling kalp foundation admin's address
func GetKalpFoundationAdminAddress(ctx kalpsdk.TransactionContextInterface) string {
	return constants.KalpFoundationAddress
}

// DenyAddress adds the given address to the denylist
func DenyAddress(ctx kalpsdk.TransactionContextInterface, address string) error {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if err := ctx.PutStateWithoutKYC(addressDenyKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to put state in deny list: %v", err)
	}
	if err := ctx.SetEvent(constants.Denied, []byte(address)); err != nil {
		return ginierr.ErrFailedToEmitEvent
	}
	return nil
}

// AllowAddress removes the given address from the denylist
func AllowAddress(ctx kalpsdk.TransactionContextInterface, address string) error {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if err := ctx.PutStateWithoutKYC(addressDenyKey, []byte("false")); err != nil {
		return fmt.Errorf("failed to put state in deny list: %v", err)
	}
	if err := ctx.SetEvent(constants.Approved, []byte(address)); err != nil {
		return ginierr.ErrFailedToEmitEvent
	}
	return nil
}

// IsDenied checks if an address is denied
func IsDenied(ctx kalpsdk.TransactionContextInterface, address string) (bool, error) {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return false, fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if bytes, err := ctx.GetState(addressDenyKey); err != nil {
		return false, fmt.Errorf("failed to get state from deny list: %v", err)
	} else if bytes == nil {
		// GetState() returns nil, nil when key is not found
		return false, nil
	} else if string(bytes) == "false" {
		return false, nil
	}
	return true, nil
}

// Mint mints given amount at a given address
func Mint(ctx kalpsdk.TransactionContextInterface, addresses []string, amounts []string) error {

	logger.Log.Infof("Mint operation started")

	// Validate input amount
	accAmount1, ok := big.NewInt(0).SetString(amounts[0], 10)
	if !ok {
		return fmt.Errorf("error with status code %v,can't convert amount to big int %s", http.StatusConflict, amounts)
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 { // if amount is not greater than 0 return error
		return ginierr.ErrInvalidAmount(amounts[0])
	}
	accAmount2, ok := big.NewInt(0).SetString(amounts[1], 10)
	if !ok {
		return fmt.Errorf("error with status code %v,can't convert amount to big int %s", http.StatusConflict, amounts)
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 { // if amount is not greater than 0 return error
		return ginierr.ErrInvalidAmount(amounts[1])
	}

	// Validate input address
	if !helper.IsValidAddress(addresses[0]) {
		return ginierr.ErrIncorrectAddress(addresses[0])
	}
	if !helper.IsValidAddress(addresses[1]) {
		return ginierr.ErrIncorrectAddress(addresses[1])
	}

	// checking if contract is already initialized
	if bytes, err := ctx.GetState(constants.NameKey); err != nil {
		return ginierr.ErrFailedToGetName
	} else if bytes != nil {
		return fmt.Errorf("contract already initialized, minting not allowed")
	}
	if bytes, err := ctx.GetState(constants.SymbolKey); err != nil {
		return ginierr.ErrFailedToGetName
	} else if bytes != nil {
		return fmt.Errorf("contract already initialized, minting not allowed")
	}

	// TODO: check balance if required here

	// Mint tokens
	if err := MintUtxoHelperWithoutKYC(ctx, addresses[0], accAmount1); err != nil {
		return fmt.Errorf("error with status code %v, failed to mint tokens: %v", http.StatusBadRequest, err)
	}
	if err := MintUtxoHelperWithoutKYC(ctx, addresses[1], accAmount2); err != nil {
		return fmt.Errorf("error with status code %v, failed to mint tokens: %v", http.StatusBadRequest, err)
	}
	logger.Log.Infof("MintToken Amount---->%v\n", amounts)
	return nil

}

// As of now, we are not supporting usecases where asset is owned by multiple owners.
func MintUtxoHelperWithoutKYC(ctx kalpsdk.TransactionContextInterface, account string, amount *big.Int) error {
	if account == "0x0" {
		return fmt.Errorf("mint to the zero address")
	}

	fmt.Println("account & amount in mintutxohelper -", account, amount)

	err := AddUtxo(ctx, account, amount)
	if err != nil {
		return err
	}
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  amount.String(),
	}
	utxoJSON, err := json.Marshal(utxo)
	if err != nil {
		return fmt.Errorf("failed to marshal owner with ID %s and account address %s to JSON: %v", constants.GINI, account, err)
	}
	if err := ctx.SetEvent("Mint", utxoJSON); err != nil {
		return ginierr.ErrFailedToEmitEvent
	}
	return nil
}

func AddUtxo(sdk kalpsdk.TransactionContextInterface, account string, iamount interface{}) error {
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
	}
	amount, err := helper.CustomBigIntConvertor(iamount)
	if err != nil {
		return fmt.Errorf("error in CustomBigInt %v", err)
	}
	fmt.Printf("add amount: %v\n", amount)
	fmt.Printf("utxoKey: %v\n", utxoKey)
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  amount.String(),
	}

	utxoJSON, err := json.Marshal(utxo)
	if err != nil {
		return fmt.Errorf("failed to marshal owner with ID %s and account address %s to JSON: %v", constants.GINI, account, err)
	}
	fmt.Printf("utxoJSON: %s\n", utxoJSON)

	err = sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
	if err != nil {
		return fmt.Errorf("failed to put owner with ID %s and account address %s to world state: %v", constants.GINI, account, err)

	}
	return nil
}
func RemoveUtxo(sdk kalpsdk.TransactionContextInterface, account string, iamount interface{}) error {

	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
	}
	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexIdDocType"}`
	amount, err := helper.CustomBigIntConvertor(iamount)
	if err != nil {
		return fmt.Errorf("error in CustomBigInt %v", err)
	}
	fmt.Printf("queryString: %s\n", queryString)
	resultsIterator, err := sdk.GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	var utxo []models.Utxo
	amt := big.NewInt(0)
	for resultsIterator.HasNext() {
		var u models.Utxo
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return err
		}
		fmt.Printf("query Value %s\n", queryResult.Value)
		fmt.Printf("query key %s\n", queryResult.Key)
		err = json.Unmarshal(queryResult.Value, &u)
		if err != nil {
			return fmt.Errorf("failed to unmarshal value %v", err)
		}
		u.Key = queryResult.Key
		am, s := big.NewInt(0).SetString(u.Amount, 10)
		if !s {
			return fmt.Errorf("failed to set string")
		}
		amt.Add(amt, am)
		utxo = append(utxo, u)
		if amt.Cmp(amount) == 0 || amt.Cmp(amount) == 1 { // >= amount {
			break
		}
	}
	fmt.Printf("amount: %v\n", amount)
	fmt.Printf("total balance: %v\n", amt)
	if amount.Cmp(amt) == 1 {
		return fmt.Errorf("account %v has insufficient balance for token %v, required balance: %v, available balance: %v", account, constants.GINI, amount, amt)
	}

	for i := 0; i < len(utxo); i++ {
		am, s := big.NewInt(0).SetString(utxo[i].Amount, 10)
		if !s {
			return fmt.Errorf("failed to set string")
		}
		if amount.Cmp(am) == 0 || amount.Cmp(am) == 1 { // >= utxo[i].Amount {
			fmt.Printf("amount> delete: %s\n", utxo[i].Amount)
			amount = amount.Sub(amount, am)
			if err := sdk.DelStateWithoutKYC(utxo[i].Key); err != nil {
				return fmt.Errorf("%v", err)
			}
		} else if amount.Cmp(am) == -1 { // < utxo[i].Amount {
			fmt.Printf("amount<: %s\n", utxo[i].Amount)
			if err := sdk.DelStateWithoutKYC(utxo[i].Key); err != nil {
				return fmt.Errorf("%v", err)
			}
			// Create a new utxo object
			utxo := models.Utxo{
				DocType: constants.UTXO,
				Account: account,
				Amount:  am.Sub(am, amount).String(),
			}
			utxoJSON, err := json.Marshal(utxo)
			if err != nil {
				return fmt.Errorf("failed to marshal owner with  and account address %s to JSON: %v", account, err)
			}

			err = sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
			if err != nil {
				return fmt.Errorf("failed to put owner with  and account address %s to world state: %v", account, err)
			}

		}
	}

	return nil
}

func EmitTransferSingle(ctx kalpsdk.TransactionContextInterface, transferSingleEvent models.TransferSingle) error {
	transferSingleEventJSON, err := json.Marshal(transferSingleEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}

	err = ctx.SetEvent("models.TransferSingle", transferSingleEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

func IsCallerKalpBridge(ctx kalpsdk.TransactionContextInterface, KalpBridgeContractName string) (bool, error) {
	signedProposal, err := ctx.GetSignedProposal()
	if signedProposal == nil {
		return false, fmt.Errorf("could not retrieve proposal details")
	}
	if err != nil {
		return false, fmt.Errorf("error in getting signed proposal")
	}

	data := signedProposal.GetProposalBytes()
	if data == nil {
		return false, fmt.Errorf("error in fetching signed proposal")
	}

	proposal := &peer.Proposal{}
	err = proto.Unmarshal(data, proposal)
	if err != nil {
		return false, fmt.Errorf("error in parsing signed proposal")
	}

	payload := &common.Payload{}
	err = proto.Unmarshal(proposal.Payload, payload)
	if err != nil {
		return false, fmt.Errorf("error in parsing payload")
	}

	paystring := payload.GetHeader().GetChannelHeader()
	if paystring == nil {
		return false, fmt.Errorf("channel header is empty")
	}

	fmt.Println(paystring, KalpBridgeContractName)
	return strings.Contains(string(paystring), KalpBridgeContractName), nil
}

func GetTotalUTXO(ctx kalpsdk.TransactionContextInterface, account string) (string, error) {

	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"}}`
	logger.Log.Infof("queryString: %s\n", queryString)
	resultsIterator, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	amt := big.NewInt(0)
	for resultsIterator.HasNext() {
		var u map[string]interface{}
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return "", err
		}
		logger.Log.Infof("query Value %s\n", string(queryResult.Value))
		logger.Log.Infof("query key %s\n", queryResult.Key)
		err = json.Unmarshal(queryResult.Value, &u)
		if err != nil {
			logger.Log.Infof("%v", err)
			return amt.String(), err
		}
		logger.Log.Debugf("%v\n", u["amount"])
		amount := new(big.Int)
		if uamount, ok := u["amount"].(string); ok {
			amount.SetString(uamount, 10)
		}

		amt = amt.Add(amt, amount) // += u.Amount

	}

	return amt.String(), nil
}

func UpdateAllowance(sdk kalpsdk.TransactionContextInterface, owner string, spender string, spent string) error {
	approvalKey, err := sdk.CreateCompositeKey(constants.Approval, []string{owner, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner with address %s and account address %s: %v", owner, spender, err)
	}
	// Get the current balance of the owner
	approvalByte, err := sdk.GetState(approvalKey)
	if err != nil {
		return fmt.Errorf("failed to read current balance of owner with address %s and account address %s from world state: %v", owner, spender, err)
	}
	var approval models.Allow
	if approvalByte != nil {
		err = json.Unmarshal(approvalByte, &approval)
		if err != nil {
			return fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", owner, spender, err)
		}
		approvalAmount, s := big.NewInt(0).SetString(approval.Amount, 10)
		if !s {
			return fmt.Errorf("failed to convert approvalAmount to big int")
		}
		amountSpent, s := big.NewInt(0).SetString(spent, 10)
		if !s {
			return fmt.Errorf("failed to convert approvalAmount to big int")
		}
		if amountSpent.Cmp(approvalAmount) == 1 { // amountToAdd > approvalAmount {
			return fmt.Errorf("failed to convert approvalAmount to float64")
		}
		approval.Amount = fmt.Sprint(approvalAmount.Sub(approvalAmount, amountSpent))
	}
	approvalJSON, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	// Update the state of the smart contract by adding the allowanceKey and value
	err = sdk.PutStateWithoutKYC(approvalKey, approvalJSON)
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", approvalKey, err)
	}
	err = sdk.SetEvent(constants.Approval, approvalJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}
	return nil
}

func TransferUTXOFrom(ctx kalpsdk.TransactionContextInterface, owner []string, spender []string, receiver string, iamount interface{}, docType string) error {

	// Get ID of submitting client identity
	operator, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	logger.Log.Debugf("owner: %v\n", owner[0])
	logger.Log.Debugf("spender: %v\n", spender[0])
	approved, err := models.GetAllowance(ctx, owner[0], spender[0])
	if err != nil {
		return fmt.Errorf("error in getting allowance: %v", err)
	}
	approvedAmount, s := big.NewInt(0).SetString(approved, 10)
	if !s {
		return fmt.Errorf("failed to convert approvalAmount to big int")
	}
	var am string
	if a, ok := iamount.(string); ok {
		am = a
		logger.Log.Debugf("String found: %s\n", am)
	}
	amount, s := big.NewInt(0).SetString(am, 10)
	if !s {
		return fmt.Errorf("failed to convert approvalAmount to big int")
	}

	if approvedAmount.Cmp(amount) == -1 { //approvedAmount < amount {
		logger.Log.Debugf("approvedAmount: %f\n", approvedAmount)
		logger.Log.Debugf("amount: %f\n", amount)
		return fmt.Errorf("transfer amount can not be greater than allowed amount")
	}
	if spender[0] == owner[0] {
		return fmt.Errorf("owner and spender can not be same account")
	}
	logger.Log.Debugf("spender check")

	err = RemoveUtxo(ctx, owner[0], amount)
	if err != nil {
		return err
	}
	logger.Log.Debugf("removed utxo")
	if receiver == "0x0" {
		return fmt.Errorf("transfer to the zero address")
	}

	// Deposit the fund to the recipient address
	err = AddUtxo(ctx, receiver, amount)
	if err != nil {
		return err
	}

	err = UpdateAllowance(ctx, owner[0], spender[0], fmt.Sprint(amount))
	if err != nil {
		return err
	}
	// Emit models.TransferSingle event
	transferSingleEvent := models.TransferSingle{Operator: operator, From: owner[0], To: receiver, Value: amount}
	return EmitTransferSingle(ctx, transferSingleEvent)
}

func InitializeRoles(ctx kalpsdk.TransactionContextInterface, id string, role string) (bool, error) {
	userRole := models.UserRole{
		Id:      id,
		Role:    role,
		DocType: constants.UserRoleMap,
	}
	roleJson, e := json.Marshal(userRole)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "error in marshaling user role", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userRole.Id, constants.UserRoleMap})
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to create the composite key for user role", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(key, roleJson); e != nil {
		err := ginierr.NewWithInternalError(e, "unable to put user role struct in statedb", http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	return true, nil
}

// SetUserRoles is a smart contract function which is used to setup a role for user.
func SetUserRoles(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {
	//check if contract has been intilized first

	fmt.Println("SetUserRoles", data)

	// Parse input data into Role struct.
	var userRole models.UserRole
	errs := json.Unmarshal([]byte(data), &userRole)
	if errs != nil {
		return "", fmt.Errorf("failed to parse data: %v", errs)
	}

	userValid, err := ValidateUserRole(ctx, constants.KalpFoundationRole)
	if err != nil {
		return "", fmt.Errorf("error in validating the role %v", err)
	}
	if !userValid {
		return "", fmt.Errorf("error in setting role %s, only %s can set the roles", userRole.Role, constants.KalpFoundationRole)
	}

	// Validate input data.
	if userRole.Id == "" {
		return "", fmt.Errorf("user Id can not be null")
	}

	if userRole.Role == "" {
		return "", fmt.Errorf("role can not be null")
	}

	ValidRoles := []string{constants.KalpFoundationRole, constants.KalpGateWayAdminRole}
	if !slices.Contains(ValidRoles, userRole.Role) {
		return "", fmt.Errorf("invalid input role")
	}

	key, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userRole.Id, constants.UserRoleMap})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, err)
	}
	// Generate JSON representation of Role struct.
	usrRoleJSON, err := json.Marshal(userRole)
	if err != nil {
		return "", fmt.Errorf("unable to Marshal userRole struct : %v", err)
	}
	// Store the Role struct in the state database
	if err := ctx.PutStateWithoutKYC(key, usrRoleJSON); err != nil {
		return "", fmt.Errorf("unable to put user role struct in statedb: %v", err)
	}
	return GetTransactionTimestamp(ctx)

}

// GetTransactionTimestamp retrieves the transaction timestamp from the context and returns it as a string.
func GetTransactionTimestamp(ctx kalpsdk.TransactionContextInterface) (string, error) {
	timestamp, err := ctx.GetTxTimestamp()
	if err != nil {
		return "", err
	}

	return timestamp.AsTime().String(), nil
}

func ValidateUserRole(ctx kalpsdk.TransactionContextInterface, Role string) (bool, error) {

	// Check if operator is authorized to create Role.
	operator, err := ctx.GetUserID()
	if err != nil {
		return false, fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("operator---------------", operator)
	userRole, err1 := GetUserRoles(ctx, operator)
	if err1 != nil {
		return false, fmt.Errorf("error: %v", err1)
	}

	if userRole != Role {
		return false, fmt.Errorf("this transaction can be performed by %v only", Role)
	}
	return true, nil
}

// GetUserRoles is a smart contract function which is used to get a role of a user.
func GetUserRoles(ctx kalpsdk.TransactionContextInterface, id string) (string, error) {
	// Get the asset from the ledger using id & check if asset exists
	key, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{id, constants.UserRoleMap})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, err)
	}

	userJSON, err := ctx.GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if userJSON == nil {
		return "", nil
	}

	// Unmarshal asset from JSON to struct
	var userRole models.UserRole
	err = json.Unmarshal(userJSON, &userRole)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal user role struct : %v", err)
	}

	return userRole.Role, nil
}

func IsAmountProper(amount string) bool {
	// Parse the amount as a big.Int
	bigAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		// Return false if amount cannot be converted to big.Int
		return false
	}

	// Check if the amount is less than 0
	return bigAmount.Sign() >= 0
}
