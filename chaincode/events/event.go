package events

import (
	"encoding/json"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/logger"
	"net/http"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

type DeniedEvent struct {
	Address string `json:"address"`
}
type AllowedEvent struct {
	Address string `json:"address"`
}

type ApprovalEvent struct {
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
	Value   string `json:"value"`
}
type TransferEvent struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
}

type MintEvent struct {
	Account string `json:"account"`
	Value   string `json:"value"`
}

func EmitDenied(ctx kalpsdk.TransactionContextInterface, address string) error {
	deniedEvent := DeniedEvent{
		Address: address,
	}
	deniedEventJSON, e := json.Marshal(deniedEvent)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal Denied event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Denied, deniedEventJSON); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to emit Denied event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func EmitAllowed(ctx kalpsdk.TransactionContextInterface, address string) error {
	allowedEvent := AllowedEvent{
		Address: address,
	}
	allowedEventJSON, e := json.Marshal(allowedEvent)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal Allowed event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Allowed, allowedEventJSON); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to emit Allowed event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func EmitTransfer(ctx kalpsdk.TransactionContextInterface, from string, to string, value string) error {
	transferEvent := TransferEvent{
		From:  from,
		To:    to,
		Value: value,
	}
	transferEventJSON, e := json.Marshal(transferEvent)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal Transfer event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Transfer, transferEventJSON); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to emit Transfer event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func EmitApproval(ctx kalpsdk.TransactionContextInterface, owner string, spender string, value string) error {
	approvalEvent := ApprovalEvent{
		Owner:   owner,
		Spender: spender,
		Value:   value,
	}
	approvalEventJSON, e := json.Marshal(approvalEvent)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal Approval event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Approval, approvalEventJSON); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to emit Approval event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}

func EmitMint(ctx kalpsdk.TransactionContextInterface, account string, value string) error {
	mintEvent := MintEvent{
		Account: account,
		Value:   value,
	}
	mintEventJSON, e := json.Marshal(mintEvent)
	if e != nil {
		err := ginierr.NewWithInternalError(e, "failed to marshal Mint event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	if e := ctx.SetEvent(constants.Mint, mintEventJSON); e != nil {
		err := ginierr.NewWithInternalError(e, "failed to emit Mint event", http.StatusInternalServerError)
		logger.Log.Error(err.FullError())
		return err
	}
	return nil
}
