package inverse

import (
	"errors"
	"math/big"

	v3utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	ErrState         = errors.New("inverse: invalid or unsupported snapshot")
	ErrExactOutput   = errors.New("inverse: exact output unsupported")
	ErrBounds        = errors.New("inverse: trade outside contract bounds")
	ErrRounding      = errors.New("inverse: native rounding or custody limit")
	ErrController    = errors.New("inverse: unsupported protocol fee controller")
	ray              = new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil)
	maxDelta         = sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
	maxReserve       = sub(new(big.Int).Lsh(big.NewInt(1), 112), big.NewInt(1))
	q128             = new(big.Int).Lsh(big.NewInt(1), 128)
	q192             = new(big.Int).Lsh(big.NewInt(1), 192)
	maxInitialShares = new(big.Int).Exp(n(10), n(30), nil)
	maxRelativePrice = new(big.Int).Exp(n(10), n(36), nil)
	maxIndex         = new(big.Int).Exp(n(10), n(45), nil)
	lowerSqrt        = sqrtTick(minTick)
	upperSqrt        = sqrtTick(maxTick)
)

const minTick = -887220
const maxTick = 887220

// Arbitrary-width intermediates implement Solidity FullMath (512-bit products),
// with explicit EVM result bounds below. Helpers never mutate their arguments.
func n(x int64) *big.Int                   { return big.NewInt(x) }
func add(a, b *big.Int) *big.Int           { return new(big.Int).Add(a, b) }
func sub(a, b *big.Int) *big.Int           { return new(big.Int).Sub(a, b) }
func mul(a, b *big.Int) *big.Int           { return new(big.Int).Mul(a, b) }
func div(a, b *big.Int) *big.Int           { return new(big.Int).Quo(a, b) }
func md(a, b, c *big.Int) *big.Int         { return div(mul(a, b), c) }
func up(a, b, c *big.Int) *big.Int         { return div(add(mul(a, b), sub(c, n(1))), c) }
func absdiff(a, b *big.Int) *big.Int       { return new(big.Int).Abs(sub(a, b)) }
func u(a *big.Int) *uint256.Int            { return uint256.MustFromBig(a) }
func positiveBound(a, limit *big.Int) bool { return a.Sign() > 0 && a.Cmp(limit) <= 0 }

func (s Extra) validate() error {
	if s.Version != 1 || !s.Live || s.BlockNumber == 0 || s.ProtocolFee&0xfff > 1000 || s.ProtocolFee>>12 > 1000 {
		return ErrState
	}
	if s.InitialShares.CmpUint64(1e18) < 0 || s.InitialShares.ToBig().Cmp(maxInitialShares) > 0 ||
		s.InitialQuote.CmpUint64(1e12) < 0 || !positiveBound(s.InitialQuote.ToBig(), maxReserve) || s.InitialQuote.Cmp(&s.InitialShares) >= 0 {
		return ErrState
	}
	if s.ReserveShares.CmpUint64(1e9) < 0 || s.ReserveShares.Cmp(&s.InitialShares) > 0 || !positiveBound(s.ReserveQuote.ToBig(), maxReserve) ||
		s.Index.ToBig().Cmp(ray) < 0 || !positiveBound(s.Index.ToBig(), maxIndex) ||
		s.Index.ToBig().Cmp(md(maxDelta, ray, s.InitialShares.ToBig())) > 0 || s.CustodiedShares.Cmp(&s.ReserveShares) < 0 || s.CustodiedShares.Cmp(&s.InitialShares) > 0 ||
		!positiveBound(s.Liquidity.ToBig(), maxDelta) || s.Tick < minTick || s.Tick >= maxTick {
		return ErrState
	}
	for _, v := range []*uint256.Int{&s.NativeQuote, &s.HookQuote, &s.NativeInverse, &s.Fees0, &s.Fees1} {
		if v.ToBig().Cmp(maxDelta) > 0 {
			return ErrState
		}
	}
	if s.RoundingQuote.ToBig().Cmp(maxReserve) > 0 || s.Sequence == (uint256.Int{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}) {
		return ErrState
	}
	lo, hi := lowerSqrt, upperSqrt
	if s.SqrtPriceX96.ToBig().Cmp(lo) <= 0 || s.SqrtPriceX96.ToBig().Cmp(hi) >= 0 ||
		s.NativeInverse.ToBig().Cmp(md(s.CustodiedShares.ToBig(), s.Index.ToBig(), ray)) > 0 ||
		add(s.HookQuote.ToBig(), s.NativeQuote.ToBig()).Cmp(add(s.ReserveQuote.ToBig(), s.RoundingQuote.ToBig())) < 0 {
		return ErrState
	}
	tick, err := v3utils.GetTickAtSqrtRatioV2(&s.SqrtPriceX96)
	if err != nil || (s.Tick != tick && !(s.Tick == tick-1 && s.SqrtPriceX96.ToBig().Cmp(sqrtTick(tick)) == 0)) {
		return ErrState
	}
	return nil
}

