// R2CI is an acronym for Right to Create Intellectual Property. It is a platform for maintaining the IP and showcasing those IP rights to others.
//
// This package provides the functions to create and maintain R2CI Assets and Token in the blokchain.
package kalpAccounting

import (
	//Standard Libs

	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
	"golang.org/x/exp/slices"
)

// Deployment notes for GINI contract:
// Initialize with name and symbol as GINI, GINI
// Create Admin user (admin privilege role will be given during NGL register/enroll)
// Create 3 users for GINI-ADMIN, GasFeeAdmin, GatewayAdmin (using Kalp wallet)
// Admin user to invoke setuserrole with enrollment id of user and GINI-ADMIN role,   (only blockchain Admin can set GINI-ADMIN)
// Admin user to invoke setuserrole with enrollment id of user and GasFeeAdmin role   (only GINI-ADMIN can set Gasfee)
// Admin user to invoke setuserrole with enrollment id of user and GatewayAdmin role  (only GINI-ADMIN can set Gasfee)
const attrRole = "hf.Type"
const intialgasfeesadmin = ""
const intialkalpGateWayadmin = ""
const nameKey = "name"
const symbolKey = "symbol"
const gasFeesKey = "gasFees"
const kalpFoundation = "fb9185edc0e4bdf6ce9b46093dc3fcf4eea61c40"
const GINI = "GINI"
const env = "dev"
const totalSupply = "2000000000000000000000000000"
const giniAdmin = "KalpFoundation"
const gasFeesAdminRole = "GasFeesAdmin"
const kalpGateWayAdmin = "KalpGatewayAdmin"
const userRolePrefix = "ID~UserRoleMap"
const UserRoleMap = "UserRoleMap"
const BridgeContractAddress = "klp-6b616c70627269646765-cc"

// const legalPrefix = "legal~tokenId"
type SmartContract struct {
	kalpsdk.Contract
}

type Response struct {
	Status     string      `json:"status"`
	StatusCode uint        `json:"statusCode"`
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Response   interface{} `json:"response" `
}
type UserRole struct {
	Id      string `json:"User"`
	Role    string `json:"Role"`
	DocType string `json:"DocType"`
	Desc    string `json:"Desc"`
}

type Sender struct {
	Sender string `json:"sender"`
}

func (s *SmartContract) InitLedger(ctx kalpsdk.TransactionContextInterface) error {
	logger := kalpsdk.NewLogger()
	logger.Infof("InitLedger invoked...")
	return nil
}

// Initializing smart contract
func (s *SmartContract) Initialize(ctx kalpsdk.TransactionContextInterface, name string, symbol string) (bool, error) {
	//check contract options are not already set, client is not authorized to change them once intitialized

	operator, err := GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	role := UserRole{
		Id:      kalpFoundation,
		Role:    giniAdmin,
		DocType: "UserRoleMap",
	}
	roleJson, err := json.Marshal(role)
	if err != nil {
		fmt.Println("Error marshaling struct:", err)
		return false, fmt.Errorf("error marsheling user role")
	}
	key, err := ctx.CreateCompositeKey(userRolePrefix, []string{role.Id, UserRoleMap})
	if err != nil {
		return false, fmt.Errorf("failed to create the composite key for prefix %s: %v", userRolePrefix, err)
	}
	if err := ctx.PutStateWithoutKYC(key, roleJson); err != nil {
		return false, fmt.Errorf("unable to put user role struct in statedb: %v", err)
	}
	userRole, err := s.GetUserRoles(ctx, operator)
	if err != nil {
		return false, fmt.Errorf("error with status code %v,error checking sponsor's role: %v", http.StatusBadRequest, err)
	}
	if userRole != giniAdmin {
		return false, fmt.Errorf("error with status code %v, error:only gini admin is allowed to mint", http.StatusInternalServerError)
	}

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

	gasFeesAdminRole := UserRole{
		Id:      intialgasfeesadmin,
		Role:    gasFeesAdminRole,
		DocType: "UserRoleMap",
	}
	gasFeesAdminRoleJson, err := json.Marshal(gasFeesAdminRole)
	if err != nil {
		fmt.Printf("error marshaling struct: %v\n", err)
		return false, fmt.Errorf("error marsheling user role")
	}

	_, err = s.SetUserRoles(ctx, string(gasFeesAdminRoleJson))
	if err != nil {
		fmt.Printf("error setting roles gini admin: %v", err)
		return false, fmt.Errorf("error setting roles gini admin: %v", err)
	}

	kalpGateWayAdminRole := UserRole{
		Id:      intialkalpGateWayadmin,
		Role:    kalpGateWayAdmin,
		DocType: "UserRoleMap",
	}
	kalpGateWayAdminRoleJson, err := json.Marshal(kalpGateWayAdminRole)
	if err != nil {
		fmt.Println("Error marshaling struct:", err)
		return false, fmt.Errorf("error marsheling user role")
	}

	_, err = s.SetUserRoles(ctx, string(kalpGateWayAdminRoleJson))
	if err != nil {
		fmt.Printf("error setting roles gini admin: %v", err)
		return false, fmt.Errorf("error setting roles gini admin: %v", err)
	}

	err = s.mint(ctx, BridgeContractAddress, totalSupply)
	if err != nil {
		return false, fmt.Errorf("error with status code %v,error in minting: %v", http.StatusInternalServerError, err)
	}
	return true, nil
}

