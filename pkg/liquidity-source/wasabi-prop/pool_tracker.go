package wasabiprop

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	propamm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/prop-amm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPoolState fetches on-chain quotes to build a discrete price curve (Samples).
//
// Strategy:
//   - Cold start (1st run): broad 10^k grid → refine around cap boundary (2 RPC rounds).
//   - Incremental (subsequent runs): log-spaced points in previous valid range + edge probes (1 RPC round).
//
// Only samples with amountOut > 0 are kept — they define the tradeable range.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var prevExtra Extra
	if p.Extra != "" {
		_ = json.Unmarshal([]byte(p.Extra), &prevExtra)
	}

	// Round 1: query on-chain quotes at chosen sample points.
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	samples := make([][][2]*big.Int, len(p.Tokens))
	incremental := make([]bool, len(p.Tokens))

	for i := range p.Tokens {
		prevMin, prevMax := propamm.ValidRangeFromSamples(prevExtra.Samples, i)
		incremental[i] = prevMin != nil && prevMax != nil && prevMax.Cmp(prevMin) > 0

		for _, amt := range propamm.BuildQueryPoints(p.Tokens[i].Decimals, prevMin, prevMax) {
			samples[i] = append(samples[i], [2]*big.Int{amt, new(big.Int)})
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: "quoteExactInput",
				Params: []any{common.HexToAddress(p.Tokens[i].Address), amt},
			}, []any{&samples[i][len(samples[i])-1][1]})
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	propamm.ApplyBuffer(samples, t.cfg.Buffer)

	// Round 2 (cold start only): refine between last-positive and first-zero to narrow cap.
	if err := t.refineColdStartCap(ctx, res.BlockNumber, p, samples, incremental); err != nil {
		return p, err
	}

	for i := range samples {
		samples[i] = propamm.CleanSamples(samples[i])
	}

	extraBytes, err := json.Marshal(Extra{Samples: samples})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	var reserves getReservesResult
	reqRes := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	reqRes.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "getReserves",
	}, []any{&reserves})
	if _, err := reqRes.Call(); err != nil {
		return p, err
	}
	if reserves.BaseTokenReserves == nil || reserves.QuoteTokenReserves == nil {
		return p, ErrInsufficientLiquidity
	}

	p.Reserves = []string{reserves.BaseTokenReserves.String(), reserves.QuoteTokenReserves.String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = res.BlockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) refineColdStartCap(
	ctx context.Context, blockNumber *big.Int, p entity.Pool,
	samples [][][2]*big.Int, incremental []bool,
) error {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	refined := make([][][2]*big.Int, len(samples))
	hasCall := false

	for dir := range samples {
		if dir < len(incremental) && incremental[dir] {
			continue
		}
		points := propamm.RefineCapPoints(propamm.FindCapBoundary(samples[dir]))
		if len(points) == 0 {
			continue
		}

		refined[dir] = lo.Map(points, func(amt *big.Int, _ int) [2]*big.Int {
			return [2]*big.Int{amt, new(big.Int)}
		})
		for j := range refined[dir] {
			hasCall = true
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: "quoteExactInput",
				Params: []any{common.HexToAddress(p.Tokens[dir].Address), refined[dir][j][0]},
			}, []any{&refined[dir][j][1]})
		}
	}

	if !hasCall {
		return nil
	}
	if _, err := req.TryAggregate(); err != nil {
		return err
	}
	for dir := range refined {
		samples[dir] = append(samples[dir], lo.Filter(refined[dir], func(r [2]*big.Int, _ int) bool {
			return r[0] != nil && r[1] != nil && r[1].Sign() > 0
		})...)
	}
	return nil
}
