package wasabiprop

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

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
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	// Fetch reserves first so we can generate reserve-aware sample points
	var reserves getReservesResult
	reqRes := t.ethrpcClient.NewRequest().SetContext(ctx)
	reqRes.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "getReserves",
	}, []any{&reserves})

	resBlock, err := reqRes.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	if reserves.BaseTokenReserves == nil || reserves.QuoteTokenReserves == nil {
		return p, ErrInsufficientLiquidity
	}

	tokenReserves := []*big.Int{reserves.BaseTokenReserves, reserves.QuoteTokenReserves}

	// Build sample points: power-of-10 levels + reserve-based levels
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resBlock.BlockNumber)
	samples := make([][][2]*big.Int, 2)
	for i := range p.Tokens {
		points := make([]*big.Int, 0, sampleSize+len(reserveSampleBps))

		// Power-of-10 levels (covers small to large orders of magnitude)
		start := lo.Ternary(p.Tokens[i].Decimals < sampleSize/2, 0, p.Tokens[i].Decimals-sampleSize/2)
		for k := start; k <= start+sampleSize-1; k++ {
			points = append(points, bignumber.TenPowInt(k))
		}

		// Reserve-based levels (fine-grained coverage near the liquidity boundary)
		for _, bps := range reserveSampleBps {
			pt := new(big.Int).Mul(tokenReserves[i], big.NewInt(int64(bps)))
			pt.Div(pt, bignumber.BasisPoint)
			if pt.Sign() > 0 {
				points = append(points, pt)
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

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	if t.cfg.Buffer > 0 {
		buf := big.NewInt(t.cfg.Buffer)
		for i := range samples {
			for j := range samples[i] {
				if samples[i][j][1] != nil {
					samples[i][j][1].Mul(samples[i][j][1], buf)
					samples[i][j][1].Div(samples[i][j][1], bignumber.BasisPoint)
				}
			}
		}
	}

	// Filter out failed samples (nil outputs from reverted on-chain calls)
	for i := range samples {
		valid := samples[i][:0]
		for _, s := range samples[i] {
			if s[0] != nil && s[1] != nil {
				valid = append(valid, s)
			}
		}
		samples[i] = valid
	}

	extra := Extra{Samples: samples}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{
		reserves.BaseTokenReserves.String(),
		reserves.QuoteTokenReserves.String(),
	}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = res.BlockNumber.Uint64()

	return p, nil
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