func (s *SmartContract) Name(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(nameKey)
	if err != nil {
		return "", fmt.Errorf("failed to get Name: %v", err)
	}
	return string(bytes), nil
}

func (s *SmartContract) Symbol(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(symbolKey)
	if err != nil {
		return "", fmt.Errorf("failed to get Name: %v", err)
	}
	return string(bytes), nil
}

func (s *SmartContract) Decimals(ctx kalpsdk.TransactionContextInterface) uint8 {
	return 18
}

func (s *SmartContract) GetGasFees(ctx kalpsdk.TransactionContextInterface) (string, error) {
	bytes, err := ctx.GetState(gasFeesKey)
	if err != nil {
		// return "", fmt.Errorf("failed to get Name: %v", err)
		fmt.Printf("failed to get Gas Fee: %v", err)
		return "", fmt.Errorf("failed to get Gas Fee: %v", err)
	}
	if bytes == nil {
		return "", fmt.Errorf("gas fee not set")
	}
	return string(bytes), nil
}

func (s *SmartContract) SetGasFees(ctx kalpsdk.TransactionContextInterface, gasFees string) error {
	logger := kalpsdk.NewLogger()
	operator, err := GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to get client id: %v", http.StatusBadRequest, err)
	}
	userRole, err := s.GetUserRoles(ctx, operator)
	if err != nil {
		logger.Infof("error checking sponsor's role: %v", err)
		return fmt.Errorf("error checking sponsor's role: %v", err)
	}
	logger.Infof("useRole: %s\n", userRole)
	if userRole != gasFeesAdminRole {
		return fmt.Errorf("error with status code %v, error: only gas fees admin is allowed to update gas fees", http.StatusInternalServerError)
	}
	err = ctx.PutStateWithoutKYC(gasFeesKey, []byte(gasFees))
	if err != nil {
		return fmt.Errorf("failed to set gasfees: %v", err)
	}
	return nil
}

func (s *SmartContract) mint(ctx kalpsdk.TransactionContextInterface, address string, amount string) error {
	logger := kalpsdk.NewLogger()
	logger.Infof("Mint---->")

	accAmount, su := big.NewInt(0).SetString(amount, 10)
	if !su {
		return fmt.Errorf("error with status code %v,can't convert amount to big int %s", http.StatusConflict, amount)
	}
	if accAmount.Cmp(big.NewInt(0)) == -1 || accAmount.Cmp(big.NewInt(0)) == 0 { // <= 0 {
		return fmt.Errorf("error with status code %v,amount can't be less then 0", http.StatusBadRequest)
	}

	balance, _ := GetTotalUTXO(ctx, address)
	logger.Infof("balance: %s", balance)
	balanceAmount, su := big.NewInt(0).SetString(balance, 10)
	if !su {
		logger.Infof("amount can't be converted to string ")
		return fmt.Errorf("amount can't be converted to string: ")
	}
	if balanceAmount.Cmp(big.NewInt(0)) == 1 {
		return fmt.Errorf("internal error %v: error can't call mint request twice", http.StatusBadRequest)
	}

	// Mint tokens
	err := MintUtxoHelperWithoutKYC(ctx, []string{address}, accAmount, DocTypeNIU)
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to mint tokens: %v", http.StatusBadRequest, err)
	}

	logger.Infof("MintToken Amount---->%v\n", amount)
	balanceAfterMint, err := GetTotalUTXO(ctx, address)
	if err != nil {
		return fmt.Errorf("error with status code %v, failed to mint tokens: %v", http.StatusInternalServerError, err)
	}
	balanceAfterMintAmount, su := big.NewInt(0).SetString(balanceAfterMint, 10)
	if !su {
		logger.Infof("balanceAfterMint amount can't be converted to string ")
		return fmt.Errorf("balanceAfterMint amount can't be converted to string ")
	}
	totalSupplyAmount, su := big.NewInt(0).SetString(totalSupply, 10)
	if !su {
		logger.Infof("totalSupplyAmount amount can't be converted to string ")
		return fmt.Errorf("totalSupplyAmount amount can't be converted to string ")
	}
	if balanceAfterMintAmount.Cmp(totalSupplyAmount) != 0 {
		return fmt.Errorf("error with status code %v,error: minitng failed", http.StatusInternalServerError)
	}
	logger.Infof("balanceAfterMint: %s", balanceAfterMint)
	return nil

}

