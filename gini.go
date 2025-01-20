/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	"gini-contract/chaincode"
	"gini-contract/chaincode/constants"

	"github.com/muditp2e/kalp-sdk-public/kalpsdk"
)

func main() {
	contract := kalpsdk.Contract{}
	contract.Contract.Name = constants.GiniContractAddress
	contract.Logger = kalpsdk.NewLogger()
	giniChaincode, err := kalpsdk.NewChaincode(&chaincode.SmartContract{Contract: contract})
	if err != nil {
		log.Panicf("Error creating gini chaincode: %v", err)
	}

	if err := giniChaincode.Start(); err != nil {
		log.Panicf("Error starting gini chaincode: %v", err)
	}
}
