package gluehook

import (
	"github.com/ethereum/go-ethereum/common"
)

// GlueHook is deployed via CREATE from a nonce-0 deployer, so it lives at the SAME address on
// every chain it supports (Ethereum, Base, Unichain, Arbitrum, Optimism, BNB, Polygon,
// Avalanche, X Layer, World Chain, Soneium, MegaETH, Robinhood).
var HookAddresses = []common.Address{
	common.HexToAddress("0x0F41715dc432692b66A5aDF8dCfef6Ac407b20c8"),
}

// Worst-case extra gas a GlueHook pot action can add to a swap, from the audited gas suite
// (idle ~8-12k, shield ~+38k, pump ~+88k, in-swap auto-harvest + compound ~+111k).
const maxHookGas = 120_000
