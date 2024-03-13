// R2CI is an acronym for Right to Create Intellectual Property. It is a platform for maintaining the IP and showcasing those IP rights to others.
//
// This package provides the functions to create and maintain R2CI Assets and Token in the blokchain.
package kalpAccounting

import (
	//Standard Libs

	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/p2eengineering/kalp-sdk/kalpsdk"
	"github.com/p2eengineering/kalp-sdk/kaps"
)

const attrRole = "hf.Type"

// const admintype = "client"
const nameKey = "name"
const symbolKey = "symbol"
const SMART_ADMIN = "SMART_ADMIN"
const SPONSOR = "SPONSOR"
const BROKER = "BROKER"
const INVESTOR = "INVESTOR"
const TRUSTEE = "TRUSTEE"
const OwnerPrefix = "ownerId~assetId"

// const legalPrefix = "legal~tokenId"

const IPO = "IPO"
const PRESALES = "PRESALES"
const MARKET = "MARKET"

// const statusBlackListed = "BLACKLISTED"

type SmartContract struct {
	kalpsdk.Contract
}

type NIU struct {
	Id       string      `json:"Id"`
	DocType  string      `json:"DocType"`
	Name     string      `json:"Name"`
	Type     string      `json:"Type"`
	Desc     string      `json:"Desc"`
	Status   string      `json:"Status"`
	Account  string      `json:"Account"`
	MetaData interface{} `json:"Metadata"`
	Amount   uint64      `json:"Amount"`
}

type Account struct {
	OffchainTxnId string `json:"OffchainTxnId"`
	Id            string `json:"Id"`
	Account       string `json:"Account"`
	DocType       string `json:"DocType"`
	Amount        uint64 `json:"Amount"`
	Desc          string `json:"Desc"`
	IsLocked      string `json:"IsLocked"`
}

