package lglclob

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})
	l.Info("Start getting new state")

	var orderBookRPC OrderBookRPC
	poolAddr := common.HexToAddress(p.Address)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    onchainClobHelperABI,
		Target: t.config.HelperAddress,
		Method: "assembleOrderbookFromOrders",
		Params: []any{poolAddr, false, bMaxPriceLevels},
	}, []any{&orderBookRPC.Bids}).AddCall(&ethrpc.Call{
		ABI:    onchainClobHelperABI,
		Target: t.config.HelperAddress,
		Method: "assembleOrderbookFromOrders",
		Params: []any{poolAddr, true, bMaxPriceLevels},
	}, []any{&orderBookRPC.Asks}).TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to aggregate RPC requests")
		return entity.Pool{}, err
	}

	orderBook := OrderBook{
		Bids: OrderBookLevels{
			ArrayPrices: orderBookRPC.Bids.ArrayPrices,
			ArrayShares: orderBookRPC.Bids.ArrayShares,
		},
		Asks: OrderBookLevels{
			ArrayPrices: orderBookRPC.Asks.ArrayPrices,
			ArrayShares: orderBookRPC.Asks.ArrayShares,
		},
	}
	extraBytes, err := json.Marshal(orderBook)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	var reserveX, reserveY uint256.Int
	for i, share := range orderBook.Bids.ArrayShares {
		reserveY.Add(&reserveY, reserveX.Mul(share, orderBook.Bids.ArrayPrices[i]))
	}
	reserveX.Clear()
	for _, share := range orderBook.Asks.ArrayShares {
		reserveX.Add(&reserveX, share)
	}

	p.Reserves = entity.PoolReserves{reserveX.Mul(&reserveX, staticExtra.ScalingFactorX).String(),
		reserveY.Mul(&reserveY, staticExtra.ScalingFactorY).String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	l.Info("Finish updating state of pool")
	return p, nil
}
