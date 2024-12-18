# Gini Contract

## TODOs
- TODO: call intialize for vesting contract address
- TODO: Do we need to take bridge contract address as input in initialize
- TODO: use deny list in transferFrom
- Emit event after Update Allowance?
- Setting roles in intialize for Foundation admin and gateway admin but its not getting used
- Write setter getter for UserRole struct if we are going to use it
- Can a valid Transfer or TransferFrom request have amount = 0?
- In minting, check if name & symbol are initialized or check only one?
- Why there is need to check for balance in minting? will checking name and symbol not cover all the cases?
- Emit event "Transfer" in TransferFrom?
- Remove all the unrequired logs later  
- Add starting and ending logs for functions
- Deprecate GetUserID() after changes are made in kalp-sdk
- Set Gini contract address in main()

## RemoveUtxo Implementation
```go
func RemoveUtxo(ctx kalpsdk.TransactionContextInterface, account string, amountVal *big.Int) error {
	amount := new(big.Int).Set(amountVal)

	utxoKey, e := ctx.CreateCompositeKey(constants.UTXO, []string{account, ctx.GetTxID()})
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to create the composite key for owner:"+account, http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	queryString := `{"selector":{"account":"` + account + `","docType":"` + constants.UTXO + `"},"use_index": "indexIdDocType"}`

	logger.Log.Debugf("queryString: %s\n", queryString)
	resultsIterator, e := ctx.GetQueryResult(queryString)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "error creating iterator while removing UTXO", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	var utxo []models.Utxo
	currentBalance := big.NewInt(0)
	for resultsIterator.HasNext() {
		var u models.Utxo
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return err
		}
		logger.Log.Debugf("query Value %s\n", queryResult.Value)
		logger.Log.Debugf("query key %s\n", queryResult.Key)

		if e := json.Unmarshal(queryResult.Value, &u); e != nil {
			err := ginierr.NewWithInternalError(e, "failed to unmarshal UTXO data while removing UTXO", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return err
		}
		u.Key = queryResult.Key // TODO:: check if this is needed
		am, ok := big.NewInt(0).SetString(u.Amount, 10)
		if !ok {
			err := ginierr.New("failed to convert UTXO amount to big int while removing UTXO", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return err
		}
		currentBalance.Add(currentBalance, am)
		utxo = append(utxo, u)
		if currentBalance.Cmp(amount) >= 0 { // >= amount {
			break
		}
	}
	logger.Log.Debugf("amount: %v, total balance: %v\n", amount, currentBalance)
	if amount.Cmp(currentBalance) == 1 {
		return fmt.Errorf("account %v has insufficient balance for token %v, required balance: %v, available balance: %v", account, constants.GINI, amount, currentBalance)
	}

	for i := 0; i < len(utxo); i++ {
		am, ok := big.NewInt(0).SetString(utxo[i].Amount, 10)
		if !ok {
			err := ginierr.New("failed to convert UTXO amount to big int while removing UTXO", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return err
		}
		if amount.Cmp(am) >= 0 { // >= utxo[i].Amount {
			logger.Log.Debugf("amount> delete: %s\n", utxo[i].Amount)
			amount = amount.Sub(amount, am)
			if e := ctx.DelStateWithoutKYC(utxo[i].Key); e != nil {
				err := ginierr.ErrFailedToDeleteState(e)
				logger.Log.Error(err.FullError())
				return err
			}
		} else if amount.Cmp(am) == -1 { // < utxo[i].Amount {
			logger.Log.Debugf("amount<: %s\n", utxo[i].Amount)
			if err := ctx.DelStateWithoutKYC(utxo[i].Key); err != nil {
				err := ginierr.ErrFailedToDeleteState(e)
				logger.Log.Error(err.FullError())
				return err
			}
			// Create a new utxo object
			utxo := models.Utxo{
				DocType: constants.UTXO,
				Account: account,
				Amount:  am.Sub(am, amount).String(),
			}
			utxoJSON, e := json.Marshal(utxo)
			if e != nil {
				err := ginierr.NewWithInternalError(e, "failed to marshal UTXO data while removing UTXO", http.StatusInternalServerError)
				logger.Log.Error(err.FullError())
				return err
			}

			if e := ctx.PutStateWithoutKYC(utxoKey, utxoJSON); e != nil {
				err := ginierr.ErrFailedToPutState(e)
				logger.Log.Error(err.FullError())
				return err
			}

		}
	}

	return nil
}
```
## AddUtxo Implementation

