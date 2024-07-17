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
const GatewayRoleValue = "GatewayAdmin"
const PaymentRoleValue = "PaymentAdmin"
const GINI = "GINI"
const GINI_PAYMENT_TXN = "GINI_PAYMENT_TXN"

// const legalPrefix = "legal~tokenId"
type SmartContract struct {
	kalpsdk.Contract
}

type GiniTransaction struct {
	OffchainTxnId string  `json:"OffchainTxnId" validate:"required"`
	Id            string  `json:"Id" `
	Account       string  `json:"Account" validate:"required"`
	DocType       string  `json:"DocType" `
	Amount        float64 `json:"Amount" validate:"required,gt=0"`
	Desc          string  `json:"Desc" validate:"required"`
}

type TransferNIU struct {
	TxnId     string  `json:"TxnId" validate:"required"`
	Sender    string  `json:"Sender" validate:"required"`
	Receiver  string  `json:"Receiver" validate:"required"`
	Id        string  `json:"Id" `
	DocType   string  `json:"DocType" validate:"required"`
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
	fmt.Println("InitLedger invoked...")
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

// 	fmt.Println("MintToken Amount---->", niuData.Amount)

// 	if err := ctx.PutStateWithKYC(niuData.Id, niuJSON); err != nil {
// 		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
// 	}

// 	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: niuData.Account, ID: niuData.Id, Value: niuData.Amount}
// 	return kaps.EmitTransferSingle(ctx, transferSingleEvent)

// }

func (s *SmartContract) Mint(ctx kalpsdk.TransactionContextInterface, data string) Response {
	//check if contract has been intilized first
	fmt.Println("AddFunds---->")

	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}
	if !initialized {
		return Response{
			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	fmt.Println("AddFunds CheckInitialized---->")
	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("payment admin role check failed: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Parse input data into NIU struct.
	var acc GiniTransaction
	errs := json.Unmarshal([]byte(data), &acc)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("failed to parse data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}
	validate := validator.New()

	err = validate.Struct(acc)
	if err != nil {
		fmt.Println("Validation failed:")
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}
		}
	}
	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", acc.OffchainTxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusConflict,
		}
	}

	if acc.Amount <= 0 {
		return Response{
			Message:    "amount can't be less then 0",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	acc.Id = GINI
	acc.DocType = GINI_PAYMENT_TXN

	fmt.Println("GINI amount", acc.Amount)
	accJSON, err := json.Marshal(acc)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to Marshal Token struct : %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to get client id: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	fmt.Println("MintToken operator---->", operator)

	// Mint tokens
	err = kaps.MintHelperWithoutKYC(ctx, operator, []string{acc.Account}, acc.Id, acc.Amount, kaps.DocTypeNIU)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return Response{
			Message:    fmt.Sprintf("failed to mint tokens: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithoutKYC(acc.OffchainTxnId, accJSON); err != nil {
		fmt.Printf("error: %v\n", err)
		return Response{
			Message:    fmt.Sprintf("unable to put Asset struct in statedb: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	fmt.Printf("Transfer single event: %v\n", acc.Account)
	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: acc.Account, ID: acc.Id, Value: acc.Amount}
	if err := kaps.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to add funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
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
	}

}

func (s *SmartContract) Burn(ctx kalpsdk.TransactionContextInterface, data string) Response {
	//check if contract has been intilized first

	fmt.Println("RemoveFunds---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}

	}
	if !initialized {
		return Response{
			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, GatewayRoleValue)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("gateway admin role check failed: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
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
		}
	}

	fmt.Println("acc---->", acc)

	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", acc.OffchainTxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusConflict,
		}
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
		}
	}

	err = kaps.RemoveBalance(ctx, acc.Id, []string{acc.Account}, acc.Amount)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("Remove balance in burn has error: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to Marshal Token struct : %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	validate := validator.New()
	err = validate.Struct(accJSON)
	if err != nil {
		fmt.Println("Validation failed:")
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}
		}
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to put Asset struct in statedb: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	if err := kaps.EmitTransferSingle(ctx, kaps.TransferSingle{Operator: operator, From: acc.Account, To: "0x0", ID: acc.Id, Value: acc.Amount}); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to remove funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
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
	}
}

