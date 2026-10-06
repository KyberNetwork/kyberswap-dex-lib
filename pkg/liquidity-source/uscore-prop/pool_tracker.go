package uscoreprop

import (
	"context"
	"math"
	"math/big"
	"sort"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct{ client *ethrpc.Client }
type reservesResult struct{ ReserveQuote, ReserveBase *big.Int }
type quoteResult struct {
	AmountOut, Fee *big.Int
	Status         uint8
}
type ladderResult struct {
	Outs, Fees []*big.Int
	Status     uint8
}

func NewPoolTracker(_ *Config, client *ethrpc.Client) *PoolTracker {
	return &PoolTracker{client: client}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.track(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateWithOverridesParams) (entity.Pool, error) {
	return t.track(ctx, p, params.Overrides)
}

func (t *PoolTracker) track(ctx context.Context, p entity.Pool, overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return p, ErrInvalidState
	}
	var seed reservesResult
	var probes [2]quoteResult
	var units [2]*big.Int
	req := t.client.R().SetContext(ctx).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "getReserves"}, []any{&seed})
	for i, token := range p.Tokens {
		if token.Decimals > 38 || !common.IsHexAddress(token.Address) {
			return p, ErrInvalidState
		}
		units[i] = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(token.Decimals)), nil)
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteExactIn", Params: []any{common.HexToAddress(token.Address), units[i]}}, []any{&probes[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return p, err
	}
	if seed.ReserveQuote == nil || seed.ReserveBase == nil {
		return p, ErrInvalidState
	}
	seedReserves := [2]*big.Int{seed.ReserveQuote, seed.ReserveBase}
	var points [2][]*big.Int
	for i := range 2 {
		basis := seedReserves[i]
		var minimum *big.Int
		if seedReserves[1-i].Sign() > 0 && probes[i].Status == 3 {
			rateIn, rateOut, low, err := t.smallerProbe(ctx, p.Address, p.Tokens[i].Address, units[i], overrides)
			if err != nil {
				return p, err
			}
			if rateOut != nil {
				minimum = low
				basis = new(big.Int).Mul(seedReserves[1-i], rateIn)
				basis.Div(basis, rateOut)
			}
		}
		if probes[i].Status == 0 && probes[i].AmountOut != nil && probes[i].AmountOut.Sign() > 0 {
			basis = new(big.Int).Mul(seedReserves[1-i], units[i])
			basis.Div(basis, probes[i].AmountOut)
		}
		maxInput := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
		if basis.Cmp(maxInput) > 0 {
			basis = maxInput
		}
		points[i] = ladder.BuildSamplePointsN(basis, ladder.SampleSize)
		if minimum != nil {
			filtered := []*big.Int{minimum}
			for _, amount := range points[i] {
				if amount.Cmp(minimum) > 0 {
					filtered = append(filtered, amount)
				}
			}
			points[i] = filtered
		}
	}

	var reserves reservesResult
	var quotes [2]ladderResult
	var timestamp *big.Int
	var blockNumber *big.Int
	var ladders [2][]ladder.Point
	for round := 0; ; round++ {
		reserves, quotes, timestamp = reservesResult{}, [2]ladderResult{}, nil
		req = t.client.R().SetContext(ctx).SetOverrides(overrides)
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "getReserves"}, []any{&reserves})
		req.AddCall(&ethrpc.Call{ABI: clockABI, Target: multicall3, Method: "getCurrentBlockTimestamp"}, []any{&timestamp})
		for i := range 2 {
			if len(points[i]) > 0 {
				req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteLadder", Params: []any{common.HexToAddress(p.Tokens[i].Address), points[i]}}, []any{&quotes[i]})
			}
		}
		resp, err := req.Aggregate()
		if err != nil {
			return p, err
		}
		if reserves.ReserveQuote == nil || reserves.ReserveBase == nil || timestamp == nil || !timestamp.IsInt64() || timestamp.Sign() <= 0 || resp.BlockNumber == nil || resp.BlockNumber.Sign() <= 0 {
			return p, ErrInvalidState
		}
		blockNumber = resp.BlockNumber
		for i := range 2 {
			ladders[i], err = collectQuotes(points[i], quotes[i])
			if err != nil {
				return p, err
			}
		}
		if round == maxRefinementRounds {
			break
		}
		changed := false
		for i := range 2 {
			refined := refineUpper(points[i], quotes[i])
			if round == 2 || len(refined) <= len(points[i]) {
				refined = refineCurve(points[i], quotes[i])
			}
			if len(refined) > len(points[i]) {
				points[i], changed = refined, true
			}
		}
		if !changed {
			break
		}
	}
	extra, err := json.Marshal(ladder.Extra{Ladders: ladders})
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserves.ReserveQuote.String(), reserves.ReserveBase.String()}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = timestamp.Int64()
	return p, nil
}

