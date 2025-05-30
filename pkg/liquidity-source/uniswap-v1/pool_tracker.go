package uniswapv1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	reserves, blockNumber, err := d.getReserves(ctx, p.Address, p.Tokens, overrides)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	oldReserves := p.Reserves

	newReserves := make(entity.PoolReserves, 0, len(reserves))
	for _, reserve := range reserves {
		if reserve == nil {
			newReserves = append(newReserves, "0")
		} else {
			newReserves = append(newReserves, reserve.String())
		}
	}

	p.Reserves = newReserves
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      oldReserves,
				"new_reserve":      p.Reserves,
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return p, nil
}

func (d *PoolTracker) getReserves(
	ctx context.Context,
	poolAddress string,
	tokens []*entity.PoolToken,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*big.Int, *big.Int, error) {
	var reserves = make([]*big.Int, len(tokens))

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	for i, token := range tokens {
		if valueobject.IsWrappedNative(token.Address, d.config.ChainID) {
			req.AddCall(&ethrpc.Call{
				ABI:    multicallABI,
				Target: d.config.MulticallContractAddress,
				Method: multicallGetEthBalanceMethod,
				Params: []interface{}{common.HexToAddress(poolAddress)},
			}, []interface{}{&reserves[i]})
		} else {
			req.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: token.Address,
				Method: erc20BalanceOfMethod,
				Params: []interface{}{common.HexToAddress(poolAddress)},
			}, []interface{}{&reserves[i]})
		}
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return reserves, resp.BlockNumber, nil
}