func (s *SmartContract) TransferToken(ctx kalpsdk.TransactionContextInterface, data string) Response {

	var transferNIU TransferNIU

	errs := kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, GatewayRoleValue)
	if errs != nil {

		return Response{
			Message:    fmt.Sprintf("gateway admin role check failed: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusUnauthorized,
		}
	}

	errs = json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return Response{
			Message:    fmt.Sprintf("error is parsing transfer request data: %v", errs),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	validate := validator.New()
	err := validate.Struct(transferNIU)
	if err != nil {
		fmt.Println("Validation failed:")
		for _, e := range err.(validator.ValidationErrors) {
			return Response{
				Message:    fmt.Sprintf("field: %s, Error: %s", e.Field(), e.Tag()),
				Success:    false,
				Status:     "Failure",
				StatusCode: http.StatusBadRequest,
			}
		}
	}

	fmt.Println("transferNIU", transferNIU)

	txnJSON, err := ctx.GetState(transferNIU.TxnId)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to read from world state: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}
	if txnJSON != nil {
		return Response{
			Message:    fmt.Sprintf("transaction %v already accounted", transferNIU.TxnId),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	if transferNIU.Sender == transferNIU.Receiver {
		return Response{
			Message:    "transfer to self is not allowed",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("failed to get client id: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	fmt.Println("operator-->", operator, transferNIU.Sender)
	kycCheck, err := kaps.IsKyced(ctx, transferNIU.Sender)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("not able to do KYC check for user:%s, error:%v", transferNIU.Sender, err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}
	if !kycCheck {
		return Response{
			Message:    fmt.Sprintf("user %s is not kyced", transferNIU.Sender),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Check KYC status for each recipient
	kycCheck, err = kaps.IsKyced(ctx, transferNIU.Receiver)
	if err != nil {
		return Response{
			Message:    fmt.Sprintf("not able to do KYC check for user:%s, error:%v", transferNIU.Receiver, err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}
	if !kycCheck {
		return Response{
			Message:    fmt.Sprintf("user %s is not kyced", transferNIU.Receiver),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusBadRequest,
		}
	}

	transferNIU.Id = GINI
	transferNIU.DocType = GINI_PAYMENT_TXN

	// Withdraw the funds from the sender address
	err = kaps.RemoveBalance(ctx, transferNIU.Id, []string{transferNIU.Sender}, transferNIU.Amount)
	if err != nil {
		return Response{
			Message:    "error while reducing balance",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Deposit the fund to the recipient address
	err = kaps.AddBalance(ctx, transferNIU.Id, []string{transferNIU.Receiver}, transferNIU.Amount)
	if err != nil {
		return Response{
			Message:    "error while adding balance",
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: transferNIU.Sender, To: transferNIU.Receiver, ID: transferNIU.Id, Value: transferNIU.Amount}
	if err := kaps.EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		return Response{
			Message:    fmt.Sprintf("unable to remove funds: %v", err),
			Success:    false,
			Status:     "Failure",
			StatusCode: http.StatusInternalServerError,
		}
	}

	funcName, _ := ctx.GetFunctionAndParameters()
	response := map[string]interface{}{
		"txId":            ctx.GetTxID(),
		"txFcn":           funcName,
		"txType":          "Invoke",
		"transactionData": transferNIU,
	}

	//return kaps.EmitTransferSingle(ctx, kaps.TransferSingle{Operator: operator, From: acc.Account, To: "0x0", ID: acc.Id, Value: acc.Amount})
	return Response{
		Message:    "Funds transfered successfully",
		Success:    true,
		Status:     "Success",
		StatusCode: http.StatusCreated,
		Response:   response,
	}

}

func (s *SmartContract) GetBalanceForAccount(ctx kalpsdk.TransactionContextInterface, account string) (float64, error) {
	var owner kaps.Owner
	id := GINI
	ownerKey, err := ctx.CreateCompositeKey(OwnerPrefix, []string{account, id})
	fmt.Println("ownerKey", ownerKey)
	if err != nil {
		return 0, fmt.Errorf("failed to create composite key for account %v and token %v: %v", account, id, err)
	}

	// Retrieve the current balance for the account and token ID
	ownerBytes, err := ctx.GetState(ownerKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read balance for account %v and token %v: %v", account, id, err)
	}

	if ownerBytes != nil {
		fmt.Println("unmarshelling balance bytes")
		// Unmarshal the current balance into an Owner struct
		err = json.Unmarshal(ownerBytes, &owner)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", account, id, err)
		}
		fmt.Println("owner", owner)

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
