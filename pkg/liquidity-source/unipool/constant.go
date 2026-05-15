package unipool

const (
	DexType = "unipool"

	// defaultGas is a conservative placeholder for routing cost estimation.
	// Realistic mainnet measurements suggest 100-140k for a clean swap; we keep
	// 150k as a safe upper bound. TODO: refine empirically once Unipool is
	// deployed (forge test --gas-report on /test or measured swaps on-chain).
	defaultGas = 150_000

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
