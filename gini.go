/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	"gini-contract/chaincode"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

func main() {
	contract := kalpsdk.Contract{IsPayableContract: false}
	contract.Contract.Name = "klp-6b616c70616373-cc"
	contract.Logger = kalpsdk.NewLogger()
	giniChaincode, err := kalpsdk.NewChaincode(&chaincode.SmartContract{Contract: contract})
	if err != nil {
		log.Panicf("Error creating gini chaincode: %v", err)
	}

	if err := giniChaincode.Start(); err != nil {
		log.Panicf("Error starting gini chaincode: %v", err)
	}
}