// func (s *SmartContract) Burn(ctx kalpsdk.TransactionContextInterface, address string) (Response, error) {
// 	//check if contract has been intilized first
// 	logger := kalpsdk.NewLogger()
// 	logger.Infof("RemoveFunds---->%s", env)
// 	initialized, err := CheckInitialized(ctx)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to check if contract is already initialized: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v,error: failed to check if contract is already initialized: %v", http.StatusInternalServerError, err)

// 	}
// 	if !initialized {
// 		return Response{
// 			Message:    "contract options need to be set before calling any function, call Initialize() to initialize contract",
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v,error: contract options need to be set before calling any function, call Initialize() to initialize contract", http.StatusInternalServerError)
// 	}

// 	err = InvokerAssertAttributeValue(ctx, MailabRoleAttrName, PaymentRoleValue)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("payment admin role check failed in Brun request: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error: payment admin role check failed in Brun request: %v", http.StatusInternalServerError, err)
// 	}

// 	operator, err := GetUserId(ctx)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to get client id: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusBadRequest,
// 		}, fmt.Errorf("error with status code %v, error:failed to get client id: %v", http.StatusBadRequest, err)
// 	}

// 	amount, err := s.BalanceOf(ctx, address)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("failed to get balance : %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusBadRequest,
// 		}, fmt.Errorf("error with status code %v, error:failed to get balance : %v", http.StatusBadRequest, err)
// 	}
// 	accAmount, su := big.NewInt(0).SetString(amount, 10)
// 	if !su {
// 		return Response{
// 			Message:    fmt.Sprintf("can't convert amount to big int %s", amount),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusConflict,
// 		}, fmt.Errorf("error with status code %v,can't convert amount to big int %s", http.StatusConflict, amount)
// 	}
// 	err = RemoveUtxo(ctx, address, false, accAmount)
// 	if err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("Remove balance in burn has error: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error:Remove balance in burn has error: %v", http.StatusBadRequest, err)
// 	}

// 	if err := EmitTransferSingle(ctx, TransferSingle{Operator: operator, From: address, To: "0x0", Value: accAmount}); err != nil {
// 		return Response{
// 			Message:    fmt.Sprintf("unable to remove funds: %v", err),
// 			Success:    false,
// 			Status:     "Failure",
// 			StatusCode: http.StatusInternalServerError,
// 		}, fmt.Errorf("error with status code %v, error:unable to remove funds: %v", http.StatusBadRequest, err)
// 	}

// 	funcName, _ := ctx.GetFunctionAndParameters()
// 	response := map[string]interface{}{
// 		"txId":            ctx.GetTxID(),
// 		"txFcn":           funcName,
// 		"txType":          "Invoke",
// 		"transactionData": address,
// 	}
// 	return Response{
// 		Message:    "Funds removed successfully",
// 		Success:    true,
// 		Status:     "Success",
// 		StatusCode: http.StatusCreated,
// 		Response:   response,
// 	}, nil
// }

