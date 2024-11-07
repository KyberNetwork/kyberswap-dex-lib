package iziswap

import "math/big"

const (
	DexTypeiZiSwap = "iziswap"

	methodGetState              = "state"
	methodGetLiquiditySnapshot  = "liquiditySnapshot"
	methodGetLimitOrderSnapshot = "limitOrderSnapshot"
	erc20MethodBalanceOf        = "balanceOf"

	defaultTokenWeight = 50
	emptyString        = ""
	zeroString         = "0"

	SNAPSHOT_BATCH     = 256
	RIGHT_MOST_PT  int = 800000

	LEFT_MOST_PT int = -800000

	DEFAULT_PT_RANGE   = 2000
	SIMULATOR_PT_RANGE = 2000
)

var zeroBI = big.NewInt(0)

var pointDeltas = map[int]int{
	100:   1,
	400:   8,
	500:   10,
	2000:  40,
	3000:  60,
	10000: 200,
}

// // Fee can be ignored for now
// var feeBase = big.NewInt(1e6)
// var boneFloat, _ = new(big.Float).SetString("1000000000000000000")
