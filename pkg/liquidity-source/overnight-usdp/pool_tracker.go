package overnightusdp

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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
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
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	var (
		isPaused          bool
		buyFee, redeemFee *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: t.config.Exchange,
		Method: exchangeMethodPaused,
		Params: nil,
	}, []interface{}{&isPaused})

	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: t.config.Exchange,
		Method: exchangeMethodBuyFee,
		Params: nil,
	}, []interface{}{&buyFee})

	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: t.config.Exchange,
		Method: exchangeMethodRedeemFee,
		Params: nil,
	}, []interface{}{&redeemFee})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(&Extra{
		IsPaused:  isPaused,
		BuyFee:    buyFee,
		RedeemFee: redeemFee,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      t.config.DexID,
				"pool_id":     p.Address,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return p, nil
}