func (s *SmartContract) Transfer(ctx kalpsdk.TransactionContextInterface, address string, amount string) (bool, error) {
	logger := kalpsdk.NewLogger()
	logger.Infof("Transfer---->%s", env)
	address = strings.Trim(address, " ")
	if address == "" {
		return false, fmt.Errorf("invalid input address")
	}
	sender, err := ctx.GetUserID()
	if err != nil {
		return false, fmt.Errorf("error in getting user id: %v", err)
	}
	userRole, err := s.GetUserRoles(ctx, sender)
	if err != nil {
		logger.Infof("error checking sponsor's role: %v", err)
		return false, fmt.Errorf("error checking sponsor's role:: %v", err)
	}
	gasFees, err := s.GetGasFees(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get gas gee: %v", err)
	}
	gasFeesAmount, su := big.NewInt(0).SetString(gasFees, 10)
	if !su {
		return false, fmt.Errorf("gasfee can't be converted to big int")
	}
	validateAmount, su := big.NewInt(0).SetString(amount, 10)
	if !su {
		logger.Infof("Amount can't be converted to string")
		return false, fmt.Errorf("error with status code %v,Amount can't be converted to string", http.StatusConflict)
	}
	if validateAmount.Cmp(big.NewInt(0)) == -1 || validateAmount.Cmp(big.NewInt(0)) == 0 { // <= 0 {
		return false, fmt.Errorf("error with status code %v,amount can't be less then 0", http.StatusBadRequest)
	}
	logger.Infof("useRole: %s\n", userRole)
	// In this scenario sender is kalp gateway we will credit amount to kalp foundation as gas fees
	if userRole == kalpGateWayAdmin {
		var send Sender
		errs := json.Unmarshal([]byte(address), &send)
		if errs != nil {
			logger.Info("internal error: error in parsing sender data")
			return false, fmt.Errorf("internal error: error in parsing sender data")
		}
		if send.Sender != kalpFoundation {
			gAmount, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Infof("amount can't be converted to string ")

				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}
			err = RemoveUtxo(ctx, send.Sender, false, gAmount)
			if err != nil {
				logger.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}

			err = AddUtxo(ctx, kalpFoundation, false, gAmount)
			if err != nil {
				logger.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Infof("foundation transfer : %s\n", userRole)
		}
	} else if b, err := IsCallerKalpBridge(ctx, BridgeContractAddress); b && err == nil {
		// In this scenario sender is Kalp Bridge we will credit amount to kalp foundation and remove amount from sender
		logger.Infof("sender address changed to Bridge contract addres: \n", BridgeContractAddress)
		// In this scenario sender is kalp foundation is bridgeing will credit amount to kalp foundation and remove amount from sender without gas fees
		if sender == kalpFoundation {
			sender = BridgeContractAddress
			am, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}

			err = RemoveUtxo(ctx, sender, false, am)
			if err != nil {
				logger.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}

			err = AddUtxo(ctx, kalpFoundation, false, am)
			if err != nil {
				logger.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Infof("foundation transfer to self : %s\n", userRole)
		} else {
			// In this scenario sender is Kalp Bridge we will credit gas fees to kalp foundation and remove amount from bridge contract
			// address. Reciver will recieve amount after gas fees deduction
			sender = BridgeContractAddress
			am, su := big.NewInt(0).SetString(amount, 10)
			if !su {
				logger.Infof("amount can't be converted to string ")
				return false, fmt.Errorf("amount can't be converted to string: %v ", err)
			}

			err = RemoveUtxo(ctx, sender, false, am)
			if err != nil {
				logger.Infof("transfer remove err: %v", err)
				return false, fmt.Errorf("transfer remove err: %v", err)
			}
			err = AddUtxo(ctx, address, false, am)
			if err != nil {
				logger.Infof("err: %v\n", err)
				return false, fmt.Errorf("transfer add err: %v", err)
			}
			logger.Infof("foundation transfer to self : %s\n", userRole)
		}
	} else if sender == kalpFoundation && address == kalpFoundation {
		//In this scenario sender is kalp foundation and address is the kalp foundation so no addition or removal is required
		logger.Infof("foundation transfer to sender : %s address:%s\n", sender, address)

	} else if sender == kalpFoundation {
		//In this scenario sender is kalp foundation and address is the reciver so no gas fees deduction in code
		am, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		err := RemoveUtxo(ctx, sender, false, am)
		if err != nil {
			logger.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("transfer remove err: %v", err)
		}
		err = AddUtxo(ctx, address, false, am)
		if err != nil {
			logger.Infof("err: %v\n", err)
			return false, fmt.Errorf("transfer add err: %v", err)
		}
		logger.Infof("foundation transfer to user : %s\n", userRole)

	} else if address == kalpFoundation {
		//In this scenario sender is normal user and address is the kap foundation so gas fees+amount will be credited to kalp foundation
		removeAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		removeAmount = removeAmount.Add(removeAmount, gasFeesAmount)
		err := RemoveUtxo(ctx, sender, false, removeAmount)
		if err != nil {
			logger.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("transfer remove err: %v", err)
		}
		logger.Infof("addAmount: %v\n", removeAmount)
		addAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Infof("amount can't be converted to string ")
			return false, fmt.Errorf("amount can't be converted to string: %v ", err)
		}
		addAmount.Add(addAmount, gasFeesAmount)
		logger.Infof("addAmount: %v\n", addAmount)
		err = AddUtxo(ctx, address, false, addAmount)
		if err != nil {
			logger.Infof("err: %v\n", err)
			return false, fmt.Errorf("transfer add err: %v", err)
		}
		logger.Infof("foundation transfer to user : %s\n", userRole)
	} else {
		//This is normal scenario where gas fees+ amount will be deducted from sender and amount will credited to address and gas fees will be credited to kalp foundation
		logger.Infof("operator-->", sender)
		logger.Info("transfer transferAmount")
		transferAmount, su := big.NewInt(0).SetString(amount, 10)
		if !su {
			logger.Infof("Amount can't be converted to string")
			return false, fmt.Errorf("error with status code %v,Amount can't be converted to string", http.StatusConflict)
		}
		fmt.Printf("transferAmount %v\n", transferAmount)
		fmt.Printf("gasFeesAmount %v\n", gasFeesAmount)

		// Withdraw the funds from the sender address
		err = RemoveUtxo(ctx, sender, false, transferAmount)
		if err != nil {
			logger.Infof("transfer remove err: %v", err)
			return false, fmt.Errorf("error with status code %v, error:error while reducing balance %v", http.StatusBadRequest, err)
		}
		recieverAmount := transferAmount.Add(transferAmount, gasFeesAmount)
		fmt.Printf("recieverAmount amount %v\n", recieverAmount)
		// Deposit the fund to the recipient address
		err = AddUtxo(ctx, address, false, recieverAmount)
		if err != nil {
			logger.Infof("err: %v\n", err)
			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
		}
		logger.Infof("gasFeesAmount %v\n", gasFeesAmount)
		err = AddUtxo(ctx, kalpFoundation, false, gasFeesAmount)
		if err != nil {
			logger.Infof("err: %v\n", err)
			return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
		}
	}
	transferSingleEvent := TransferSingle{Operator: sender, From: sender, To: address, Value: amount}
	if err := EmitTransferSingle(ctx, transferSingleEvent); err != nil {
		logger.Infof("err: %v\n", err)
		return false, fmt.Errorf("error with status code %v, error:error while adding balance %v", http.StatusBadRequest, err)
	}
	return true, nil

}

