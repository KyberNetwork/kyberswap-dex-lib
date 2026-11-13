package pools

import (
	"errors"
	"fmt"
	"math/bits"
	"sort"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math/twamm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// MEVCapture v2.1 (https://github.com/EkuboProtocol/evm-contracts/blob/main/src/extensions/MEVCaptureV21.sol) only
// allows swaps through `Core.forward`. Each forwarded swap runs as a sequence of Core swap calls: an uncharged call
// toward a time-decaying anchor, then away-from-anchor calls cut one representable sqrt ratio step before every
// initialized tick and at every surcharge rate boundary, each charged a surcharge on the calculated token through
// `Core.accumulateAsFees`, with every initialized tick crossed by its own uncharged one-step bridge call. The calls
// depend on Core's tick bitmap words, so the pool emulates Core's initialized tick search exactly.

const (
	mevCaptureV21MaxTickSpacing uint32 = 698_605
	mevCaptureV21MaxHalfLife    uint32 = 1 << 20
	mevCaptureV21MaxSlopeK      uint8  = 16
	mevCaptureV21MaxSegmentExp  uint8  = 8
	mevCaptureV21MaxJLin        uint8  = 64
	mevCaptureV21MaxMaxFee      uint64 = 1 << 63
	mevCaptureV21MaxMGate       uint8  = 16
)

var (
	ErrMevCaptureV21InvalidConfig           = errors.New("invalid MEVCapture v2.1 configuration")
	ErrMevCaptureV21InvalidState            = errors.New("invalid MEVCapture v2.1 pool state")
	ErrMevCaptureV21InvalidSqrtRatioLimit   = errors.New("invalid sqrt ratio limit")
	ErrMevCaptureV21SqrtRatioLimitDirection = errors.New("sqrt ratio limit in the wrong direction")
	ErrMevCaptureV21AmountOverflow          = errors.New("amount overflow")
	ErrMevCaptureV21LiquidityOverflow       = errors.New("liquidity overflow")
	ErrMevCaptureV21NoProgress              = errors.New("swap makes no progress")
	ErrMevCaptureV21RefreshRequired         = errors.New("MEVCapture v2.1 pool state must be refreshed from the chain")
)

var (
	zeroInt256 int256.Int
	int128Min  = new(int256.Int).Neg(new(int256.Int).Lsh(int256.NewInt(1), 127))
	int128Max  = new(int256.Int).Sub(new(int256.Int).Lsh(int256.NewInt(1), 127), int256.NewInt(1))
)

type (
	// MevCaptureV21Config is the extension's `getConfig()`; the values are constructor immutables.
	MevCaptureV21Config struct {
		HalfLife   uint32 `json:"halfLife"`
		SlopeK     uint8  `json:"slopeK"`
		SegmentExp uint8  `json:"segmentExp"`
		JLin       uint8  `json:"jLin"`
		MaxFee     uint64 `json:"maxFee"`
		ClampTicks uint32 `json:"clampTicks"`
		MGate      uint8  `json:"mGate"`
	}

	// MevCaptureV21Anchor is the extension's per-pool state, as returned by `getAnchor(poolId)`.
	MevCaptureV21Anchor struct {
		LastUpdateTime uint32 `json:"lastUpdateTime"`
		LRefTime       uint32 `json:"lRefTime"`
		LastPosTime    uint32 `json:"lastPosTime"`
		LRefBits       uint8  `json:"lRefBits"`
		SnapBits       uint8  `json:"snapBits"`
		LRefRaiseTime  uint32 `json:"lRefRaiseTime"`
		AnchorX16      int64  `json:"anchorX16"`
	}

	// MevCaptureV21Gate is the result of the first-touch anchor update of a timestamp; a failed gate emits
	// `AnchorGateFailed(poolId, lObs, eRef)`.
	MevCaptureV21Gate struct {
		Passed bool
		LObs   uint8
		ERef   uint8
	}

	MevCaptureV21PoolSwapState struct {
		SqrtRatio *uint256.Int        `json:"sqrtRatio"`
		Liquidity *uint256.Int        `json:"liquidity"`
		Tick      int32               `json:"tick"`
		Anchor    MevCaptureV21Anchor `json:"anchor"`

		// The block timestamp of the simulated swap that produced this state: swaps simulated after it on the same
		// state execute in the same block, at the same timestamp.
		swapTime uint32
	}

	MevCaptureV21PoolState struct {
		*MevCaptureV21PoolSwapState
		Config MevCaptureV21Config `json:"config"`
		// All initialized ticks within TickBounds, including those whose net liquidity delta is zero.
		SortedTicks []Tick   `json:"sortedTicks"`
		TickBounds  [2]int32 `json:"tickBounds"`
	}

	MevCaptureV21Pool struct {
		key *ConcentratedPoolKey
		*MevCaptureV21PoolState
	}

	// MevCaptureV21Resources counts what one forwarded swap executes, for gas estimation.
	MevCaptureV21Resources struct {
		CoreCalls               uint32
		CoreSteps               uint32
		InitializedTicksCrossed uint32
		Bridges                 uint32
		TickEventSearches       uint32
		AccumulateAsFees        uint32
		AnchorUpdates           uint32
		StateWrites             uint32
	}

	MevCaptureV21CallKind uint8

	mevCaptureV21CoreState struct {
		sqrtRatio uint256.Int
		liquidity uint256.Int
		tick      int32
	}

	// MevCaptureV21CoreCall is one `Core.swap` made by the extension.
	MevCaptureV21CoreCall struct {
		Kind           MevCaptureV21CallKind
		Surcharge      uint64
		SqrtRatioLimit uint256.Int
		Amount         int256.Int
		Delta0         int256.Int
		Delta1         int256.Int
		Surcharge0     uint256.Int
		Surcharge1     uint256.Int
		Before         mevCaptureV21CoreState
		After          mevCaptureV21CoreState
	}

	MevCaptureV21SwapParams struct {
		IsToken1 bool
		// Positive for exact input, negative for exact output.
		Amount *int256.Int
		// Fixed point; must be a valid sqrt ratio.
		SqrtRatioLimit *uint256.Int
		SkipAhead      uint32
	}

	MevCaptureV21Swap struct {
		Gate      *MevCaptureV21Gate
		AnchorX16 int64
		// Only recorded when requested.
		Calls       []MevCaptureV21CoreCall
		Delta0      int256.Int
		Delta1      int256.Int
		Surcharge0  uint256.Int
		Surcharge1  uint256.Int
		FeesPaid    uint256.Int
		StateAfter  mevCaptureV21CoreState
		AnchorAfter MevCaptureV21Anchor
		Resources   MevCaptureV21Resources
	}

	mevCaptureV21CoreSwap struct {
		ticks     []Tick
		spacing   int64
		skipAhead uint32
		fee       uint64
	}

	mevCaptureV21CoreSwapResult struct {
		delta0, delta1 int256.Int
		feesPaid       uint256.Int
		state          mevCaptureV21CoreState
		steps, crossed uint32
	}
)

const (
	MevCaptureV21CallPassThrough MevCaptureV21CallKind = iota
	MevCaptureV21CallToward
	MevCaptureV21CallCharged
	MevCaptureV21CallBridge
)

func (k MevCaptureV21CallKind) String() string {
	switch k {
	case MevCaptureV21CallPassThrough:
		return "passThrough"
	case MevCaptureV21CallToward:
		return "toward"
	case MevCaptureV21CallCharged:
		return "charged"
	case MevCaptureV21CallBridge:
		return "bridge"
	default:
		return "unknown"
	}
}

// Validate mirrors the extension constructor's checks.
func (c MevCaptureV21Config) Validate() error {
	if c.HalfLife == 0 || c.HalfLife > mevCaptureV21MaxHalfLife ||
		c.SlopeK == 0 || c.SlopeK > mevCaptureV21MaxSlopeK ||
		c.SegmentExp > mevCaptureV21MaxSegmentExp ||
		c.JLin == 0 || c.JLin > mevCaptureV21MaxJLin ||
		c.MaxFee == 0 || c.MaxFee > mevCaptureV21MaxMaxFee ||
		c.ClampTicks == 0 || uint64(c.ClampTicks) > 2*uint64(ekubomath.MaxTick) ||
		c.MGate > mevCaptureV21MaxMGate {
		return ErrMevCaptureV21InvalidConfig
	}
	return nil
}

func liquidityBitLength(liquidity *uint256.Int) uint8 {
	return uint8(liquidity.BitLen())
}

func (c *MevCaptureV21Config) decayedReference(state *MevCaptureV21Anchor, now uint32) uint8 {
	decay := (now - state.LRefTime) / c.HalfLife
	if decay >= uint32(state.LRefBits) {
		return 0
	}
	return state.LRefBits - uint8(decay)
}

func (c *MevCaptureV21Config) raise(base, observed uint8, now, raiseTime uint32) (uint8, uint32) {
	if observed > base && now-raiseTime >= c.HalfLife {
		return base + 1, now
	}
	return base, raiseTime
}

// BeforeSwap is the extension's `_anchorUpdate`, run only on the first swap of a timestamp; it returns nil otherwise.
func (c *MevCaptureV21Config) BeforeSwap(
	state *MevCaptureV21Anchor,
	tick int32,
	liquidity *uint256.Int,
	now uint32,
) *MevCaptureV21Gate {
	if state.LastUpdateTime == now {
		return nil
	}

	dt := now - state.LastUpdateTime
	lObs := liquidityBitLength(liquidity)
	if state.LastPosTime == now {
		lObs = state.SnapBits
	}
	eRef := c.decayedReference(state, now)
	passed := uint16(lObs)+uint16(c.MGate) >= uint16(eRef)

	if passed {
		// |anchorX16| and |tick << 16| are below 2^43, so the offset and the moves fit in an int64
		offset := state.AnchorX16 - int64(tick)<<16

		var magnitude, divisor uint256.Int
		if offset < 0 {
			magnitude.SetUint64(uint64(-offset))
		} else {
			magnitude.SetUint64(uint64(offset))
		}
		magnitude.Rsh(&magnitude, uint(dt/c.HalfLife))
		magnitude.Lsh(&magnitude, 64)
		fraction, _ := bits.Div64(uint64(dt%c.HalfLife), 0, uint64(c.HalfLife))
		divisor.SetUint64(fraction)
		magnitude.Div(&magnitude, twamm.Exp2(&divisor))

		decayed := int64(magnitude.Uint64())
		if offset < 0 {
			decayed = -decayed
		}

		// rounded up: every update that consumes elapsed time grants at least one 2^-16 tick
		limit := int64((uint64(c.ClampTicks)<<16*uint64(min(dt, c.HalfLife)) + uint64(c.HalfLife) - 1) /
			uint64(c.HalfLife))
		move := min(max(decayed-offset, -limit), limit)
		state.AnchorX16 += move

		base := max(eRef, min(state.LRefBits, lObs))
		state.LRefBits, state.LRefRaiseTime = c.raise(base, lObs, now, state.LRefRaiseTime)
		state.LRefTime = now
	}
	state.LastUpdateTime = now

	return &MevCaptureV21Gate{Passed: passed, LObs: lObs, ERef: eRef}
}

// AfterSwap is the extension's `_refresh`: it raises the reference from the liquidity after the swap.
func (c *MevCaptureV21Config) AfterSwap(state *MevCaptureV21Anchor, liquidity *uint256.Int, now uint32) {
	if state.LastPosTime == now {
		return
	}
	e := c.decayedReference(state, now)
	raised, raiseTime := c.raise(e, liquidityBitLength(liquidity), now, state.LRefRaiseTime)
	if raised != e {
		state.LRefBits = raised
		state.LRefTime = now
		state.LRefRaiseTime = raiseTime
	}
}

// mevCaptureV21Grid holds the away segment boundaries in mirrored Q16 tick coordinates (the swap always moves toward
// larger values). All values stay below 2^51 in magnitude: the anchor is bounded by the tick range and the tick
// spacing by MAX_TICK_SPACING.
type mevCaptureV21Grid struct {
	w16, g1, gJ int64
	jLin        uint32
}

func newMevCaptureV21Grid(a int64, spacing uint32, config *MevCaptureV21Config) mevCaptureV21Grid {
	w16 := int64(spacing) << (uint(config.SegmentExp) + 16)
	q := a / w16
	if a < 0 && a%w16 != 0 {
		q--
	}
	g1 := (q + 1) * w16
	return mevCaptureV21Grid{
		w16:  w16,
		g1:   g1,
		gJ:   g1 + int64(config.JLin-1)*w16,
		jLin: uint32(config.JLin),
	}
}

func (g *mevCaptureV21Grid) boundary(k uint32) int64 {
	if k <= g.jLin {
		return g.g1 + int64(k-1)*g.w16
	}
	return g.gJ + (int64(1)<<(k-g.jLin+1)-2)*g.w16
}

func (g *mevCaptureV21Grid) firstSegment(p int64) uint32 {
	p16 := p << 16
	if p16 < g.g1 {
		return 1
	} else if p16 < g.gJ {
		return uint32((p16-g.g1)/g.w16 + 2)
	}
	return g.jLin - 1 + uint32(bits.Len64(uint64((p16-g.gJ)/g.w16+2)))
}

// surcharge is `_surcharge`: min(maxFee - poolFee, ceil(slopeK * poolFee * twiceMid / (spacing << 18))).
func (c *MevCaptureV21Config) surcharge(poolFee uint64, twiceMidX16 int64, spacing uint32) uint64 {
	var num, den, tmp uint256.Int
	den.SetUint64(uint64(spacing)).Lsh(&den, 18)
	num.SetUint64(uint64(c.SlopeK))
	num.Mul(&num, tmp.SetUint64(poolFee))
	num.Mul(&num, tmp.SetUint64(uint64(twiceMidX16)))
	num.Add(&num, &den).SubUint64(&num, 1).Div(&num, &den)
	capped := c.MaxFee - poolFee
	if !num.IsUint64() || num.Uint64() > capped {
		return capped
	}
	return num.Uint64()
}

const mevCaptureV21MantissaBits = 94

var (
	mevCaptureV21MantissaMax = new(uint256.Int).SubUint64(big256.U2Pow94, 1)
	mevCaptureV21MantissaMin = new(uint256.Int).Lsh(big256.U1, 62)
)

// toRawSqrtRatio splits a representable fixed point sqrt ratio into its region and mantissa.
func toRawSqrtRatio(fixed *uint256.Int) (uint, uint256.Int) {
	var region uint
	switch {
	case !fixed.Lt(big256.U2Pow160):
		region = 3
	case !fixed.Lt(big256.U2Pow128):
		region = 2
	case !fixed.Lt(big256.U2Pow96):
		region = 1
	}
	var mantissa uint256.Int
	mantissa.Rsh(fixed, 2+32*region)
	return region, mantissa
}

// PredSqrtRatio is `predSqrtRatio`: the largest valid sqrt ratio strictly below x.
func PredSqrtRatio(x *uint256.Int) *uint256.Int {
	region, mantissa := toRawSqrtRatio(x)
	if region != 0 && mantissa.Eq(mevCaptureV21MantissaMin) {
		region--
		mantissa.Set(mevCaptureV21MantissaMax)
	} else {
		mantissa.SubUint64(&mantissa, 1)
	}
	return mantissa.Lsh(&mantissa, 2+32*region)
}

// SuccSqrtRatio is `succSqrtRatio`: the smallest valid sqrt ratio strictly above x.
func SuccSqrtRatio(x *uint256.Int) *uint256.Int {
	region, mantissa := toRawSqrtRatio(x)
	if region != 3 && mantissa.Eq(mevCaptureV21MantissaMax) {
		region++
		mantissa.Set(mevCaptureV21MantissaMin)
	} else {
		mantissa.AddUint64(&mantissa, 1)
	}
	return mantissa.Lsh(&mantissa, 2+32*region)
}

// SqrtRatioFromRaw converts a raw contract SqrtRatio (region << 94 | mantissa) to its fixed point value.
func SqrtRatioFromRaw(raw *uint256.Int) *uint256.Int {
	return ekubomath.FloatSqrtRatioToFixed(raw)
}

// SqrtRatioToRaw converts a representable fixed point sqrt ratio to the raw contract SqrtRatio.
func SqrtRatioToRaw(fixed *uint256.Int) *uint256.Int {
	region, mantissa := toRawSqrtRatio(fixed)
	var regionBits uint256.Int
	regionBits.SetUint64(uint64(region)).Lsh(&regionBits, mevCaptureV21MantissaBits)
	return mantissa.Or(&mantissa, &regionBits)
}

const tickBitmapStorageOffsetI64 = int64(tickBitmapStorageOffset)

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// raw is `tickToBitmapWordAndIndex` as one index (rounding toward negative infinity).
func (c *mevCaptureV21CoreSwap) raw(tick int64) int64 {
	return floorDiv(tick, c.spacing) + tickBitmapStorageOffsetI64
}

func (c *mevCaptureV21CoreSwap) tickOf(raw int64) int64 {
	return (raw - tickBitmapStorageOffsetI64) * c.spacing
}

// next is Core's `findNextInitializedTick`; it returns the tick and the index of the initialized tick, or -1.
func (c *mevCaptureV21CoreSwap) next(from int32) (int32, int) {
	next := int64(from)
	skip := c.skipAhead
	for {
		raw := c.raw(next + c.spacing)
		wordEnd := raw>>8<<8 + 255
		i := sort.Search(len(c.ticks), func(i int) bool { return c.raw(int64(c.ticks[i].Number)) >= raw })
		if i < len(c.ticks) && c.raw(int64(c.ticks[i].Number)) <= wordEnd {
			return c.ticks[i].Number, i
		}
		next = c.tickOf(wordEnd)
		if next >= int64(ekubomath.MaxTick) {
			return ekubomath.MaxTick, -1
		}
		if skip == 0 {
			return int32(next), -1
		}
		skip--
	}
}

// prev is Core's `findPrevInitializedTick`.
func (c *mevCaptureV21CoreSwap) prev(from int32) (int32, int) {
	prev := int64(from)
	skip := c.skipAhead
	for {
		raw := c.raw(prev)
		wordStart := raw >> 8 << 8
		i := sort.Search(len(c.ticks), func(i int) bool { return c.raw(int64(c.ticks[i].Number)) > raw })
		if i > 0 && c.raw(int64(c.ticks[i-1].Number)) >= wordStart {
			return c.ticks[i-1].Number, i - 1
		}
		prev = c.tickOf(wordStart)
		if prev <= int64(ekubomath.MinTick) {
			return ekubomath.MinTick, -1
		}
		if skip == 0 {
			return int32(prev), -1
		}
		skip--
		prev--
	}
}

func (c *mevCaptureV21CoreSwap) search(from int32, increasing bool) (int32, int) {
	if increasing {
		return c.next(from)
	}
	return c.prev(from)
}

func toSqrtRatioChecked(tick int64) (*uint256.Int, error) {
	if tick < int64(ekubomath.MinTick) || tick > int64(ekubomath.MaxTick) {
		return nil, fmt.Errorf("invalid tick %d", tick)
	}
	return ekubomath.ToSqrtRatio(int32(tick)), nil
}

func nearerSqrtRatio(x, y *uint256.Int, increasing bool) *uint256.Int {
	if (x.Lt(y)) == increasing {
		return x
	}
	return y
}

// addLiquidityDelta adds a signed liquidity delta to a uint128 liquidity, failing on overflow and underflow.
func addLiquidityDelta(liquidity *uint256.Int, delta *int256.Int, negate bool) error {
	var d int256.Int
	d.Set(delta)
	if negate {
		if d.Eq(int128Min) {
			return ErrMevCaptureV21LiquidityOverflow
		}
		d.Neg(&d)
	}
	var abs uint256.Int
	if d.Sign() < 0 {
		abs.Neg((*uint256.Int)(&d))
		if _, underflow := liquidity.SubOverflow(liquidity, &abs); underflow {
			return ErrMevCaptureV21LiquidityOverflow
		}
		return nil
	}
	liquidity.Add(liquidity, (*uint256.Int)(&d))
	if liquidity.BitLen() > 128 {
		return ErrMevCaptureV21LiquidityOverflow
	}
	return nil
}

func isInt128(x *int256.Int) bool {
	return !x.Lt(int128Min) && !x.Gt(int128Max)
}

// swap is `Core.swap` for a concentrated pool, with the exact rounding and tick search of the current Core.
func (c *mevCaptureV21CoreSwap) swap(
	state *mevCaptureV21CoreState,
	isToken1 bool,
	amount *int256.Int,
	limit *uint256.Int,
	result *mevCaptureV21CoreSwapResult,
) error {
	if !ekubomath.IsValidSqrtRatio(limit) {
		return ErrMevCaptureV21InvalidSqrtRatioLimit
	}
	*result = mevCaptureV21CoreSwapResult{state: *state}
	if amount.IsZero() || state.sqrtRatio.Eq(limit) {
		return nil
	}
	exactOut := amount.Sign() < 0
	increasing := isToken1 != exactOut
	if limit.Lt(&state.sqrtRatio) == increasing {
		return ErrMevCaptureV21SqrtRatioLimitDirection
	}

	cur := &result.state
	var remaining int256.Int
	var calculated uint256.Int
	remaining.Set(amount)
	for {
		nextTick, initialized := c.search(cur.tick, increasing)
		nextRatio := ekubomath.ToSqrtRatio(nextTick)
		limited := nearerSqrtRatio(nextRatio, limit, increasing)

		step, err := ekubomath.ComputeStep(&cur.sqrtRatio, &cur.liquidity, limited,
			(*uint256.Int)(&remaining), isToken1, c.fee)
		if err != nil {
			return fmt.Errorf("swap step computation: %w", err)
		}
		remaining.Sub(&remaining, (*int256.Int)(step.ConsumedAmount))
		calculated.Add(&calculated, step.CalculatedAmount)
		if calculated.BitLen() > 128 {
			return ErrMevCaptureV21AmountOverflow
		}
		result.feesPaid.Add(&result.feesPaid, step.FeeAmount)
		if result.feesPaid.BitLen() > 128 {
			result.feesPaid.Set(big256.UMaxU128)
		}
		result.steps++

		if step.SqrtRatioNext.Eq(nextRatio) {
			cur.sqrtRatio.Set(nextRatio)
			cur.tick = nextTick
			if !increasing {
				cur.tick--
			}
			if initialized >= 0 {
				if err := addLiquidityDelta(&cur.liquidity, c.ticks[initialized].LiquidityDelta,
					!increasing); err != nil {
					return err
				}
				result.crossed++
			}
		} else if !step.SqrtRatioNext.Eq(&cur.sqrtRatio) {
			cur.sqrtRatio.Set(step.SqrtRatioNext)
			cur.tick = ekubomath.SqrtRatioToTick(&cur.sqrtRatio)
		}

		if remaining.IsZero() || cur.sqrtRatio.Eq(limit) {
			break
		}
	}

	var specified, calc int256.Int
	specified.Sub(amount, &remaining)
	if exactOut {
		if calculated.BitLen() > 127 {
			return ErrMevCaptureV21AmountOverflow
		}
		calc.Set((*int256.Int)(&calculated))
	} else if calculated.BitLen() > 127 {
		calc.Set(int128Min)
	} else {
		calc.Neg((*int256.Int)(&calculated))
	}
	if isToken1 {
		result.delta0, result.delta1 = calc, specified
	} else {
		result.delta0, result.delta1 = specified, calc
	}
	return nil
}

func (s *mevCaptureV21CoreState) SqrtRatio() *uint256.Int { return &s.sqrtRatio }
func (s *mevCaptureV21CoreState) Liquidity() *uint256.Int { return &s.liquidity }
func (s *mevCaptureV21CoreState) Tick() int32             { return s.tick }

func (s *mevCaptureV21CoreState) eq(o *mevCaptureV21CoreState) bool {
	return s.tick == o.tick && s.sqrtRatio.Eq(&o.sqrtRatio) && s.liquidity.Eq(&o.liquidity)
}

// SwapThroughSegments is the extension's `_swapThroughSegments` with a given anchor (no anchor update or refresh).
func (p *MevCaptureV21Pool) SwapThroughSegments(
	core *mevCaptureV21CoreState,
	params *MevCaptureV21SwapParams,
	anchorX16 int64,
	recordCalls bool,
) (*MevCaptureV21Swap, error) {
	exactOut := params.Amount.Sign() < 0
	increasing := params.IsToken1 != exactOut
	userLimit := params.SqrtRatioLimit
	if !ekubomath.IsValidSqrtRatio(userLimit) {
		return nil, ErrMevCaptureV21InvalidSqrtRatioLimit
	}
	config := &p.Config
	poolFee := p.key.Config.Fee
	spacing := p.key.Config.TypeConfig.TickSpacing
	coreSwap := mevCaptureV21CoreSwap{
		ticks:     p.SortedTicks,
		spacing:   int64(spacing),
		skipAhead: params.SkipAhead,
		fee:       poolFee,
	}

	swap := &MevCaptureV21Swap{AnchorX16: anchorX16, StateAfter: *core}
	current := &swap.StateAfter
	var remaining int256.Int
	remaining.Set(params.Amount)
	resources := &swap.Resources
	var sum0, sum1 int256.Int
	var r mevCaptureV21CoreSwapResult

	call := func(kind MevCaptureV21CallKind, surcharge uint64, limit *uint256.Int) error {
		if err := coreSwap.swap(current, params.IsToken1, &remaining, limit, &r); err != nil {
			return err
		}
		var fee uint256.Int
		if kind == MevCaptureV21CallCharged && surcharge != 0 {
			calculated := &r.delta1
			if params.IsToken1 {
				calculated = &r.delta0
			}
			if exactOut {
				input := (*uint256.Int)(calculated)
				var inputNet uint256.Int
				inputNet.Sub(input, ekubomath.ComputeFee(input, poolFee))
				gross, err := ekubomath.AmountBeforeFee(&inputNet, surcharge)
				if err != nil {
					return ErrMevCaptureV21AmountOverflow
				}
				fee.Sub(gross, &inputNet)
			} else {
				var output uint256.Int
				output.Neg((*uint256.Int)(calculated))
				fee.Set(ekubomath.ComputeFee(&output, surcharge))
			}
		}
		specified := &r.delta0
		if params.IsToken1 {
			specified = &r.delta1
		}
		progressed := !r.state.eq(current) || !specified.IsZero()

		feeSigned := (*int256.Int)(&fee)
		fee0, fee1 := &zeroInt256, feeSigned
		if params.IsToken1 {
			fee0, fee1 = feeSigned, &zeroInt256
			swap.Surcharge0.Add(&swap.Surcharge0, &fee)
		} else {
			swap.Surcharge1.Add(&swap.Surcharge1, &fee)
		}
		for _, add := range [...]struct{ sum, delta, fee *int256.Int }{
			{&sum0, &r.delta0, fee0},
			{&sum1, &r.delta1, fee1},
		} {
			if !isInt128(add.fee) {
				return ErrMevCaptureV21AmountOverflow
			}
			if add.sum.Add(add.sum, add.delta); !isInt128(add.sum) {
				return ErrMevCaptureV21AmountOverflow
			}
			if add.sum.Add(add.sum, add.fee); !isInt128(add.sum) {
				return ErrMevCaptureV21AmountOverflow
			}
		}
		swap.FeesPaid.Add(&swap.FeesPaid, &r.feesPaid)
		if swap.FeesPaid.BitLen() > 128 {
			swap.FeesPaid.Set(big256.UMaxU128)
		}
		resources.CoreCalls++
		resources.CoreSteps += r.steps
		resources.InitializedTicksCrossed += r.crossed
		if kind == MevCaptureV21CallBridge {
			resources.Bridges++
		}
		if !fee.IsZero() {
			resources.AccumulateAsFees++
		}

		if recordCalls {
			c := MevCaptureV21CoreCall{
				Kind:   kind,
				Amount: remaining,
				Delta0: r.delta0,
				Delta1: r.delta1,
				Before: *current,
				After:  r.state,
			}
			if kind == MevCaptureV21CallCharged {
				c.Surcharge = surcharge
			}
			c.SqrtRatioLimit.Set(limit)
			if params.IsToken1 {
				c.Surcharge0 = fee
			} else {
				c.Surcharge1 = fee
			}
			swap.Calls = append(swap.Calls, c)
		}

		remaining.Sub(&remaining, specified)
		*current = r.state
		if !progressed && !remaining.IsZero() && !current.sqrtRatio.Eq(userLimit) {
			return ErrMevCaptureV21NoProgress
		}
		return nil
	}

	if remaining.IsZero() || current.sqrtRatio.Eq(userLimit) {
		if err := call(MevCaptureV21CallPassThrough, 0, userLimit); err != nil {
			return nil, err
		}
	} else {
		a := anchorX16
		if !increasing {
			a = -a
		}
		done := false

		anchorTick := a >> 16
		if !increasing {
			anchorTick = -anchorTick
		}
		anchorTick = min(max(anchorTick, int64(ekubomath.MinTick)), int64(ekubomath.MaxTick))
		anchorRatio := ekubomath.ToSqrtRatio(int32(anchorTick))
		if (increasing && current.sqrtRatio.Lt(anchorRatio)) || (!increasing && current.sqrtRatio.Gt(anchorRatio)) {
			limit := nearerSqrtRatio(anchorRatio, userLimit, increasing)
			if err := call(MevCaptureV21CallToward, 0, limit); err != nil {
				return nil, err
			}
			done = remaining.IsZero() || limit.Eq(userLimit) || !current.sqrtRatio.Eq(limit)
		}

		if !done {
			grid := newMevCaptureV21Grid(a, spacing, config)
			pos := int64(current.tick)
			if !increasing {
				pos = -pos - 1
			}
			k := grid.firstSegment(pos)
			lo := a
			if k != 1 {
				lo = grid.boundary(k - 1)
			}
			capped := config.MaxFee - poolFee
			for {
				hi := grid.boundary(k)
				sur := config.surcharge(poolFee, lo+hi-2*a, spacing)
				limit := userLimit
				if sur != capped {
					if hiTick := hi >> 16; hiTick <= int64(ekubomath.MaxTick) {
						t := hiTick
						if !increasing {
							t = -hiTick
						}
						boundary, err := toSqrtRatioChecked(t)
						if err != nil {
							return nil, err
						}
						if (increasing && !boundary.Gt(&current.sqrtRatio)) ||
							(!increasing && !boundary.Lt(&current.sqrtRatio)) {
							lo = hi
							k++
							continue
						}
						limit = nearerSqrtRatio(boundary, limit, increasing)
					}
				}

				eventTick, initialized := coreSwap.search(current.tick, increasing)
				resources.TickEventSearches++
				eventRatio := ekubomath.ToSqrtRatio(eventTick)
				if eventRatio.Eq(&current.sqrtRatio) {
					from := eventTick
					if !increasing {
						from--
					}
					eventTick, initialized = coreSwap.search(from, increasing)
					resources.TickEventSearches++
					eventRatio = ekubomath.ToSqrtRatio(eventTick)
				}

				kind := MevCaptureV21CallCharged
				if initialized >= 0 {
					var cut *uint256.Int
					if increasing {
						cut = PredSqrtRatio(eventRatio)
					} else {
						cut = SuccSqrtRatio(eventRatio)
					}
					if cut.Eq(&current.sqrtRatio) {
						limit = nearerSqrtRatio(eventRatio, userLimit, increasing)
						kind = MevCaptureV21CallBridge
					} else {
						limit = nearerSqrtRatio(cut, limit, increasing)
					}
				} else {
					limit = nearerSqrtRatio(eventRatio, limit, increasing)
				}

				if err := call(kind, sur, limit); err != nil {
					return nil, err
				}
				if remaining.IsZero() || current.sqrtRatio.Eq(userLimit) {
					break
				}
			}
		}
	}

	swap.Delta0, swap.Delta1 = sum0, sum1
	return swap, nil
}

// SimulateSwap simulates `forward` exactly: the first-touch anchor update, the toward call, the away state machine
// and the post-swap refresh.
func (p *MevCaptureV21Pool) SimulateSwap(
	swapState *MevCaptureV21PoolSwapState,
	params *MevCaptureV21SwapParams,
	now uint32,
	recordCalls bool,
) (*MevCaptureV21Swap, error) {
	var core mevCaptureV21CoreState
	core.sqrtRatio.Set(swapState.SqrtRatio)
	core.liquidity.Set(swapState.Liquidity)
	core.tick = swapState.Tick

	anchor := swapState.Anchor
	gate := p.Config.BeforeSwap(&anchor, core.tick, &core.liquidity, now)
	swap, err := p.SwapThroughSegments(&core, params, anchor.AnchorX16, recordCalls)
	if err != nil {
		return nil, err
	}
	p.Config.AfterSwap(&anchor, &swap.StateAfter.liquidity, now)
	swap.Gate = gate
	if gate != nil {
		swap.Resources.AnchorUpdates = 1
	}
	if anchor != swapState.Anchor {
		swap.Resources.StateWrites = 1
	}
	swap.AnchorAfter = anchor
	return swap, nil
}

// priceLimit is the sqrt ratio limit the quote and the executed swap use: the edge of the range whose initialized
// ticks are known, so that a quote never depends on ticks outside it.
func (p *MevCaptureV21Pool) priceLimit(increasing bool) *uint256.Int {
	if increasing {
		return ekubomath.ToSqrtRatio(p.TickBounds[1])
	}
	return ekubomath.ToSqrtRatio(p.TickBounds[0])
}

func (p *MevCaptureV21Pool) quoteAt(amount *uint256.Int, isToken1 bool, now uint32) (*quoting.Quote, error) {
	signedAmount := (*int256.Int)(amount)
	if !isInt128(signedAmount) || signedAmount.Eq(int128Min) {
		return nil, ErrMevCaptureV21AmountOverflow
	}

	increasing := ekubomath.IsPriceIncreasing(amount, isToken1)
	limit := p.priceLimit(increasing)
	swap, err := p.SimulateSwap(p.MevCaptureV21PoolSwapState, &MevCaptureV21SwapParams{
		IsToken1:       isToken1,
		Amount:         signedAmount,
		SqrtRatioLimit: limit,
	}, now, false)
	if err != nil {
		return nil, err
	}

	specified, calculated := &swap.Delta0, &swap.Delta1
	if isToken1 {
		specified, calculated = calculated, specified
	}
	var consumed, calculatedAbs uint256.Int
	consumed.Set((*uint256.Int)(specified))
	calculatedAbs.Abs((*uint256.Int)(calculated))

	after := &swap.StateAfter
	return &quoting.Quote{
		ConsumedAmount:   &consumed,
		CalculatedAmount: &calculatedAbs,
		FeesPaid:         &swap.FeesPaid,
		Gas:              mevCaptureV21Gas(&swap.Resources),
		SwapInfo: quoting.SwapInfo{
			SkipAhead:  0,
			IsToken1:   isToken1,
			PriceLimit: ekubomath.FixedSqrtRatioToFloat(limit, false),
			Forward:    &p.key.Config.Extension,
			SwapStateAfter: &MevCaptureV21PoolSwapState{
				SqrtRatio: after.sqrtRatio.Clone(),
				Liquidity: after.liquidity.Clone(),
				Tick:      after.tick,
				Anchor:    swap.AnchorAfter,
				swapTime:  now,
			},
			TickSpacingsCrossed: ekubomath.ApproximateNumberOfTickSpacingsCrossed(p.SqrtRatio, &after.sqrtRatio,
				p.key.Config.TypeConfig.TickSpacing),
		},
	}, nil
}

func mevCaptureV21Gas(r *MevCaptureV21Resources) int64 {
	return quoting.BaseGasCostOfOneMevCaptureV21Swap +
		quoting.GasCostOfOneMevCaptureV21CoreCall*int64(r.CoreCalls) +
		quoting.GasCostOfOneMevCaptureV21ExtraCoreStep*int64(r.CoreSteps-min(r.CoreSteps, r.CoreCalls)) +
		quoting.GasCostOfOneMevCaptureV21TickCrossed*int64(r.InitializedTicksCrossed) +
		quoting.GasCostOfOneMevCaptureV21FeeAccumulation*int64(r.AccumulateAsFees) +
		quoting.GasCostOfMevCaptureV21AnchorUpdate*int64(r.AnchorUpdates)
}

// nextSwapTime is the block timestamp the next swap executes at: the timestamp of a swap already simulated on this
// state (same transaction), otherwise the estimated timestamp of the next block.
func (p *MevCaptureV21Pool) nextSwapTime() uint32 {
	if p.swapTime != 0 {
		return p.swapTime
	}
	return uint32(estimatedBlockTimestamp())
}

func (p *MevCaptureV21Pool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteAt(amount, isToken1, p.nextSwapTime())
}

func (p *MevCaptureV21Pool) GetKey() IPoolKey {
	return p.key
}

func (p *MevCaptureV21Pool) GetState() any {
	return p.MevCaptureV21PoolState
}

// CloneSwapStateOnly copies the state struct whose swap state SetSwapState replaces; swap states are never modified in
// place.
func (p *MevCaptureV21Pool) CloneSwapStateOnly() Pool {
	cloned := *p
	clonedState := *p.MevCaptureV21PoolState
	cloned.MevCaptureV21PoolState = &clonedState
	return &cloned
}

func (p *MevCaptureV21Pool) SetSwapState(state quoting.SwapState) {
	p.MevCaptureV21PoolSwapState = state.(*MevCaptureV21PoolSwapState)
}

// ApplyEvent never applies an event locally: the extension updates its anchor state without emitting it, so the
// pool state is refetched on any event of the pool.
func (p *MevCaptureV21Pool) ApplyEvent(Event, []byte, uint64) error {
	return ErrMevCaptureV21RefreshRequired
}

func (p *MevCaptureV21Pool) NewBlock() {}

func (p *MevCaptureV21Pool) CalcBalances() ([]uint256.Int, error) {
	ticks := make([]Tick, len(p.SortedTicks))
	for i, tick := range p.SortedTicks {
		ticks[i] = Tick{Number: tick.Number, LiquidityDelta: tick.LiquidityDelta.Clone()}
	}
	state := NewConcentratedPoolState(
		NewConcentratedPoolSwapState(p.SqrtRatio.Clone(), p.Liquidity.Clone(), 0),
		ticks,
		p.TickBounds,
		p.Tick,
	)
	state.AddLiquidityCutoffs()
	return state.CalcBalances()
}

// Validate checks that the state can be quoted: a valid configuration and price, ticks sorted and within the
// bounds, and an anchor within the tick range (which the extension guarantees).
func (s *MevCaptureV21PoolState) Validate(key *ConcentratedPoolKey) error {
	if err := s.Config.Validate(); err != nil {
		return err
	}
	if s.MevCaptureV21PoolSwapState == nil || s.SqrtRatio == nil || s.Liquidity == nil {
		return ErrMevCaptureV21InvalidState
	}
	spacing := key.Config.TypeConfig.TickSpacing
	fee := key.Config.Fee
	if key.Config.Extension == (common.Address{}) || fee == 0 || fee >= s.Config.MaxFee ||
		spacing == 0 || spacing > mevCaptureV21MaxTickSpacing {
		return ErrMevCaptureV21InvalidState
	}
	if !ekubomath.IsValidSqrtRatio(s.SqrtRatio) || s.Liquidity.BitLen() > 128 ||
		s.Tick < ekubomath.MinTick || s.Tick > ekubomath.MaxTick ||
		ekubomath.ToSqrtRatio(s.Tick).Gt(s.SqrtRatio) ||
		(s.Tick < ekubomath.MaxTick && ekubomath.ToSqrtRatio(s.Tick+1).Lt(s.SqrtRatio)) {
		return ErrMevCaptureV21InvalidState
	}
	if s.TickBounds[0] < ekubomath.MinTick || s.TickBounds[1] > ekubomath.MaxTick ||
		s.TickBounds[0] > s.Tick || s.TickBounds[1] < s.Tick {
		return ErrMevCaptureV21InvalidState
	}
	for i, tick := range s.SortedTicks {
		if tick.LiquidityDelta == nil || !isInt128(tick.LiquidityDelta) ||
			tick.Number < s.TickBounds[0] || tick.Number > s.TickBounds[1] ||
			tick.Number%int32(spacing) != 0 || (i > 0 && s.SortedTicks[i-1].Number >= tick.Number) {
			return ErrMevCaptureV21InvalidState
		}
	}
	maxAnchor := int64(ekubomath.MaxTick) << 16
	if s.Anchor.AnchorX16 < -maxAnchor-(1<<16) || s.Anchor.AnchorX16 > maxAnchor+(1<<16) {
		return ErrMevCaptureV21InvalidState
	}
	return nil
}

func NewMevCaptureV21PoolState(
	swapState *MevCaptureV21PoolSwapState,
	config MevCaptureV21Config,
	sortedTicks []Tick,
	tickBounds [2]int32,
) *MevCaptureV21PoolState {
	return &MevCaptureV21PoolState{
		MevCaptureV21PoolSwapState: swapState,
		Config:                     config,
		SortedTicks:                sortedTicks,
		TickBounds:                 tickBounds,
	}
}

func NewMevCaptureV21Pool(key *ConcentratedPoolKey, state *MevCaptureV21PoolState) (*MevCaptureV21Pool, error) {
	if err := state.Validate(key); err != nil {
		return nil, err
	}
	return &MevCaptureV21Pool{key: key, MevCaptureV21PoolState: state}, nil
}
