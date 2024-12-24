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
		err := ginierr.NewInternalError(e, "failed to get public address", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if signer != constants.KalpFoundationAddress {
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
		err := ginierr.New("channel header is empty", http.StatusNotFound)
		logger.Log.Error(err.FullError())
		return "", err
	}

	logger.Log.Debug("paystring", paystring, string(paystring))

	printableASCIIPaystring := helper.FilterPrintableASCII(string(paystring))
	logger.Log.Debug("printableASCIIPaystring", printableASCIIPaystring)

	contractAddress := helper.FindContractAddress(printableASCIIPaystring)
	if contractAddress == "" {
		err := ginierr.New("contract address not found", http.StatusNotFound)
		logger.Log.Error(err.FullError())
		return "", err
	}
	return contractAddress, nil
}

func GetGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface) ([]string, error) {
	iterator, err := ctx.GetStateByPartialCompositeKey(constants.UserRolePrefix, []string{constants.UserRoleMap})
	if err != nil {
		return nil, fmt.Errorf("failed to get data for gateway admin: %v", err)
	}
	defer iterator.Close()

	gatewayAdmins := []string{}

	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error reading next item: %v", err)
		}

		var userRole models.UserRole
		if err := json.Unmarshal(response.Value, &userRole); err != nil {
			return nil, fmt.Errorf("failed to parse user role data: %v", err)
		}

		gatewayAdmins = append(gatewayAdmins, userRole.Id)

		fmt.Println("here are the gatewayAdmins ====================>", gatewayAdmins, userRole.Id)
	}

	return gatewayAdmins, nil
}

func IsGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface, userID string) (bool, error) {
	prefix := constants.UserRolePrefix
	iterator, err := ctx.GetStateByPartialCompositeKey(prefix, []string{userID, constants.UserRoleMap})
	if err != nil {
		return false, fmt.Errorf("failed to get data for gateway admin: %v", err)
	}
	defer iterator.Close()

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
			if userRole.Role == constants.KalpGateWayAdminRole {
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	return false, nil
}

func DenyAddress(ctx kalpsdk.TransactionContextInterface, address string) error {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if err := ctx.PutStateWithoutKYC(addressDenyKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to put data in deny list: %v", err)
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
		return fmt.Errorf("failed to put data in deny list: %v", err)
	}
	if err := events.EmitAllowed(ctx, address); err != nil {
		return err
	}
	return nil
}

func IsDenied(ctx kalpsdk.TransactionContextInterface, address string) (bool, error) {
	addressDenyKey, err := ctx.CreateCompositeKey(constants.DenyListKey, []string{address})
	if err != nil {
		return false, fmt.Errorf("failed to create composite key for deny list: %v", err)
	}
	if bytes, err := ctx.GetState(addressDenyKey); err != nil {
		return false, fmt.Errorf("failed to get data from deny list: %v", err)
	} else if bytes == nil {
		return false, nil
	} else if string(bytes) == "false" {
		return false, nil
	}
	return true, nil
}

func Mint(ctx kalpsdk.TransactionContextInterface, addresses []string, amounts []string) error {

	logger.Log.Infof("Mint invoked.... with arguments", addresses, amounts)

	accAmount1, ok := big.NewInt(0).SetString(amounts[0], 10)
	if !ok {
		return ginierr.ErrConvertingAmountToBigInt(amounts[0])
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 {
		return ginierr.ErrInvalidAmount(amounts[0])
	}
	accAmount2, ok := big.NewInt(0).SetString(amounts[1], 10)
	if !ok {
		return ginierr.ErrConvertingAmountToBigInt(amounts[1])
	}
	if accAmount1.Cmp(big.NewInt(0)) != 1 {
		return ginierr.ErrInvalidAmount(amounts[1])
	}

	if !helper.IsValidAddress(addresses[0]) {
		return ginierr.ErrInvalidAddress(addresses[0])
	}
	if !helper.IsValidAddress(addresses[1]) {
		return ginierr.ErrInvalidAddress(addresses[1])
	}

	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		return ginierr.ErrFailedToGetKey(constants.NameKey)
	} else if bytes != nil {
		return ginierr.New(fmt.Sprintf("cannot mint again,%s already set: %s", constants.NameKey, string(bytes)), http.StatusBadRequest)
	}
	if bytes, e := ctx.GetState(constants.SymbolKey); e != nil {
		return ginierr.ErrFailedToGetKey(constants.SymbolKey)
	} else if bytes != nil {
		return ginierr.New(fmt.Sprintf("cannot mint again,%s already set: %s", constants.SymbolKey, string(bytes)), http.StatusBadRequest)
	}

	if err := MintUtxoHelperWithoutKYC(ctx, addresses[0], accAmount1); err != nil {
		return err
	}
	if err := MintUtxoHelperWithoutKYC(ctx, addresses[1], accAmount2); err != nil {
		return err
	}
	logger.Log.Infof("Mint Invoke complete amount: %v\n", amounts)
	return nil

}

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
		return fmt.Errorf("failed to put owner with ID %s and account address %s: %v", constants.GINI, account, err)

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
		return fmt.Errorf("failed to read: %v", err)
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
		if amt.Cmp(amount) == 0 || amt.Cmp(amount) == 1 {
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
		if amount.Cmp(am) == 0 || amount.Cmp(am) == 1 {
			amount = amount.Sub(amount, am)
			if err := sdk.DelStateWithoutKYC(utxo[i].Key); err != nil {
				return fmt.Errorf("%v", err)
			}
		} else if amount.Cmp(am) == -1 {
			if err := sdk.DelStateWithoutKYC(utxo[i].Key); err != nil {
				return fmt.Errorf("%v", err)
			}

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
				return fmt.Errorf("failed to put owner with  and account address %s: %v", account, err)
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
		return "", fmt.Errorf("failed to read: %v", err)
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

		amt = amt.Add(amt, amount)
	}

	return amt.String(), nil
}

func UpdateAllowance(sdk kalpsdk.TransactionContextInterface, owner string, spender string, spent string) error {
	approvalKey, err := sdk.CreateCompositeKey(constants.Approval, []string{owner, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner with address %s and spender with address %s: %v", owner, spender, err)
	}

	approvalByte, err := sdk.GetState(approvalKey)
	if err != nil {
		return fmt.Errorf("failed to read current balance of owner with address %s and spender with address %s : %v", owner, spender, err)
	}
	var approval models.Allow
	if approvalByte != nil {
		err = json.Unmarshal(approvalByte, &approval)
		if err != nil {
			return fmt.Errorf("failed to unmarshal balance for owner address : %v and spender address: %v: %v", owner, spender, err)
		}
		approvalAmount, s := big.NewInt(0).SetString(approval.Amount, 10)
		if !s {
			return ginierr.ErrConvertingAmountToBigInt(approval.Amount)
		}
		amountSpent, s := big.NewInt(0).SetString(spent, 10)
		if !s {
			return ginierr.ErrConvertingAmountToBigInt(spent)
		}
		if amountSpent.Cmp(approvalAmount) == 1 {
			return fmt.Errorf("the amount spent :%s , is greater than allowance :%s ", spent, approval.Amount)
		}
		approval.Amount = fmt.Sprint(approvalAmount.Sub(approvalAmount, amountSpent))
	}
	approvalJSON, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}

	err = sdk.PutStateWithoutKYC(approvalKey, approvalJSON)
	if err != nil {
		return fmt.Errorf("failed to update data of smart contract for key %s: %v", approvalKey, err)
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
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key: user ID '%s', role '%s'", userRole.Id, userRole.Role), http.StatusInternalServerError)
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(key, roleJson); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role: %s , id: %s ", role, id), http.StatusInternalServerError)
		return false, err
	}
	return true, nil
}

func GetTransactionTimestamp(ctx kalpsdk.TransactionContextInterface) (string, error) {
	timestamp, err := ctx.GetTxTimestamp()
	if err != nil {
		return "", err
	}

	return timestamp.AsTime().String(), nil
}

func GetUserRoles(ctx kalpsdk.TransactionContextInterface, id string) (string, error) {

	key, err := ctx.CreateCompositeKey(constants.UserRolePrefix, []string{id, constants.UserRoleMap})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, err)
	}

	userJSON, err := ctx.GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read: %v", err)
	}
	if userJSON == nil {
		return "", nil
	}

	var userRole models.UserRole
	err = json.Unmarshal(userJSON, &userRole)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal user role struct : %v", err)
	}

	return userRole.Role, nil
}
