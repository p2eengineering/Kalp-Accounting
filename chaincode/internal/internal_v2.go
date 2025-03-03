package internal

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/logger"
	"gini-contract/chaincode/models"
	"math/big"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func RemoveUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, amount string) error {
	amountInt := new(big.Int)
	amountInt, ok := amountInt.SetString(amount, 10)
	if !ok {
		return fmt.Errorf("error converting amount: %v to big.Int", amount)
	}
	if amountInt.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	totalAmount := big.NewInt(0)
	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexAccountDocType"}`
	resultsIterator, err := sdk.GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("failed to read UTXO: %v", err)
	}
	defer resultsIterator.Close()
	// Keep fetching until required amount is accumulated
	for totalAmount.Cmp(amountInt) < 0 {
		if !resultsIterator.HasNext() {
			return fmt.Errorf("insufficient balance for account %v, required: %v, available: %v", account, amountInt, totalAmount)
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

	if totalAmount.Cmp(amountInt) > 0 {
		remainingAmount := new(big.Int).Sub(totalAmount, amountInt)
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
			return fmt.Errorf("failed to create the composite key for account %s: %v", account, err)
		}
		if err := sdk.PutStateWithoutKYC(utxoKey, utxoJSON); err != nil {
			return fmt.Errorf("failed to put UTXO for account address %s: %v", account, err)
		}
	}

	return nil
}
func AddUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, amount string) error {
	amountInt, e := strconv.ParseUint(amount, 10, 64)
	if e != nil {
		logger.Log.Error("error parsing amount ", amount, e)
		return ginierr.ErrInvalidAmount(amount)
	}
	if amountInt == 0 {
		return nil
	}
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for account %s: %v", account, err)
	}
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  amount,
	}
	utxoJSON, err := json.Marshal(utxo)
	if err != nil {
		return fmt.Errorf("failed to marshal UTXO for account %s to JSON: %v", account, err)
	}
	err = sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
	if err != nil {
		return fmt.Errorf("failed to put UTXO for account address %s: %v", account, err)
	}
	return nil
}

func GetTotalUTXOWithPagination(ctx kalpsdk.TransactionContextInterface, account string) (string, error) {
	totalAmount := big.NewInt(0)
	bookmark := ""
	for {

		// Create the paginated query
		queryString := fmt.Sprintf(`{
			"selector": {
				"account": "%s",
				"docType": "%s"
			}
		}`, account, constants.UTXO)

		resultsIterator, metadata, err := ctx.GetQueryResultWithPagination(queryString, constants.PageSize, bookmark)
		if err != nil {
			return "", fmt.Errorf("failed to execute query: %v", err)
		}
		bookmark = metadata.Bookmark

		// Track if this page has any results
		hasResults := false

		for resultsIterator.HasNext() {
			hasResults = true
			queryResult, err := resultsIterator.Next()
			if err != nil {
				resultsIterator.Close()
				return "", fmt.Errorf("failed to iterate over query results: %v", err)
			}

			var utxo models.Utxo
			if err := json.Unmarshal(queryResult.Value, &utxo); err != nil {
				resultsIterator.Close()
				return "", fmt.Errorf("failed to unmarshal query result: %v", err)
			}

			// Accumulate the UTXO amount
			amount := new(big.Int)
			if _, ok := amount.SetString(utxo.Amount, 10); !ok {
				resultsIterator.Close()
				return "", ginierr.ErrInvalidAmount(utxo.Amount)
			}

			totalAmount.Add(totalAmount, amount)
			resultsIterator.Close()
		}

		// If no results were found in the current page, exit the loop
		if !hasResults {
			break
		}
	}

	return totalAmount.String(), nil
}
