package iziswap

import "math/big"

const DexTypeiZiSwap = "iziswap"

const methodGetState = "state"
const methodGetLiquiditySnapshot = "liquiditySnapshot"
const methodGetLimitOrderSnapshot = "limitOrderSnapshot"
const erc20MethodBalanceOf = "balanceOf"

const defaultTokenWeight = 50
const emptyString = ""
const zeroString = "0"

const SNAPSHOT_BATCH = 256

var zeroBI = big.NewInt(0)
var boneFloat, _ = new(big.Float).SetString("1000000000000000000")

// var feeBase = big.NewInt(1e6)

var pointDeltas = map[int]int{
	100:   1,
	400:   8,
	500:   10,
	2000:  40,
	3000:  60,
	10000: 200,
}

const RIGHT_MOST_PT int = 800000

const LEFT_MOST_PT int = -800000

const DEFAULT_PT_RANGE = 2000
const SIMULATOR_PT_RANGE = 2000

const POOL_LIST_LIMIT = 1000
const POOL_TYPE_VALUE = "10"
