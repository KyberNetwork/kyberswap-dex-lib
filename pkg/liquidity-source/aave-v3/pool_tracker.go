package aavev3

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	staticExtra := StaticExtra{}
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to unmarshal staticExtra")
		return p, err
	}

	rpcData, err := d.getPoolData(ctx, staticExtra.AavePoolAddress, p.Tokens[0].Address, overrides)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(p, rpcData)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to updatePool")
		return p, err
	}

	return newPool, nil
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress,
	assetToken string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (RPCData, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var rpcData RPCData

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetConfiguration,
		Params: []any{common.HexToAddress(assetToken)},
	}, []any{&rpcData.Configuration})

	resp, err := req.Aggregate()
	if err != nil {
		return RPCData{}, err
	}

	rpcData.BlockNumber = resp.BlockNumber.Uint64()

	return rpcData, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, data RPCData) (entity.Pool, error) {
	extra := parseConfiguration(data.Configuration.Data.Data)
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return entity.Pool{}, err
	}

	isBlocked := !extra.IsActive && extra.IsFrozen && extra.IsPaused

	pool.Reserves = d.calculateReserves(pool, isBlocked)

	pool.BlockNumber = data.BlockNumber
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extraBytes)

	return pool, nil
}

func (d *PoolTracker) calculateReserves(pool entity.Pool, isBlocked bool) entity.PoolReserves {
	if isBlocked {
		return entity.PoolReserves{"0", "0"}
	}

	return entity.PoolReserves{
		strconv.Itoa(getReserve(pool.Tokens[0].Decimals)),
		strconv.Itoa(getReserve(pool.Tokens[1].Decimals)),
	}
}

func getReserve(decimals uint8) int {
	return max(100*int(math.Pow(10, float64(decimals))), defaultReserve)
}
