// R2CI is an acronym for Right to Create Intellectual Property. It is a platform for maintaining the IP and showcasing those IP rights to others.
//
// This package provides the functions to create and maintain R2CI Assets and Token in the blokchain.
package kalpAccounting

import (
	//Standard Libs

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/p2eengineering/kalp-kaps/kaps"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

const attrRole = "hf.Type"

// const admintype = "client"
const nameKey = "name"
const symbolKey = "symbol"
const OwnerPrefix = "ownerId~assetId"
const MailabRoleAttrName = "MailabUserRole"
const PaymentRoleValue = "PaymentAdmin"
const GINI = "GINI"
const GINI_PAYMENT_TXN = "GINI_PAYMENT_TXN"
const env = "dev"

// const legalPrefix = "legal~tokenId"
type SmartContract struct {
	kalpsdk.Contract
}

type GiniTransaction struct {
	OffchainTxnId string `json:"OffchainTxnId" validate:"required"`
	Id            string
	Account       string `json:"Account" validate:"required"`
	DocType       string
	Amount        float64 `json:"Amount" validate:"required,gt=0"`
	Desc          string  `json:"Desc"`
}

type TransferNIU struct {
	TxnId     string  `json:"TxnId" validate:"required"`
	Sender    string  `json:"Sender" validate:"required"`
	Receiver  string  `json:"Receiver" validate:"required"`
	Id        string  `json:"Id" `
	DocType   string  `json:"DocType"`
	Amount    float64 `json:"Amount" validate:"required,gt=0"`
	TimeStamp string  `json:"TimeStamp" `
}
type Response struct {
	Status     string      `json:"status"`
	StatusCode uint        `json:"statusCode"`
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Response   interface{} `json:"response" `
}

func (s *SmartContract) InitLedger(ctx kalpsdk.TransactionContextInterface) error {
	logger := kalpsdk.NewLogger()
	logger.Infof("InitLedger invoked...")
	return nil
}

// Initializing smart contract
func (s *SmartContract) Initialize(ctx kalpsdk.TransactionContextInterface, name string, symbol string) (bool, error) {
	//check contract options are not already set, client is not authorized to change them once intitialized
	bytes, err := ctx.GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("failed to get Name: %v", err)
	}
	if bytes != nil {
		return false, fmt.Errorf("contract options are already set, client is not authorized to change them")
	}

	err = ctx.PutStateWithoutKYC(nameKey, []byte(name))
	if err != nil {
		return false, fmt.Errorf("failed to set token name: %v", err)
	}

	err = ctx.PutStateWithoutKYC(symbolKey, []byte(symbol))
	if err != nil {
		return false, fmt.Errorf("failed to set symbol: %v", err)
	}
	return true, nil
}

// 	if err := ctx.PutStateWithKYC(niuData.Id, niuJSON); err != nil {
// 		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
// 	}

// 	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: niuData.Account, ID: niuData.Id, Value: niuData.Amount}
// 	return kaps.EmitTransferSingle(ctx, transferSingleEvent)