func (s *SmartContract) BalanceOf(ctx kalpsdk.TransactionContextInterface, owner string) (string, error) {
	logger := kalpsdk.NewLogger()
	owner = strings.Trim(owner, " ")
	if owner == "" {
		return big.NewInt(0).String(), fmt.Errorf("invalid input account is required")
	}
	amt, err := GetTotalUTXO(ctx, owner)
	if err != nil {
		return big.NewInt(0).String(), fmt.Errorf("error: %v", err)
	}

	logger.Infof("total balance%v\n", amt)

	return amt, nil
}

// GetHistoryForAsset is a smart contract function which list the complete history of particular R2CI asset from blockchain ledger.
func (s *SmartContract) GetHistoryForAsset(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return "", err
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return "", err
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

func (s *SmartContract) Approve(ctx kalpsdk.TransactionContextInterface, spender string, value string) (bool, error) {
	owner, err := ctx.GetUserID()
	if err != nil {
		return false, err
	}

	err = Approve(ctx, owner, spender, value)
	if err != nil {
		fmt.Printf("error unable to approve funds: %v", err)
		return false, err
	}
	return true, nil
}

func (s *SmartContract) TransferFrom(ctx kalpsdk.TransactionContextInterface, from string, to string, value string) (bool, error) {
	logger := kalpsdk.NewLogger()
	logger.Infof("TransferFrom---->%s", env)
	spender, err := ctx.GetUserID()
	if err != nil {
		return false, fmt.Errorf("error iin getting spender's id: %v", err)
	}
	err = TransferUTXOFrom(ctx, []string{from}, []string{spender}, to, value, UTXO)
	if err != nil {
		logger.Infof("err: %v\n", err)
		return false, fmt.Errorf("error: unable to transfer funds: %v", err)
	}
	return true, nil
}

func (s *SmartContract) Allowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string) (string, error) {

	allowance, err := Allowance(ctx, owner, spender)
	if err != nil {
		return "", fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err) //big.NewInt(0).String(), fmt.Errorf("internal error %v: failed to get allowance: %v", http.StatusBadRequest, err)
	}
	return allowance, nil
}

