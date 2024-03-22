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
const SPONSOR = "SPONSOR"
const BROKER = "BROKER"
const INVESTOR = "INVESTOR"
const TRUSTEE = "TRUSTEE"
const OwnerPrefix = "ownerId~assetId"
const MailabRoleAttrName = "MailabUserRole"
const GatewayRoleValue = "GatewayAdmin"
const PaymentRoleValue = "PaymentAdmin"
const GINI = "GINI"

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
	Amount   float64     `json:"Amount"`
}

type Account struct {
	OffchainTxnId string  `json:"OffchainTxnId"`
	Id            string  `json:"Id"`
	Account       string  `json:"Account"`
	DocType       string  `json:"DocType"`
	Amount        float64 `json:"Amount"`
	Desc          string  `json:"Desc"`
	IsLocked      string  `json:"IsLocked"`
}

type TransferNIU struct {
	TxnId     string  `json:"TxnId"`
	Sender    string  `json:"Sender"`
	Receiver  string  `json:"Receiver"`
	Id        string  `json:"Id"`
	DocType   string  `json:"DocType"`
	Amount    float64 `json:"Amount"`
	TimeStamp string  `json:"TimeStamp"`
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

	niuData.DocType = kaps.DocTypeNIU

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
	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
	if err != nil {
		return fmt.Errorf("payment admin role check failed: %v", err)
	}

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

	acc.Id = GINI
	acc.DocType = kaps.DocTypeNIU

	fmt.Println("GINI amount", acc.Amount)
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

	err = kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, GatewayRoleValue)
	if err != nil {
		return fmt.Errorf("gateway admin role check failed: %v", err)
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

	acc.Id = GINI
	acc.DocType = kaps.DocTypeNIU

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

	errs := kaps.InvokerAssertAttributeValue(ctx, MailabRoleAttrName, GatewayRoleValue)
	if errs != nil {
		return fmt.Errorf("gateway admin role check failed: %v", errs)
	}

	errs = json.Unmarshal([]byte(data), &transferNIU)
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

	if transferNIU.Sender == transferNIU.Receiver {
		return fmt.Errorf("transfer to self is not allowed")
	}

	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("operator-->", operator, transferNIU.Sender)
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

	transferNIU.Id = GINI
	transferNIU.DocType = kaps.DocTypeNIU

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
