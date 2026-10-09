package fablecurve

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is FableHook, the single hook of every coin launched on Fable (source: robin.etherscan.io).
var HookAddresses = []common.Address{
	common.HexToAddress("0x376aA4C2e48AC2498CaaAc06bd7A6b6ef373f8E0"),
}

const (
	sniperStartPips = 990_000 // FableHook.SNIPER_START_PIPS: 99% at the second a pool is created
	sniperSeconds   = 3       // FableHook.SNIPER_SECONDS: the fee reaches the pool's own fee 3 s after creation
	maxFeePips      = 200_000 // FableHook.MAX_FEE_PIPS: beforeInitialize refuses a pool fee above 20%

	// gasBeforeSwap is the locker checkpointCreatorFees call beforeSwap adds to a plain fee override: V4Quoter at
	// Robinhood block 81852934 measured 65,915 gas against 48,493 for the same buy through MofoHook.
	gasBeforeSwap = 17_500
)
