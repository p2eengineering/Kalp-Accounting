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

	//Custom Build Libs
	// "golang.org/x/exp/slices"

	//Third party Libs

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/p2eengineering/kalp-sdk/kalpsdk"
	"github.com/p2eengineering/kalp-sdk/kaps"
	"golang.org/x/exp/slices"
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

// const legalPrefix = "legal~tokenId"

const IPO = "IPO"
const PRESALES = "PRESALES"
const MARKET = "MARKET"

// const statusBlackListed = "BLACKLISTED"

type SmartContract struct {
	kalpsdk.Contract
}

type SmartAsset struct {
	Id       string      `json:"AssetCode"`
	DocType  string      `json:"DocType"`
	Name     string      `json:"Name"`
	Type     string      `json:"Type"`
	Desc     string      `json:"Description"`
	Status   string      `json:"CreditRating"`
	Account  []string    `json:"PrimarySponsorId"`
	Uri      string      `json:"Address"`
	MetaData interface{} `json:"Metadata"`
	Price    string      `json:"PresalePrice"`
}

type NIU struct {
	Id              string      `json:"Id"`
	AssetCode       string      `json:"AssetCode"`
	DocType         string      `json:"DocType"`
	Name            string      `json:"Name"`
	Type            string      `json:"Type"`
	Desc            string      `json:"Desc"`
	Status          string      `json:"Status"`
	Account         []string    `json:"Account"`
	MetaData        interface{} `json:"Metadata"`
	Amount          uint64      `json:"Amount,omitempty" metadata:",optional"`
	Uri             string      `json:"Uri"`
	AssetDigest     string      `json:"AssetDigest"`
	TotalTokens     uint64      `json:"TotalTokens,omitempty" metadata:",optional"`
	ReservedTokens  uint64      `json:"ReservedTokens,omitempty" metadata:",optional"`
	AvailableTokens uint64      `json:"AvailableTokens,omitempty" metadata:",optional"`
	BlockedTokens   uint64      `json:"BlockedTokens,omitempty" metadata:",optional"`
	ReservedDays    int         `json:"ReservedDays,omitempty" metadata:",optional"`
	ReserveEndDate  time.Time   `json:"ReserveEndDate,omitempty" metadata:",optional"`
}

type Account struct {
	OffchainTxnId string   `json:"OffchainTxnId"`
	Id            string   `json:"Id"`
	Account       []string `json:"Account"`
	DocType       string   `json:"DocType"`
	Amount        uint64   `json:"Amount"`
	Desc          string   `json:"Desc"`
	IsLocked      string   `json:"IsLocked"`
}

type UserRole struct {
	Id      string `json:"User"`
	Role    string `json:"Role"`
	DocType string `json:"DocType"`
	Desc    string `json:"Desc"`
}

type TransferNIU struct {
	OffchainTxnId string   `json:"OffchainTxnId"`
	Senders       []string `json:"Senders"`
	Receivers     []string `json:"Receivers"`
	Id            string   `json:"Id"`
	DocType       string   `json:"DocType"`
	Amount        uint64   `json:"Amount"`
	TimeStamp     string   `json:"TimeStamp"`
}

type BlockNIU struct {
	Id string `json:"Id"`
	// AllocationId string `json:"AllocationId"`
	DocType   string `json:"DocType"`
	Type      string `json:"Type"`
	Amount    uint64 `json:"Amount"`
	TimeStamp string `json:"TimeStamp"`
}

func (s *SmartContract) InitLedger(ctx kalpsdk.TransactionContextInterface) error {
	fmt.Println("InitLedger invoked...")
	return nil
}

//km remove setURI from MintWithToken
// Also change int64 to float64

