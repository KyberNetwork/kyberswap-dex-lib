package uscoreprop

import (
	"context"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct{ client *ethrpc.Client }

type stateOverrides = map[common.Address]gethclient.OverrideAccount

var (
	scoreThreshold = new(uint256.Int).Div(one, big256.U100000) // 0.00001
	scoreCap       = new(uint256.Int).Lsh(big256.U1, maxShapeBits)
)

func NewPoolTracker(_ *Config, client *ethrpc.Client) *PoolTracker {
	return &PoolTracker{client: client}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.track(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context, p entity.Pool, params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.track(ctx, p, params.Overrides)
}

func (t *PoolTracker) track(ctx context.Context, p entity.Pool, overrides stateOverrides) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return p, ErrInvalidState
	}
	points, err := t.initialPoints(ctx, p, overrides)
	if err != nil {
		return p, err
	}

	// Quote all points at one block per round; refine where the curve needs more samples and
	// keep only the last round, so the stored ladders and reserves share a block.
	var reserves reservesResult
	var results [2]ladderResult
	var quotes [2]ladderQuote
	var timestamp, blockNumber *big.Int
	var ladders [2][]Point
	for round := 0; ; round++ {
		reserves, results, timestamp = reservesResult{}, [2]ladderResult{}, nil
		req := t.client.R().SetContext(ctx).SetOverrides(overrides)
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "getReserves"}, []any{&reserves})
		req.AddCall(&ethrpc.Call{ABI: clockABI, Target: multicall3, Method: "getCurrentBlockTimestamp"}, []any{&timestamp})
		for i := range 2 {
			if len(points[i]) > 0 {
				amounts := lo.Map(points[i], func(x *uint256.Int, _ int) *big.Int { return x.ToBig() })
				req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteLadder",
					Params: []any{common.HexToAddress(p.Tokens[i].Address), amounts}}, []any{&results[i]})
			}
		}
		resp, err := req.Aggregate()
		if err != nil {
			return p, err
		}
		if reserves.ReserveQuote == nil || reserves.ReserveBase == nil || timestamp == nil || !timestamp.IsInt64() ||
			timestamp.Sign() <= 0 || resp.BlockNumber == nil || resp.BlockNumber.Sign() <= 0 {
			return p, ErrInvalidState
		}
		blockNumber = resp.BlockNumber
		for i := range 2 {
			if quotes[i], err = toLadderQuote(points[i], results[i]); err != nil {
				return p, err
			}
			ladders[i] = collectQuotes(points[i], quotes[i])
		}
		if round == maxRefinementRounds {
			break
		}
		changed := false
		for i := range 2 {
			// Locate the capacity edge first; from the third round on, only reshape the curve.
			refined := refineUpper(points[i], quotes[i])
			if round == 2 || len(refined) <= len(points[i]) {
				refined = refineCurve(points[i], quotes[i])
			}
			if refined != nil && !slices.EqualFunc(refined, points[i], (*uint256.Int).Eq) {
				points[i], changed = refined, true
			}
		}
		if !changed {
			break
		}
	}
	extra, err := json.Marshal(Extra{Ladders: ladders, Time: timestamp.Int64()})
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserves.ReserveQuote.String(), reserves.ReserveBase.String()}
	for dir := range ladders {
		if len(ladders[dir]) == 0 { // nothing obtainable: don't report it as liquidity
			p.Reserves[1-dir] = "0"
		}
	}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = timestamp.Int64()
	return p, nil
}

