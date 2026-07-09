package chaincode

import (
	"gini-contract/chaincode/ginierr"
	"gini-contract/chaincode/logger"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

// Deprecated: GINI-2.1  KWALA credit movement is now managed in Web2.
func (s *SmartContract) GasFeesTransfer(ctx kalpsdk.TransactionContextInterface, gasFeesAccount string, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("GasFeesTransfer")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.2  KWALA minting is now managed in Web2.
func (s *SmartContract) MintByFaucetAdmin(ctx kalpsdk.TransactionContextInterface, address string, amount string) error {
	err := ginierr.ErrDeprecatedFunction("MintByFaucetAdmin")
	logger.Log.Error(err.FullError())
	return err
}

// Deprecated: GINI-2.4  KWALA admin management is now managed in Web2.
func (s *SmartContract) SetKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	err := ginierr.ErrDeprecatedFunction("SetKwalaAdmin")
	logger.Log.Error(err.FullError())
	return err
}

// Deprecated: GINI-2.4  KWALA admin management is now managed in Web2.
func (s *SmartContract) DeleteKwalaAdmin(ctx kalpsdk.TransactionContextInterface, userID string) error {
	err := ginierr.ErrDeprecatedFunction("DeleteKwalaAdmin")
	logger.Log.Error(err.FullError())
	return err
}

// Deprecated: GINI-3.1  disabled LAST, after KWALA->Web2 migration was confirmed complete.
// This was intentionally the LAST KWALA function disabled, since it was the fn used
// by the migration to burn/settle remaining KWALA gas-fee balances.
func (s *SmartContract) TransferGasFeesToFoundation(ctx kalpsdk.TransactionContextInterface, fromAddress string, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferGasFeesToFoundation")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.1  KWALA credit movement is now managed in Web2.
func (s *SmartContract) TransferKWALACredits(ctx kalpsdk.TransactionContextInterface, fromAddress string, toAddress string, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferKWALACredits")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.3  the Kalp<->Kwala bridge is now managed in Web2.
func (s *SmartContract) TransferKalpToKwala(ctx kalpsdk.TransactionContextInterface, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferKalpToKwala")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.3  the Kalp<->Kwala bridge is now managed in Web2.
func (s *SmartContract) TransferFromAnyKalpToKwala(ctx kalpsdk.TransactionContextInterface, to, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("TransferFromAnyKalpToKwala")
	logger.Log.Error(err.FullError())
	return false, err
}

// Deprecated: GINI-2.2  KWALA minting is now managed in Web2.
func (s *SmartContract) MintToKwalaAccountOnBehalfOfKawalaAdmin(ctx kalpsdk.TransactionContextInterface, to, amount string) (bool, error) {
	err := ginierr.ErrDeprecatedFunction("MintToKwalaAccountOnBehalfOfKawalaAdmin")
	logger.Log.Error(err.FullError())
	return false, err
}