// }
func (g *GiniTransaction) Validation() error {
	offchainTxnId := strings.Trim(g.OffchainTxnId, " ")
	if offchainTxnId == "" {
		return fmt.Errorf("invalid input OffchainTxnId")
	}

	account := strings.Trim(g.Account, " ")
	if account == "" {
		return fmt.Errorf("invalid input Account")
	}

	// desc := strings.Trim(g.Desc, " ")
	// if desc == "" {
	// 	return fmt.Errorf("invalid input desc")
	// }

	// if len(account) < 8 || len(account) > 60 {
	// 	return fmt.Errorf("account must be at least 8 characters long and shorter than 60 characters")
	// }
	//`~!@#$%^&*()-_+=[]{}\|;':",./<>?
	if strings.ContainsAny(account, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") {
		return fmt.Errorf("invalid Account")
	}
	return nil
}
func (t *TransferNIU) TransferNIUValidation() error {
	txnId := strings.Trim(t.TxnId, " ")
	if txnId == "" {
		return fmt.Errorf("invalid input TxnId")
	}

	sender := strings.Trim(t.Sender, " ")
	if sender == "" {
		return fmt.Errorf("invalid input Sender")
	}
	// if len(sender) < 8 || len(sender) > 60 {
	// 	return fmt.Errorf("sender must be at least 8 characters long and shorter than 60 characters")
	// }
	//`~!@#$%^&*()-_+=[]{}\|;':",./<>?
	if strings.ContainsAny(sender, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") {
		return fmt.Errorf("invalid sender")
	}
	receiver := strings.Trim(t.Receiver, " ")
	if receiver == "" {
		return fmt.Errorf("invalid input Receiver")
	}

	// docType := strings.Trim(t.DocType, " ")
	// if docType == "" {
	// 	return fmt.Errorf("invalid input DocType")
	// }
	// if len(receiver) < 8 || len(receiver) > 60 {
	// 	return fmt.Errorf("receiver must be at least 8 characters long and shorter than 60 characters")
	// }
	//`~!@#$%^&*()-_+=[]{}\|;':",./<>?
	if strings.ContainsAny(receiver, "`~!@#$%^&*()-_+=[]{}\\|;':\",./<>? ") {
		return fmt.Errorf("invalid receiver")
	}
	return nil
}
func (s *SmartContract) Mint(ctx kalpsdk.TransactionContextInterface, data string) (Response, error) {
	//check if contract has been intilized first
	logger := kalpsdk.NewLogger()
	logger.Infof("AddFunds---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, err:failed to check if contract is already initialized: %v ", http.StatusInternalServerError, err)
	}
	if !initialized {
		return Response{
			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, contract options need to be set before calling any function, call Initialize() to initialize contract", http.StatusInternalServerError)
	}

	logger.Infof("AddFunds CheckInitialized---->")
	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("payment admin role check failed: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error: payment admin role check failed: %v", http.StatusInternalServerError, err)
	}

	// Parse input data into NIU struct.
	var acc GiniTransaction
	errs := json.Unmarshal([]byte(data), &acc)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("failed to parse data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error: failed to parse data: %v", http.StatusBadRequest, err)
	}
	validate := validator.New()

	err = validate.Struct(acc)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error: inavalid input %s %s", http.StatusBadRequest, e.Field(), e.Tag())
		}
	}
	err = acc.Validation()
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("%v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:%v", http.StatusBadRequest, err)

	}
	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v,failed to read from world state: %v", http.StatusBadRequest, err)
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", acc.OffchainTxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusConflict,
		}, fmt.Errorf("error with status code %v,transaction %v already accounted", http.StatusConflict, acc.OffchainTxnId)
	}

	if acc.Amount <= 0 {
		return Response{
			Message:    "amount can't be less then 0",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v,amount can't be less then 0", http.StatusBadRequest)
	}

	acc.Id = GINI
	acc.DocType = GINI_PAYMENT_TXN

	logger.Infof("GINI amount %v\n", acc.Amount)
	// accJSON, err := json.Marshal(acc)
	// if err != nil {
	// 	return Response{
	// 		Message:    fmt.Sprintf("unable to Marshal Token struct : %v", err),
	// 		Success:    false,
	// 		Status:     "Failure",
	// 		StatusCode: http.StatusBadRequest,
	// 	}, fmt.Errorf("error with tatus code %v, unable to Marshal Token struct : %v", http.StatusBadRequest, err)
	// }

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to get client id: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}

	logger.Infof("MintToken operator---->%v\n", operator)
	// Mint tokens
	err = kaps.AddBalanceCredit(ctx, operator, []string{acc.Account}, acc.Amount, acc.OffchainTxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to mint tokens: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, failed to mint tokens: %v", http.StatusBadRequest, err)
	}

	logger.Infof("MintToken Amount---->%v\n", acc.Amount)
	// if err := ctx.PutStateWithoutKYC(acc.OffchainTxnId, accJSON); err != nil {
	// 	logger.Errorf("error: %v\n", err)
	// 	return Response{
	// 		Message:    fmt.Sprintf("Mint: unable to store GINI transaction data in blockchain: %v", err),
	// 		Success:    false,
	// 		Status:     "Failure",
	// 		StatusCode: http.StatusInternalServerError,
	// 	}, fmt.Errorf("error with status code %v, Mint: unable to store GINI transaction data in blockchain: %v", http.StatusInternalServerError, err)
	// }
	logger.Infof("Transfer single event: %v\n", acc.Amount)
	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: acc.Account, ID: acc.Id, Value: acc.Amount}
	if err := kaps.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to add funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v,error: unable to add funds: %v", http.StatusInternalServerError, err)
	}
	funcName, _ := ctx.GetFunctionAndParameters()
	response := map[string]interface{}{
		"txId":            ctx.GetTxID(),
		"txFcn":           funcName,
		"txType":          "Invoke",
		"transactionData": acc,
	}

	return Response{
		Message:    "Added funds successfully",
		Success:    true,
		Status:     "Success",
		StatusCode: http.StatusCreated,
		Response:   response,
	}, nil

}

