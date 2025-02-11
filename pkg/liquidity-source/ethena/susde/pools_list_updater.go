package susde

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	if u.initialized {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	pool := entity.Pool{
		Address:  StakedUSDeV2,
		Exchange: string(valueobject.ExchangeEthenaSusde),
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   USDe,
				Name:      "USDe",
				Symbol:    "USDe",
				Decimals:  18,
				Weight:    1,
				Swappable: true,
			},
			{
				Address:   StakedUSDeV2,
				Name:      "Staked USDe",
				Symbol:    "sUSDe",
				Decimals:  18,
				Weight:    1,
				Swappable: true,
			},
		},
	}

	u.initialized = true

	return []entity.Pool{pool}, metadataBytes, nil
}
