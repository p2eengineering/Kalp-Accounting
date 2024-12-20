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
- Check this condition in Transfer !helper.IsValidAddress(signer) 
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

```go
// TODO: needs to be deleted just for testing purposes
func (s *SmartContract) DeleteDocTypes(ctx kalpsdk.TransactionContextInterface, queryString string) (string, error) {

	logger := kalpsdk.NewLogger()

	// queryString := `{"selector":{"DocType":"` + docType + `","Id":"GINI"}}`

	logger.Infof("queryString: %s\n", queryString)
	resultsIterator, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return "fail", fmt.Errorf("err:failed to fetch UTXO tokens for: %v", err)
	}
	if !resultsIterator.HasNext() {
		return "fail", fmt.Errorf("error with status code %v, err:no records to delete", http.StatusInternalServerError)

	}

	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return "fail", fmt.Errorf("error with status code %v, err:failed to fetch unlocked tokens: %v ", http.StatusInternalServerError, err)
		}

		logger.Infof("deleting %s\n", queryResult.Key)

		if err = ctx.DelStateWithoutKYC(queryResult.Key); err != nil {
			logger.Errorf("Error in deleting %s\n", err.Error())
		}
	}
	return "success", nil

}

func (s *SmartContract) GetTotalSUMUTXO(ctx kalpsdk.TransactionContextInterface) (string, error) {

	queryString := `{"selector":{"docType":"` + constants.UTXO + `"}}`
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
```

