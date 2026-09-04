package tidefiprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// noLimitPrice is passed as TideFi's limitPrice cap when sampling: the docs
// specify type(uint256).max as "no limit", which lets the sampled ladder
// reflect the engine's own pricing/inventory caps rather than an arbitrary
// probe-time price ceiling.
var noLimitPrice = bignumber.MaxUint256

type quoteResult struct {
	AmountIn  *big.Int `abi:"amountIn"`
	AmountOut *big.Int `abi:"amountOut"`
}

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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	balances, blockNumber, err := t.fetchBalances(ctx, p.Tokens)
	if err != nil {
		return p, err
	}

	points := [2][]*big.Int{
		ladder.SamplePoints(p, 0, balances[0], balances[1]),
		ladder.SamplePoints(p, 1, balances[1], balances[0]),
	}

	results, err := t.fetchQuotes(ctx, p, staticExtra.Address, points, blockNumber)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, results)
	t.applyBuffer(results)

	ladders := [2][]ladder.Point{
		collectLadder(results[0]),
		collectLadder(results[1]),
	}

	r0, r1 := uint256.MustFromBig(balances[0]), uint256.MustFromBig(balances[1])
	return t.persist(p, ladder.Extra{Ladders: ladders}, r0, r1, blockNumber), nil
}

// fetchBalances reads each pool token's balanceOf(swapper) directly: TideFi
// is a single global contract backing every pair, not a per-pool
// router/lens, so on-chain reserves are just the swapper's own token
// balances.
func (t *PoolTracker) fetchBalances(
	ctx context.Context, tokens []*entity.PoolToken,
) ([]*big.Int, *big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	balances := make([]*big.Int, len(tokens))
	for i, tok := range tokens {
		balances[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: tok.Address,
			Method: "balanceOf",
			Params: []any{common.HexToAddress(t.cfg.Address)},
		}, []any{&balances[i]})
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return balances, res.BlockNumber, nil
}

// fetchQuotes probes the swapper's quote(tokenIn, tokenOut, maxAmountIn,
// limitPrice) for every point in each direction's grid, returning the raw
// (amountIn, amountOut) results aligned index-for-index with points.
// amountIn can come back below the requested maxAmountIn -- TideFi's quote
// stops filling once the marginal price would exceed limitPrice or the
// engine's own inventory runs out -- so the returned amountIn (not the
// requested point) is what must be used as the ladder's x-value.
func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	swapperAddr string,
	points [2][]*big.Int,
	blockNumber *big.Int,
) ([2][]quoteResult, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	tokenAddr := [2]common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	var results [2][]quoteResult
	callCount := 0
	for dir := range points {
		tokenIn, tokenOut := tokenAddr[dir], tokenAddr[1-dir]
		results[dir] = make([]quoteResult, len(points[dir]))
		for i, pt := range points[dir] {
			req.AddCall(&ethrpc.Call{
				ABI:    swapperABI,
				Target: swapperAddr,
				Method: "quote",
				Params: []any{tokenIn, tokenOut, pt, noLimitPrice},
			}, []any{&results[dir][i]})
			callCount++
		}
	}
	if callCount == 0 {
		return results, nil
	}

	if _, err := req.TryAggregate(); err != nil {
		return [2][]quoteResult{}, err
	}

	return results, nil
}

func (t *PoolTracker) applyBuffer(results [2][]quoteResult) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for dir := range results {
		for _, r := range results[dir] {
			if r.AmountOut == nil {
				continue
			}
			r.AmountOut.Mul(r.AmountOut, buf)
			r.AmountOut.Div(r.AmountOut, bignumber.BasisPoint)
		}
	}
}

func (t *PoolTracker) warnGapInQuotes(p entity.Pool, results [2][]quoteResult) {
	for dir := range results {
		seenPositive := false
		zeroRunStart := -1

		for i, r := range results[dir] {
			out := r.AmountOut
			if out == nil {
				continue
			}
			if out.Sign() > 0 {
				if zeroRunStart >= 0 {
					logger.WithFields(logger.Fields{
						"pool":     p.Address,
						"dir":      dir,
						"tokenIn":  p.Tokens[dir].Address,
						"tokenOut": p.Tokens[1-dir].Address,
						"holeAt":   zeroRunStart,
						"resumeAt": i,
					}).Warn("tidefi-prop quote gap detected (positive -> zero -> positive)")
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

// collectLadder pairs each quote result's actual returned amountIn with its
// amountOut, dropping any point that reverted or returned zero. Unlike
// ladder.CollectLadder, the x-value comes from the result itself, not the
// requested probe point, since TideFi's quote can cap amountIn below what
// was asked.
func collectLadder(results []quoteResult) []ladder.Point {
	pts := make([]ladder.Point, 0, len(results))
	for _, r := range results {
		if r.AmountIn == nil || r.AmountOut == nil || r.AmountIn.Sign() <= 0 || r.AmountOut.Sign() <= 0 {
			continue
		}
		inF, _ := r.AmountIn.Float64()
		outF, _ := r.AmountOut.Float64()
		pts = append(pts, ladder.Point{inF, outF})
	}
	return pts
}

func (t *PoolTracker) persist(p entity.Pool, extra ladder.Extra, r0, r1 *uint256.Int, blockNumber *big.Int) entity.Pool {
	extraBytes, _ := json.Marshal(extra)
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{r0.Dec(), r1.Dec()}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p
}
