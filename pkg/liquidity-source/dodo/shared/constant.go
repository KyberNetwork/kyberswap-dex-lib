package shared

const (
	// SubgraphPoolType DodoV1
	SubgraphPoolTypeDodoClassical = "CLASSICAL"
	// SubgraphPoolType DodoV2
	SubgraphPoolTypeDodoVendingMachine = "DVM"
	SubgraphPoolTypeDodoStable         = "DSP"
	SubgraphPoolTypeDodoPrivate        = "DPP"

	defaultTokenWeight   = 50
	defaultTokenDecimals = 18

	zeroString = "0"

	// Dodo Classical contract methods
	dodoV1MethodGetExpectedTarget = "getExpectedTarget"
	dodoV1MethodK                 = "_K_"
	dodoV1MethodRStatus           = "_R_STATUS_"
	dodoV1MethodGetOraclePrice    = "getOraclePrice"
	dodoV1MethodLpFeeRate         = "_LP_FEE_RATE_"
	dodoV1MethodMtFeeRate         = "_MT_FEE_RATE_"
	dodoV1MethodBaseBalance       = "_BASE_BALANCE_"
	dodoV1MethodQuoteBalance      = "_QUOTE_BALANCE_"
	dodoV1MethodTradeAllowed      = "_TRADE_ALLOWED_"
	dodoV1MethodSellingAllowed    = "_SELLING_ALLOWED_"
	dodoV1MethodBuyingAllowed     = "_BUYING_ALLOWED_"

	// Dodo V2 contract methods
	dodoV2MethodGetPMMStateForCall = "getPMMStateForCall"
	dodoV2MethodLpFeeRate          = "_LP_FEE_RATE_"
	dodoV2MethodGetUserFeeRate     = "getUserFeeRate"
)

var (
	V2DefaultGas = V2Gas{
		SellBase:  128000,
		SellQuote: 116000,
	}
)