func quote(s Extra, zeroForOne bool, input *big.Int) (*big.Int, Extra, error) {
	if err := s.validate(); err != nil {
		return nil, s, err
	}
	if input == nil || !positiveBound(input, maxDelta) {
		return nil, s, ErrBounds
	}
	buy := zeroForOne != s.Inverse0
	pf := s.ProtocolFee >> 12
	if zeroForOne {
		pf = s.ProtocolFee & 0xfff
	}
	fee := int64(pf) + 3000 - int64(pf)*3000/1_000_000
	x, y, idx := s.ReserveShares.ToBig(), s.ReserveQuote.ToBig(), s.Index.ToBig()
	shares, qt := new(big.Int), new(big.Int)
	refInput := input
	rin, rout := y, x
	if !buy {
		refInput = up(input, ray, idx)
		rin, rout = x, y
	}
	if !positiveBound(refInput, maxReserve) {
		return nil, s, ErrBounds
	}
	effective := mul(refInput, n(1_000_000-fee))
	output := md(effective, rout, add(mul(rin, n(1_000_000)), effective))
	if output.Sign() == 0 {
		return nil, s, ErrBounds
	}
	if buy {
		shares, qt = output, input
	} else {
		shares, qt = refInput, output
	}
	protocolInput := up(refInput, n(int64(pf)), n(1_000_000))
	nx, ny := sub(x, shares), sub(add(y, qt), protocolInput)
	if !buy {
		nx, ny = sub(add(x, shares), protocolInput), sub(y, qt)
	}
	if nx.Cmp(n(1e9)) < 0 || nx.Cmp(s.InitialShares.ToBig()) > 0 || !positiveBound(ny, maxReserve) {
		return nil, s, ErrBounds
	}
	relative := md(ny, mul(s.InitialShares.ToBig(), ray), mul(nx, s.InitialQuote.ToBig()))
	if relative.Cmp(n(1e18)) < 0 || relative.Cmp(maxRelativePrice) > 0 {
		return nil, s, ErrBounds
	}
	ni := md(relative, relative, ray)
	if ni.Cmp(ray) < 0 || ni.Cmp(maxIndex) > 0 || ni.Cmp(md(maxDelta, ray, s.InitialShares.ToBig())) > 0 ||
		(buy && ni.Cmp(idx) <= 0) || (!buy && ni.Cmp(idx) >= 0) {
		return nil, s, ErrBounds
	}
	nativeIn, out := sub(input, n(1)), md(shares, ni, ray)
	if !buy {
		nativeIn, out = md(shares, ni, ray), qt
	}
	if nativeIn.Sign() <= 0 || nativeIn.Cmp(input) >= 0 || !positiveBound(out, maxDelta) {
		return nil, s, ErrBounds
	}
	vs, vq := mul(x, n(1_000_000-fee)), mul(y, n(1_000_000-fee))
	if buy {
		vs = add(vs, mul(shares, n(fee-int64(pf))))
	} else {
		vq = add(vq, mul(qt, n(fee-int64(pf))))
	}
	vi := md(div(vs, n(1_000_000-int64(pf))), ni, ray)
	vq = div(vq, n(1_000_000-int64(pf)))
	if !positiveBound(vi, maxDelta) || !positiveBound(vq, maxDelta) {
		return nil, s, ErrBounds
	}
	// Remove the previous full-range position, including collectable LP fees.
	recovered0, recovered1, err := positionAmounts(s.SqrtPriceX96.ToBig(), s.Liquidity.ToBig(), false)
	if err != nil {
		return nil, s, err
	}
	recovered0 = add(recovered0, s.Fees0.ToBig())
	recovered1 = add(recovered1, s.Fees1.ToBig())
	recoveredInv, recoveredQuote := recovered0, recovered1
	if !s.Inverse0 {
		recoveredInv, recoveredQuote = recovered1, recovered0
	}
	if recovered0.Cmp(maxDelta) > 0 || recovered1.Cmp(maxDelta) > 0 {
		return nil, s, ErrRounding
	}
	loss := sub(s.NativeQuote.ToBig(), recoveredQuote)
	if loss.Sign() < 0 || loss.Cmp(n(32)) > 0 || loss.Cmp(s.RoundingQuote.ToBig()) > 0 || recoveredInv.Cmp(s.NativeInverse.ToBig()) > 0 {
		return nil, s, ErrRounding
	}
	// Re-denominate idle custody and install the next full-range position.
	start, err := sqrtPrice(vi, vq, s.Inverse0)
	if err != nil {
		return nil, s, err
	}
	liq := new(big.Int).Sqrt(mul(vi, vq))
	d0, d1, err := positionAmounts(start, liq, true)
	if err != nil {
		return nil, s, err
	}
	depositInv, depositQuote := d0, d1
	if !s.Inverse0 {
		depositInv, depositQuote = d1, d0
	}
	hookQuote := sub(add(s.HookQuote.ToBig(), recoveredQuote), depositQuote)
	if hookQuote.Sign() < 0 || depositInv.Cmp(md(s.CustodiedShares.ToBig(), ni, ray)) > 0 || !positiveBound(d0, maxDelta) || !positiveBound(d1, maxDelta) {
		return nil, s, ErrRounding
	}
	tick, err := v3utils.GetTickAtSqrtRatioV2(u(start))
	if err != nil {
		return nil, s, err
	}
	if start.Cmp(s.SqrtPriceX96.ToBig()) == 0 {
		tick = s.Tick
	} else if start.Cmp(s.SqrtPriceX96.ToBig()) < 0 && tick%15360 == 0 && start.Cmp(sqrtTick(tick)) == 0 {
		tick--
	}
	end, nt, nativeOut, proto, lpFees, err := nativeSwap(start, tick, liq, nativeIn, zeroForOne, fee, int64(pf))
	if err != nil {
		return nil, s, err
	}
	bound := add(up(n(32), vq, vi), n(32))
	if buy {
		bound = add(add(up(n(32), vi, vq), up(n(32), ni, ray)), n(32))
	}
	if !positiveBound(nativeOut, maxDelta) || absdiff(nativeOut, out).Cmp(bound) > 0 {
		return nil, s, ErrRounding
	}
	down := buy == s.Inverse0
	if (down && end.Cmp(s.SqrtPriceX96.ToBig()) >= 0) || (!down && end.Cmp(s.SqrtPriceX96.ToBig()) <= 0) {
		return nil, s, ErrBounds
	}
	expected := md(md(ny, ray, nx), ray, ni)
	actual := md(end, mul(end, ray), q192)
	if !s.Inverse0 {
		if actual.Sign() == 0 {
			return nil, s, ErrBounds
		}
		actual = md(ray, ray, actual)
	}
	if absdiff(actual, expected).Cmp(add(div(expected, n(10_000_000)), n(2))) > 0 {
		return nil, s, ErrRounding
	}
	custody := s.CustodiedShares.ToBig()
	nativeQuote, nativeInv := depositQuote, depositInv
	if buy {
		if proto.Cmp(protocolInput) > 0 {
			return nil, s, ErrRounding
		}
		custody = sub(custody, shares)
		nativeQuote = add(nativeQuote, sub(nativeIn, proto))
		nativeInv = sub(nativeInv, nativeOut)
		hookQuote = add(hookQuote, n(1))
	} else {
		if proto.Sign() > 0 && !s.FeeControllerSupported {
			return nil, s, ErrController
		}
		feeShares := up(proto, ray, ni)
		if feeShares.Cmp(protocolInput) > 0 {
			return nil, s, ErrRounding
		}
		custody = sub(add(custody, shares), feeShares)
		nativeQuote = sub(nativeQuote, nativeOut)
		nativeInv = add(nativeInv, sub(nativeIn, proto))
		hookQuote = add(hookQuote, sub(nativeOut, out))
	}
	rounding := sub(s.RoundingQuote.ToBig(), loss)
	if nativeQuote.Sign() < 0 || nativeInv.Sign() < 0 || hookQuote.Sign() < 0 || custody.Cmp(nx) < 0 ||
		nativeInv.Cmp(md(custody, ni, ray)) > 0 || add(nativeQuote, hookQuote).Cmp(add(ny, rounding)) < 0 {
		return nil, s, ErrRounding
	}
	next := s
	next.ReserveShares = *u(nx)
	next.ReserveQuote = *u(ny)
	next.Index = *u(ni)
	next.CustodiedShares = *u(custody)
	next.NativeInverse = *u(nativeInv)
	next.NativeQuote = *u(nativeQuote)
	next.HookQuote = *u(hookQuote)
	next.RoundingQuote = *u(rounding)
	next.SqrtPriceX96 = *u(end)
	next.Liquidity = *u(liq)
	next.Tick = nt
	next.Sequence.AddUint64(&s.Sequence, 1)
	next.Fees0.Clear()
	next.Fees1.Clear()
	if zeroForOne {
		next.Fees0 = *u(lpFees)
	} else {
		next.Fees1 = *u(lpFees)
	}
	return out, next, nil
}

