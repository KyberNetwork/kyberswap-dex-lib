package kalos

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4Kalos

// HookAddresses is KalosDynamic (the source names the contract KalosV4FeeHook), the hook of every Kalos V4 pool on the
// shared Uniswap v4 PoolManager of Kinetiq Elysium (99901), 0xE2e105d3F7209A9DA11f83cCf3E7398a753823F1. Deployed in
// block 6,897,730; verified source https://github.com/fugakux/kalos-v4-hook. Its runtime code hashes to
// 0x295107603ee96a130e57cdbf0516c875972c45503a8528fae3a1ebe65d6f3cc1 (hook_live_test.go checks it).
var HookAddresses = []common.Address{
	common.HexToAddress("0x0559EF6bC07974bfaB2149d7fbC0E898914115CF"),
}

// Mirrors of KalosV4FeeHookBase's constants.
const (
	kindStandard uint8 = 0
	kindCreator  uint8 = 1

	feeNone      uint8 = 0
	feeCurrency0 uint8 = 1
	feeCurrency1 uint8 = 2

	pips = 1_000_000
	bps  = 10_000

	variableFeeDenominator = 1_000_000
)

// clockMarginSeconds: how far the wall clock may run ahead of the block timestamp the hook sees. A pool's fee never
// rises with time while no swap lands, so pricing a second early never charges less than the chain.
const clockMarginSeconds = 1

// Hook gas: V4Quoter.quoteExactInputSingle's gasEstimate on each live Kalos pool less the same quote on the unhooked
// HYPE/USDC 0.3% pool of the same PoolManager at the same block, the mean over both pools of the hook, three sizes each
// (1e-6 to 1e-2 of the input reserve; Elysium blocks ~7.62M, TestLiveHookGas): 64.7k-82.5k with the fee asset in (the
// beforeSwap charge), 72.4k-99.2k with it out (the afterSwap charge, and the LPs' part booked). The hook's fee book
// replays every swap of a pool with a fee asset, which is what is measured; no creator pool or pool without a fee
// asset is live yet: they skip the book and cost less, so these overstate them. The swap's tick crossings are
// counted by the v4 simulator as for any pool.
const (
	gasFeeAssetIn  int64 = 73_500
	gasFeeAssetOut int64 = 88_800
)

var (
	ErrPoolNotRegistered = errors.New("kalos: pool is not registered with the hook or not tracked yet")
	ErrInvalidFeeState   = errors.New("kalos: invalid fee state")
	ErrAmountTooLarge    = errors.New("kalos: amount above int128 max")
	ErrStateMoved        = errors.New("kalos: the pool's price moved between the tracker's read and the hook's")
)
