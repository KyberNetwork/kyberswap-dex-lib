package polmatic

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrFailedToGetReserves = errors.New("failed to get reserves")
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool": p.Address}).Debug("Start getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      t.config.DexID,
					"pool":        p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}).
			Debug("Finish getting new pool state")
	}()

	var (
		maticReserves   *big.Int
		polygonReserves *big.Int
	)

	poolAddress := common.HexToAddress(p.Address)

	getReserves := t.ethrpcClient.NewRequest().SetContext(ctx)
	getReserves.AddCall(
		&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{poolAddress},
		}, []interface{}{&maticReserves})
	getReserves.AddCall(
		&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{poolAddress},
		}, []interface{}{&polygonReserves})
	if _, err := getReserves.TryAggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": t.config.DexID,
					"poolAddress":     p.Address,
					"error":           err,
				}).
			Error("failed to get reserves")

		return p, ErrFailedToGetReserves
	}

	p.Reserves = []string{maticReserves.String(), polygonReserves.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
