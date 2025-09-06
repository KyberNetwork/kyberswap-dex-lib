package xpress

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/logger"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

const (
	maxPriceLevels = 50
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*OrderBook, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexId,
	})
	l.Info("Start fetching RPC data of onchainclob pool")

	result := &OrderBook{
		Bids: OrderBookLevels{
			ArrayPrices: make([]*big.Int, 0, maxPriceLevels),
			ArrayShares: make([]*big.Int, 0, maxPriceLevels),
		},
		Asks: OrderBookLevels{
			ArrayPrices: make([]*big.Int, 0, maxPriceLevels),
			ArrayShares: make([]*big.Int, 0, maxPriceLevels),
		},
	}

	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    onchainClobHelperABI,
		Target: t.config.HelperAddress,
		Method: "assembleOrderbookFromOrders",
		Params: []any{common.HexToAddress(p.Address), false, big.NewInt(int64(maxPriceLevels))},
	}, []any{&result.Bids})

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    onchainClobHelperABI,
		Target: t.config.HelperAddress,
		Method: "assembleOrderbookFromOrders",
		Params: []any{common.HexToAddress(p.Address), true, big.NewInt(int64(maxPriceLevels))},
	}, []any{&result.Asks})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to aggregate RPC requests")
		return nil, err
	}

	return result, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexId,
	})
	l.Info("Start getting new state of onchainclob pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		orderBook *OrderBook
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		orderBook, err = t.FetchRPCData(ctx, &p, 0)
		if err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(orderBook)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber

	l.Infof("Finish updating state of pool")
	return p, nil
}
