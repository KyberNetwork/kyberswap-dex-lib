package umbraedlmm

import "github.com/holiman/uint256"

// StaticExtra holds immutable pair config the tracker/simulator need. scaleX/scaleY are derived
// from the decimals (10^(18-decimals)); reserves are carried in the contract's internal 18-decimal
// normalized space.
type StaticExtra struct {
	BinStep   uint16 `json:"binStep"`
	DecimalsX uint8  `json:"decimalsX"`
	DecimalsY uint8  `json:"decimalsY"`
	// Router is the DLMM Router address — the swap entry point KyberSwap's executor approves and
	// calls. Persisted by the lister from Config.RouterAddress so GetApprovalAddress is deployment-
	// driven rather than hardcoded.
	Router string `json:"router"`
}

// FeeParameters mirrors the DEPLOYED FeeHelper.FeeParameters (7 fields; no protocolShare —
// protocol share is global on the factory). maxVolatilityAccumulator is uint24 on-chain.
type FeeParameters struct {
	BaseFactor               uint16 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint16 `json:"variableFeeControl"`
	MaxVolatilityAccumulator uint32 `json:"maxVolatilityAccumulator"`
	MinSwapBps               uint16 `json:"minSwapBps"`
}

// Bin is one discrete liquidity bin. Reserves are in the 18-decimal normalized space (native
// getBin() reserves multiplied by the token scale factor).
type Bin struct {
	ID       uint32       `json:"id"`
	ReserveX *uint256.Int `json:"reserveX"`
	ReserveY *uint256.Int `json:"reserveY"`
}

func (b Bin) isEmpty() bool {
	return (b.ReserveX == nil || b.ReserveX.IsZero()) && (b.ReserveY == nil || b.ReserveY.IsZero())
}

// Extra is the mutable per-pool state refreshed each block. The dynamic fee depends on the
// volatility accumulator + reference, which the deployed pair exposes via getQuoteState(); the
// tracker decays the accumulator to the tracked block and the simulator ramps it from the reference
// as bins are crossed, matching the deployed quoteSwap().
type Extra struct {
	ActiveID       uint32        `json:"activeId"`
	Bins           []Bin         `json:"bins"` // sorted ascending by ID; normalized reserves
	FeeParameters  FeeParameters `json:"feeParameters"`
	VariableFeeCap uint16        `json:"variableFeeCap"` // from Factory.getVariableFeeCap(binStep, baseFactor)
	// Volatility state for the dynamic-fee ramp. VolatilityAccumulator is already decayed to the
	// tracked block, and VolatilityReference is the reference bin (== activeId when idle).
	VolatilityAccumulator uint64 `json:"volatilityAccumulator"`
	VolatilityReference   uint32 `json:"volatilityReference"`
	// Native (un-normalized) total reserves; feed the min-swap-for-volatility threshold exactly as
	// the deployed _getMinSwapForVolatility does (native sum * minSwapBps / 10000).
	NativeReserveX string `json:"nativeReserveX"`
	NativeReserveY string `json:"nativeReserveY"`
}

// binUpdate records a post-swap bin reserve (normalized) for UpdateBalance to apply by index,
// so UpdateBalance consumes CalcAmountOut's result rather than recomputing it.
type binUpdate struct {
	index    int
	reserveX *uint256.Int
	reserveY *uint256.Int
}

type SwapInfo struct {
	newActiveID uint32
	binUpdates  []binUpdate
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// decimalsResult decodes the pair's getDecimals() two-value return (ABI multi-returns unpack into
// a single struct by position).
type decimalsResult struct {
	DecimalsX uint8
	DecimalsY uint8
}