func (s *SmartContract) AddFunds(ctx kalpsdk.TransactionContextInterface, data string) error {
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

	niuJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON != nil {
		return fmt.Errorf("transaction %v already accounted", acc.OffchainTxnId)
	}

	if acc.Amount <= 0 {
		return fmt.Errorf("Amount can't be less then 0")
	}

	if acc.DocType == "" || (acc.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("Not a valid DocType")
	}

	// // Check if token is already minted.
	// minted, err := s.IsMinted(ctx, acc.Id, acc.DocType)
	// if err != nil {
	// 	return err // form fmt.Errorf()
	// }
	// if minted {
	// 	return fmt.Errorf("Transaction Id '%v' is already accounted", acc.Id)
	// }

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	// Mint token and store the JSON representation in the state database.
	err = kaps.MintWithTokenURIMetadata(ctx, acc.Account, acc.Id, acc.Amount, acc.Desc, accJSON, acc.DocType)
	if err != nil {
		return err
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	return nil
}

func (s *SmartContract) RemoveFunds(ctx kalpsdk.TransactionContextInterface, data string) error {
	//check if contract has been intilized first

	fmt.Println("RemoveFunds---->")
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

	fmt.Println("acc---->", acc)

	niuJSON, err := ctx.GetState(acc.OffchainTxnId)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON != nil {
		return fmt.Errorf("transaction %v already accounted", acc.OffchainTxnId)
	}

	if acc.DocType == "" || (acc.DocType != kaps.DocTypeNIU) {
		return fmt.Errorf("Not a valid DocType")
	}

	// Mint token and store the JSON representation in the state database.
	err = kaps.Burn(ctx, acc.Account, acc.Id, acc.Amount, acc.DocType)
	if err != nil {
		return fmt.Errorf("error while reducing balance : %v", err)
	}

	accJSON, err := json.Marshal(acc)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	fmt.Println("MintToken Amount---->", acc.Amount)

	if err := ctx.PutStateWithKYC(acc.OffchainTxnId, accJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	return nil
}

func (s *SmartContract) TransferFunds(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {

	var transferNIU TransferNIU

	errs := json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return "", fmt.Errorf("error is parsing transfer request data: %v", errs)
	}

	fmt.Println("transferNIU", transferNIU)

	niuJSON, err := ctx.GetState(transferNIU.OffchainTxnId)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON != nil {
		return "", fmt.Errorf("transaction %v already accounted", transferNIU.OffchainTxnId)
	}

	if transferNIU.DocType == "" || (transferNIU.DocType != kaps.DocTypeNIU) {
		return "", fmt.Errorf("DocType is either null or not NIU")
	}
	if len(transferNIU.Senders) > 1 || len(transferNIU.Receivers) > 1 {
		return "", fmt.Errorf("currently one receiver and one sender is only supported for transfer operation")
	}

	// // Retrieve asset from the world state using its ID
	// niuJSON, err := ctx.GetState(transferNIU.Id)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to read from world state: %v", err)
	// }
	// if niuJSON == nil {
	// 	return "", fmt.Errorf("the Asset %v does not exist", transferNIU.Id)
	// }

	// Unmarshal the asset JSON data into a struct
	// var niu NIU
	// err = json.Unmarshal(niuJSON, &niu)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to unmarshal struct: %v", err)
	// }

	// Check KYC status for each recipient
	for i := 0; i < len(transferNIU.Receivers); i++ {
		kycCheck, err := kaps.IsKyced(ctx, transferNIU.Receivers[i])
		if err != nil {
			return "", fmt.Errorf("not able to do KYC check for user:%s, error:%v", transferNIU.Receivers[i], err)
		}
		if !kycCheck {
			return "", fmt.Errorf("user %s is not kyced", transferNIU.Receivers[i])
		}
		// if slices.Contains(niu.Account, transferNIU.Receivers[i]) {
		// 	return "", fmt.Errorf("transfer to self is not allowed")
		// }
		receiver := transferNIU.Receivers[i]
		fmt.Println("receiver", receiver)

		// user, err1 := s.GetUserRoles(ctx, receiver)
		// if err1 != nil || user.Role == BROKER {
		// 	return "", fmt.Errorf("receiver %s role not defined or receiver can not be broker ", transferNIU.Receivers[i], err1)
		// }
		// fmt.Println("user", user.Role)

		// if transferNIU.Receivers[i] == niu.Account[0] {
		// 	niu.AvailableTokens = niu.AvailableTokens + transferNIU.Amount
		// }

	}
	// for i := 0; i < len(transferNIU.Senders); i++ {
	// 	user, err1 := s.GetUserRoles(ctx, transferNIU.Senders[i])
	// 	if err1 != nil || user.Role == BROKER {
	// 		return "", fmt.Errorf("sender %s role not defined or receuver can not be broker", transferNIU.Senders[i], err1)
	// 	}

	// 	// if transferNIU.Senders[i] == niu.Account[0] {
	// 	// 	fmt.Println("------------transferNIU.Senders----------", transferNIU.Senders[i])
	// 	// 	if niu.AvailableTokens < transferNIU.Amount {
	// 	// 		return "", fmt.Errorf("tokens are not available for transfer.")
	// 	// 	} else {
	// 	// 		niu.AvailableTokens = niu.AvailableTokens - transferNIU.Amount
	// 	// 	}
	// 	// }
	// }

	// Update the asset owner to the new recipient
	var OrgSenders = transferNIU.Senders
	// niu.Account = transferNIU.Receivers

	// Marshal the updated asset struct into JSON data and verify the asset hash
	// newniuJSON, err := json.Marshal(niu)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to marshal struct: %v", err)
	// }
	fmt.Println("calling TransferFrom")

	// Transfer tokens using the KAPS contract functionality
	err = kaps.TransferFrom(ctx, transferNIU.Senders, transferNIU.Receivers, transferNIU.Id, uint64(transferNIU.Amount), transferNIU.DocType, OrgSenders)
	if err != nil {
		return "", fmt.Errorf("failed to transfer tokens: %v", err)
	}

	transferNIUJSON, err := json.Marshal(transferNIU)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %v", err)
	}

	// Save the updated asset state in the world state
	if err := ctx.PutStateWithKYC(transferNIU.OffchainTxnId, transferNIUJSON); err != nil {
		return "", fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	// Emit an event
	if err := ctx.SetEvent("TransferNIU", transferNIUJSON); err != nil {
		return "", fmt.Errorf("unable to setEvent TransferNIU: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)
}

// // BurnTokens is a smart contract function which will delete the R2CI asset and its owner details from blockchain world state.
// func (s *SmartContract) RemoveFunds1(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {
// 	// Retrieve asset from the world state using its ID
// 	niuJSON, err := ctx.GetStub().GetState(id)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read from world state: %v", err)
// 	}
// 	if niuJSON == nil {
// 		return "", fmt.Errorf("the Asset %v does not exist", id)
// 	}

// 	// Unmarshal the asset JSON data into a struct
// 	var niu NIU
// 	if err := json.Unmarshal(niuJSON, &niu); err != nil {
// 		return "", fmt.Errorf("failed to unmarshal struct: %v", err)
// 	}

// 	// Get the operator's client ID
// 	operator, err := kaps.GetUserId(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to get client id: %v", err)
// 	}

// 	// Check if operator is a valid owner of the asset
// 	if !slices.Contains(niu.Account, operator) {
// 		return "", fmt.Errorf("Asset owner can only initiate Burn")
// 	}

// 	// Burn the tokens using kaps contract
// 	if niu.DocType == kaps.DocTypeNIU {
// 		fmt.Println(niu.DocType)
// 		// Burn token (if you want to burn amount with 0x0)
// 		err = kaps.Burn(ctx, niu.Account, id, 0, niu.DocType)
// 		if err != nil {
// 		}
// 			return "", err
// 	} else if niu.DocType == kaps.DocTypeAsset {
// 		err = kaps.Burn(ctx, niu.Account, id, 0, niu.DocType)
// 		if err != nil {
// 			return "", err
// 		}
// 	} else {
// 		return "", fmt.Errorf("unknown docType: %s", niu.DocType)
// 	}

// 	// Delete metadata and URI  JSON representation in the state database.
// 	err = kaps.DeleteWithTokenURIMetadata(ctx, niu.Id)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Delete the asset from the world state
// 	if err := ctx.GetStub().DelState(id); err != nil {
// 		return "", fmt.Errorf("unable to delete Asset struct in statedb: %v", err)
// 	}

// 	// Emit an event
// 	if err := ctx.GetStub().SetEvent("DeleteNIU", niuJSON); err != nil {
// 		return "", fmt.Errorf("unable to setEvent DeleteNIU: %v", err)
// 	}
// 	return GetTransactionTimestamp(ctx)
// }

// Set information for a token and intialize contract.
// param {String} name The name of the token
// param {String} symbol The symbol of the token
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

// SetUserRoles is a smart contract function which is used to setup a role for user.
func (s *SmartContract) SetUserRoles(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {
	//check if contract has been intilized first

	fmt.Println("SetUserRoles", data)
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return "", fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Parse input data into NIU struct.
	var userRole UserRole
	errs := json.Unmarshal([]byte(data), &userRole)
	if errs != nil {
		return "", fmt.Errorf("failed to parse data: %v", errs)
	}

	err = kaps.IsAdmin(ctx)
	if userValid, _ := s.ValidateUserRole(ctx, SMART_ADMIN); !userValid && err != nil {
		return "", fmt.Errorf("user roles can be defined by %v only", SMART_ADMIN)
	}

	// Validate input data.
	if userRole.Id == "" {
		return "", fmt.Errorf("user Id can not be null")
	}
	if userRole.Role == "" {
		return "", fmt.Errorf("role can not be null")
	}

	// Generate JSON representation of NIU struct.
	usrRoleJSON, err := json.Marshal(userRole)
	if err != nil {
		return "", fmt.Errorf("unable to Marshal userRole struct : %v", err)
	}
	// Store the NIU struct in the state database
	if err := ctx.PutStateWithKYC(userRole.Id, usrRoleJSON); err != nil {
		return "", fmt.Errorf("unable to put user role struct in statedb: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)
}

// GetUserRoles is a smart contract function which is used to get a role of a user.
func (s *SmartContract) GetUserRoles(ctx kalpsdk.TransactionContextInterface, id string) (*UserRole, error) {
	// Get the asset from the ledger using id & check if asset exists

	userJSON, err := ctx.GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if userJSON == nil {
		return nil, fmt.Errorf("the user role %v does not exist", id)
	}

	// Unmarshal asset from JSON to struct
	var userRole UserRole
	err = json.Unmarshal(userJSON, &userRole)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal user role struct : %v", err)
	}

	return &userRole, nil
}

// CreateSmartAsset is a smart contract function which takes the Asset input as JSON and store it in blockchain.
// CreateSmartAsset also inherits the KAPS contract functionality and validates onwer KYC.  Also it stores the hash of the Intellectual Property in blockchain.
func (s *SmartContract) CreateSmartAsset(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {
	//check if contract has been intilized first

	fmt.Println("CreateSmartAsset", data)
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return "", fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Parse input data into NIU struct.
	var sAsset SmartAsset
	errs := json.Unmarshal([]byte(data), &sAsset)
	if errs != nil {
		return "", fmt.Errorf("failed to parse data: %v", errs)
	}

	// Check if operator is authorized to create NIU.
	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get client id: %v", err)
	}
	if !slices.Contains(sAsset.Account, operator) {
		return "", fmt.Errorf("asset owner can only initiate create transaction")
	}

	if userValid, _ := s.ValidateUserRole(ctx, SPONSOR); !userValid {
		return "", fmt.Errorf("create smart asset can be performed by %v only", SPONSOR)
	}

	// Validate input data.
	if sAsset.Name == "" {
		return "", fmt.Errorf("asset name can not be null")
	}
	if sAsset.DocType == "" || sAsset.DocType != kaps.DocTypeSmartAsset {
		return "", fmt.Errorf("not a valid DocType")
	}
	if sAsset.Status != IPO && sAsset.Status != PRESALES && sAsset.Status != MARKET {
		return "", fmt.Errorf("not a valid status")
	}

	// Check if token is already minted.
	minted, err := s.IsMinted(ctx, sAsset.Id, sAsset.DocType)
	if err != nil {
		return "", err
	}
	if minted {
		return "", fmt.Errorf("the Token with Id '%v' is already minted ", sAsset.Id)
	}

	// Generate JSON representation of NIU struct.
	niuJSON, err := json.Marshal(sAsset)
	if err != nil {
		return "", fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	// Mint token and store the JSON representation in the state database.
	err = kaps.MintWithTokenURIMetadata(ctx, sAsset.Account, sAsset.Id, 0, sAsset.Uri, niuJSON, sAsset.DocType)
	if err != nil {
		return "", err
	}

	// Store the NIU struct in the state database
	if err := ctx.PutStateWithKYC(sAsset.Id, niuJSON); err != nil {
		return "", fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	// Emit an event
	if err := ctx.SetEvent("CreateSmartAsset", niuJSON); err != nil {
		return "", fmt.Errorf("unable to setEvent CreateSmartAsset: %v", err)
	}

	return s.GetTransactionTimestamp(ctx)
}

// Mint token create number of token which are request in totaltoken attribute.
// All tokens will be minted and assigned to SPONSOR by TRUSTEE
func (s *SmartContract) MintToken(ctx kalpsdk.TransactionContextInterface, data string) error {
	//check if contract has been intilized first

	fmt.Println("MintToken---->")
	initialized, err := kaps.CheckInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	fmt.Println("MintToken CheckInitialized---->")

	// Parse input data into NIU struct.
	var niu NIU
	errs := json.Unmarshal([]byte(data), &niu)
	if errs != nil {
		return fmt.Errorf("failed to parse data: %v", errs)
	}

	if niu.TotalTokens <= 0 {
		return fmt.Errorf("TotalTokes can't be less then 0")
	}

	// Check if operator is authorized to create NIU.
	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("operator---------------", operator)

	user, err1 := s.GetUserRoles(ctx, operator)
	if err1 != nil {
		return fmt.Errorf("failed to get user role: %v", err1)
	}

	if user.Role != TRUSTEE {
		return fmt.Errorf("token minting is allowed to %v only", TRUSTEE)
	}

	if !slices.Contains(niu.Account, operator) {
		return fmt.Errorf("asset owner should initiate mint transaction")
	}

	if niu.DocType == "" || (niu.DocType != kaps.DocTypeNIU && niu.DocType != kaps.DocTypeAsset) {
		return fmt.Errorf("Not a valid DocType")
	}

	// Check if token is already minted.
	minted, err := s.IsMinted(ctx, niu.Id, niu.DocType)
	if err != nil {
		return err // form fmt.Errorf()
	}
	if minted {
		return fmt.Errorf("either Token '%v' or Smart Asset '%v' is already minted", niu.Id, niu.AssetCode)
	}

	sAssetJSON, err := ctx.GetState(niu.AssetCode)
	if err != nil {
		return fmt.Errorf("failed to read asset from world state: %v", err)
	}
	if sAssetJSON == nil {
		return fmt.Errorf("the Asset %v does not exist during minting", niu.AssetCode)
	}

	// Unmarshal the asset JSON data into a struct
	var sAsset SmartAsset
	err = json.Unmarshal(sAssetJSON, &sAsset)
	if err != nil {
		return fmt.Errorf("failed to unmarshal asset struct: %v", err)
	}

	// assignig token to SPONSOR of the asset
	niu.Account = sAsset.Account
	fmt.Println("niu.Account", niu.Account)
	niu.ReserveEndDate = time.Now().AddDate(0, 0, niu.ReservedDays)
	fmt.Println("niu.ReserveEndDate", niu.ReserveEndDate)
	niu.Amount = niu.TotalTokens

	if niu.ReservedTokens <= 0 {
		fmt.Println("inside niu.ReservedTokens <=0", niu.ReservedTokens)
		niu.ReservedTokens = (niu.TotalTokens * 26 / 100)
	}

	niuJSON, err := json.Marshal(niu)
	if err != nil {
		return fmt.Errorf("unable to Marshal Token struct : %v", err)
	}

	// Mint token and store the JSON representation in the state database.
	err = kaps.MintWithTokenURIMetadata(ctx, niu.Account, niu.Id, niu.Amount, niu.Uri, niuJSON, niu.DocType)
	if err != nil {
		return err
	}

	fmt.Println("MintToken Amount---->", niu.Amount)

	if err := ctx.PutStateWithKYC(niu.Id, niuJSON); err != nil {
		return fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	return nil
}

// BlockNIU is function which allows token to be blocked for presales, IPO or market.
// Later these tokens can be transferred to investors
func (s *SmartContract) BlockNIU(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {

	var blockNIU BlockNIU

	errs := json.Unmarshal([]byte(data), &blockNIU)
	if errs != nil {
		return "", fmt.Errorf("error is parsing transfer request data: %v", errs)
	}

	if blockNIU.DocType == "" || (blockNIU.DocType != kaps.DocTypeNIU) {
		return "", fmt.Errorf("DocType is either null or not NIU")
	}

	// Retrieve asset from the world state using its ID
	niuJSON, err := ctx.GetState(blockNIU.Id)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON == nil {
		return "", fmt.Errorf("the NIU %v does not exist", blockNIU.Id)
	}

	// Unmarshal the asset JSON data into a struct
	var niu NIU
	err = json.Unmarshal(niuJSON, &niu)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	fmt.Println("------------totalBlockecTokens----------", niu.BlockedTokens+blockNIU.Amount)
	fmt.Println("------------totalAvailable to Block----------", niu.TotalTokens-niu.ReservedTokens)

	if niu.BlockedTokens+blockNIU.Amount > niu.TotalTokens-niu.ReservedTokens {
		today := time.Now()

		if niu.ReservedTokens > 0 {
			if today.Before(niu.ReserveEndDate) {
				return "", fmt.Errorf("Not enough tokens are available to block.")
			} else if niu.BlockedTokens+blockNIU.Amount > niu.TotalTokens {
				fmt.Println("------------Number are requested tokens are more than total available tokes----------")
				return "", fmt.Errorf("Number are requested tokens are more than total available tokes.")
			} else {
				niu.AvailableTokens = niu.AvailableTokens + blockNIU.Amount
				niu.BlockedTokens = niu.BlockedTokens + blockNIU.Amount
				niu.ReservedTokens = niu.TotalTokens - niu.BlockedTokens
			}
		} else {
			return "", fmt.Errorf("All tokens are already blocked.")
		}
	} else {
		fmt.Println("------------Allocating avaiable tokens----------")
		niu.AvailableTokens = niu.AvailableTokens + blockNIU.Amount
		niu.BlockedTokens = niu.BlockedTokens + blockNIU.Amount

	}

	// Update the asset owner to the new recipient
	// var OrgSenders = niu.Account
	fmt.Println("------------niu----------", niu)
	// Marshal the updated asset struct into JSON data and verify the asset hash
	newniuJSON, err := json.Marshal(niu)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %v", err)
	}
	fmt.Println("niu.Status-", niu.Status)

	// Save the updated asset state in the world state
	if err := ctx.PutStateWithKYC(blockNIU.Id, newniuJSON); err != nil {
		return "", fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	// Emit an event
	if err := ctx.SetEvent("BlockNIU", newniuJSON); err != nil {
		return "", fmt.Errorf("unable to setEvent BlockNIU: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)
}

// TransferNIU is a smart contract function which transfers tokens from current ownere to new owner.
// TransferNIU requires the Token id, current owner and new owner details to perform the transfer operation in blockchain.
// TransferNIU also inherits the KAPS contract functionality and validates current onwer and new owner's KYC before performing transfer.
func (s *SmartContract) TransferNIU(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {

	var transferNIU TransferNIU

	errs := json.Unmarshal([]byte(data), &transferNIU)
	if errs != nil {
		return "", fmt.Errorf("error is parsing transfer request data: %v", errs)
	}

	if transferNIU.DocType == "" || (transferNIU.DocType != kaps.DocTypeNIU) {
		return "", fmt.Errorf("DocType is either null or not NIU")
	}
	if len(transferNIU.Senders) > 1 || len(transferNIU.Receivers) > 1 {
		return "", fmt.Errorf("currently one receiver and one sender is only supported for transfer operation")
	}

	// Retrieve asset from the world state using its ID
	niuJSON, err := ctx.GetState(transferNIU.Id)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON == nil {
		return "", fmt.Errorf("the Asset %v does not exist", transferNIU.Id)
	}

	// Unmarshal the asset JSON data into a struct
	var niu NIU
	err = json.Unmarshal(niuJSON, &niu)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	// Check KYC status for each recipient
	for i := 0; i < len(transferNIU.Receivers); i++ {
		kycCheck, err := kaps.IsKyced(ctx, transferNIU.Receivers[i])
		if err != nil {
			return "", fmt.Errorf("not able to do KYC check for user:%s, error:%v", transferNIU.Receivers[i], err)
		}
		if !kycCheck {
			return "", fmt.Errorf("user %s is not kyced", transferNIU.Receivers[i])
		}
		// if slices.Contains(niu.Account, transferNIU.Receivers[i]) {
		// 	return "", fmt.Errorf("transfer to self is not allowed")
		// }
		receiver := transferNIU.Receivers[i]
		fmt.Println("receiver", receiver)

		user, err1 := s.GetUserRoles(ctx, receiver)
		if err1 != nil || user.Role == BROKER {
			return "", fmt.Errorf("receiver %s role not defined or receiver can not be broker ", transferNIU.Receivers[i], err1)
		}
		fmt.Println("user", user.Role)

		// if transferNIU.Receivers[i] == niu.Account[0] {
		// 	niu.AvailableTokens = niu.AvailableTokens + transferNIU.Amount
		// }

	}
	for i := 0; i < len(transferNIU.Senders); i++ {
		user, err1 := s.GetUserRoles(ctx, transferNIU.Senders[i])
		if err1 != nil || user.Role == BROKER {
			return "", fmt.Errorf("sender %s role not defined or receuver can not be broker", transferNIU.Senders[i], err1)
		}

		if transferNIU.Senders[i] == niu.Account[0] {
			fmt.Println("------------transferNIU.Senders----------", transferNIU.Senders[i])
			if niu.AvailableTokens < transferNIU.Amount {
				return "", fmt.Errorf("tokens are not available for transfer.")
			} else {
				niu.AvailableTokens = niu.AvailableTokens - transferNIU.Amount
			}
		}
	}

	// Update the asset owner to the new recipient
	var OrgSenders = transferNIU.Senders
	// niu.Account = transferNIU.Receivers

	// Marshal the updated asset struct into JSON data and verify the asset hash
	newniuJSON, err := json.Marshal(niu)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %v", err)
	}

	// Transfer tokens using the KAPS contract functionality
	err = kaps.TransferFrom(ctx, transferNIU.Senders, transferNIU.Receivers, transferNIU.Id, uint64(transferNIU.Amount), transferNIU.DocType, OrgSenders)
	if err != nil {
		return "", fmt.Errorf("failed to transfer tokens: %v", err)
	}

	transferNIUJSON, err := json.Marshal(transferNIU)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %v", err)
	}

	// Save the updated asset state in the world state
	if err := ctx.PutStateWithKYC(transferNIU.Id, newniuJSON); err != nil {
		return "", fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	// Emit an event
	if err := ctx.SetEvent("TransferNIU", transferNIUJSON); err != nil {
		return "", fmt.Errorf("unable to setEvent TransferNIU: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)
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

// // DeleteNIU is a smart contract function which will delete the R2CI Asset / Token  without deleting owner details from blockchain world state.
func (s *SmartContract) DeleteNIU(ctx kalpsdk.TransactionContextInterface, id string) error {
	// check if invoker is admin
	// if err := s.IsClient(ctx); err != nil {
	// 	return err
	// }

	fmt.Println("Inside DeleteNIU!")

	// Get the JSON data of the asset from the world state and check if exisits
	niuJSON, err := ctx.GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if niuJSON == nil {
		return fmt.Errorf("the Asset %v does not exist", id)
	}

	// Unmarshal the JSON data to an NIU struct
	var niu NIU
	if err := json.Unmarshal(niuJSON, &niu); err != nil {
		return fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	// Delete the asset from the world state
	if err := ctx.DelStateWithKYC(id); err != nil {
		return fmt.Errorf("unable to delete Asset struct in statedb: %v", err)
	}

	// Emits an event for the deleted asset
	if err := ctx.SetEvent("DeleteNIU", niuJSON); err != nil {
		return fmt.Errorf("unable to setEvent DeleteNIU: %v", err)
	}

	return nil
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

// // UpdateStatus is a smart contract function that only the admin user can use to update the status of an asset on the blockchain.
func (s *SmartContract) UpdateStatus(ctx kalpsdk.TransactionContextInterface, id string, status string) (string, error) {
	// check if invoker is admin
	// if err := kaps.IsAdmin(ctx); err != nil {
	// 	return "", err
	// }

	if userValid, _ := s.ValidateUserRole(ctx, SPONSOR); !userValid {
		return "", fmt.Errorf("update smart asset can be performed by %v only", SPONSOR)
	}

	sAssetJSON, err := ctx.GetState(id)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if sAssetJSON == nil {
		return "", fmt.Errorf("the Asset %v does not exist", id)
	}

	var sAsset SmartAsset
	err = json.Unmarshal(sAssetJSON, &sAsset)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	if sAsset.Status != IPO && sAsset.Status != PRESALES && sAsset.Status != MARKET {
		return "", fmt.Errorf("not a valid status")
	}

	sAsset.Status = status
	newAssetJSON, err := json.Marshal(sAsset)
	if err != nil {
		return "", fmt.Errorf("unable to parse asset data : %v", err)
	}

	if err := ctx.PutStateWithKYC(id, newAssetJSON); err != nil {
		return "", fmt.Errorf("unable to put Asset struct in statedb: %v", err)
	}

	if err := ctx.SetEvent("UpdateStatus", sAssetJSON); err != nil {
		return "", fmt.Errorf("unable to setEvent UpdateStatus: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)
}

// GetTransactionTimestamp retrieves the transaction timestamp from the context and returns it as a string.
func (s *SmartContract) GetTransactionTimestamp(ctx kalpsdk.TransactionContextInterface) (string, error) {
	timestamp, err := ctx.GetTxTimestamp()
	if err != nil {
		return "", err
	}

	return timestamp.AsTime().String(), nil
}

func (s *SmartContract) ValidateUserRole(ctx kalpsdk.TransactionContextInterface, Role string) (bool, error) {

	// Check if operator is authorized to create NIU.
	operator, err := kaps.GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get client id: %v", err)
	}
	// if !slices.Contains(sAsset.Account, operator) {
	// 	return "", fmt.Errorf("asset owner can only initiate create transaction")
	// }

	fmt.Println("operator---------------", operator)
	user, err1 := s.GetUserRoles(ctx, operator)
	if err1 != nil {
		return false, fmt.Errorf("failed to get user role: %v", err1)
	}

	if user.Role != Role {
		return false, fmt.Errorf("this transaction can be performed by %v only", Role)
	}

	return true, nil
}