func (s *SmartContract) TotalSupply(ctx kalpsdk.TransactionContextInterface) (string, error) {
	return totalSupply, nil
}

// SetUserRoles is a smart contract function which is used to setup a role for user.
func (s *SmartContract) SetUserRoles(ctx kalpsdk.TransactionContextInterface, data string) (string, error) {
	//check if contract has been intilized first

	fmt.Println("SetUserRoles", data)

	// Parse input data into Role struct.
	var userRole UserRole
	errs := json.Unmarshal([]byte(data), &userRole)
	if errs != nil {
		return "", fmt.Errorf("failed to parse data: %v", errs)
	}

	userValid, err := s.ValidateUserRole(ctx, giniAdmin)
	if err != nil {
		return "", fmt.Errorf("error in validating the role %v", err)
	}
	if !userValid {
		return "", fmt.Errorf("error in setting role %s, only GINI-ADMIN can set the roles", userRole.Role)
	}

	// Validate input data.
	if userRole.Id == "" {
		return "", fmt.Errorf("user Id can not be null")
	}

	if userRole.Role == "" {
		return "", fmt.Errorf("role can not be null")
	}

	ValidRoles := []string{giniAdmin, gasFeesAdminRole, kalpGateWayAdmin}
	if !slices.Contains(ValidRoles, userRole.Role) {
		return "", fmt.Errorf("invalid input role")
	}

	key, err := ctx.CreateCompositeKey(userRolePrefix, []string{userRole.Id, UserRoleMap})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", userRolePrefix, err)
	}
	// Generate JSON representation of Role struct.
	usrRoleJSON, err := json.Marshal(userRole)
	if err != nil {
		return "", fmt.Errorf("unable to Marshal userRole struct : %v", err)
	}
	// Store the Role struct in the state database
	if err := ctx.PutStateWithoutKYC(key, usrRoleJSON); err != nil {
		return "", fmt.Errorf("unable to put user role struct in statedb: %v", err)
	}
	return s.GetTransactionTimestamp(ctx)

}

func (s *SmartContract) ValidateUserRole(ctx kalpsdk.TransactionContextInterface, Role string) (bool, error) {

	// Check if operator is authorized to create Role.
	operator, err := GetUserId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get client id: %v", err)
	}

	fmt.Println("operator---------------", operator)
	userRole, err1 := s.GetUserRoles(ctx, operator)
	if err1 != nil {
		return false, fmt.Errorf("error: %v", err1)
	}

	if userRole != Role {
		return false, fmt.Errorf("this transaction can be performed by %v only", Role)
	}
	return true, nil
}

// GetUserRoles is a smart contract function which is used to get a role of a user.
func (s *SmartContract) GetUserRoles(ctx kalpsdk.TransactionContextInterface, id string) (string, error) {
	// Get the asset from the ledger using id & check if asset exists
	key, err := ctx.CreateCompositeKey(userRolePrefix, []string{id, UserRoleMap})
	if err != nil {
		return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", userRolePrefix, err)
	}

	userJSON, err := ctx.GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if userJSON == nil {
		return "", nil
	}

	// Unmarshal asset from JSON to struct
	var userRole UserRole
	err = json.Unmarshal(userJSON, &userRole)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal user role struct : %v", err)
	}

	return userRole.Role, nil
}
