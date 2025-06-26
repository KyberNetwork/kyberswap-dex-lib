package v2

import (
	"context"
	"log"
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

	extra, blockNumber, err := d.getPoolExtraData(ctx, p.Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to getPoolExtraData")
		return p, err
	}

	newPool, err := d.updatePool(p, extra, blockNumber)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to updatePool")
		return p, err
	}

	return newPool, nil
}

func (d *PoolTracker) getPoolExtraData(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var extra Extra

	req.AddCall(&ethrpc.Call{
		ABI:    cTokenABI,
		Target: poolAddress,
		Method: CTokenMethodExchangeRateStored,
		Params: nil,
	}, []any{&extra.ExchangeRateStored})

	resp, err := req.Aggregate()
	if err != nil {
		log.Fatalln(poolAddress)
		return Extra{}, 0, err
	}

	return extra, resp.BlockNumber.Uint64(), nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, extra Extra, blockNumber uint64) (entity.Pool, error) {
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Reserves = entity.PoolReserves{
		strconv.Itoa(max(100*int(pool.Tokens[0].Decimals), defaultReserve)),
		strconv.Itoa(max(100*int(pool.Tokens[1].Decimals), defaultReserve)),
	}

	pool.BlockNumber = blockNumber
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extraBytes)

	return pool, nil
}
