package volatile

import (
	"github.com/holiman/uint256"
)

// Port of VolatileFeeCalculation.sol, RelativeMoveLibrary.sol and the fee path of VolatilePairHook.sol
// (impl 0x1ddbf7215be4f8a7623b9b1b0255c6e1e3f6c000). All *E6 values are uint24 on-chain, held as uint32.

const (
	oneE6    = 1_000_000
	maxFeeE6 = oneE6 - 2
	q96Shift = 96
	q48Shift = 48
)

var q96 = *new(uint256.Int).Lsh(uint256.NewInt(1), q96Shift)

// FeeConfig is the hook's feeConfig(poolId).
type FeeConfig struct {
	RestingSpread          uint32 `json:"rs"`
	BackrunCaptureFraction uint32 `json:"bc"`
	MidFadeRate            uint32 `json:"mr"`
	MaxMidFade             uint32 `json:"mm"`
	SkewRate               uint32 `json:"sr"`
	MaxSkewStep            uint32 `json:"ss"`
	MaxSkew                uint32 `json:"ms"`
	BackrunThreshold       uint32 `json:"bt"`
	RestoreFraction        uint32 `json:"rf"`
	RestoreThreshold       uint32 `json:"rt"`
}

// FeeState is the hook's feeState(poolId) minus blockNumber: off-chain a swap always runs in a later
// block than the tracked one, so the top-of-block advance always applies once (see Hook.advanced).
type FeeState struct {
	SqrtPriceTop uint256.Int `json:"p"`
	AskPermFee   uint32      `json:"a"`
	TempFee      uint32      `json:"t,omitempty"`
	TempOnBid    bool        `json:"b,omitempty"`
}

// relativeMove is |P_new - P_ref| / P_ref as an exact ratio (num < 2**96, den <= 2**96) plus its direction.
type relativeMove struct {
	num, den  uint256.Int
	increased bool
}

// relMove ports VolatileFeeCalculation.relativeMove. ref must be nonzero.
func relMove(m *relativeMove, newP, ref *uint256.Int) {
	m.den.Set(&q96)
	if newP.Eq(ref) {
		m.num.Clear()
		m.increased = false
		return
	}
	// Q48 sqrt ratio of smaller/larger, rounded down; ref*2**48 < 2**208 so no overflow.
	m.increased = newP.Gt(ref)
	small, large := newP, ref
	if m.increased {
		small, large = ref, newP
	}
	var r uint256.Int
	r.Lsh(small, q48Shift)
	r.Div(&r, large)
	r.Mul(&r, &r) // ratioX96 <= 2**96
	if r.IsZero() {
		r.SetOne() // extreme up-move: clamp, the resulting fee is capped downstream
	}
	m.num.Sub(&m.den, &r)
	if m.increased {
		m.den.Set(&r)
	}
}

// mulUp is rateE6 * move rounded up. rate < 2**24 and num < 2**96, so the product fits.
func (m *relativeMove) mulUp(res *uint256.Int, rateE6 uint32) *uint256.Int {
	var rem uint256.Int
	res.SetUint64(uint64(rateE6))
	res.Mul(res, &m.num)
	res.DivMod(res, &m.den, &rem)
	if !rem.IsZero() {
		res.AddUint64(res, 1)
	}
	return res
}

// gte is move >= thresholdE6 pips, exact. num*1e6 < 2**116.
func (m *relativeMove) gte(thresholdE6 uint32) bool {
	var l, r uint256.Int
	l.SetUint64(oneE6)
	l.Mul(&l, &m.num)
	r.SetUint64(uint64(thresholdE6))
	r.Mul(&r, &m.den)
	return l.Cmp(&r) >= 0
}

func clampFee(x *uint256.Int) uint32 {
	if x.GtUint64(maxFeeE6) {
		return maxFeeE6
	}
	return uint32(x.Uint64())
}

// newPermanentFee ports VolatileFeeCalculation.newPermanentFee.
func (c *FeeConfig) newPermanentFee(oldPerm uint32, move *relativeMove) uint32 {
	var step uint256.Int
	move.mulUp(&step, c.SkewRate)
	if step.GtUint64(uint64(c.MaxSkewStep)) {
		step.SetUint64(uint64(c.MaxSkewStep))
	}
	return min(oldPerm+uint32(step.Uint64()), c.MaxSkew)
}

// temporaryBackrunFee ports VolatileFeeCalculation.temporaryBackrunFee.
func (c *FeeConfig) temporaryBackrunFee(newLean, oldLean uint32, move *relativeMove) uint32 {
	var temp uint32
	if move.gte(c.RestoreThreshold) {
		// newLean >= oldLean (config bounds), delta * fraction < 2**44, mulDivUp by 1e6.
		temp = uint32((uint64(newLean-oldLean)*uint64(c.RestoreFraction) + oneE6 - 1) / oneE6)
	}
	if move.gte(c.BackrunThreshold) {
		var backrun uint256.Int
		move.mulUp(&backrun, c.BackrunCaptureFraction)
		backrun.AddUint64(&backrun, uint64(temp))
		temp = clampFee(&backrun)
	}
	return temp
}

// advance ports _loadTopOfBlockState's new-block branch: steps the permanent skew, prices the one-block
// backrun fee from the move since the stored block-open price, and sets the block-open price to cur.
func (c *FeeConfig) advance(s *FeeState, cur *uint256.Int) {
	if !s.SqrtPriceTop.IsZero() {
		var move relativeMove
		relMove(&move, cur, &s.SqrtPriceTop)
		// An up move steps the ask perm fee up and puts the temp fee on the bid; a down move reverses.
		s.TempOnBid = move.increased
		oldLean := c.RestingSpread - s.AskPermFee
		if s.TempOnBid {
			oldLean = s.AskPermFee
		}
		newLean := c.newPermanentFee(oldLean, &move)
		s.TempFee = c.temporaryBackrunFee(newLean, oldLean, &move)
		if s.TempOnBid {
			s.AskPermFee = newLean
		} else {
			s.AskPermFee = c.RestingSpread - newLean
		}
	}
	s.SqrtPriceTop.Set(cur)
}

// fee ports _getFee: the block baseline plus the displacement of cur from the block-open price.
func (c *FeeConfig) fee(s *FeeState, cur *uint256.Int, zeroForOne bool) uint32 {
	var move relativeMove
	relMove(&move, cur, &s.SqrtPriceTop)

	// A sell (zeroForOne) pays the bid baseline, a buy the ask baseline.
	baseline := s.AskPermFee
	if zeroForOne {
		baseline = c.RestingSpread - s.AskPermFee
	}
	var x uint256.Int
	if zeroForOne == s.TempOnBid {
		x.SetUint64(uint64(baseline) + uint64(s.TempFee))
		baseline = clampFee(&x)
	}

	if zeroForOne == move.increased { // against the displacement: backrun
		move.mulUp(&x, c.BackrunCaptureFraction)
	} else { // with the displacement: continuation, fade capped
		move.mulUp(&x, c.MidFadeRate)
		if x.GtUint64(uint64(c.MaxMidFade)) {
			x.SetUint64(uint64(c.MaxMidFade))
		}
	}
	x.AddUint64(&x, uint64(baseline))
	return clampFee(&x)
}
