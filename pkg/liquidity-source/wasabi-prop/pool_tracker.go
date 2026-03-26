package wasabiprop

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	reserves, blockNumber, err := t.fetchReserves(ctx, p.Address)
	if err != nil {
		return p, err
	}

	if reserves.BaseTokenReserves == nil || reserves.QuoteTokenReserves == nil {
		return p, ErrInsufficientLiquidity
	}

	tokenReserves := []*big.Int{reserves.BaseTokenReserves, reserves.QuoteTokenReserves}

	samples, err := t.fetchQuotes(ctx, p, tokenReserves, blockNumber)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, samples)
	t.applyBuffer(samples)

	samples = filterSamples(samples)

	p.Reserves = []string{
		reserves.BaseTokenReserves.String(),
		reserves.QuoteTokenReserves.String(),
	}

	extraBytes, err := json.Marshal(Extra{Samples: samples})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	tokenReserves []*big.Int,
	blockNumber *big.Int,
) ([][][2]*big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	samples := make([][][2]*big.Int, 2)

	for i := range p.Tokens {
		points := make([]*big.Int, 0, sampleSize+len(reserveSampleBps))

		// Power-of-10 levels (covers small to large orders of magnitude)
		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)
		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			points = append(points, bignumber.TenPowInt(k))
		}

		// Reserve-based levels (fine-grained coverage near the liquidity boundary)
		if tokenReserves != nil && tokenReserves[i] != nil && tokenReserves[i].Sign() > 0 {
			for _, bps := range reserveSampleBps {
				pt := new(big.Int).Mul(tokenReserves[i], big.NewInt(int64(bps)))
				pt.Div(pt, bignumber.BasisPoint)
				if pt.Sign() > 0 {
					points = append(points, pt)
				}
			}
		}

		// Sort and deduplicate
		sort.Slice(points, func(a, b int) bool {
			return points[a].Cmp(points[b]) < 0
		})
		points = dedupSorted(points)

		samples[i] = make([][2]*big.Int, len(points))
		for j, pt := range points {
			samples[i][j] = [2]*big.Int{new(big.Int).Set(pt), big.NewInt(0)}
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: "quoteExactInput",
				Params: []any{
					common.HexToAddress(p.Tokens[i].Address),
					samples[i][j][0],
				},
			}, []any{&samples[i][j][1]})
		}
	}

	if _, err := req.TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	return samples, nil
}

func (t *PoolTracker) applyBuffer(samples [][][2]*big.Int) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for i := range samples {
		for j := range samples[i] {
			if s1 := samples[i][j][1]; s1 != nil {
				s1.Mul(s1, buf)
				s1.Div(s1, bignumber.BasisPoint)
			}
		}
	}
}

func (t *PoolTracker) fetchReserves(ctx context.Context, poolAddr string) (getReservesResult, *big.Int, error) {
	var reserves getReservesResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: "getReserves",
	}, []any{&reserves})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return getReservesResult{}, nil, err
	}

	return reserves, res.BlockNumber, nil
}

func filterSamples(samples [][][2]*big.Int) [][][2]*big.Int {
	for dir := range samples {
		valid := samples[dir][:0]
		for _, s := range samples[dir] {
			if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
				continue
			}
			valid = append(valid, s)
		}
		samples[dir] = valid
	}
	return samples
}

func (t *PoolTracker) warnGapInQuotes(p entity.Pool, samples [][][2]*big.Int) {
	for dir := range samples {
		seenPositive := false
		zeroRunStart := -1

		for i := range samples[dir] {
			pt := samples[dir][i]
			if pt[0] == nil || pt[1] == nil {
				continue
			}
			if pt[1].Sign() > 0 {
				if zeroRunStart >= 0 {
					startAmt := samples[dir][zeroRunStart][0]
					endAmt := samples[dir][i-1][0]
					logger.WithFields(logger.Fields{
						"pool":           p.Address,
						"dir":            dir,
						"tokenIn":        p.Tokens[dir].Address,
						"tokenOut":       p.Tokens[1-dir].Address,
						"holeFromAmount": startAmt.String(),
						"holeToAmount":   endAmt.String(),
						"resumeAmount":   pt[0].String(),
					}).Warn("wasabi-prop quote gap detected (positive -> zero -> positive)")
					zeroRunStart = -1
				}
				seenPositive = true
				continue
			}
			if seenPositive && zeroRunStart < 0 {
				zeroRunStart = i
			}
		}
	}
}

// dedupSorted removes consecutive duplicates from a sorted []*big.Int slice.
func dedupSorted(sorted []*big.Int) []*big.Int {
	if len(sorted) <= 1 {
		return sorted
	}
	result := sorted[:1]
	for _, v := range sorted[1:] {
		if v.Cmp(result[len(result)-1]) != 0 {
			result = append(result, v)
		}
	}
	return result
}