// initialPoints spreads ladder.SampleSize geometric samples per direction up to the input that
// would drain the output reserve at the current rate.
func (t *PoolTracker) initialPoints(ctx context.Context, p entity.Pool, overrides stateOverrides) ([2][]*uint256.Int, error) {
	var points [2][]*uint256.Int
	var seed reservesResult
	var probes [2]quoteResult
	var units [2]*uint256.Int
	req := t.client.R().SetContext(ctx).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "getReserves"}, []any{&seed})
	for i, token := range p.Tokens {
		if token.Decimals > 38 || !common.IsHexAddress(token.Address) {
			return points, ErrInvalidState
		}
		units[i] = big256.TenPow(token.Decimals)
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteExactIn",
			Params: []any{common.HexToAddress(token.Address), units[i].ToBig()}}, []any{&probes[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return points, err
	}
	if seed.ReserveQuote == nil || seed.ReserveBase == nil {
		return points, ErrInvalidState
	}
	seedReserves := [2]*uint256.Int{uint256.MustFromBig(seed.ReserveQuote), uint256.MustFromBig(seed.ReserveBase)}
	for i := range 2 {
		var basis uint256.Int
		basis.Set(seedReserves[i])
		var minimum *uint256.Int
		if !seedReserves[1-i].IsZero() && probes[i].Status == statusInsufficient {
			rateIn, rateOut, low, err := t.smallerProbe(ctx, p.Address, p.Tokens[i].Address, units[i], overrides)
			if err != nil {
				return points, err
			}
			if rateOut != nil {
				minimum = low
				mulDivCapped(&basis, seedReserves[1-i], rateIn, rateOut)
			}
		}
		if probes[i].Status == statusOK && probes[i].AmountOut != nil && probes[i].AmountOut.Sign() > 0 {
			mulDivCapped(&basis, seedReserves[1-i], units[i], uint256.MustFromBig(probes[i].AmountOut))
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
	return points, nil
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
// smallest quotable amount, for pools too shallow to quote one whole token.
func (t *PoolTracker) smallerProbe(
	ctx context.Context, address, token string, unit *uint256.Int, overrides stateOverrides,
) (rateIn, rateOut, minimum *uint256.Int, err error) {
	var amounts []*uint256.Int
	for amount := new(uint256.Int).Rsh(unit, 1); !amount.IsZero(); amount = new(uint256.Int).Rsh(amount, 1) {
		amounts = append(amounts, amount)
	}
	if len(amounts) == 0 {
		return nil, nil, nil, nil
	}
	results := make([]quoteResult, len(amounts))
	req := t.client.R().SetContext(ctx).SetOverrides(overrides)
	for i, amount := range amounts {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: "quoteExactIn",
			Params: []any{common.HexToAddress(token), amount.ToBig()}}, []any{&results[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, nil, err
	}
	for i, result := range results {
		if result.Status == statusOK && result.AmountOut != nil && result.AmountOut.Sign() > 0 {
			if rateIn == nil {
				rateIn, rateOut = amounts[i], uint256.MustFromBig(result.AmountOut)
			}
			minimum = amounts[i]
		}
	}
	return rateIn, rateOut, minimum, nil
}

// toLadderQuote validates a quoteLadder response against the sampled inputs.
func toLadderQuote(points []*uint256.Int, result ladderResult) (ladderQuote, error) {
	if len(points) == 0 {
		return ladderQuote{}, nil
	}
	if result.Status > maxStatus || len(result.Outs) != len(points) || len(result.Fees) != len(points) ||
		slices.Contains(result.Outs, nil) {
		return ladderQuote{}, ErrInvalidState
	}
	return ladderQuote{outs: big256.MustFromBigs(result.Outs), status: result.Status}, nil
}

// collectQuotes keeps the quotable prefix: it stops at the first zero, decreasing or oversized output.
func collectQuotes(points []*uint256.Int, q ladderQuote) []Point {
	if q.status != statusOK && q.status != statusInsufficient {
		return nil
	}
	curve := make([]Point, 0, len(q.outs))
	for i, out := range q.outs {
		if out.IsZero() || out.Gt(maxPoint) || i > 0 && out.Lt(q.outs[i-1]) {
			break
		}
		curve = append(curve, Point{*points[i], *out})
	}
	return curve
}
