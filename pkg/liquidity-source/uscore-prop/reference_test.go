package uscoreprop

import (
	"slices"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// This file is the Go reference of abi/USCoreSampler.sol: the same sampling, driven by a quoter,
// so tests can check the on-chain sampler against it point by point.

var (
	scoreThreshold = new(uint256.Int).Div(one, big256.U100000) // 0.00001
	scoreCap       = new(uint256.Int).Lsh(big256.U1, maxShapeBits)
)

// quoter is what the reference sampler reads of a pool, all at one block.
type quoter interface {
	reserves() [2]*uint256.Int
	quoteExactIn(dir int, amount *uint256.Int) (*uint256.Int, uint8)
	quoteLadder(dir int, amounts []*uint256.Int) ([]*uint256.Int, uint8)
}

type refSnapshot struct {
	reserves [2]*uint256.Int
	points   [2][]*uint256.Int
	quotes   [2]ladderQuote
	rounds   int
}

func referenceSample(q quoter, units [2]*uint256.Int) refSnapshot {
	points := referenceInitialPoints(q, units)
	var s refSnapshot
	for round := 0; ; round++ {
		s.reserves = q.reserves()
		for i := range 2 {
			s.quotes[i] = ladderQuote{status: statusOK}
			if len(points[i]) > 0 {
				s.quotes[i].outs, s.quotes[i].status = q.quoteLadder(i, points[i])
			}
		}
		s.rounds = round + 1
		if round == maxRefinementRounds || s.quotes[0].status > maxStatus || s.quotes[1].status > maxStatus {
			break
		}
		changed := false
		for i := range 2 {
			refined := refineUpper(points[i], s.quotes[i])
			if round == 2 || len(refined) <= len(points[i]) {
				refined = refineCurve(points[i], s.quotes[i])
			}
			if refined != nil && !slices.EqualFunc(refined, points[i], (*uint256.Int).Eq) {
				points[i], changed = refined, true
			}
		}
		if !changed {
			break
		}
	}
	s.points = points
	return s
}

func referenceInitialPoints(q quoter, units [2]*uint256.Int) [2][]*uint256.Int {
	reserves := q.reserves()
	var points [2][]*uint256.Int
	for i := range 2 {
		out, status := q.quoteExactIn(i, units[i])
		var basis uint256.Int
		basis.Set(reserves[i])
		var minimum *uint256.Int
		if !reserves[1-i].IsZero() && status == statusInsufficient {
			var rateIn, rateOut *uint256.Int
			for amount := new(uint256.Int).Rsh(units[i], 1); !amount.IsZero(); amount = new(uint256.Int).Rsh(amount, 1) {
				if o, st := q.quoteExactIn(i, amount); st == statusOK && !o.IsZero() {
					if rateIn == nil {
						rateIn, rateOut = amount, o
					}
					minimum = amount
				}
			}
			if rateOut != nil {
				mulDivCapped(&basis, reserves[1-i], rateIn, rateOut)
			}
		}
		if status == statusOK && !out.IsZero() {
			mulDivCapped(&basis, reserves[1-i], units[i], out)
		}
		if basis.Gt(maxPoint) {
			basis.Set(maxPoint)
		}
		points[i] = big256.MustFromBigs(ladder.BuildSamplePointsN(basis.ToBig(), ladder.SampleSize))
		if minimum != nil {
			above := slices.DeleteFunc(points[i], func(x *uint256.Int) bool { return !x.Gt(minimum) })
			points[i] = append([]*uint256.Int{minimum}, above...)
		}
	}
	return points
}

// mulDivCapped sets z = x*y/d, or maxPoint when that overflows.
func mulDivCapped(z, x, y, d *uint256.Int) {
	if _, overflow := z.MulDivOverflow(x, y, d); overflow {
		z.Set(maxPoint)
	}
}

// refineUpper bisects the gap where a partial (statusInsufficient) ladder first fails, to find
// how far the pool can actually quote.
func refineUpper(points []*uint256.Int, q ladderQuote) []*uint256.Int {
	if q.status != statusInsufficient || len(points) >= maxSamplePoints {
		return nil
	}
	for i, out := range q.outs {
		if !out.IsZero() {
			if i > 0 && out.Lt(q.outs[i-1]) {
				return nil
			}
			continue
		}
		if i == 0 {
			return nil
		}
		lo, hi := points[i-1], points[i]
		var gap, n uint256.Int
		gap.Sub(hi, lo)
		refined := append([]*uint256.Int(nil), points[:i]...)
		for k := uint64(1); k < 8 && len(refined)+len(points)-i < maxSamplePoints; k++ {
			x := new(uint256.Int)
			x.MulDivOverflow(&gap, n.SetUint64(k), big256.U8)
			if x.Add(x, lo).Gt(refined[len(refined)-1]) && x.Lt(hi) {
				refined = append(refined, x)
			}
		}
		return append(refined, points[i:]...)
	}
	return nil
}

// refineCurve spends the remaining sample budget on segments whose rate differs most from their
// neighbours'. score = |rate_i - rate_j| * width_i / out_i = |delta_i - delta_j*width_i/width_j| / out_i,
// kept in fixed point.
func refineCurve(points []*uint256.Int, q ladderQuote) []*uint256.Int {
	if len(points) == 0 || (q.status != statusOK && q.status != statusInsufficient) {
		return nil
	}
	type segment struct {
		lo, hi, out         *uint256.Int
		width, delta, score uint256.Int // delta is signed
		parts               uint64
	}
	var segments []segment
	for i, out := range q.outs {
		if out.IsZero() || out.Gt(maxPoint) {
			break
		}
		lo, prev := big256.U0, big256.U0
		if i > 0 {
			lo, prev = points[i-1], q.outs[i-1]
		}
		segments = append(segments, segment{lo: lo, hi: points[i], out: out, parts: 1})
		s := &segments[len(segments)-1]
		s.width.Sub(points[i], lo)
		if s.delta.Sub(out, prev).Sign() < 0 {
			break
		}
	}
	// Samples past the first zero or decreasing output never reach the ladder: keep that first one
	// as the upper bound and give their budget to the curve.
	end := len(segments) + 1
	if len(segments) > 0 && segments[len(segments)-1].delta.Sign() < 0 {
		end--
	}
	points = points[:min(end, len(points))]
	var scaled, diff, score uint256.Int
	for i := range segments {
		s := &segments[i]
		if i == 0 || s.width.CmpUint64(2) < 0 {
			continue
		}
		for _, j := range []int{i - 1, i + 1} {
			if j >= len(segments) {
				continue
			}
			if !mulDiv(&scaled, &segments[j].delta, &s.width, &segments[j].width) {
				s.score.Set(scoreCap)
				break
			}
			abs(&diff, diff.Sub(&s.delta, &scaled))
			if _, overflow := score.MulDivOverflow(&diff, one, s.out); overflow || score.Gt(scoreCap) {
				score.Set(scoreCap)
			}
			if score.Gt(&s.score) {
				s.score.Set(&score)
			}
		}
	}
	budget := uint64(maxSamplePoints - len(points))
	for budget > 0 {
		best := -1
		for i := range segments {
			s := &segments[i]
			if s.parts <= budget && s.score.Gt(scoreThreshold) && (best < 0 || s.score.Gt(&segments[best].score)) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		s := &segments[best]
		budget -= s.parts
		s.parts *= 2
		s.score.Rsh(&s.score, 2)
	}
	refined := append([]*uint256.Int(nil), points...)
	var n, parts uint256.Int
	for i := range segments {
		s := &segments[i]
		parts.SetUint64(s.parts)
		for k := uint64(1); k < s.parts; k++ {
			x := new(uint256.Int)
			x.MulDivOverflow(&s.width, n.SetUint64(k), &parts)
			if x.Add(x, s.lo).Gt(s.lo) && x.Lt(s.hi) {
				refined = append(refined, x)
			}
		}
	}
	slices.SortFunc(refined, func(a, b *uint256.Int) int { return a.Cmp(b) })
	return slices.CompactFunc(refined, func(a, b *uint256.Int) bool { return a.Eq(b) })
}

// smallerProbe quotes unit/2, unit/4, ... to find a quotable rate (the largest such amount) and the

// mulDiv sets z = x*y/d (truncated) for two's-complement x, y, d; false if d is 0 or |z| is too big.
func mulDiv(z, x, y, d *uint256.Int) bool {
	var ax, ay, ad uint256.Int
	neg := abs(&ax, x) != abs(&ay, y)
	if abs(&ad, d) {
		neg = !neg
	}
	if ad.IsZero() {
		return false
	}
	if _, overflow := z.MulDivOverflow(&ax, &ay, &ad); overflow || z.BitLen() > maxShapeBits {
		return false
	}
	if neg {
		z.Neg(z)
	}
	return true
}
