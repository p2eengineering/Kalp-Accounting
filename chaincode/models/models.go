package models

import (
	"encoding/json"
	"fmt"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/helper"
	"gini-contract/chaincode/logger"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

type UserRole struct {
	Id      string `json:"user"`
	Role    string `json:"role"`
	DocType string `json:"docType"`
	Desc    string `json:"desc"`
}

type Sender struct {
	Sender string `json:"sender"`
}

type Utxo struct {
	Key     string `json:"_id,omitempty"`
	Account string `json:"account"`
	DocType string `json:"docType"`
	Amount  string `json:"amount"`
}

type Allow struct {
	Owner   string `json:"owner"`
	Amount  string `json:"amount"`
	DocType string `json:"docType"`
	Spender string `json:"spendor"`
}

func SetAllowance(ctx kalpsdk.TransactionContextInterface, spender string, amount string) error {
	signer, err := helper.GetUserId(ctx)
	if err != nil {
		return ginierr.ErrFailedToGetPublicAddress
	}
	if !helper.IsValidAddress(spender) {
		return ginierr.ErrInvalidAddress(spender)
	}
	if !helper.IsAmountProper(amount) {
		return ginierr.ErrInvalidAmount(amount)
	}
	approvalKey, err := ctx.CreateCompositeKey(constants.Approval, []string{signer, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for owner with address %s and spender with address %s: %v", signer, spender, err)
	}

	var approval = Allow{
		Owner:   signer,
		Amount:  amount,
		DocType: constants.Allowance,
		Spender: spender,
	}
	approvalJSON, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}

	err = ctx.PutStateWithoutKYC(approvalKey, approvalJSON)
	if err != nil {
		return fmt.Errorf("failed to update data of smart contract: %v", err)
	}

	logger.Log.Debugf("owner %s approved a withdrawal allowance of %s for spender %s", signer, amount, spender)

	return nil
}

func GetAllowance(ctx kalpsdk.TransactionContextInterface, signer string, spender string) (string, error) {
	approvalKey, err := ctx.CreateCompositeKey(constants.Approval, []string{signer, spender})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for owner with address %s and spender with address %s: %v", signer, spender, err)
	}
	approvalByte, err := ctx.GetState(approvalKey)
	if err != nil {
		return "", fmt.Errorf("failed to read current balance of owner with address %s and spender with address %s from world state: %v", signer, spender, err)
	}
	var approval Allow
	if approvalByte != nil {
		err = json.Unmarshal(approvalByte, &approval)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal allow struct for owner %v and spender %v: %v", signer, spender, err)
		}
	}
	if approval.Amount == "" {
		return "0", nil
	}

	return approval.Amount, nil
}