```go
func AddUtxo(ctx kalpsdk.TransactionContextInterface, account string, amountVal *big.Int) error {
	amount := new(big.Int).Set(amountVal)

	utxoKey, e := ctx.CreateCompositeKey(constants.UTXO, []string{account, ctx.GetTxID()})
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to create the composite key for owner:"+account, http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}

	logger.Log.Debugf("add amount: %v\n", amount)
	logger.Log.Debugf("utxoKey: %v\n", utxoKey)
	utxo := models.Utxo{
		DocType: constants.UTXO,
		Account: account,
		Amount:  amount.String(),
	}

	utxoJSON, e := json.Marshal(utxo)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal UTXO data while adding UTXO", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	logger.Log.Debugf("utxoJSON: %s\n", utxoJSON)

	if e := ctx.PutStateWithoutKYC(utxoKey, utxoJSON); e != nil {
		err := ginierr.ErrFailedToPutState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

```

## UpdateAllowance implementation
```go
func UpdateAllowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string, spent string) error {
	approvalKey, e := ctx.CreateCompositeKey(constants.Approval, []string{owner, spender})
	if e != nil {
		err := ginierr.ErrCreatingCompositeKey(e)
		logger.Log.Error(err.FullError())
		return err

	}
	// Get the current balance of the owner
	approvalByte, e := ctx.GetState(approvalKey)
	if e != nil {
		err := ginierr.ErrFailedToGetState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	var approval models.Allow
	if approvalByte != nil {
		if e := json.Unmarshal(approvalByte, &approval); e != nil {
			return fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", owner, spender, e)
		}
		approvalAmount, ok := big.NewInt(0).SetString(approval.Amount, 10)
		if !ok {
			err := ginierr.ErrConvertingStringToBigInt(approval.Amount)
			logger.Log.Error(err.FullError())
			return err
		}
		amountSpent, ok := big.NewInt(0).SetString(spent, 10)
		if !ok {
			err := ginierr.ErrConvertingStringToBigInt(spent)
			logger.Log.Error(err.FullError())
			return err
		}
		if amountSpent.Cmp(approvalAmount) == 1 { // amountToAdd > approvalAmount {
			err := ginierr.New("amount spent cannot be greater than allowance", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return err
		}
		approval.Amount = fmt.Sprint(approvalAmount.Sub(approvalAmount, amountSpent))
	}
	approvalJSON, e := json.Marshal(approval)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal approval data", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	// Update the state of the smart contract by adding the allowanceKey and value
	if e = ctx.PutStateWithoutKYC(approvalKey, approvalJSON); e != nil {
		err := ginierr.ErrFailedToPutState(e)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Approval, approvalJSON); e != nil {
		err := ginierr.ErrFailedToSetEvent(e, constants.Approval)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}
```

## Simple logic to replace complex if else in TransferFrom
```go
// check for allowance of spender first & 
// balances before calling this function
func UpdateBalances(signer, contractAdmin,sender, recipient string, gasFees, amount *big.Int) error {
 transactions := make(map[string]big.Int)

 transactions[contractAdmin] += gasFees // use actual addition function for big integer
 transactions[signer] -= gasFees

 transactions[sender] -= amount
 transactions[recipient] += amount

 for key,val := range transactions {
  if val < 0 {
   RemoveUTXO(key, val * -1)
  } else {
            AddUTXO(key, val)
        }
 }
}
```