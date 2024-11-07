package common

const (
	LRTDepositPool = "0x036676389e48133b63a802f8635ad39e752d375d"
	LRTOracle      = "0x349a73444b1a310bae67ef67973022020d70020d"
	LRTConfig      = "0x947cb49334e6571ccbfef1f1f1178d8469d65ec7"
	ETH            = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	WETH           = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	RSETH          = "0xa1290d69c65a6fe4df752f95823fae25cb99e5a7"
)

const (
	LRTDepositPoolMethodMinAmountToDeposit    = "minAmountToDeposit"
	LRTDepositPoolMethodGetTotalAssetDeposits = "getTotalAssetDeposits"

	LRTOracleMethodRSETHPrice    = "rsETHPrice"
	LRTOracleMethodGetAssetPrice = "getAssetPrice"

	LRTConfigMethodGetSupportedAssetList = "getSupportedAssetList"
	LRTConfigMethodDepositLimitByAsset   = "depositLimitByAsset"

	Erc20MethodDecimals = "decimals"
)
