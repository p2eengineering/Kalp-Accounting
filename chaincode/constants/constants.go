package constants

const (
	KalpFoundationAddress         = "0b87970433b22494faff1cc7a819e71bddc7880c"
	KalpGateWayAdminAddress       = "67c30fcb223182fef1c471a26527bfc4c50d093c"
	InitialVestingContractBalance = "1988800000000000000000000000"
	InitialFoundationBalance      = "11200000000000000000000000"
	InitialGasFees                = "1000000000000000"
	InitialGatewayMaxGasFee       = "100000000000000000"
	NameKey                       = "name"
	SymbolKey                     = "symbol"
	GasFeesKey                    = "gasFees"
	GatewayMaxFee                 = "gatewayMaxFee"
	DenyListKey                   = "denyList"
	GINI                          = "GINI"
	TotalSupply                   = "2000000000000000000000000000"
	KalpFoundationRole            = "KalpFoundation"
	KalpGateWayAdminRole          = "KalpGatewayAdmin"
	UserRolePrefix                = "ID~UserRoleMap"
	UserRoleMap                   = "UserRoleMap"
	UTXO                          = "UTXO"
	Allowance                     = "Allowance"
	Approval                      = "Approval"
	Denied                        = "Denied"
	Allowed                       = "Allowed"
	Mint                          = "Mint"
	Transfer                      = "Transfer"
	VestingContractKey            = "vestingContract"
	BridgeContractKey             = "bridgeContract"
	InitialBridgeContractAddress  = "klp-6b616c70627269646765-cc"
	GiniContractAddress           = "klp-abab101-cc"
	ContractAddressRegex          = `^klp-[a-fA-F0-9]+-cc`
	UserAddressRegex              = `^[0-9a-fA-F]{40}$`
	IsContractAddressRegex        = `^klp-[a-fA-F0-9]+-cc$`
)