func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, recipient string, value string) (bool, error) {
	logger.Log.Info("Transfer operation initiated")

	signer, err := ctx.GetUserID()
	if err != nil {
		err := ginierr.NewWithInternalError(err, "error getting signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	sender := signer

	var gasFees *big.Int
	if gasFeesString, err := s.GetGasFees(ctx); err != nil {
		return false, err
	} else if val, ok := big.NewInt(0).SetString(gasFeesString, 10); !ok {
		return false, ginierr.New("invalid gas fees found:"+gasFeesString, http.StatusInternalServerError)
	} else {
		gasFees = val
	}

	// Determine if the call is from a contract
	callingContractAddress, err := internal.GetCallingContractAddress(ctx, s.GetName())
	if err != nil || len(callingContractAddress) == 0 {
		callingContractAddress = s.GetName()
	}
	// TODO: check if error needs to be handled here
	logger.Log.Info("callingContractAddress => ", callingContractAddress, err)

	var spender string
	var e error

	vestingContract, err := s.GetVestingContract(ctx)
	if err != nil {
		return false, err
	}
	bridgeContract, err := s.GetBridgeContract(ctx)
	if err != nil {
		return false, err
	}

	if callingContractAddress != s.GetName() && callingContractAddress != "" {
		if callingContractAddress != bridgeContract && callingContractAddress != vestingContract {
			err := ginierr.New("The calling contract is not bridge contract or vesting contract", http.StatusBadRequest)
			logger.Log.Error(err.FullError())
			return false, err
		}
		sender = callingContractAddress
		if signer, e = ctx.GetUserID(); e != nil {
			err := ginierr.NewWithInternalError(e, "error getting signer", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return false, err
		}
	} else {
		if sender, e = ctx.GetUserID(); e != nil {
			err := ginierr.NewWithInternalError(e, "error getting signer", http.StatusInternalServerError)
			logger.Log.Error(err.FullError())
			return false, err
		}
		signer = sender
	}

	logger.Log.Info("signer ==> ", signer, spender)
	gatewayAdmin := internal.GetGatewayAdminAddress(ctx)

	if signer == gatewayAdmin {
		var gasDeductionAccount models.Sender
		err := json.Unmarshal([]byte(recipient), &gasDeductionAccount)
		if err == nil {
			sender = gasDeductionAccount.Sender
			recipient = constants.KalpFoundationAddress

			gasFees = big.NewInt(0)
		} else {
			return false, fmt.Errorf("failed to unmarshal recipient: %v", err)
		}
	}

	logger.Log.Info("signer ==> ", signer, spender, recipient, helper.IsValidAddress(sender), helper.IsValidAddress(recipient))

	// Input validation
	if helper.IsContractAddress(signer) {
		return false, ginierr.New("signer cannot be a contract", http.StatusBadRequest)
	} else if !helper.IsValidAddress(signer) {
		return false, ginierr.ErrIncorrectAddress("signer")
	}
	if helper.IsContractAddress(sender) && helper.IsContractAddress(recipient) {
		return false, ginierr.New("both sender and recipient cannot be contracts", http.StatusBadRequest)
	}
	if !helper.IsValidAddress(sender) {
		return false, ginierr.ErrIncorrectAddress("sender")
	} else if !helper.IsValidAddress(recipient) {
		return false, ginierr.ErrIncorrectAddress("recipient")
	} else if !helper.IsAmountProper(value) {
		return false, ginierr.ErrInvalidAmount(value)
	}

	if denied, err := internal.IsDenied(ctx, signer); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.DeniedAddress(signer)
	}

	if denied, err := internal.IsDenied(ctx, sender); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.DeniedAddress(sender)
	}

	if denied, err := internal.IsDenied(ctx, recipient); err != nil {
		return false, err
	} else if denied {
		return false, ginierr.DeniedAddress(recipient)
	}

	var kycSender, kycSigner bool
	if kycSender, e = ctx.GetKYC(sender); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for sender", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if kycSigner, e = ctx.GetKYC(signer); e != nil {
		err := ginierr.NewWithInternalError(e, "error fetching KYC for signer", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return false, err
	}

	if !(kycSender || kycSigner) {
		err := ginierr.New(fmt.Sprintf("IsSender kyced: %v, IsSigner kyced: %v", kycSender, kycSigner), http.StatusForbidden)
		logger.Log.Error(err.FullError())
		return false, err
	}

	amount, ok := big.NewInt(0).SetString(value, 10)
	if !ok || amount.Cmp(big.NewInt(0)) != 1 {
		return false, ginierr.ErrInvalidAmount(value)
	}

	senderBalance, err := s.balance(ctx, sender)
	if err != nil {
		return false, err
	}

	signerBalance, err := s.balance(ctx, signer)
	if err != nil {
		return false, err
	}

	if sender == signer {
		if senderBalance.Cmp(new(big.Int).Add(amount, gasFees)) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount + gas fees", http.StatusBadRequest)
		}
	} else {
		if senderBalance.Cmp(amount) < 0 {
			return false, ginierr.New("insufficient balance in sender's account for amount", http.StatusBadRequest)
		}
		if signerBalance.Cmp(gasFees) < 0 {
			return false, ginierr.New("insufficient balance in signer's account for gas fees", http.StatusBadRequest)
		}
	}

	if sender == signer && sender == recipient {
		if sender != constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, gasFees); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
				return false, err
			}
		}
	} else if sender == signer && sender != recipient {
		if sender == constants.KalpFoundationAddress {
			if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
				return false, err
			}
			if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
				return false, err
			}
		} else {
			if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, sender, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	} else {
		if signer == gatewayAdmin {
			if sender != constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, amount); err != nil {
					return false, err
				}
			}
		} else {
			if signer == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
			} else if recipient == constants.KalpFoundationAddress {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, new(big.Int).Add(amount, gasFees)); err != nil {
					return false, err
				}
			} else {
				if err = internal.RemoveUtxo(ctx, signer, gasFees); err != nil {
					return false, err
				}
				if err = internal.RemoveUtxo(ctx, sender, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, recipient, amount); err != nil {
					return false, err
				}
				if err = internal.AddUtxo(ctx, constants.KalpFoundationAddress, gasFees); err != nil {
					return false, err
				}
			}
		}
	}

	eventPayload := map[string]interface{}{
		"from":  sender,
		"to":    recipient,
		"value": amount.String(),
	}
	eventBytes, _ := json.Marshal(eventPayload)
	_ = ctx.SetEvent("Transfer", eventBytes)

	return true, nil
}




