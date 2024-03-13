/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	kalpAccounting "KAPS-NIU/niu"

	"github.com/p2eengineering/kalp-sdk/kalpsdk"
)

func main() {
	contract := kalpsdk.Contract{IsPayableContract: false}
	contract.Logger = kalpsdk.NewLogger()
	nftChaincode, err := kalpsdk.NewChaincode(&kalpAccounting.SmartContract{Contract: contract})
	if err != nil {
		log.Panicf("Error creating nft chaincode: %v", err)
	}

	if err := nftChaincode.Start(); err != nil {
		log.Panicf("Error starting nft chaincode: %v", err)
	}
}