func sqrtTick(tick int) *big.Int {
	var x uint256.Int
	_ = v3utils.GetSqrtRatioAtTickV2(tick, &x)
	return x.ToBig()
}
func sqrtPrice(inverse, quote *big.Int, inverse0 bool) (*big.Int, error) {
	num, den := quote, inverse
	if !inverse0 {
		num, den = inverse, quote
	}
	result := new(big.Int)
	if div(num, den).BitLen() <= 64 {
		result.Sqrt(md(num, q192, den))
	} else {
		result.Sqrt(md(num, q128, den))
		result.Lsh(result, 32)
	}
	if result.Cmp(lowerSqrt) <= 0 || result.Cmp(upperSqrt) >= 0 {
		return nil, ErrBounds
	}
	return result, nil
}
func positionAmounts(sqrt, liq *big.Int, roundUp bool) (*big.Int, *big.Int, error) {
	var a, b uint256.Int
	if err := v3utils.GetAmount0DeltaV2(u(sqrt), u(upperSqrt), u(liq), roundUp, &a); err != nil {
		return nil, nil, err
	}
	if err := v3utils.GetAmount1DeltaV2(u(lowerSqrt), u(sqrt), u(liq), roundUp, &b); err != nil {
		return nil, nil, err
	}
	return a.ToBig(), b.ToBig(), nil
}

