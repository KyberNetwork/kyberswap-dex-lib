package kalos

import (
	"math"
	"math/bits"

	"github.com/holiman/uint256"
)

// FeeParams is KalosV4FeeHookBase.FeeParams, a pool's fee parameters (feeParams(id)): a calm minimum and a cap per
// side (zeroForOne sells currency0), and the volatility engine's settings. Fees are in pips, times in seconds,
// accumulators in ticks.
type FeeParams struct {
	MinFeeZeroForOne         uint32 `json:"m0"`
	MaxFeeZeroForOne         uint32 `json:"x0"`
	MinFeeOneForZero         uint32 `json:"m1"`
	MaxFeeOneForZero         uint32 `json:"x1"`
	Dynamic                  bool   `json:"d,omitempty"`
	Directional              bool   `json:"dr,omitempty"`
	FilterPeriod             uint64 `json:"fp,omitempty"`
	DecayPeriod              uint64 `json:"dp,omitempty"`
	ReductionFactor          uint64 `json:"rf,omitempty"`
	VariableFeeControl       uint64 `json:"vc,omitempty"`
	MaxVolatilityAccumulator uint64 `json:"mv,omitempty"`
	ResetPeriod              uint64 `json:"rp,omitempty"`
	ResetTickFilter          uint64 `json:"rt,omitempty"`
	SameMomentSeconds        uint64 `json:"sm,omitempty"`
}

// FeeState is KalosV4FeeHookBase.FeeState, a pool's volatility state (feeState(id)), which every swap of a dynamic
// pool moves.
type FeeState struct {
	ReferenceTick       int32  `json:"r,omitempty"`
	LastTick            int32  `json:"l,omitempty"`
	Peak                int32  `json:"p,omitempty"`
	VolatilityReference uint64 `json:"v,omitempty"`
	LastSwap            uint64 `json:"s,omitempty"`
	ResetTime           uint64 `json:"t,omitempty"`
	MoveTime            uint64 `json:"mt,omitempty"`
	PeakTime            uint64 `json:"pt,omitempty"`
}

// computeFee mirrors KalosV4FeeHook.computeFee: the total fee in pips of a swap in a direction landing at timestamp,
// at the pool's tick before the swap, the accumulator it pays, and the state it leaves (Trader Joe Liquidity Book
// v2.1's volatility accumulator held at the burst's peak, Clanker v4's reset check, a directional rule). Every
// division rounds down. ok is false where the contract would revert (a state whose reset time is ahead of the swap).
func (p *FeeParams) computeFee(s FeeState, tick int32, timestamp uint64, zeroForOne bool) (fee uint32, va uint64,
	next FeeState, ok bool) {
	next = s
	var variable uint64
	if p.Dynamic {
		// Time never runs backward on chain; if it ever did, it counts as no time.
		timestamp = max(timestamp, s.LastSwap)
		moveTicks := uint64(min(p.MinFeeZeroForOne, p.MinFeeOneForZero) / 100)
		if distance(tick, s.LastTick) >= max(moveTicks, 1) {
			next.LastTick = tick
			next.MoveTime = s.LastSwap
		}
		if next.MoveTime > timestamp || s.ResetTime > timestamp { // the contract's subtractions underflow
			return 0, 0, next, false
		}
		sameMoment := timestamp-next.MoveTime < p.SameMomentSeconds
		peak := raisePeak(&s, tick, p.MaxVolatilityAccumulator)
		if peak != s.Peak {
			next.PeakTime = s.LastSwap
		}
		if factor := p.windowFactor(&s, peak, tick, timestamp, next.PeakTime); factor <= bps {
			vr := abs(peak) * factor / bps
			next.VolatilityReference = vr
			if peak < 0 {
				peak = -int32(vr)
			} else {
				peak = int32(vr)
			}
			next.ReferenceTick = tick
			next.ResetTime = timestamp
		}
		next.Peak = peak
		next.LastSwap = timestamp
		var pays bool
		if va, pays = p.paidAccumulator(&next, tick, zeroForOne, sameMoment); pays {
			variable = variableFee(p.VariableFeeControl, va)
		}
	}
	minFee, maxFee := p.MinFeeOneForZero, p.MaxFeeOneForZero
	if zeroForOne {
		minFee, maxFee = p.MinFeeZeroForOne, p.MaxFeeZeroForOne
	}
	// min(minFee + variable, maxFee) without overflow.
	fee = maxFee
	if minFee < maxFee && variable < uint64(maxFee-minFee) {
		fee = minFee + uint32(variable)
	}
	return fee, va, next, true
}

// variableFee is variableFeeControl * va^2 / 1e6, rounded down. On chain both are below 2^24 (va is capped by
// maxVolatilityAccumulator < 2^23), so the product is below 2^71; a larger one saturates (the fee is capped anyway).
func variableFee(control, va uint64) uint64 {
	hi, lo := bits.Mul64(control, va)
	if hi != 0 {
		return math.MaxUint64
	}
	hi, lo = bits.Mul64(lo, va)
	if hi >= variableFeeDenominator {
		return math.MaxUint64
	}
	q, _ := bits.Div64(hi, lo, variableFeeDenominator)
	return q
}

