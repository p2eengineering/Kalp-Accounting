package internal

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/models"
	"math/big"
	"net/http"
	"strconv"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func RemoveUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, amount string) error {
	amountInt := new(big.Int)
	amountInt, ok := amountInt.SetString(amount, 10)
	if !ok {
		return ginierr.ErrConvertingAmountToBigInt(amount)
	}
	if amountInt.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	totalAmount := big.NewInt(0)
	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexAccountDocType"}`
	resultsIterator, err := sdk.GetQueryResult(queryString)
	if err != nil {
		return ginierr.ErrFailedToGetState(err)
	}
	defer resultsIterator.Close()

	for totalAmount.Cmp(amountInt) < 0 {
		if !resultsIterator.HasNext() {
			return ginierr.New(
				fmt.Sprintf("insufficient balance for account %v, required: %v, available: %v", account, amountInt, totalAmount),
				http.StatusBadRequest,
			)
		}
		var u models.Utxo
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return ginierr.NewInternalError(err, "failed to get next UTXO result", http.StatusInternalServerError)
		}
		err = json.Unmarshal(queryResult.Value, &u)
		if err != nil {
			return ginierr.NewInternalError(err, "failed to unmarshal UTXO value", http.StatusInternalServerError)
		}
		u.Key = queryResult.Key
		utxoAmount := new(big.Int)
		utxoAmount, ok = utxoAmount.SetString(u.Amount, 10)
		if !ok {
			return ginierr.ErrConvertingAmountToBigInt(u.Amount)
		}
		totalAmount.Add(totalAmount, utxoAmount)

		if err := sdk.DelStateWithoutKYC(u.Key); err != nil {
			return ginierr.ErrFailedToPutState(err)
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
			return ginierr.NewInternalError(err, fmt.Sprintf("failed to marshal new UTXO for account %s", account), http.StatusInternalServerError)
		}
		utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
		if err != nil {
			return ginierr.NewInternalError(err, fmt.Sprintf("failed to create the composite key for account %s", account), http.StatusInternalServerError)
		}
		if err := sdk.PutStateWithoutKYC(utxoKey, utxoJSON); err != nil {
			return ginierr.ErrFailedToPutState(err)
		}
	}

	return nil
}

func AddUtxoForGasFees(sdk kalpsdk.TransactionContextInterface, account string, amount string) error {
	amountInt, e := strconv.ParseUint(amount, 10, 64)
	if e != nil {
		return ginierr.ErrInvalidAmount(amount)
	}
	if amountInt == 0 {
		return nil
	}
	utxoKey, err := sdk.CreateCompositeKey(constants.UTXO, []string{account, sdk.GetTxID()})
	if err != nil {
		return ginierr.NewInternalError(err, fmt.Sprintf("failed to create the composite key for account %s", account), http.StatusInternalServerError)
	}
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  amount,
	}
	utxoJSON, err := json.Marshal(utxo)
	if err != nil {
		return ginierr.NewInternalError(err, fmt.Sprintf("failed to marshal UTXO for account %s to JSON", account), http.StatusInternalServerError)
	}
	err = sdk.PutStateWithoutKYC(utxoKey, utxoJSON)
	if err != nil {
		return ginierr.ErrFailedToPutState(err)
	}
	return nil
}
