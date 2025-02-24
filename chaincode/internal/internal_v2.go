package internal

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/models"
	"math/big"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func RemoveUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, iamount string) error {
	// Convert input amount to big.Int
	amount := new(big.Int)
	amount, ok := amount.SetString(iamount, 10)
	if !ok {
		return fmt.Errorf("error converting amount: %v to big.Int", iamount)
	}
	if amount.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	totalAmount := big.NewInt(0)
	// Keep fetching until required amount is accumulated
	for totalAmount.Cmp(amount) < 0 {
		queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexAccountDocType"}`
		resultsIterator, err := sdk.GetQueryResult(queryString)
		if err != nil {
			return fmt.Errorf("failed to read UTXO: %v", err)
		}
		if !resultsIterator.HasNext() {
			return fmt.Errorf("insufficient balance for account %v, required: %v, available: %v", account, amount, totalAmount)
		}
		var u models.Utxo
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return err
		}
		err = json.Unmarshal(queryResult.Value, &u)
		if err != nil {
			return fmt.Errorf("failed to unmarshal UTXO value: %v", err)
		}
		u.Key = queryResult.Key
		utxoAmount := new(big.Int)
		utxoAmount, ok = utxoAmount.SetString(u.Amount, 10)
		if !ok {
			return fmt.Errorf("failed to parse UTXO amount: %v", u.Amount)
		}
		totalAmount.Add(totalAmount, utxoAmount)

		if err := sdk.DelStateWithoutKYC(u.Key); err != nil {
			return fmt.Errorf("failed to delete UTXO: %v", err)
		}
	}

	if totalAmount.Cmp(amount) > 0 {
		remainingAmount := new(big.Int).Sub(totalAmount, amount)
		newUtxo := models.Utxo{
			DocType: constants.UTXO,
			Account: account,
			Amount:  remainingAmount.String(),
		}
		utxoJSON, err := json.Marshal(newUtxo)
		if err != nil {
			return fmt.Errorf("failed to marshal new UTXO for account %s: %v", account, err)
		}
		utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
		if err != nil {
			return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
		}
		if err := sdk.PutStateWithoutKYC(utxoKey, utxoJSON); err != nil {
			return fmt.Errorf("failed to update balance for account %s: %v", account, err)
		}
	}

	return nil
}
func AddUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, iamount string) error {
	amount, err := strconv.ParseUint(iamount, 10, 64)
	if err != nil {
		return fmt.Errorf("error converting amount: %v to uint: %v", iamount, err)
	}
	if amount == 0 {
		return nil
	}
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner %s: %v", account, err)
	}
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  iamount,
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
