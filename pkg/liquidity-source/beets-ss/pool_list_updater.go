package beets_ss

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config         *Config
		ethrpcClient   *ethrpc.Client
		hasInitialized bool
	}
)

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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	pools, err := u.initPool()
	if err != nil {
		return nil, metadataBytes, err
	}

	u.hasInitialized = true

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) initPool() ([]entity.Pool, error) {
	return []entity.Pool{
		{
			Address:   Beets_Staked_Sonic_Address,
			Exchange:  string(valueobject.ExchangeBeetsSS),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{defaultReserve, defaultReserve},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic]),
					Name:      "Wrapped Sonic",
					Symbol:    "wS",
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   Beets_Staked_Sonic_Address,
					Name:      "Beets Staked Sonic",
					Symbol:    "stS",
					Decimals:  18,
					Swappable: true,
				},
			},
		},
	}, nil
}
