package constants

const (
	KalpFoundationAddress                   = "0b87970433b22494faff1cc7a819e71bddc7880c"
	KalpGateWayAdminAddress                 = "67c30fcb223182fef1c471a26527bfc4c50d093c"
	TestnetFaucetAdmin                      = "88016ab3510adc3905d858e08d3c08d8a78041bd"
	InitialVestingContractBalance           = "1988800000000000000000000000"
	InitialFoundationBalance                = "11200000000000000000000000"
	InitialGasFees                          = "1000000000000000"
	InitialGatewayMaxGasFee                 = "100000000000000000"
	NameKey                                 = "name"
	SymbolKey                               = "symbol"
	GasFeesKey                              = "gasFees"
	GatewayMaxFee                           = "gatewayMaxFee"
	DenyListKey                             = "denyList"
	GINI                                    = "GINI"
	TotalSupply                             = "2000000000000000000000000000"
	KalpGateWayAdminRole                    = "KalpGatewayAdmin"
	UserRolePrefix                          = "ID~UserRoleMap"
	UserRoleMap                             = "UserRoleMap"
	UTXO                                    = "UTXO"
	Allowance                               = "Allowance"
	Approval                                = "Approval"
	Denied                                  = "Denied"
	Allowed                                 = "Allowed"
	Mint                                    = "Mint"
	Transfer                                = "Transfer"
	VestingContractKey                      = "vestingContract"
	BridgeContractKey                       = "bridgeContract"
	InitialBridgeContractAddress            = "klp-519fe60d6e-cc"
	GiniContractAddress                     = "klp-f02611a93e-cc"
	ContractAddressRegex                    = `^klp-[a-fA-F0-9]+-cc`
	UserAddressRegex                        = `^[0-9a-fA-F]{40}$`
	IsContractAddressRegex                  = `^klp-[a-fA-F0-9]+-cc$`
	KwalaAccountRegex                       = `^kwl-[0-9a-fA-F]{40}-cc$`
	KwalaAccountPrefix                      = "kwl"
	KwalaAccountSuffix                      = "cc"
	KwalaAdminRole                          = "KwalaAdmin"
	TransferGasFromKwalaAccountToFoundation = "TransferGasFromKwalaAccountToFoundation"
	TransferKalpToKwala                     = "TransferKalpToKwala"
	TransferFromAnyKalpToKwala              = "TransferFromAnyKalpToKwala"
	MintToKwalaAccountOnBehalfOfKawalaAdmin = "MintToKwalaAccountOnBehalfOfKawalaAdmin"
	KwalaDenyListKey                        = "kwalaDenyList"
)
