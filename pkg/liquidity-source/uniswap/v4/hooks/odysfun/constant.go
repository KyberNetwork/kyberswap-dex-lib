package odysfun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses lists every OdysHook deployment that gates ODYS Elite (Uniswap v4) markets.
// See https://odys.fun docs: OdysHook2 is the current line (tax schedule), OdysHook is the
// first Elite line (static 1/3/5% tiers, no schedule). Both are only deployed on Arbitrum One.
var HookAddresses = []common.Address{
	common.HexToAddress("0x18a6ba193352036ef8a7c22be2cb288bb26da8cc"), // OdysHook2 (tax schedule)
	common.HexToAddress("0xBC40c7492F5823785c4dcC1EC2F9D1b7406b28Cc"), // OdysHook (first Elite line)
}

// BpsDenominator is the fee-bps base used by OdysHook2 (BASIS_POINTS() == 10_000).
var BpsDenominator = big.NewInt(10_000)