type TransferNIU struct {
	TxnId     string `json:"TxnId"`
	Sender    string `json:"Sender"`
	Receiver  string `json:"Receiver"`
	Id        string `json:"Id"`
	DocType   string `json:"DocType"`
	Amount    uint64 `json:"Amount"`
	TimeStamp string `json:"TimeStamp"`
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

func (s *SmartContract) DefineToken(ctx kalpsdk.TransactionContextInterface, data string) error {
	//check if contract has been intilized first
	fmt.Println("AddFunds---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	fmt.Println("AddFunds CheckInitialized---->")

	// Parse input data into NIU struct.
	var niuData NIU
	errs := json.Unmarshal([]byte(data), &niuData)
	if errs != nil {
		return fmt.Errorf("failed to parse data: %v", errs)
	}

	niu, err := ctx.GetState(niuData.Id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if niu != nil {
		return fmt.Errorf("token %v already defined", niuData.Id)
	}

	if niuData.Amount <= 0 {
		return fmt.Errorf("amount can't be less then 0")
	}

	if niuData.DocType == "" || (niuData.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("not a valid DocType")
	}

	niuJSON, err := json.Marshal(niu)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("MintToken operator---->", operator)

	// Mint tokens
	err = kaps.MintHelper(ctx, operator, []string{niuData.Account}, niuData.Id, niuData.Amount, niuData.DocType)
	if err != nil {
		return err
	}

	fmt.Println("MintToken Amount---->", niuData.Amount)

	if err := ctx.PutStateWithKYC(niuData.Id, niuJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: niuData.Account, ID: niuData.Id, Value: niuData.Amount}
	return kaps.EmitTransferSingle(ctx, transferSingleEvent)

}

func (s *SmartContract) Mint(ctx kalpsdk.TransactionContextInterface, data string) error {
	//check if contract has been intilized first
	fmt.Println("AddFunds---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	fmt.Println("AddFunds CheckInitialized---->")

	// Parse input data into NIU struct.
	var acc Account
	errs := json.Unmarshal([]byte(data), &acc)
	if errs != nil {
		return fmt.Errorf("failed to parse data: %v", errs)
	}

	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if txnJSON != nil {
		return fmt.Errorf("transaction %v already accounted", acc.OffchainTxnId)
	}

	if acc.Amount <= 0 {
		return fmt.Errorf("amount can't be less then 0")
	}

	if acc.DocType == "" || (acc.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("not a valid DocType")
	}

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("MintToken operator---->", operator)

	// Mint tokens
	err = kaps.MintHelper(ctx, operator, []string{acc.Account}, acc.Id, acc.Amount, acc.DocType)
	if err != nil {
		return err
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: "0x0", To: acc.Account, ID: acc.Id, Value: acc.Amount}
	return kaps.EmitTransferSingle(ctx, transferSingleEvent)

}

func (s *SmartContract) Burn(ctx kalpsdk.TransactionContextInterface, data string) error {
	//check if contract has been intilized first

	fmt.Println("RemoveFunds---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Parse input data into NIU struct.
	var acc Account
	errs := json.Unmarshal([]byte(data), &acc)
	if errs != nil {
		return fmt.Errorf("failed to parse data: %v", errs)
	}

	fmt.Println("acc---->", acc)

	txnJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if txnJSON != nil {
		return fmt.Errorf("transaction %v already accounted", acc.OffchainTxnId)
	}

	if acc.DocType == "" || (acc.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("not a valid DocType")
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	err = kaps.RemoveBalance(ctx, acc.Id, []string{acc.Account}, acc.Amount)
	if err != nil {
		return fmt.Errorf("RemoveBalance in burn has error: %v", err)
	}

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	return kaps.EmitTransferSingle(ctx, kaps.TransferSingle{Operator: operator, From: acc.Account, To: "0x0", ID: acc.Id, Value: acc.Amount})
}

func (s *SmartContract) TransferToken(ctx kalpsdk.TransactionContextInterface, data string) error {

	var transferNIU TransferNIU

	errs := json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return fmt.Errorf("error is parsing transfer request data: %v", errs)
	}

	fmt.Println("transferNIU", transferNIU)

	txnJSON, err := ctx.GetState(transferNIU.TxnId)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if txnJSON != nil {
		return fmt.Errorf("transaction %v already accounted", transferNIU.TxnId)
	}

	if transferNIU.DocType == "" || (transferNIU.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("DocType is either null or not NIU")
	}
	if transferNIU.Sender == transferNIU.Receiver {
		return fmt.Errorf("transfer to self is not allowed")
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("operator-->", operator, transferNIU.Sender)

	// Check whether operator is owner or approved
	// if operator != transferNIU.Sender {
	// 	approved, err := kaps.IsApprovedForAll(ctx, transferNIU.Sender, operator)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if !approved {
	// 		return fmt.Errorf("caller is not owner nor is approved")
	// 	}
	// }

	kycCheck, err := kaps.IsKyced(ctx, transferNIU.Sender)
	if err != nil {
		return fmt.Errorf("not able to do KYC check for user:%s, error:%v", transferNIU.Sender, err)
	}
	if !kycCheck {
		return fmt.Errorf("user %s is not kyced", transferNIU.Sender)
	}

	// Check KYC status for each recipient
	kycCheck, err = kaps.IsKyced(ctx, transferNIU.Receiver)
	if err != nil {
		return fmt.Errorf("not able to do KYC check for user:%s, error:%v", transferNIU.Receiver, err)
	}
	if !kycCheck {
		return fmt.Errorf("user %s is not kyced", transferNIU.Receiver)
	}

	// Withdraw the funds from the sender address
	err = kaps.RemoveBalance(ctx, transferNIU.Id, []string{transferNIU.Sender}, transferNIU.Amount)
	if err != nil {
		return fmt.Errorf("error while reducing balance")
	}

	// Deposit the fund to the recipient address
	err = kaps.AddBalance(ctx, transferNIU.Id, []string{transferNIU.Receiver}, transferNIU.Amount)
	if err != nil {
		return fmt.Errorf("error while adding balance")
	}

	transferSingleEvent := kaps.TransferSingle{Operator: operator, From: transferNIU.Sender, To: transferNIU.Receiver, ID: transferNIU.Id, Value: transferNIU.Amount}
	return kaps.EmitTransferSingle(ctx, transferSingleEvent)

}

func (s *SmartContract) GetBalanceForAccount(ctx kalpsdk.TransactionContextInterface, id string, account string) (uint64, error) {
	var owner kaps.Owner
	ownerKey, err := ctx.CreateCompositeKey(OwnerPrefix, []string{account, id})

	if err != nil {
		return 0, fmt.Errorf("failed to create composite key for account %v and token %v: %v", account, id, err)
	}

	// Retrieve the current balance for the account and token ID
	ownerBytes, err := ctx.GetState(ownerKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read balance for account %v and token %v: %v", account, id, err)
	}

	if ownerBytes != nil {
		// Unmarshal the current balance into an Owner struct
		err = json.Unmarshal(ownerBytes, &owner)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal balance for account %v and token %v: %v", account, id, err)
		}

		return owner.Amount, nil
	}

	return 0, nil
}

func (s *SmartContract) GetBalance(ctx kalpsdk.TransactionContextInterface, id string, account string) (uint64, error) {

	queryString := `{"selector": {"id":"` + id + `", "account":"` + account + `", "docType":"Owner"}, "fields": ["amount"] }`
	fmt.Println("queryString-", queryString)
	resultsIterator, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	if resultsIterator.HasNext() {
		fmt.Println("Inside iterator")
		result, err := resultsIterator.Next()
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve query result: %v", err)
		}
		fmt.Println("no err in balance")

		type AmountStruct struct{ Amount uint64 }
		var amountJSON AmountStruct
		err = json.Unmarshal(result.Value, &amountJSON)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal token: %v", err)
		}
		fmt.Println("balance", amountJSON.Amount)

		return amountJSON.Amount, nil
	}

	return 0, nil
}

// // Internal chaincode function to check if the Asset already exists or Token is already minted.
func (s *SmartContract) IsMinted(ctx kalpsdk.TransactionContextInterface, id string, docType string) (bool, error) {
	queryString := `{"selector": {"_id":"` + id + `"}}`

	resultsIterator, err := ctx.GetQueryResult(queryString)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	if resultsIterator.HasNext() {
		return true, nil
	}

	return false, nil
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
