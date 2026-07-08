package unipool

const (
	DexType = "unipool"

	defaultGas = 217659

	bpsDivisor = 10_000

	factoryMethodGetAllPairsLength = "getAllPairsLength"
	factoryMethodGetPairAtIndex    = "getPairAtIndex"

	pairMethodGetTokens                = "getTokens"
	pairMethodGetReserves              = "getReserves"
	pairMethodGetVirtualReserves       = "getVirtualReserves"
	pairMethodGetLastUpdateTimestamp   = "getLastUpdateTimestamp"
	pairMethodGetPriceDecay            = "getPriceDecay"
	pairMethodGetFeesBps               = "getFeesBps"
	pairMethodGetTotalBorrowed0        = "getTotalBorrowed0"
	pairMethodGetTotalBorrowed1        = "getTotalBorrowed1"
	pairMethodGetSwapPriceToleranceBps = "getSwapPriceToleranceBps"

	factoryEventPairCreated = "PairCreated"
)

// swapPriceToleranceDisabled mirrors the on-chain `type(uint16).max` sentinel
// for which UniPoolPairSwap.swap skips the _validateSpreads check.
const swapPriceToleranceDisabled uint16 = 0xFFFF
