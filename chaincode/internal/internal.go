package internal

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
	"math/big"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func IsSignerKalpFoundation(ctx kalpsdk.TransactionContextInterface) (bool, error) {
	signer, e := helper.GetUserId(ctx)
	if e != nil {
		err := ginierr.NewInternalError(e, "failed to get client id", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}
	kalpFoundationAddress, err := GetKalpFoundationAdminAddress(ctx)
	if err != nil {
		return false, err
	}

	if signer != kalpFoundationAddress {
		return false, nil
	}
	return true, nil
}

// TODO: remove debug logs later
func GetCalledContractAddress(ctx kalpsdk.TransactionContextInterface) (string, error) {
	signedProposal, e := ctx.GetSignedProposal()
	if signedProposal == nil {
		err := ginierr.New("could not retrieve signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	if e != nil {
		err := ginierr.NewInternalError(e, "error in getting signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	logger.Log.Debug("signedProposal:", signedProposal)

	data := signedProposal.GetProposalBytes()
	if data == nil {
		err := ginierr.New("error in fetching proposal bytes", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	logger.Log.Debug("signedProposal.GetProposalBytes()", data, string(data))

	proposal := &peer.Proposal{}
	e = proto.Unmarshal(data, proposal)
	if e != nil {
		err := ginierr.NewInternalError(e, "error in parsing signed proposal", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	logger.Log.Debug("peer.Proposal{}", proposal)

	payload := &common.Payload{}
	e = proto.Unmarshal(proposal.Payload, payload)
	if e != nil {
		err := ginierr.NewInternalError(e, "error in parsing payload", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	logger.Log.Debug("common.Payload{}", payload)

	paystring := payload.GetHeader().GetChannelHeader()
	if len(paystring) == 0 {
		err := ginierr.New("channel header is empty", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}

	logger.Log.Debug("paystring", paystring, string(paystring))

	printableASCIIPaystring := helper.FilterPrintableASCII(string(paystring))
	logger.Log.Debug("printableASCIIPaystring", printableASCIIPaystring)

	contractAddress := helper.FindContractAddress(printableASCIIPaystring)
	if contractAddress == "" {
		err := ginierr.New("contract address not found", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return "", err
	}
	return contractAddress, nil
}

// GetGatewayAdminAddress retrieves the address of the Kalp Gateway Admin dynamically from the blockchain.
func GetGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface, userID string) ([]models.UserRole, error) {
	// Define the composite key to retrieve the gateway admin role
	prefix := constants.UserRolePrefix
	iterator, err := ctx.GetStateByPartialCompositeKey(prefix, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get state for gateway admin: %v", err)
	}
	defer iterator.Close()

	gatewayAdmins := []models.UserRole{}

	// Iterate through keys to find the gateway admin
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error reading next item: %v", err)
		}

		var userRole models.UserRole
		if err := json.Unmarshal(response.Value, &userRole); err != nil {
			return nil, fmt.Errorf("failed to parse user role data: %v", err)
		}

		gatewayAdmins = append(gatewayAdmins, userRole)
	}

	return gatewayAdmins, nil
}

// IsGatewayAdminAddress retrieves the address of the Kalp Gateway Admin dynamically from the blockchain.
func IsGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface, userID string) (bool, error) {
	// Define the composite key to retrieve the gateway admin role
	prefix := constants.UserRolePrefix
	iterator, err := ctx.GetStateByPartialCompositeKey(prefix, []string{})
	if err != nil {
		return false, fmt.Errorf("failed to get state for gateway admin: %v", err)
	}
	defer iterator.Close()

	// Iterate through keys to find the gateway admin
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return false, fmt.Errorf("error reading next item: %v", err)
		}

		var userRole models.UserRole
		if err := json.Unmarshal(response.Value, &userRole); err != nil {
			return false, fmt.Errorf("failed to parse user role data: %v", err)
		}

		if userRole.Id == userID {
			return true, nil
		}
	}

	return false, fmt.Errorf("no gateway admin address found")
}

// GetKalpFoundationAdminAddress retrieves the Kalp Foundation Admin address from the blockchain state.
func GetKalpFoundationAdminAddress(ctx kalpsdk.TransactionContextInterface) (string, error) {
	// Create the composite key for the Kalp Foundation Admin
	key, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{constants.KalpFoundationRole})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key for foundation admin: %v", err)
	}

	roleData, err := ctx.GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to fetch foundation admin role: %v", err)
	}

	if roleData == nil {
		return "", fmt.Errorf("no foundation admin role found in the state")
	}

	var userRole models.UserRole
	if err := json.Unmarshal(roleData, &userRole); err != nil {
		return "", fmt.Errorf("failed to parse foundation admin role data: %v", err)
	}

	// Return the address (assuming it's stored in the Id field)
	return userRole.Id, nil
}

func DenyAddress(ctx kalpsdk.TransactionContextInterface, address string) error {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if err := ctx.PutStateWithoutKYC(addressDenyKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to put state in deny list: %v", err)
	}
	if err := events.EmitDenied(ctx, address); err != nil {
		return err
	}
	return nil
}

func AllowAddress(ctx kalpsdk.TransactionContextInterface, address string) error {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if err := ctx.PutStateWithoutKYC(addressDenyKey, []byte("false")); err != nil {
		return fmt.Errorf("failed to put state in deny list: %v", err)
	}
	if err := events.EmitAllowed(ctx, address); err != nil {
		return err
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

	logger.Log.Infof("Mint invoked.... with arguments", addresses, amounts)

	// Validate input amount
	accAmount1, ok := big.NewInt(0).SetString(amounts[0], 10)
	if !ok {
		return ginierr.ErrConvertingAmountToBigInt(amounts[0])
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 { // if amount is not greater than 0 return error
		return ginierr.ErrInvalidAmount(amounts[0])
	}
	accAmount2, ok := big.NewInt(0).SetString(amounts[1], 10)
	if !ok {
		return ginierr.ErrConvertingAmountToBigInt(amounts[1])
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 { // if amount is not greater than 0 return error
		return ginierr.ErrInvalidAmount(amounts[1])
	}

	// Validate input address
	if !helper.IsValidAddress(addresses[0]) {
		return ginierr.ErrInvalidAddress(addresses[0])
	}
	if !helper.IsValidAddress(addresses[1]) {
		return ginierr.ErrInvalidAddress(addresses[1])
	}

	// checking if contract is already initialized
	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		return ginierr.ErrFailedToGetKey("namekey", constants.NameKey)
	} else if bytes != nil {
		return ginierr.New(fmt.Sprintf("cannot mint again,%s already set: %s", constants.NameKey, string(bytes)), http.StatusBadRequest)
	}
	if bytes, e := ctx.GetState(constants.SymbolKey); e != nil {
		return ginierr.ErrFailedToGetKey("symbolkey", constants.SymbolKey)
	} else if bytes != nil {
		return ginierr.New(fmt.Sprintf("cannot mint again,%s already set: %s", constants.SymbolKey, string(bytes)), http.StatusBadRequest)
	}

	// Mint tokens
	if err := MintUtxoHelperWithoutKYC(ctx, addresses[0], accAmount1); err != nil {
		return err
	}
	if err := MintUtxoHelperWithoutKYC(ctx, addresses[1], accAmount2); err != nil {
		return err
	}
	logger.Log.Infof("Mint Invoke complete amount: %v\n", amounts)
	return nil

}

// As of now, we are not supporting usecases where asset is owned by multiple owners.
func MintUtxoHelperWithoutKYC(ctx kalpsdk.TransactionContextInterface, account string, amount *big.Int) error {
	err := AddUtxo(ctx, account, amount)
	if err != nil {
		return err
	}
	if err := events.EmitMint(ctx, account, amount.String()); err != nil {
		return err
	}
	return nil
}

func AddUtxo(sdk kalpsdk.TransactionContextInterface, account string, iamount interface{}) error {
	amount, err := helper.ConvertToBigInt(iamount)
	if err != nil {
		return fmt.Errorf("error in coverting amount: %v to big int: %v", iamount, err)
	}
	if amount.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("amount to add cannot be negative")
	}
	if amount.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
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

	err = sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
	if err != nil {
		return fmt.Errorf("failed to put owner with ID %s and account address %s to world state: %v", constants.GINI, account, err)

	}
	return nil
}
func RemoveUtxo(sdk kalpsdk.TransactionContextInterface, account string, iamount interface{}) error {
	amount, err := helper.ConvertToBigInt(iamount)
	if err != nil {
		return fmt.Errorf("error in coverting amount: %v to big int: %v", iamount, err)
	}
	if amount.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("amount to remove cannot be negative")
	}
	if amount.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
	}
	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexIdDocType"}`

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
	if amount.Cmp(amt) == 1 {
		return fmt.Errorf("account %v has insufficient balance for token %v, required balance: %v, available balance: %v", account, constants.GINI, amount, amt)
	}

	for i := 0; i < len(utxo); i++ {
		am, s := big.NewInt(0).SetString(utxo[i].Amount, 10)
		if !s {
			return fmt.Errorf("failed to set string")
		}
		if amount.Cmp(am) == 0 || amount.Cmp(am) == 1 { // >= utxo[i].Amount {
			amount = amount.Sub(amount, am)
			if err := sdk.DelStateWithoutKYC(utxo[i].Key); err != nil {
				return fmt.Errorf("%v", err)
			}
		} else if amount.Cmp(am) == -1 { // < utxo[i].Amount {
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
			return ginierr.ErrConvertingAmountToBigInt(approval.Amount)
		}
		amountSpent, s := big.NewInt(0).SetString(spent, 10)
		if !s {
			return ginierr.ErrConvertingAmountToBigInt(spent)
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
	return nil
}

func InitializeRoles(ctx kalpsdk.TransactionContextInterface, id string, role string) (bool, error) {
	userRole := models.UserRole{
		Id:      id,
		Role:    role,
		DocType: constants.UserRoleMap,
	}
	roleJson, err := json.Marshal(userRole)
	if err != nil {
		return false, ginierr.New("error in marshaling user role: "+role, http.StatusInternalServerError)
	}
	key, e := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{userRole.Id, constants.UserRoleMap})
	if e != nil {
		err := ginierr.NewInternalError(e, "failed to create the composite key for user role: "+role, http.StatusInternalServerError)
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(key, roleJson); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role: %s struct in statedb", role), http.StatusInternalServerError)
		return false, err
	}
	return true, nil
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
	operator, err := helper.GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get client id: %v", err)
	}

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