// calmFee is a direction's minimum, the whole fee of a plain pool (_calmFee).
func (p *FeeParams) calmFee(zeroForOne bool) uint32 {
	if zeroForOne {
		return p.MinFeeZeroForOne
	}
	return p.MinFeeOneForZero
}

// raisePeak mirrors _raisePeak: the burst's peak after the price reached tick. On the peak's side (or with no peak)
// it rises to the accumulator; on the other side of the reference it flips only once the price is further that way
// than the peak.
func raisePeak(s *FeeState, tick int32, maxVA uint64) int32 {
	peak := s.Peak
	if tick == s.ReferenceTick {
		return peak
	}
	d := distance(tick, s.ReferenceTick)
	magnitude := abs(peak)
	up := tick > s.ReferenceTick
	va := min(s.VolatilityReference+d, maxVA)
	if sameSide := peak == 0 || (peak > 0) == up; sameSide && va <= magnitude || !sameSide && d <= magnitude {
		return peak
	}
	if up {
		return int32(va)
	}
	return -int32(va)
}

// windowFactor mirrors _windowFactor: the factor a new window keeps of the peak (above bps: no new window). peak is
// after raisePeak, peakTime when it last rose.
func (p *FeeParams) windowFactor(s *FeeState, peak, tick int32, t, peakTime uint64) uint64 {
	resetDue := t-s.ResetTime >= p.ResetPeriod
	magnitude := abs(peak)
	if resetDue && distance(tick, s.ReferenceTick) <= p.ResetTickFilter && magnitude <= p.ResetTickFilter {
		return 0
	}
	var age uint64
	if t > peakTime {
		age = t - peakTime
	}
	if age >= p.DecayPeriod {
		return 0
	}
	if age >= p.FilterPeriod && magnitude != s.VolatilityReference || resetDue {
		return p.ReductionFactor
	}
	return bps + 1
}

// paidAccumulator mirrors _paidAccumulator: the accumulator this swap pays (the burst's, or the distance past the
// reference against the burst) and whether it pays one.
func (p *FeeParams) paidAccumulator(n *FeeState, tick int32, zeroForOne, sameMoment bool) (uint64, bool) {
	lb := min(n.VolatilityReference+distance(tick, n.ReferenceTick), p.MaxVolatilityAccumulator)
	va := max(abs(n.Peak), lb)
	if !p.Directional || sameMoment {
		return va, true
	}
	up := !zeroForOne
	if n.Peak != 0 && (n.Peak > 0) == up { // with the burst
		return va, true
	}
	if tick != n.ReferenceTick && (tick > n.ReferenceTick) == up { // further past the reference
		return lb, true
	}
	return va, false // back toward the burst's start: the minimum
}

// splitFee mirrors KalosV4FeeHookBase.splitFee: the hook's part h and the LP fee lp of a total fee for a swap in a
// direction, in pips. No fee asset: all LP fee. A creator pool, or the fee asset as the swap's output: all the hook's
// (the LP fee is zero). The fee asset as the input: h = total * share / 1e4 and lp = (total - h) * 1e6 / (1e6 - h),
// so the swapper pays total.
func splitFee(kind, feeSide uint8, shareBps uint16, total uint32, zeroForOne bool) (hookFee, lpFee uint32) {
	if feeSide == feeNone {
		return 0, total
	}
	if kind == kindCreator || (feeSide == feeCurrency0) != zeroForOne {
		return total, 0
	}
	hookFee = uint32(uint64(total) * uint64(shareBps) / bps)
	lpFee = uint32(uint64(total-hookFee) * pips / uint64(pips-hookFee))
	return hookFee, lpFee
}

var uPips = uint256.NewInt(pips)

// hookFeeOf is KalosV4FeeHook.hookFeeOf for exact input: h of a gross amount of the fee asset, rounded up. amount is
// below 2^128 and h at most 5e4, so the product fits 256 bits.
func hookFeeOf(amount *uint256.Int, h uint32) (fee uint256.Int) {
	fee.Mul(amount, fee.SetUint64(uint64(h)))
	fee.AddUint64(&fee, pips-1)
	return *fee.Div(&fee, uPips)
}

// grossOf is the least gross with gross - hookFeeOf(gross, h) >= net: gross - ceil(gross*h/1e6) is
// floor(gross*(1e6-h)/1e6), so it is ceil(net*1e6/(1e6-h)).
func grossOf(net *uint256.Int, h uint32) (gross uint256.Int) {
	var denom uint256.Int
	denom.SetUint64(pips - uint64(h))
	gross.Mul(net, uPips)
	gross.Add(&gross, denom.SubUint64(&denom, 1))
	return *gross.Div(&gross, denom.AddUint64(&denom, 1))
}

func distance(a, b int32) uint64 {
	if a > b {
		return uint64(int64(a) - int64(b))
	}
	return uint64(int64(b) - int64(a))
}

func abs(x int32) uint64 {
	if x < 0 {
		return uint64(-int64(x))
	}
	return uint64(x)
}
