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


func IsGatewayAdminAddress(ctx kalpsdk.TransactionContextInterface, userID string) (bool, error) {
    // Construct the key to fetch the gateway admin role
	key, e := ctx.CreateCompositeKey(constants.KalpGateWayAdminRole, []string{userID})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for prefix %s: %v", constants.UserRolePrefix, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}

    // Fetch the state from the context
    data, err := ctx.GetState(key)
    if err != nil {
        wrappedErr := ginierr.NewInternalError(err, fmt.Sprintf("Failed to fetch gateway admin data for userID: %s", userID), http.StatusInternalServerError)
        logger.Log.Errorf(wrappedErr.FullError())
        return false, wrappedErr
    }

    if data == nil {
        // No data found for the given key
        logger.Log.Infof("No data found for userID: %s", userID)
        return false, nil
    }

    // Unmarshal the state data into a UserRole model
    var userRole models.UserRole
    if err := json.Unmarshal(data, &userRole); err != nil {
        wrappedErr := ginierr.NewInternalError(err, "Failed to unmarshal user role data", http.StatusInternalServerError)
        logger.Log.Errorf(wrappedErr.FullError())
        return false, wrappedErr
    }

    // Check if the user has the Gateway Admin role
    if userRole.Id == userID && userRole.Role == constants.KalpGateWayAdminRole {
        return true, nil
    }

    // No matching role found
    logger.Log.Infof("No Gateway Admin role found for userID: %s", userID)
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

	if len(addresses) < 2 || len(amounts) < 2 {
		return ginierr.New("addresses and amounts arrays cannot be empty", http.StatusBadRequest)
	}
	if len(addresses) != len(amounts) {
		return ginierr.New("length of addresses and amounts arrays must be equal", http.StatusBadRequest)
	}

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
	if accAmount2.Cmp(big.NewInt(0)) != 1 {
		return ginierr.ErrInvalidAmount(amounts[1])
	}

	if !helper.IsValidAddress(addresses[0]) {
		return ginierr.ErrInvalidAddress(addresses[0])
	}
	if !helper.IsValidAddress(addresses[1]) {
		return ginierr.ErrInvalidAddress(addresses[1])
	}

	if bytes, e := ctx.GetState(constants.NameKey); e != nil {
		logger.Log.Errorf("Error in GetState %s: %v", constants.NameKey, e)
		return ginierr.ErrFailedToGetKey(constants.NameKey)
	} else if bytes != nil {
		return ginierr.New(fmt.Sprintf("cannot mint again,%s already set: %s", constants.NameKey, string(bytes)), http.StatusBadRequest)
	}
	if bytes, e := ctx.GetState(constants.SymbolKey); e != nil {
		logger.Log.Errorf("Error in GetState %s: %v", constants.SymbolKey, e)
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

	e := sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to put owner with ID %s and account address %s: %v", constants.GINI, account, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
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

			e := sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
			if e != nil {
				err := ginierr.NewInternalError(e, fmt.Sprintf("failed to update balance for account %s and amount %s", account, amount), http.StatusInternalServerError)
				logger.Log.Errorf(err.FullError())
				return err
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
	approvalKey, e := sdk.CreateCompositeKey(constants.Approval, []string{owner, spender})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key for owner with address %s and spender with address %s: %v", owner, spender, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	approvalByte, e := sdk.GetState(approvalKey)
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to read current allowance of owner with address %s and spender with address %s : %v", owner, spender, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
	}

	if approvalByte == nil {
		err := ginierr.New(fmt.Sprintf("no allowance exists for owner with address %s and spender with address %s", owner, spender), http.StatusBadRequest)
		logger.Log.Errorf(err.FullError())
		return err
	}

	var approval models.Allow
	e = json.Unmarshal(approvalByte, &approval)
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to unmarshal allowance for owner address : %s and spender address: %s: %v", owner, spender, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
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
		return fmt.Errorf("the amount spent :%s, is greater than allowance :%s", spent, approval.Amount)
	}

	approval.Amount = fmt.Sprint(approvalAmount.Sub(approvalAmount, amountSpent))

	approvalJSON, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}

	e = sdk.PutStateWithoutKYC(approvalKey, approvalJSON)
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to update data of smart contract for key %s: %v", approvalKey, e), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return err
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
	key, e := ctx.CreateCompositeKey(role, []string{userRole.Id})
	if e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("failed to create the composite key: user ID '%s', role '%s'", userRole.Id, userRole.Role), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	if e := ctx.PutStateWithoutKYC(key, roleJson); e != nil {
		err := ginierr.NewInternalError(e, fmt.Sprintf("unable to put user role: %s , id: %s ", role, id), http.StatusInternalServerError)
		logger.Log.Errorf(err.FullError())
		return false, err
	}
	return true, nil
}
