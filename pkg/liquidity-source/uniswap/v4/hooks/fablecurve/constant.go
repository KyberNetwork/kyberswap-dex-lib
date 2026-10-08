package fablecurve

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is FableHook, the single hook every coin launched on Fable (https://fablecurve.fun) trades through
// on Robinhood chain. Source verified on robin.etherscan.io: src/FableHook.sol.
var HookAddresses = []common.Address{
	common.HexToAddress("0x376aA4C2e48AC2498CaaAc06bd7A6b6ef373f8E0"),
}

const (
	sniperStartPips = 990_000 // FableHook.SNIPER_START_PIPS: 99% at the second a pool is created
	sniperSeconds   = 3       // FableHook.SNIPER_SECONDS: the fee reaches the pool's own fee 3 s after creation
	maxFeePips      = 200_000 // FableHook.MAX_FEE_PIPS: beforeInitialize refuses a pool fee above 20%

	// gasBeforeSwap is what beforeSwap adds on top of a plain fee override: the call to the LiquidityLocker's
	// checkpointCreatorFees. Measured through V4Quoter at Robinhood block 81852934: 65,915 gas for a 1e14 wei buy
	// of FABLE against 48,493 for the same buy through MofoHook, whose beforeSwap is otherwise identical.
	gasBeforeSwap = 17_500
)
