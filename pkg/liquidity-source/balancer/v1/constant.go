package balancerv1

var (
	DexType = "balancer-v1"
)

var (
	defaultGas = Gas{SwapExactAmountIn: 117092}
)

var (
	bPoolMethodGetCurrentTokens      = "getCurrentTokens"
	bPoolMethodGetDenormalizedWeight = "getDenormalizedWeight"
	bPoolMethodGetBalance            = "getBalance"
	bPoolMethodGetSwapFee            = "getSwapFee"
	bPoolMethodIsBound               = "isBound"
	bPoolMethodIsPublicSwap          = "isPublicSwap"
)
