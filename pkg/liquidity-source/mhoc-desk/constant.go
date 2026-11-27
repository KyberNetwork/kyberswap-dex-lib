package mhocdesk

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeMhocDesk

	// defaultGas is the gas of one swap() through the approval path. The contract's gas script measures 74,555
	// (sell base) and 74,134 / 74,230 (buy base, first / later swap on a card) on a local anvil chain with plain
	// ERC-20 mocks, receipt gasUsed including the 21k intrinsic. Base's FiatToken USDC proxy and WETH9 cost more per
	// transfer; 90k is an estimate until it is measured with the real tokens. The executor's own overhead is not
	// included.
	defaultGas = 90_000

	// window is the length of the per-minute notional budget: one calendar minute of block.timestamp.
	window = 60
)

// Contract constants, from PropAMM.sol.
var (
	wad         = uint256.NewInt(1e18)
	spreadToWad = uint256.NewInt(1e12) // dBid/dAsk are in 1e-6
	kToWad      = uint256.NewInt(1e6)  // k is 1e-12 per whole QUOTE token
	maxAmountIn = new(uint256.Int).Lsh(uint256.NewInt(1), 128)
)

// One error per revert selector of PropAMM._compute, in the contract's order of checks: paused, stale
// (by time or by block number), zero amount, size (2^128 and the per-swap cap), unknown token, per-card or
// per-block budget, per-minute budget, no fill, floor.
var (
	ErrPoolPaused       = errors.New("mhoc-desk: PoolPaused")
	ErrStalePrice       = errors.New("mhoc-desk: StalePrice")
	ErrZeroAmount       = errors.New("mhoc-desk: ZeroAmount")
	ErrExceedsMaxSize   = errors.New("mhoc-desk: ExceedsMaxSize")
	ErrUnknownToken     = errors.New("mhoc-desk: UnknownToken")
	ErrExceedsUpdateCap = errors.New("mhoc-desk: ExceedsUpdateCap")
	ErrExceedsWindowCap = errors.New("mhoc-desk: ExceedsWindowCap")
	ErrNoFill           = errors.New("mhoc-desk: NoFill")
	ErrBelowFloor       = errors.New("mhoc-desk: BelowFloor")

	// ErrOverflow is never expected: the contract's bounds keep every product below 2^256. It exists so that a
	// state the contract could not produce fails loudly instead of wrapping.
	ErrOverflow = errors.New("mhoc-desk: uint256 overflow")
)
