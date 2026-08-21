package prop

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	balances, blockNumber, err := t.fetchBalances(ctx, staticExtra.RouterAddress, p.Tokens)
	if err != nil {
		return p, err
	}

	points := [2][]*big.Int{
		ladder.SamplePoints(p, 0, balances[0], balances[1]),
		ladder.SamplePoints(p, 1, balances[1], balances[0]),
	}

	outputs, err := t.fetchQuotes(ctx, p, staticExtra.RouterAddress, points, blockNumber)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, points, outputs)
	t.applyBuffer(outputs)

	ladders := [2][]ladder.Point{
		ladder.CollectLadder(points[0], outputs[0]),
		ladder.CollectLadder(points[1], outputs[1]),
	}

	r0, r1 := uint256.MustFromBig(balances[0]), uint256.MustFromBig(balances[1])
	return t.persist(p, ladder.Extra{Ladders: ladders}, r0, r1, blockNumber), nil
}

// fetchBalances calls getAssetReserves on the router and returns the two balances
// that correspond to the pool's token pair, in pool token order.
func (t *PoolTracker) fetchBalances(ctx context.Context, routerAddr string, tokens []*entity.PoolToken) ([]*big.Int, *big.Int, error) {
	var assetReserves AssetReserves
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: routerAddr,
		Method: "getAssetReserves",
		Params: nil,
	}, []any{&assetReserves})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	balanceByAddr := make(map[string]*big.Int, len(assetReserves.Tokens))
	for i, addr := range assetReserves.Tokens {
		if i < len(assetReserves.Balances) {
			balanceByAddr[strings.ToLower(hexutil.Encode(addr[:]))] = assetReserves.Balances[i]
		}
	}

	balances := make([]*big.Int, len(tokens))
	for i, tok := range tokens {
		bal := balanceByAddr[strings.ToLower(tok.Address)]
		if bal == nil {
			return nil, nil, ErrInsufficientLiquidity
		}
		balances[i] = bal
	}

	return balances, res.BlockNumber, nil
}

// fetchQuotes probes the router's quote(account, tokenIn, tokenOut, amountIn) for every
// point in each direction's grid, returning the raw (possibly nil, on revert) outputs
// aligned index-for-index with points.
func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	routerAddr string,
	points [2][]*big.Int,
	blockNumber *big.Int,
) ([2][]*big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	account := common.Address{}
	tokenAddr := [2]common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	var outputs [2][]*big.Int
	callCount := 0
	for dir := range points {
		tokenIn, tokenOut := tokenAddr[dir], tokenAddr[1-dir]
		outputs[dir] = make([]*big.Int, len(points[dir]))
		for i, pt := range points[dir] {
			outputs[dir][i] = new(big.Int)
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: routerAddr,
				Method: "quote",
				Params: []any{account, tokenIn, tokenOut, pt},
			}, []any{&outputs[dir][i]})
			callCount++
		}
	}
	if callCount == 0 {
		return outputs, nil
	}

	if _, err := req.TryAggregate(); err != nil {
		return [2][]*big.Int{}, err
	}

	return outputs, nil
}

func (t *PoolTracker) applyBuffer(outputs [2][]*big.Int) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for dir := range outputs {
		for _, out := range outputs[dir] {
			if out == nil {
				continue
			}
			out.Mul(out, buf)
			out.Div(out, bignumber.BasisPoint)
		}
	}
}

func (t *PoolTracker) warnGapInQuotes(p entity.Pool, points, outputs [2][]*big.Int) {
	for dir := range outputs {
		pts, outs := points[dir], outputs[dir]
		seenPositive := false
		zeroRunStart := -1

		for i, out := range outs {
			if out == nil {
				continue
			}
			if out.Sign() > 0 {
				if zeroRunStart >= 0 {
					logger.WithFields(logger.Fields{
						"pool":           p.Address,
						"dir":            dir,
						"tokenIn":        p.Tokens[dir].Address,
						"tokenOut":       p.Tokens[1-dir].Address,
						"holeFromAmount": pts[zeroRunStart].String(),
						"holeToAmount":   pts[i-1].String(),
						"resumeAmount":   pts[i].String(),
					}).Warn("1010-prop quote gap detected (positive -> zero -> positive)")
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
