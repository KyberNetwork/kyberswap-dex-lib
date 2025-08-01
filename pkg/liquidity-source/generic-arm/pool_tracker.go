package genericarm

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
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
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	poolState, err := fetchAssetAndState(ctx, t.ethrpcClient, p.Address, t.config.Arms[p.Address])
	if err != nil {
		return p, err
	}

	extra := Extra{
		TradeRate0:         uint256.MustFromBig(poolState.TradeRate0),
		TradeRate1:         uint256.MustFromBig(poolState.TradeRate1),
		PriceScale:         uint256.MustFromBig(poolState.PriceScale),
		WithdrawsQueued:    uint256.MustFromBig(poolState.WithdrawsQueued),
		WithdrawsClaimed:   uint256.MustFromBig(poolState.WithdrawsClaimed),
		LiquidityAsset:     poolState.LiquidityAsset,
		SwapTypes:          t.config.Arms[p.Address].SwapType,
		ArmType:            t.config.Arms[p.Address].ArmType,
		HasWithdrawalQueue: t.config.Arms[p.Address].HasWithdrawalQueue,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		poolState.Reserve0.String(),
		poolState.Reserve1.String(),
	}

	p.Timestamp = time.Now().Unix()
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)
	return p, nil
}