func refineUpper(points []*big.Int, result ladderResult) []*big.Int {
	if result.Status != 3 || len(points) >= maxSamplePoints {
		return nil
	}
	for i, out := range result.Outs {
		if out != nil && out.Sign() > 0 {
			if i > 0 && out.Cmp(result.Outs[i-1]) < 0 {
				return nil
			}
			continue
		}
		if i == 0 || i >= len(points) {
			return nil
		}
		lo, hi := points[i-1], points[i]
		gap := new(big.Int).Sub(hi, lo)
		refined := append([]*big.Int(nil), points[:i]...)
		for n := int64(1); n < 8 && len(refined)+len(points)-i < maxSamplePoints; n++ {
			x := new(big.Int).Mul(gap, big.NewInt(n))
			x.Div(x, big.NewInt(8)).Add(x, lo)
			if x.Cmp(refined[len(refined)-1]) > 0 && x.Cmp(hi) < 0 {
				refined = append(refined, x)
			}
		}
		return append(refined, points[i:]...)
	}
	return nil
}

func refineCurve(points []*big.Int, result ladderResult) []*big.Int {
	if len(points) == 0 || (result.Status != 0 && result.Status != 3) {
		return nil
	}
	refined := append([]*big.Int(nil), points...)
	type segment struct {
		lo, hi                  *big.Int
		width, rate, out, score float64
		parts                   int
	}
	var segments []segment
	for i, out := range result.Outs {
		if out == nil || out.Sign() <= 0 {
			break
		}
		lo, prev := big.NewInt(0), big.NewInt(0)
		if i > 0 {
			lo, prev = points[i-1], result.Outs[i-1]
		}
		width, _ := new(big.Int).Sub(points[i], lo).Float64()
		delta, _ := new(big.Int).Sub(out, prev).Float64()
		value, _ := out.Float64()
		segments = append(segments, segment{lo: lo, hi: points[i], width: width, rate: delta / width, out: value, parts: 1})
		if delta < 0 {
			break
		}
	}
	for i := range segments {
		s := &segments[i]
		for _, j := range []int{i - 1, i + 1} {
			if j >= 0 && j < len(segments) {
				s.score = math.Max(s.score, math.Abs(s.rate-segments[j].rate)*s.width/s.out)
			}
		}
		if i == 0 || new(big.Int).Sub(s.hi, s.lo).Cmp(big.NewInt(2)) < 0 {
			s.score = 0
		}
	}
	budget := maxSamplePoints - len(points)
	for budget > 0 {
		best := -1
		for i, s := range segments {
			if s.parts <= budget && s.score > 0.00001 && (best < 0 || s.score > segments[best].score) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		s := &segments[best]
		budget -= s.parts
		s.parts *= 2
		s.score /= 4
	}
	for _, s := range segments {
		gap := new(big.Int).Sub(s.hi, s.lo)
		for n := 1; n < s.parts; n++ {
			x := new(big.Int).Mul(gap, big.NewInt(int64(n)))
			x.Div(x, big.NewInt(int64(s.parts))).Add(x, s.lo)
			if x.Cmp(s.lo) > 0 && x.Cmp(s.hi) < 0 {
				refined = append(refined, x)
			}
		}
	}
	sort.Slice(refined, func(i, j int) bool { return refined[i].Cmp(refined[j]) < 0 })
	deduped := refined[:0]
	for _, x := range refined {
		if len(deduped) == 0 || x.Cmp(deduped[len(deduped)-1]) > 0 {
			deduped = append(deduped, x)
		}
	}
	return deduped
}

func (t *PoolTracker) smallerProbe(ctx context.Context, address, token string, unit *big.Int, overrides map[common.Address]gethclient.OverrideAccount) (rateIn, rateOut, minimum *big.Int, err error) {
	var amounts []*big.Int
	for amount := new(big.Int).Rsh(new(big.Int).Set(unit), 1); amount.Sign() > 0; amount.Rsh(amount, 1) {
		amounts = append(amounts, new(big.Int).Set(amount))
	}
	if len(amounts) == 0 {
		return nil, nil, nil, nil
	}
	results := make([]quoteResult, len(amounts))
	req := t.client.R().SetContext(ctx).SetOverrides(overrides)
	for i, amount := range amounts {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: "quoteExactIn", Params: []any{common.HexToAddress(token), amount}}, []any{&results[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, nil, err
	}
	for i, result := range results {
		if result.Status == 0 && result.AmountOut != nil && result.AmountOut.Sign() > 0 {
			if rateIn == nil {
				rateIn, rateOut = amounts[i], result.AmountOut
			}
			minimum = amounts[i]
		}
	}
	return rateIn, rateOut, minimum, nil
}

func collectQuotes(points []*big.Int, result ladderResult) ([]ladder.Point, error) {
	if len(points) == 0 {
		return nil, nil
	}
	if result.Status > 5 || len(result.Outs) != len(points) || len(result.Fees) != len(points) {
		return nil, ErrInvalidState
	}
	if result.Status != 0 && result.Status != 3 {
		return nil, nil
	}
	for i, out := range result.Outs {
		if out == nil || out.Sign() <= 0 || (i > 0 && out.Cmp(result.Outs[i-1]) < 0) {
			return ladder.CollectLadder(points[:i], result.Outs[:i]), nil
		}
	}
	return ladder.CollectLadder(points, result.Outs), nil
}
