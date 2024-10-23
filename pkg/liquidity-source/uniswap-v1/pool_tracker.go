package uniswapv1

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	reserves, blockNumber, err := d.getReserves(ctx, p.Address, p.Tokens)
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
		newReserves = append(newReserves, reserve.String())
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

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, tokens []*entity.PoolToken) ([]*big.Int, *big.Int, error) {
	var reserves = make([]*big.Int, len(tokens))

	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		if strings.EqualFold(token.Address, valueobject.WETHByChainID[d.config.ChainID]) {
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
		panic(err)
		return nil, nil, err
	}

	return reserves, resp.BlockNumber, nil
}
