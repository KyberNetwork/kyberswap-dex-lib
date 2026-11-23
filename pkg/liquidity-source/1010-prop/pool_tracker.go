package prop

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"golang.org/x/sync/singleflight"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	defaultReservesCacheTTL     = 200 * time.Millisecond
	defaultReservesFetchTimeout = 5 * time.Second
)

type reservesSnapshot struct {
	balanceByAddr map[string]*big.Int // read-only after build
	blockNumber   *big.Int
	fetchedAt     time.Time
}

type reservesFetchFunc func(ctx context.Context, routerAddr string) (*reservesSnapshot, error)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	cacheMu       sync.RWMutex
	reservesCache map[string]*reservesSnapshot
	reservesGroup singleflight.Group

	fetchReserves reservesFetchFunc
	now           func() time.Time
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	t := &PoolTracker{
		cfg:           cfg,
		ethrpcClient:  ethrpcClient,
		reservesCache: make(map[string]*reservesSnapshot),
		now:           time.Now,
	}
	t.fetchReserves = t.fetchRouterReserves
	return t
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

// fetchBalances returns the two balances that correspond to the pool's token pair,
// in pool token order, backed by a short-lived per-router reserves snapshot.
func (t *PoolTracker) fetchBalances(ctx context.Context, routerAddr string, tokens []*entity.PoolToken) ([]*big.Int, *big.Int, error) {
	snap, err := t.getReservesSnapshot(ctx, routerAddr)
	if err != nil {
		return nil, nil, err
	}

	balances := make([]*big.Int, len(tokens))
	for i, tok := range tokens {
		bal := snap.balanceByAddr[strings.ToLower(tok.Address)]
		if bal == nil {
			return nil, nil, ErrInsufficientLiquidity
		}
		balances[i] = new(big.Int).Set(bal)
	}

	var blockNumber *big.Int
	if snap.blockNumber != nil {
		blockNumber = new(big.Int).Set(snap.blockNumber)
	}

	return balances, blockNumber, nil
}

func (t *PoolTracker) reservesCacheTTL() time.Duration {
	if t.cfg == nil || t.cfg.ReservesCacheTTLMs == 0 {
		return defaultReservesCacheTTL
	}
	if t.cfg.ReservesCacheTTLMs < 0 {
		return -1
	}
	return time.Duration(t.cfg.ReservesCacheTTLMs) * time.Millisecond
}

func (t *PoolTracker) currentTime() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

func (t *PoolTracker) getReservesFetcher() reservesFetchFunc {
	if t.fetchReserves != nil {
		return t.fetchReserves
	}
	return t.fetchRouterReserves
}

func (t *PoolTracker) getCachedReserves(key string, ttl time.Duration) *reservesSnapshot {
	t.cacheMu.RLock()
	snap := t.reservesCache[key]
	t.cacheMu.RUnlock()
	if snap != nil && t.currentTime().Sub(snap.fetchedAt) < ttl {
		return snap
	}
	return nil
}

func (t *PoolTracker) getReservesSnapshot(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ttl := t.reservesCacheTTL()
	if ttl < 0 {
		return t.getReservesFetcher()(ctx, routerAddr)
	}

	key := strings.ToLower(routerAddr)
	if snap := t.getCachedReserves(key, ttl); snap != nil {
		return snap, nil
	}

	ch := t.reservesGroup.DoChan(key, func() (any, error) {
		if snap := t.getCachedReserves(key, ttl); snap != nil {
			return snap, nil
		}

		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultReservesFetchTimeout)
		defer cancel()

		snap, err := t.getReservesFetcher()(fetchCtx, routerAddr)
		if err != nil {
			return nil, err
		}
		if snap == nil {
			return nil, errors.New("nil reserves snapshot")
		}

		cached := &reservesSnapshot{
			balanceByAddr: snap.balanceByAddr,
			blockNumber:   snap.blockNumber,
			fetchedAt:     t.currentTime(),
		}

		t.cacheMu.Lock()
		if t.reservesCache == nil {
			t.reservesCache = make(map[string]*reservesSnapshot)
		}
		t.reservesCache[key] = cached
		t.cacheMu.Unlock()

		return cached, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*reservesSnapshot), nil
	}
}

func (t *PoolTracker) fetchRouterReserves(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
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
		return nil, err
	}

	balanceByAddr := make(map[string]*big.Int, len(assetReserves.Tokens))
	for i, addr := range assetReserves.Tokens {
		if i < len(assetReserves.Balances) {
			balanceByAddr[hexutil.Encode(addr[:])] = assetReserves.Balances[i]
		}
	}

	return &reservesSnapshot{
		balanceByAddr: balanceByAddr,
		blockNumber:   res.BlockNumber,
	}, nil
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
	p.Reserves = ladder.ZeroUnquotedReserves(entity.PoolReserves{r0.Dec(), r1.Dec()}, extra.Ladders)
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p
}
