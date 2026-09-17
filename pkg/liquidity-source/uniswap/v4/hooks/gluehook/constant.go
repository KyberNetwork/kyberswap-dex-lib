package gluehook

import (
	"github.com/ethereum/go-ethereum/common"
)

// GlueHook V3 is deployed via CREATE from a nonce-0 deployer, so it lives at the SAME address on
// every chain it supports (Ethereum, Base, Unichain, Arbitrum, Optimism, BNB, Polygon, Avalanche,
// X Layer, World Chain, Soneium, MegaETH, Robinhood, Arc). Permission bits 0x2040 —
// beforeInitialize | afterSwap only: no beforeSwap, no return delta, so quoting is vanilla V4.
//
// This REPLACES the V2 address 0x0F41715dc432692b66A5aDF8dCfef6Ac407b20c8 registered in #1599.
// V2 stays deployed for its existing pools but is superseded — new pools land on V3 only.
var HookAddresses = []common.Address{
	common.HexToAddress("0x03D482cB3Ff339C2d29736818D0F72c66dD6A040"),
}

// Worst-case extra gas a GlueHook pot action can add to a swap, from the audited gas suite
// (idle ~8-12k, pump ~+88k, in-swap auto-harvest + compound ~+111k).
const maxHookGas = 120_000
