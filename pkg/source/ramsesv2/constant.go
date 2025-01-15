package ramsesv2

import (
	"math/big"
)

const (
	DexTypeRamsesV2       = "ramses-v2"
	graphSkipLimit        = 5000
	graphFirstLimit       = 1000
	defaultTokenDecimals  = 18
	defaultTokenWeight    = 50
)

const (
	methodV3Fee = "fee"

	methodV2GetLiquidity = "liquidity"
	methodV2GetSlot0     = "slot0"
	methodV2CurrentFee   = "currentFee"
	methodV2TickSpacing  = "tickSpacing"
	erc20MethodBalanceOf = "balanceOf"
)

var (
	zeroBI = big.NewInt(0)
	// Waiting the SC team to estimate the CrossInitTickGas at thread:
	// https://team-kyber.slack.com/archives/C05V8NL8CSF/p1702621669962399.
	// For now, keep the BaseGas = 125000 (as the previous config), CrossInitTickGas = 0.
	defaultGas = Gas{BaseGas: 125000, CrossInitTickGas: 0}
)