// Match v4-core's bitmap-word steps, not merely the two initialized ticks.
// Fee rounding (including protocol collection) happens at EVERY such step.
func nativeSwap(start *big.Int, tick int, liq, input *big.Int, zero bool, fee, pf int64) (end *big.Int, endTick int, out, protocol, lpFees *big.Int, err error) {
	end = new(big.Int).Set(start)
	out = n(0)
	protocol = n(0)
	growth := n(0)
	remaining := new(big.Int).Set(input)
	for steps := 0; remaining.Sign() > 0; steps++ {
		if steps > 120 {
			return nil, 0, nil, nil, nil, ErrBounds
		}
		compressed := tick / 60
		if tick < 0 && tick%60 != 0 {
			compressed--
		}
		var next int
		if zero {
			next = (compressed >> 8) * 256 * 60
			if next < minTick {
				next = minTick
			}
		} else {
			next = (((compressed+1)>>8)*256 + 255) * 60
			if next > maxTick {
				next = maxTick
			}
		}
		target := sqrtTick(next)
		sn, si, so, sf, e := computeSwapStepExactIn(u(end), u(target), u(liq), u(remaining), uint64(fee))
		if e != nil {
			return nil, 0, nil, nil, nil, e
		}
		paid := add(si.ToBig(), sf.ToBig())
		remaining = sub(remaining, paid)
		p := md(paid, n(pf), n(1_000_000))
		protocol = add(protocol, p)
		growth = add(growth, md(sub(sf.ToBig(), p), q128, liq))
		out = add(out, so.ToBig())
		end = sn.ToBig()
		if end.Cmp(target) == 0 {
			tick = next
			if zero {
				tick--
			}
			if next == minTick || next == maxTick {
				return nil, 0, nil, nil, nil, ErrBounds
			}
		} else {
			tick, e = v3utils.GetTickAtSqrtRatioV2(sn)
			if e != nil {
				return nil, 0, nil, nil, nil, e
			}
		}
	}
	if out.Sign() <= 0 {
		return nil, 0, nil, nil, nil, ErrBounds
	}
	return end, tick, out, protocol, md(growth, liq, q128), nil
}
