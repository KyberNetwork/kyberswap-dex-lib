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
// reductionFactor and minSwapBps are carried for ABI fidelity but are DEAD in the V2 swap path:
// nothing reads reductionFactor, and the V1 min-swap volatility gate was removed.
type FeeParameters struct {
	BaseFactor               uint16 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint16 `json:"variableFeeControl"`
	MaxVolatilityAccumulator uint32 `json:"maxVolatilityAccumulator"`
	MinSwapBps               uint16 `json:"minSwapBps"`
}

// Bin is one discrete liquidity bin. Reserves are in the 18-decimal normalized space (the pair
// stores them normalized; the viewer reports native, so the tracker scales back up).
type Bin struct {
	ID       uint32       `json:"id"`
	ReserveX *uint256.Int `json:"reserveX"`
	ReserveY *uint256.Int `json:"reserveY"`
}

// Extra is the mutable per-pool state refreshed each block. The dynamic fee depends on the
// anchored-displacement volatility model (V2 R12/M-3): the accumulator is the displacement from a
// per-window anchor bin (volatilityReference), decayed only once the filter window has lapsed. The
// tracker stores the raw on-chain state plus the tracked timestamp; the simulator derives the
// working volatility exactly as the deployed quote path does.
type Extra struct {
	ActiveID       uint32        `json:"activeId"`
	Bins           []Bin         `json:"bins"` // sorted ascending by ID; normalized reserves
	FeeParameters  FeeParameters `json:"feeParameters"`
	VariableFeeCap uint16        `json:"variableFeeCap"` // from Factory.getVariableFeeCap(binStep, baseFactor)
	// Raw volatility state from getQuoteState(), NOT pre-decayed: the filter-window decision
	// (decay vs anchor reuse) must be made at quote time against the same clock.
	VolatilityAccumulator uint64 `json:"volatilityAccumulator"`
	VolatilityReference   uint32 `json:"volatilityReference"` // anchor bin id; < 2^22 means unset
	LastVolatilityUpdate  uint64 `json:"lastVolatilityUpdate"`
	// Timestamp the state was tracked at; the simulator's clock for decay/window decisions. The
	// quote is a snapshot — fees drift as the accumulator decays between tracking and execution
	// (same caveat as the DAMM package's currentFeeBps snapshot).
	Timestamp uint64 `json:"timestamp"`
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
	// Volatility persistence mirrors the pair: the accumulator/anchor/clock are only written when
	// at least one bin was crossed.
	binsCrossed bool
	newVolAcc   uint64
	newVolRef   uint32
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
	BinStep         uint16 `json:"binStep"`
}

// decimalsResult decodes the pair's getDecimals() two-value return (ABI multi-returns unpack into
// a single struct by position).
type decimalsResult struct {
	DecimalsX uint8
	DecimalsY uint8
}
