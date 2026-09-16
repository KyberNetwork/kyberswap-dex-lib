package gluehook

import (
	"github.com/ethereum/go-ethereum/common"
)

// GlueHook is deployed via CREATE from a nonce-0 deployer, so each generation lives at the SAME
// address on every chain it supports (Ethereum, Base, Unichain, Arbitrum, Optimism, BNB, Polygon,
// Avalanche, X Layer, World Chain, Soneium, MegaETH, Robinhood).
//
//   - V3 (canonical, new pools; permission bits 0x2040 — beforeInitialize | afterSwap only)
//   - V2 (still live for its existing pools; bits 0x20C8)
//
// Both are pure passthroughs for quoting: neither changes the swapper's amounts.
var HookAddresses = []common.Address{
	common.HexToAddress("0x03D482cB3Ff339C2d29736818D0F72c66dD6A040"), // V3
	common.HexToAddress("0x0F41715dc432692b66A5aDF8dCfef6Ac407b20c8"), // V2
}

// Worst-case extra gas a GlueHook pot action can add to a swap, from the audited gas suite
// (idle ~8-12k, pump ~+88k, in-swap auto-harvest + compound ~+111k; V2's shield ~+38k).
const maxHookGas = 120_000
