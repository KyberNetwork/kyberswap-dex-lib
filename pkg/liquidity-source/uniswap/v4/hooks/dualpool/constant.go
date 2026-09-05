package dualpool

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// DualPoolHook (Uniswap Labs, v4-hooks-public/src/alf/DualPoolHook.sol) is a
// just-in-time liquidity hook: the pool holds no resident v4 liquidity. On every
// swap the hook deploys its reserves (ERC-20, ERC-6909 claims and ERC-4626 vault
// balances) as concentrated positions across a fixed set of weighted tick
// buckets, the swap executes on the ordinary v4 curve at the key's static fee,
// and the positions are removed again. Pricing therefore needs the hook's
// effective reserves and its bucket distribution rather than StateView ticks.
//
// Deployments on Robinhood Chain (chain id 4663), operated by Twofold
// (https://twofold.fi): the 2026-09-05 hook carries the ipfs metadata hash and
// is fully verified; the 2026-08 hook is the same audited bytecode.
var HookAddresses = []common.Address{
	common.HexToAddress("0xd1BcbCCa41f3bdb6b4812652959c6dF725ea2Ac0"), // 2026-09-05, fee-500 pools
	common.HexToAddress("0x127B3f3b7769f659C5eDBfF8b4005443f19FAAc0"), // 2026-08-16, fee-3000 pools
}

const (
	// StateView on Robinhood Chain, used when the source config carries none.
	defaultStateViewRobinhood = "0xf3334192d15450cdd385c8b70e03f9a6bd9e673b"

	totalWeightBps = 10_000
	maxSwapFee     = 1_000_000
	pipsDenom      = 1_000_000

	// A DualPool swap runs the JIT cycle around the v4 swap: redeem claims and
	// vault shares, mint the bucket positions, swap, burn the positions, settle,
	// re-deposit to the vault. A single-hop Universal Router swap on a Twofold
	// pool used 968,379 gas in total on Robinhood Chain
	// (tx 0xb8687b221e541fa8f10f81586bee0b4858436dd6565c5bec7c18a7860d164cc3).
	swapGas int64 = 900_000
)

var (
	ErrPoolNotLive           = errors.New("dualpool: pool is not live")
	ErrNoDistribution        = errors.New("dualpool: no distribution")
	ErrNoReserves            = errors.New("dualpool: no reserves")
	ErrExactOutUnsupported   = errors.New("dualpool: exact output is not supported")
	ErrInsufficientLiquidity = errors.New("dualpool: insufficient liquidity")
	ErrZeroOutput            = errors.New("dualpool: zero output")
	ErrStateNotSet           = errors.New("dualpool: hook state not set")
)
