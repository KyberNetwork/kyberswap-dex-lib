package twocryptong

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	policyLn2            = uint256.MustFromDecimal("693147180559945309")
	policyCapRampSeconds = uint256.NewInt(3600)
	policyTweakFactor    = uint256.NewInt(5)
)

// PriceScaleAt is YBTwocryptoPolicy's get_price_scale() at now (verified source, Ethereum): the dual-EMA
// target extrapolated by kappa, held inside the deadband, capped by a cap that ramps from MIN_CAP to MAX_CAP over
// an hour since the last update, and the capped gap multiplied by five for the pool's one-fifth step.
func (s *PolicyState) PriceScaleAt(now int64) (*uint256.Int, error) {
	current := s.PriceScale
	if current.IsZero() {
		return new(uint256.Int), nil
	}
	elapsed := uint256.NewInt(uint64(now - s.LastUpdateTs))
	rampTime := elapsed
	if policyCapRampSeconds.Lt(rampTime) {
		rampTime = policyCapRampSeconds
	}
	var currentCap uint256.Int
	currentCap.Sub(s.MaxCap, s.MinCap)
	currentCap.Mul(&currentCap, rampTime)
	currentCap.Div(&currentCap, policyCapRampSeconds)
	currentCap.Add(&currentCap, s.MinCap)

	fast, slow, err := s.projectEMAs(elapsed)
	if err != nil {
		return nil, err
	}

	var rawTarget, step uint256.Int
	if !fast.Lt(slow) {
		step.Mul(s.Kappa, step.Sub(fast, slow))
		rawTarget.Add(slow, step.Div(&step, U_1e18))
	} else {
		step.Mul(s.Kappa, step.Sub(slow, fast))
		step.Div(&step, U_1e18)
		if slow.Lt(&step) {
			step.Set(slow)
		}
		rawTarget.Sub(slow, &step)
	}

	var gap uint256.Int
	up := !rawTarget.Lt(current)
	if up {
		gap.Sub(&rawTarget, current)
	} else {
		gap.Sub(current, &rawTarget)
	}
	var lhs, rhs uint256.Int
	if !lhs.Mul(&gap, U_1e18).Gt(rhs.Mul(current, s.Deadband)) {
		return new(uint256.Int).Set(current), nil
	}
	var maxMove uint256.Int
	maxMove.Div(maxMove.Mul(current, &currentCap), U_1e18)
	if maxMove.Lt(&gap) {
		gap.Set(&maxMove)
	}
	gap.Mul(&gap, policyTweakFactor)
	if up {
		return new(uint256.Int).Add(current, &gap), nil
	}
	return new(uint256.Int).Sub(current, &gap), nil
}

// projectEMAs is _project_emas: both EMAs settled toward the stored last_prices over dt.
func (s *PolicyState) projectEMAs(dt *uint256.Int) (fast, slow *uint256.Int, err error) {
	if fast, err = s.ema(s.FastEma, dt, s.FastHalfLife); err != nil {
		return nil, nil, err
	}
	if slow, err = s.ema(s.SlowEma, dt, s.SlowHalfLife); err != nil {
		return nil, nil, err
	}
	return fast, slow, nil
}

// ema is _ema: alpha = exp(-ln 2 * dt / half_life), the observation bounded to [scale / 2, 2 * scale].
func (s *PolicyState) ema(ema, dt, halfLife *uint256.Int) (*uint256.Int, error) {
	if dt.IsZero() {
		return new(uint256.Int).Set(ema), nil
	}
	var price, bound uint256.Int
	price.Set(s.LastPrices)
	if bound.Rsh(s.PriceScale, 1); price.Lt(&bound) {
		price.Set(&bound)
	}
	if bound.Lsh(s.PriceScale, 1); bound.Lt(&price) {
		price.Set(&bound)
	}
	var exponent uint256.Int
	exponent.Div(exponent.Mul(dt, policyLn2), halfLife)
	x := int256.Int(exponent)
	alpha, err := wad_exp(x.Neg(&x))
	if err != nil {
		return nil, err
	}
	var out, tmp uint256.Int
	out.Mul(&price, tmp.Sub(U_1e18, alpha))
	out.Add(&out, tmp.Mul(ema, alpha))
	return out.Div(&out, U_1e18), nil
}

// afterUpdate is the state update_pool_state stores at now: both EMAs settled toward the previous last_prices,
// then the new observation and price_scale.
func (s *PolicyState) afterUpdate(now int64, lastPrices, priceScale, priceOracle *uint256.Int) (*PolicyState, error) {
	next := *s
	if s.PriceScale.IsZero() {
		next.FastEma, next.SlowEma = new(uint256.Int).Set(priceOracle), new(uint256.Int).Set(priceOracle)
	} else {
		fast, slow, err := s.projectEMAs(uint256.NewInt(uint64(now - s.LastUpdateTs)))
		if err != nil {
			return nil, err
		}
		next.FastEma, next.SlowEma = fast, slow
	}
	next.LastUpdateTs = now
	next.LastPrices = new(uint256.Int).Set(lastPrices)
	next.PriceScale = new(uint256.Int).Set(priceScale)
	return &next, nil
}
