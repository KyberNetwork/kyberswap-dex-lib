package rangepool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
)

// Extra is the mutable per-pool state serialized into entity.Pool.Extra, refreshed
// each tracker cycle from getRangePoolDynamicData + Vault getPoolConfig (Strategy A:
// authoritative on-chain re-read, no event reconstruction).
//
// It embeds shared.Extra so the Stage-3 simulator can hand extra.Extra straight to
// balancer/v3/base.NewPoolSimulator (which reads DecimalScalingFactors, TokenRates,
// BalancesLiveScaled18, the static/aggregate swap fees and HooksConfig off it). The
// only Range-specific additions are the virtual balances the curve prices on and the
// six liveness flags. HooksConfig is left zero-valued: the RangePoolHook sets no
// dynamic-swap-fee flag and no enableHookAdjustedAmounts, so it cannot alter a quote
// (reference §4).
type Extra struct {
	*shared.Extra

	// VirtualBalances are the concentrated-liquidity curve's virtual reserves
	// (scaled18, token registration order) that swaps are priced on.
	VirtualBalances []*uint256.Int `json:"virtualBalances"`

	// MinimumTradeAmount is the Vault's minimum-trade-amount (scaled18). Range's
	// custom Vault uses a non-canonical value (1e12 on mainnet), enforced at both
	// _ensureValidSwapAmount points; the simulator applies it in OnSwap so it
	// rejects the same dust trades the chain reverts with TradeAmountTooSmall.
	MinimumTradeAmount *uint256.Int `json:"minimumTradeAmount,omitempty"`

	// Six liveness flags from getRangePoolDynamicData. Any "unsafe" one makes the
	// tracker zero the reserves (see pool_tracker.go), which is what actually
	// disables the pool in base.PoolSimulator; they are kept here for observability.
	IsPoolRegistered     bool `json:"isPoolRegistered"`
	IsPoolInitialized    bool `json:"isPoolInitialized"`
	IsPoolPaused         bool `json:"isPoolPaused"`
	IsPoolInRecoveryMode bool `json:"isPoolInRecoveryMode"`
	IsVaultPaused        bool `json:"isVaultPaused"`
	IsHookStopped        bool `json:"isHookStopped"`
}

// StaticExtra is the immutable per-pool data serialized into entity.Pool.StaticExtra,
// written once at discovery from getRangePoolImmutableData. It embeds shared.StaticExtra
// so &staticExtra.StaticExtra can be passed to base.NewPoolSimulator. NormalizedWeights
// and DecimalScalingFactors are immutable, so the tracker copies the scaling factors
// into the (mutable) Extra each cycle rather than re-reading getRangePoolImmutableData.
type StaticExtra struct {
	shared.StaticExtra

	NormalizedWeights     []*uint256.Int `json:"normalizedWeights"`
	DecimalScalingFactors []*uint256.Int `json:"decimalScalingFactors"`
	FactoryAddress        string         `json:"factoryAddress,omitempty"`
}

// isLive reports whether the pool can currently serve swaps: all safety gates clear
// and it is initialized (reference §4).
func (e *Extra) isLive() bool {
	return e.IsPoolRegistered &&
		e.IsPoolInitialized &&
		!e.IsPoolPaused &&
		!e.IsPoolInRecoveryMode &&
		!e.IsVaultPaused &&
		!e.IsHookStopped
}

// --- ABI receiver structs ---------------------------------------------------
//
// getRangePoolDynamicData / getRangePoolImmutableData / getPoolConfig each return a
// single tuple, so go-ethereum's unpacker writes into a struct whose one field is the
// target struct (see rangePoolDynamicDataResult etc.). Field names are PascalCase to
// match the tuple component names.

type rangePoolImmutableDataABI struct {
	Tokens                []common.Address
	DecimalScalingFactors []*big.Int
	NormalizedWeights     []*big.Int
}

type rangePoolDynamicDataABI struct {
	BalancesRaw             []*big.Int
	BalancesLiveScaled18    []*big.Int
	VirtualBalances         []*big.Int
	LeverageRatios          []*big.Int
	TokenRates              []*big.Int
	StaticSwapFeePercentage *big.Int
	TotalSupply             *big.Int
	IsPoolRegistered        bool
	IsPoolInitialized       bool
	IsPoolPaused            bool
	IsPoolInRecoveryMode    bool
	IsVaultPaused           bool
	IsHookStopped           bool
}

type poolConfigABI struct {
	LiquidityManagement struct {
		DisableUnbalancedLiquidity  bool
		EnableAddLiquidityCustom    bool
		EnableRemoveLiquidityCustom bool
		EnableDonation              bool
	}
	StaticSwapFeePercentage     *big.Int
	AggregateSwapFeePercentage  *big.Int
	AggregateYieldFeePercentage *big.Int
	TokenDecimalDiffs           *big.Int
	PauseWindowEndTime          uint32
	IsPoolRegistered            bool
	IsPoolInitialized           bool
	IsPoolPaused                bool
	IsPoolInRecoveryMode        bool
}

// Single-field wrappers required by the go-ethereum ABI unpacker for a single-tuple
// return (cf. unipool's getVirtualReserves wrapper).
type (
	rangePoolImmutableDataResult struct{ Data rangePoolImmutableDataABI }
	rangePoolDynamicDataResult   struct{ Data rangePoolDynamicDataABI }
	poolConfigResult             struct{ PoolConfig poolConfigABI }
)
