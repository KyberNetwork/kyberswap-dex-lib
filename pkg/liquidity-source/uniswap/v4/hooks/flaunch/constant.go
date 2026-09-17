package flaunch

import "github.com/ethereum/go-ethereum/common"

// FeeDenom is the PositionManager's fee scale (FeeDistributor.ONE_HUNDRED_PERCENT = 100_00):
// a swapFee of 100 is 1%.
const FeeDenom = 100_00

// DefaultSwapFee is the protocol-wide FeeDistribution.swapFee every generation has shipped with
// (1%). It is only the pricing fallback for a pool whose fee has not been Track()ed yet; the live
// value comes from getPoolFeeDistribution, which a pool owner may override.
const DefaultSwapFee = 100

// HookAddresses lists every Flaunch PositionManager (Uniswap v4 hook) that serves live pools.
// Registration is address-keyed, so a CREATE3 address deployed on several chains appears once.
// Source of truth: @flaunch/sdk src/addresses.ts.
var HookAddresses = []common.Address{
	// ---- Base (8453) ----
	// PositionManager 1.0
	common.HexToAddress("0x51Bba15255406Cfe7099a42183302640ba7dAFDC"),

	// PositionManager 1.1
	common.HexToAddress("0xF785bb58059FAB6fb19bDdA2CB9078d9E546Efdc"),

	// PositionManager 1.2
	common.HexToAddress("0xB903b0AB7Bcee8f5E4D8C9b10a71aaC7135d6FdC"),

	// PositionManager 1.3
	common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),

	// AnyPositionManager 1.0
	common.HexToAddress("0x8DC3b85e1dc1C846ebf3971179a751896842e5dC"),

	// PositionManager v1.3 (paired-token generation). Also the superseded v1.3.1 hook on
	// Robinhood (4663), which keeps serving the pools it created.
	common.HexToAddress("0x588C683EcC450F8b2aAdb13D7f63792b840425DC"),
	// AnyPositionManager v1.3. Also the superseded v1.3.1 AnyPositionManager on Robinhood.
	common.HexToAddress("0x6eA0eDeE449A287504990Df8D87951B9436825Dc"),

	// ---- Arbitrum (42161) ----
	// PositionManager v1.3
	common.HexToAddress("0xCAb62e007AB6656877Ab556cEa330De27AE025DC"),
	// AnyPositionManager v1.3
	common.HexToAddress("0xbbE1b9831117E829Fe80Be97fCe82F2b49C1a5DC"),

	// ---- Robinhood Chain (4663) ----
	// PositionManager v1.3.3
	common.HexToAddress("0x8D346f24278C5CD786309161aAC0fC2bbe4c25dc"),
	// AnyPositionManager v1.3.3
	common.HexToAddress("0x9AbfbDc34A294De5210C0889f21D5Af54C4965DC"),

	// ---- Ethereum (1) ----
	// PositionManager v1.4.0
	common.HexToAddress("0xb741a710E456FC6d7f76c88F5C56B27D05e8A5DC"),
	// AnyPositionManager v1.4.0
	common.HexToAddress("0x0215C3ef94ef3e86c32e847c662ad649000965Dc"),

	// ---- Base, Arbitrum, Robinhood (CREATE3 parity, 2026-09-17) ----
	// AnyPositionManager, vested-launch generation (Game Mode). One address on all three chains.
	common.HexToAddress("0xE753a351FB498051a09Dc130fcC29aEBc76525DC"),
}