func (s *SmartContract) Burn(ctx kalpsdk.TransactionContextInterface, data string) (Response, error) {
	//check if contract has been intilized first
	logger := kalpsdk.NewLogger()
	logger.Infof("RemoveFunds---->%s", env)
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v,error: failed to check if contract is already initialized: %v", http.StatusInternalServerError, err)

	}
	if !initialized {
		return Response{
			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v,error: contract options need to be set before calling any function, call Initialize() to initialize contract", http.StatusInternalServerError)
	}

	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("payment admin role check failed in Brun request: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error: payment admin role check failed in Brun request: %v", http.StatusInternalServerError, err)
	}

	// Parse input data into NIU struct.
	var acc GiniTransaction
	errs := json.Unmarshal([]byte(data), &acc)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("failed to parse data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v,error: failed to parse data: %v", http.StatusBadRequest, err)
	}

	logger.Infof("acc---->", acc)
	err = acc.Validation()
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("%v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:%v", http.StatusBadRequest, err)

	}
	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v,error: failed to read from world state: %v", http.StatusBadRequest, err)
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", acc.OffchainTxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusConflict,
		}, fmt.Errorf("error with status code %v,error: transaction %v already accounted", http.StatusConflict, acc.OffchainTxnId)
	}

	acc.Id = GINI
	acc.DocType = GINI_PAYMENT_TXN

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to get client id: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:failed to get client id: %v", http.StatusBadRequest, err)
	}

	err = kaps.RemoveBalance(ctx, acc.Id, []string{acc.Account}, acc.Amount)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("Remove balance in burn has error: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:Remove balance in burn has error: %v", http.StatusBadRequest, err)
	}

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to Marshal Token struct : %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:unable to Marshal Token struct : %v", http.StatusBadRequest, err)
	}

	validate := validator.New()
	err = validate.Struct(acc)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error: inavalid input %s %s", http.StatusBadRequest, e.Field(), e.Tag())
		}
	}

	logger.Infof("MintToken Amount---->", acc.Amount)
	if env == "dev" {
		if err = ctx.PutStateWithoutKYC(acc.OffchainTxnId, accJSON); err != nil {
			return Response{
				Message:    fmt.Sprintf("Burn: unable to store GINI transaction data in blockchain: %v", err),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error: Burn: unable to store GINI transaction data without kyc in blockchain: %v", http.StatusBadRequest, err)
		}
	} else if err = ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return Response{
			Message:    fmt.Sprintf("Burn: unable to store GINI transaction data in blockchain: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error: Burn: unable to store GINI transaction data in blockchain: %v", http.StatusBadRequest, err)
	}

	if err := kaps.EmitTransferSingle(ctx, kaps.TransferSingle{Operator: operator, From: acc.Account, To: "0x0", ID: acc.Id, Value: acc.Amount}); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to remove funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:unable to remove funds: %v", http.StatusBadRequest, err)
	}

	funcName, _ := ctx.GetFunctionAndParameters()
	response := map[string]interface{}{
		"txId":            ctx.GetTxID(),
		"txFcn":           funcName,
		"txType":          "Invoke",
		"transactionData": acc,
	}

	//return kaps.EmitTransferSingle(ctx, kaps.TransferSingle{Operator: operator, From: acc.Account, To: "0x0", ID: acc.Id, Value: acc.Amount})
	return Response{
		Message:    "Funds removed successfully",
		Success:    true,
		Status:     "Success",
		StatusCode: http.StatusCreated,
		Response:   response,
	}, nil
}

