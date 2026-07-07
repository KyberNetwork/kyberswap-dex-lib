package stable

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/goccy/go-json"
)

type PoolTracker struct {
	curvePoolTracker *curve.PoolTracker
	config           *curve.Config
	ethrpcClient     *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *curve.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	curvePoolTracker, err := curve.NewPoolTracker(cfg, ethrpcClient)
	if err != nil {
		return nil, err
	}
	return &PoolTracker{
		curvePoolTracker: curvePoolTracker,
		config:           cfg,
		ethrpcClient:     ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	newPoolState, err := d.curvePoolTracker.GetNewPoolState(ctx, p, params)
	if err != nil {
		return entity.Pool{}, err
	}

	return d.fetchOraclePrices(ctx, newPoolState)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	newPoolState, err := d.curvePoolTracker.GetNewPoolStateWithOverrides(ctx, p, params)
	if err != nil {
		return entity.Pool{}, err
	}

	return d.fetchOraclePrices(ctx, newPoolState)
}

func (d *PoolTracker) fetchOraclePrices(
	ctx context.Context,
	p entity.Pool,
) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return entity.Pool{}, err
	}

	extra.PriceDiffThreshold = [2]*big.Int{}

	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "fetchOraclePrice",
		Params: nil,
	}, []any{&extra.OraclePrices})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "price0DiffThreshold",
		Params: nil,
	}, []any{&extra.PriceDiffThreshold[0]})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "price1DiffThreshold",
		Params: nil,
	}, []any{&extra.PriceDiffThreshold[1]})

	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)

	return p, nil
}
