package liquidcore

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("getting new state for pool %v", p.Address)
	defer logger.Infof("finished getting new state for pool %v", p.Address)

	var reserves struct {
		Reserve0 *big.Int
		Reserve1 *big.Int
	}

	token0, token1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	resp, err := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getReserves",
		}, []any{&reserves}).
		TryBlockAndAggregate()
	if err != nil {
		logger.Errorf("failed to get reserves: %v", err)
		return p, err
	}

	points0 := t.samplePoints(p, 0, reserves.Reserve0, reserves.Reserve1)
	points1 := t.samplePoints(p, 1, reserves.Reserve1, reserves.Reserve0)
	ladders, err := t.probeQuotes(ctx, p.Address, overrides, token0, token1, points0, points1)
	if err != nil {
		logger.Errorf("failed to probe quotes: %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(ladder.Extra{Ladders: ladders})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{reserves.Reserve0.String(), reserves.Reserve1.String()}

	return p, nil
}

// samplePoints builds this cycle's probe grid for one direction (dir=0 is
// token0->token1, so currentInputReserve/currentOutputReserve are
// reserve0/reserve1; dir=1 is the reverse). It prefers
// ladder.EstimateNearCapacityAmount, guided by the previous cycle's ladder
// and output-side reserve already sitting in p (no extra calls -- just
// unmarshalling), and falls back to the input-side reserve directly when
// there's nothing to guide from (first probe, or the previous ladder never
// got close to depletion).
func (t *PoolTracker) samplePoints(p entity.Pool, dir int, currentInputReserve, currentOutputReserve *big.Int) []*big.Int {
	if nearCap := t.estimateNearCapacityAmount(p, dir, currentOutputReserve); nearCap != nil {
		return ladder.BuildSamplePointsFrom(nearCap, ladder.SampleSize)
	}
	return ladder.BuildSamplePoints(currentInputReserve)
}

func (t *PoolTracker) estimateNearCapacityAmount(p entity.Pool, dir int, currentOutputReserve *big.Int) *big.Int {
	if p.Extra == "" || len(p.Reserves) != 2 {
		return nil
	}

	var prevExtra ladder.Extra
	if err := json.Unmarshal([]byte(p.Extra), &prevExtra); err != nil {
		return nil
	}

	// dir=0 (token0->token1) output is token1, reserve index 1; dir=1 the
	// reverse.
	prevOutputReserve, ok := new(big.Int).SetString(p.Reserves[1-dir], 10)
	if !ok {
		return nil
	}

	return ladder.EstimateNearCapacityAmount(prevExtra.Ladders[dir], prevOutputReserve, currentOutputReserve)
}

func (t *PoolTracker) probeQuotes(
	ctx context.Context,
	poolAddr string,
	overrides map[common.Address]gethclient.OverrideAccount,
	token0, token1 common.Address,
	points0, points1 []*big.Int,
) ([2][]ladder.Point, error) {
	var amountsOut0, amountsOut1 []*big.Int

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	if len(points0) > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddr,
			Method: "estimateSwapBatch",
			Params: []any{token0, token1, points0},
		}, []any{&amountsOut0})
	}
	if len(points1) > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddr,
			Method: "estimateSwapBatch",
			Params: []any{token1, token0, points1},
		}, []any{&amountsOut1})
	}
	if len(points0) == 0 && len(points1) == 0 {
		return [2][]ladder.Point{}, nil
	}
	if _, err := req.TryAggregate(); err != nil {
		return [2][]ladder.Point{}, err
	}

	return [2][]ladder.Point{
		ladder.CollectLadder(points0, amountsOut0),
		ladder.CollectLadder(points1, amountsOut1),
	}, nil
}
