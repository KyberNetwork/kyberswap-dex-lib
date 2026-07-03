package shared

const (
	// SubgraphPoolType DodoV1
	SubgraphPoolTypeDodoClassical = "CLASSICAL"
	// SubgraphPoolType DodoV2
	SubgraphPoolTypeDodoVendingMachine = "DVM"
	SubgraphPoolTypeDodoStable         = "DSP"
	SubgraphPoolTypeDodoPrivate        = "DPP"
	// SubgraphPoolType Dodo GasSavingPool (GSP)
	SubgraphPoolTypeDodoGasSaving = "GSP"

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
	dodoV2MethodGetUserFeeRate     = "getUserFeeRate"
	dodoV2MethodMinBaseSwapAmount  = "_MIN_BASE_SWAP_AMOUNT_"
	dodoV2MethodMinQuoteSwapAmount = "_MIN_QUOTE_SWAP_AMOUNT_"
	dodoV2MethodVersion            = "version"

	dodoV2VersionWithMinSwapAmount = "1.1.0"
)

var (
	V2DefaultGas = V2Gas{
		SellBase:  128000,
		SellQuote: 116000,
	}
)