func (s *SmartContract) TransferToken(ctx kalpsdk.TransactionContextInterface, data string) (Response, error) {
	logger := kalpsdk.NewLogger()
	logger.Infof("TransferToken---->%s", env)
	var transferNIU TransferNIU
	errs := json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("error in parsing transfer request data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:error in parsing transfer request data: %v", http.StatusBadRequest, errs)
	}

	validate := validator.New()
	err := validate.Struct(transferNIU)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error: inavalid input %s %s", http.StatusBadRequest, e.Field(), e.Tag())
		}
	}

	logger.Infof("transferNIU", transferNIU)
	err = transferNIU.TransferNIUValidation()
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("%v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:%v", http.StatusBadRequest, err)

	}
	txnJSON, err := ctx.GetState(transferNIU.TxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:failed to read from world state: %v", http.StatusBadRequest, err)
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", transferNIU.TxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:transaction %v already accounted", http.StatusBadRequest, transferNIU.TxnId)
	}

	if transferNIU.Sender == transferNIU.Receiver {
		return Response{
			Message:    "transfer to self is not allowed",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:transfer to self is not allowed", http.StatusBadRequest)
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to get client id: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:failed to get client id: %v", http.StatusBadRequest, err)
	}
	logger.Infof("operator-->", operator, transferNIU.Sender)
	transferNIUJSON, err := json.Marshal(transferNIU)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to Marshal Token struct : %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:unable to Marshal Token struct : %v", http.StatusBadRequest, err)
	}

	if env != "dev" {
		kycCheck, err := kaps.IsKyced(ctx, transferNIU.Sender)
		if err != nil {
			return Response{
				Message:    fmt.Sprintf("not able to do KYC check for sender:%s, error:%v", transferNIU.Sender, err),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error:not able to do KYC check for sender:%s, error:%v", http.StatusBadRequest, transferNIU.Sender, err)
		}
		if !kycCheck {
			return Response{
				Message:    fmt.Sprintf("sender %s is not kyced", transferNIU.Sender),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error:sender %s is not kyced", http.StatusBadRequest, transferNIU.Sender)

		}

		// Check KYC status for each recipient
		kycCheck, err = kaps.IsKyced(ctx, transferNIU.Receiver)
		if err != nil {
			return Response{
				Message:    fmt.Sprintf("not able to do KYC check for receiver:%s, error:%v", transferNIU.Receiver, err),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error:not able to do KYC check for receiver:%s, error:%v", http.StatusBadRequest, transferNIU.Receiver, err)
		}
		if !kycCheck {
			return Response{
				Message:    fmt.Sprintf("receiver %s is not kyced", transferNIU.Receiver),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error:receiver %s is not kyced", http.StatusBadRequest, transferNIU.Receiver)
		}
		if err = ctx.PutStateWithKYC(transferNIU.TxnId, transferNIUJSON); err != nil {
			return Response{
				Message:    fmt.Sprintf("Transfer: unable to store GINI transaction data in blockchain: %v", err),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}, fmt.Errorf("error with status code %v, error: Transfer: unable to store GINI transaction data without kyc in blockchain: %v", http.StatusBadRequest, err)
		}
	} else if err = ctx.PutStateWithoutKYC(transferNIU.TxnId, transferNIUJSON); err != nil {
		return Response{
			Message:    fmt.Sprintf("Transfer: unable to store GINI transaction data in blockchain: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error: Transfer: unable to store GINI transaction data in blockchain: %v", http.StatusBadRequest, err)
	}
	transferNIU.Id = GINI
	transferNIU.DocType = GINI_PAYMENT_TXN

	b, err := s.GetTransactionBalance(ctx, transferNIU.Sender, transferNIU.Amount)
	if err != nil || !b {
		return Response{
			Message:    "error while GetTransactionBalance",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:error GetTransactionBalance", http.StatusBadRequest)
	}
	// Withdraw the funds from the sender address
	// err = kaps.RemoveBalanceCredit(ctx, transferNIU.Id, []string{transferNIU.Sender}, transferNIU.Receiver, transferNIU.Amount, transferNIU.TxnId)
	// if err != nil {
	// 	return Response{
	// 		Message:    "error while reducing balance",
	// 		Success:    false,
	// 		Status:     "Failure",
	// 		StatusCode: http.StatusInternalServerError,
	// 	}, fmt.Errorf("error with status code %v, error:error while reducing balance %v", http.StatusBadRequest, err)
	// }
	reciver, _ := s.GetOwnerInfo(ctx, transferNIU.Receiver)
	reciver.Id = GINI
	reciver.DebitAccount = transferNIU.Sender
	reciver.Account = transferNIU.Receiver
	reciver.CreditAccount = transferNIU.Receiver
	reciver.Amount = transferNIU.Amount
	reciver.DocType = "Owner"
	reciverJSON, err := json.Marshal(reciver)
	fmt.Printf("recieverJSON: %s\n", reciverJSON)
	if err != nil {
		return Response{
			Message:    "error while marshalling reciever json",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:error while marshalling reciever json %v", http.StatusBadRequest, err)
	}
	err = ctx.PutStateWithoutKYC(transferNIU.TxnId, reciverJSON)
	if err != nil {
		return Response{
			Message:    "error while putting reciever value in ledger",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:error while putting reciever value in ledger %v", http.StatusBadRequest, err)
	}
	// Deposit the fund to the recipient address
	// err = kaps.AddBalanceCredit(ctx, transferNIU.Id, []string{transferNIU.Receiver}, transferNIU.Amount, transferNIU.TxnId)
	// if err != nil {
	// 	return Response{
	// 		Message:    "error while adding balance",
	// 		Success:    false,
	// 		Status:     "Failure",
	// 		StatusCode: http.StatusInternalServerError,
	// 	}, fmt.Errorf("error with status code %v, error:error while adding balance", http.StatusBadRequest)
	// }
	fmt.Println("#693 PutStateWithoutKYC")
	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: transferNIU.Sender, To: transferNIU.Receiver, ID: transferNIU.Id, Value: transferNIU.Amount}
	if err := kaps.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to remove funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}, fmt.Errorf("error with status code %v, error:unable to remove funds: %v", http.StatusBadRequest, err)
	}

	funcName, _ := ctx.GetFunctionAndParameters()
	response := map[string]interface{}{
		"txId":            ctx.GetTxID(),
		"txFcn":           funcName,
		"txType":          "Invoke",
		"transactionData": transferNIU,
	}
	fmt.Println("#711 response")
	return Response{
		Message:    "Funds transfered successfully",
		Success:    true,
		Status:     "Success",
		StatusCode: http.StatusCreated,
		Response:   response,
	}, nil

}
func (s *SmartContract) GetBalance(ctx kalpsdk.TransactionContextInterface, account string) (float64, error) {
	logger := kalpsdk.NewLogger()
	logger.Info("Get Balance---->")
	account = strings.Trim(account, " ")
	if account == "" {
		return 0, fmt.Errorf("invalid input account is required")
	}
	creditBalance, err := s.GetCreditBalance(ctx, account)
	if err != nil {
		return 0, err
	}
	logger.Infof("creditBalance: %f\n", creditBalance)
	debitBalance, err := s.GetDebitBalance(ctx, account)
	if err != nil {
		return 0, err
	}
	logger.Infof("debitBalance: %f\n", debitBalance)
	return (creditBalance - debitBalance), err
}
func (s *SmartContract) TransferMultipleToken(ctx kalpsdk.TransactionContextInterface, data string) (Response, error) {
	logger := kalpsdk.NewLogger()
	logger.Infof("TransferMutipleToken---->%s", env)
	var res Response
	var transferNIU []TransferNIU
	errs := json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("error in parsing transfer request data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}, fmt.Errorf("error with status code %v, error:error in parsing transfer request data: %v", http.StatusBadRequest, errs)
	}
	for i := 0; i < len(transferNIU); i++ {
		jsonStr, _ := json.Marshal(transferNIU[i])
		fmt.Println(string(jsonStr))
		res, err := s.TransferToken(ctx, string(jsonStr))
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
func (s *SmartContract) GetTransactionBalance(ctx kalpsdk.TransactionContextInterface, account string, amount float64) (bool, error) {
	c, err := s.GetCreditBalance(ctx, account)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	d, err := s.GetDebitBalance(ctx, account)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	fmt.Printf("c %v\n", c)
	fmt.Printf("d %v\n", d)
	if c-d >= amount {
		return true, nil
	}
	return false, nil
}
func (s *SmartContract) GetOwnerInfo(ctx kalpsdk.TransactionContextInterface, account string) (kaps.Owner, error) {
	logger := kalpsdk.NewLogger()
	var owner kaps.Owner
	id := GINI
	ownerKey, err := ctx.CreateCompositeKey(OwnerPrefix, []string{account, id})
	logger.Infof("ownerKey %v\n", ownerKey)
	if err != nil {
		return kaps.Owner{}, fmt.Errorf("failed to create composite key for account %v and token %v: %v", account, id, err)
	}

	// Retrieve the current balance for the account and token ID
	ownerBytes, err := ctx.GetState(ownerKey)
	if err != nil {
		return kaps.Owner{}, fmt.Errorf("failed to read balance for account %v and token %v: %v", account, id, err)
	}
	if ownerBytes != nil {
		logger.Infof("unmarshelling balance bytes")
		// Unmarshal the current balance into an Owner struct
		err = json.Unmarshal(ownerBytes, &owner)
		if err != nil {
			return kaps.Owner{}, fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", account, id, err)
		}
		logger.Infof("owner %v\n", owner)
		return owner, nil
	}
	return kaps.Owner{}, fmt.Errorf("owner not found")
}
func (s *SmartContract) GetCreditBalance(ctx kalpsdk.TransactionContextInterface, account string) (float64, error) {
	logger := kalpsdk.NewLogger()
	queryString := `{"selector": {"creditAccount": "` + account + `", "docType": "Owner"}	 }`
	fmt.Printf("query: %v\n", queryString)
	creditBalance := 0.0
	queryResults, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	defer queryResults.Close()

	var owners []*kaps.Owner
	for queryResults.HasNext() {
		result, err := queryResults.Next()
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve query result: %v", err)
		}

		// Use ctx.GetState correctly with 2 return values
		value, err := ctx.GetState(result.Key)
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve owners: %v", err)
		}

		var owner kaps.Owner
		err = json.Unmarshal(value, &owner)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal owners: %v", err)
		}

		owners = append(owners, &owner)
		logger.Infof("owner ----------->%v\n", owners)
	}
	for i := 0; i < len(owners); i++ {
		creditBalance += owners[i].Amount
	}
	return creditBalance, nil
}
func (s *SmartContract) GetDebitBalance(ctx kalpsdk.TransactionContextInterface, account string) (float64, error) {
	logger := kalpsdk.NewLogger()
	queryString := `{"selector": {"debitAccount": "` + account + `", "docType": "Owner"}	 }`
	fmt.Printf("query: %v\n", queryString)
	debitBalance := 0.0
	queryResults, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	defer queryResults.Close()

	var owners []*kaps.Owner
	for queryResults.HasNext() {
		result, err := queryResults.Next()
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve query result: %v", err)
		}

		// Use ctx.GetState correctly with 2 return values
		value, err := ctx.GetState(result.Key)
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve owners: %v", err)
		}

		var owner kaps.Owner
		err = json.Unmarshal(value, &owner)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal owners: %v", err)
		}

		owners = append(owners, &owner)
		logger.Infof("owner ----------->%v\n", owners)
	}
	for i := 0; i < len(owners); i++ {
		debitBalance += owners[i].Amount
	}
	return debitBalance, nil
}
func (s *SmartContract) GetBalanceForAccount(ctx kalpsdk.TransactionContextInterface, account string) (float64, error) {
	logger := kalpsdk.NewLogger()
	account = strings.Trim(account, " ")
	if account == "" {
		return 0, fmt.Errorf("invalid input account is required")
	}
	var owner kaps.Owner
	id := GINI
	ownerKey, err := ctx.CreateCompositeKey(OwnerPrefix, []string{account, id})
	logger.Infof("ownerKey %v\n", ownerKey)
	if err != nil {
		return 0, fmt.Errorf("failed to create composite key for account %v and token %v: %v", account, id, err)
	}

	// Retrieve the current balance for the account and token ID
	ownerBytes, err := ctx.GetState(ownerKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read balance for account %v and token %v: %v", account, id, err)
	}

	if ownerBytes != nil {
		logger.Infof("unmarshelling balance bytes")
		// Unmarshal the current balance into an Owner struct
		err = json.Unmarshal(ownerBytes, &owner)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", account, id, err)
		}
		logger.Infof("owner %v\n", owner)
		return owner.Amount, nil
	}

	return 0, nil
}

// GetHistoryForAsset is a smart contract function which list the complete history of particular R2CI asset from blockchain ledger.
func (s *SmartContract) GetHistoryForAsset(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return "", fmt.Errorf(err.Error())
		}
		if bArrayMemberAlreadyWritten {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")

		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return buffer.String(), nil
}

// GetTransactionTimestamp retrieves the transaction timestamp from the context and returns it as a string.
func (s *SmartContract) GetTransactionTimestamp(ctx kalpsdk.TransactionContextInterface) (string, error) {
	timestamp, err := ctx.GetTxTimestamp()
	if err != nil {
		return "", err
	}

	return timestamp.AsTime().String(), nil
}
