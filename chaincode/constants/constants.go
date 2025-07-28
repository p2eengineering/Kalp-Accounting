package constants

const (
	KalpFoundationAddress         = "951d359b6a5dbcd130ea424f4cfb875b81ae2b2f"
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
	InitialBridgeContractAddress  = "klp-519fe60d6e-cc"
	GiniContractAddress           = "klp-f02611a93e-cc"
	ContractAddressRegex          = `^klp-[a-fA-F0-9]+-cc`
	UserAddressRegex              = `^[0-9a-fA-F]{40}$`
	IsContractAddressRegex        = `^klp-[a-fA-F0-9]+-cc$`
)