1. Third party game (gasFee deduction and gateway normal transfer)
2. Direct gini transfer (user->user)
3. From vesting contract (contract(gini&bridge) -> user , user -> contract(any))
Function Transfer(ctx, recipient , amount) (bool,error):
    Logger := InitializeLogger()
    Logger.Info("Transfer operation initiated")

  KalpFoundation := GetKalpFoundationAdminAddress(ctx)
  signer:=GetSigner()
  GatewayAdmin := GetGatewayAdminAddress(ctx)
    IF IsValidAddress(recipient) 
    gas := CalculateGasFee()
      if amount<gas{
          return false,ERROR("amount is not proper ")
      }
  ELSE IF signer==GatewayAdmin && IsRecipientMarshallable()   // this function represent gas fees deduction scenario
    gasDeductionAccount,err:=Unmarshal(recipient)
    IF !IsUserAddress(gasDeductionAccount)
      return false,ERROR
        IF err==nil 
            sender=gasDeductionAccount
      recipient=CONTRACT_ADMIN
            actualAmount:=amount
            gas=0     
    if amount<0{
          return false,ERROR("amount is not proper ")
      }
  ELSE
    return false,ERROR("recipient is not proper ")

    actualAmount:=amount-gas
    sender:=signer

    calledContractAddress,err = GetCalledContractAddress(ctx)
    if err!=nil{
        return false, ERROR("getting called contract address ")
    }
    if (calledContractAddress!=GINI_CONTRACT_NAME) {
        if calledContractAddress==BRIDGE_CONTRACT_NAME || calledContractAddress=VESTING_CONTRACT_NAME{
            sender=calledContractAddress
        }else{
            return false,ERROR("Cannot use contract other than bridge contract or vesting contract")
        }
    }


    # Validate inputs
    IF IsContractAddress(signer) 
      RETURN false,ERROR("signer cannot be contract")
    ELSE IF !IsUserAddress(signer)
      RETURN false,ERROR("signer is not proper userAddress")
    IF IsContractAddress(sender)&&IsContractAddress(recipient)
      RETURN false,ERROR("both sender and recipient cannot be contract")
    IF !IsValidAddress(sender) 
      RETURN false,ERROR("sender address is not valid")

    IF IsDenied(sender) 
        return false,ERROR("sender has been denied")
    ELSE IF IsDenied(recipient)
        return false,ERROR("recipient has been denied")
    ELSE IF IsDenied(signer)
        return false,ERROR("signer has been denied") 

    IF !(IsKYCed(sender) || IsKYCed(signer))
        RETURN false,Error()   

    IF balanceOf(sender)<actualamount + gas
        return false,ERROR("insufficient amount in sender")

    // normal user transfer
    IF sender==signer AND sender==recipient
        IF sender==CONTRACT_ADMIN
            // Do Nothing
        ELSE
            RemoveUTXO(ctx,sender,gas)
            AddUTXO(ctx,CONTRACT_ADMIN,gas)
    ELSE IF sender==signer AND sender!=recipient
        IF sender==CONTRACT_ADMIN
            RemoveUTXO(ctx,sender,actualAmount)
            AddUTXO(ctx,recipient,actualAmount)
        ELSE
            IF recipient==CONTRACT_ADMIN
                RemoveUTXO(ctx,sender,actualAmount+gas)
                AddUTXO(ctx,recipient,actualAmount+gas)
            ELSE
                RemoveUTXO(ctx,sender,actualAmount+gas)
                AddUTXO(ctx,recipient,actualAmount)
                AddUTXO(ctx,CONTRACT_ADMIN,gas)
    // scenario: sender is Contract to user transfer or gateway gas fee deduction
    // 1. gateway gas fee deduction from CONTRACT_ADMIN account
    ELSE // sender!=signer // recipient is a user                                          // when sender!=signer then it is scenario for call coming from another contract or from gateway
        IF signer==gatewayAdmin
            IF sender == CONTRACT_ADMIN THEN
              // Do nothing
            ELSE
              RemoveUTXO(ctx,sender,actualAmount)
              AddUTXO(ctx,CONTRACT_ADMIN,actualAmount)
        ELSE 
      RemoveUTXO(ctx,sender,actualAmount+gas)
          IF recipient==CONTRACT_ADMIN
            AddUTXO(ctx,recipient,actualAmount+gas)
          ELSE
            AddUTXO(ctx,recipient,actualAmount)
            AddUTXO(ctx,CONTRACT_ADMIN,gas)


    # Emit transfer event
    EmitEvent("Transfer", {"from": sender, "to": recipient, "value": amount})

    RETURN