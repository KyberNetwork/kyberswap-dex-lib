package mofo

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is MofoHook, the single hook every coin launched on mofo (https://mofo.zone) trades through on
// Robinhood chain. Source verified on Blockscout and RobinScan: src/MofoHook.sol.
var HookAddresses = []common.Address{
	common.HexToAddress("0x665C52D02Ddc506dfb3158C100Edc77FE412a8c0"),
}

const (
	sniperStartPips = 990_000 // MofoHook.SNIPER_START_PIPS: 99% at the second a pool is created
	sniperSeconds   = 3       // MofoHook.SNIPER_SECONDS: the fee reaches the pool's own fee 3 s after creation
	maxFeePips      = 200_000 // MofoHook.MAX_FEE_PIPS: beforeInitialize refuses a pool fee above 20%
)
