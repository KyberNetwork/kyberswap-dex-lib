package genericarm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}
	pools := make([]entity.Pool, 0, len(d.config.Arms))
	for armAddr, armCfg := range d.config.Arms {
		pool, err := d.getNewPool(ctx, armAddr, armCfg)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, *pool)
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")
	d.hasInitialized = true
	return pools, nil, nil
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, armAddr string, armCfg ArmCfg) (*entity.Pool, error) {
	poolState, err := fetchAssetAndState(ctx, d.ethrpcClient, armAddr, armCfg)
	if err != nil {
		return nil, err
	}

	extraBytes, err := json.Marshal(Extra{
		Gas:                Gas(armCfg.Gas),
		TradeRate0:         uint256.MustFromBig(poolState.TradeRate0),
		TradeRate1:         uint256.MustFromBig(poolState.TradeRate1),
		PriceScale:         uint256.MustFromBig(poolState.PriceScale),
		WithdrawsQueued:    uint256.MustFromBig(poolState.WithdrawsQueued),
		WithdrawsClaimed:   uint256.MustFromBig(poolState.WithdrawsClaimed),
		LiquidityAsset:     poolState.LiquidityAsset,
		SwapTypes:          armCfg.SwapType,
		ArmType:            armCfg.ArmType,
		HasWithdrawalQueue: armCfg.HasWithdrawalQueue,
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra")
		return nil, err
	}

	return &entity.Pool{
		Address:   armAddr,
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{poolState.Reserve0.String(), poolState.Reserve1.String()},
		Tokens: []*entity.PoolToken{
			{
				Address:   strings.ToLower(poolState.Token0.Hex()),
				Swappable: true,
			},
			{
				Address:   strings.ToLower(poolState.Token1.Hex()),
				Swappable: true,
			},
		},
		Extra: string(extraBytes),
	}, nil
}
