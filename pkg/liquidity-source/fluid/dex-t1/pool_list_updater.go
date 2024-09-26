package dexT1

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
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

	allPools, err := u.getAllPools(ctx)

	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0)

	for _, curPool := range allPools {

		extra := PoolExtra{
			CollateralReserves: curPool.CollateralReserves,
			DebtReserves:       curPool.DebtReserves,
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
			return nil, nil, err
		}

		pool := entity.Pool{
			Address:  curPool.PoolAddress.String(),
			Exchange: string(valueobject.ExchangeFluidDexT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{
				new(big.Int).Add(curPool.CollateralReserves.Token0RealReserves, curPool.DebtReserves.Token0Debt).String(),
				new(big.Int).Add(curPool.CollateralReserves.Token1RealReserves, curPool.DebtReserves.Token1Debt).String(),
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   curPool.Token0Address.String(),
					Weight:    1,
					Swappable: true,
				},
				{
					Address:   curPool.Token1Address.String(),
					Weight:    1,
					Swappable: true,
				},
			},
			SwapFee: 0, // Todo
			Extra:   string(extraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]PoolWithReserves, error) {
	var pools []PoolWithReserves

	req := u.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: dexReservesResolver[u.config.ChainID],
		Method: DRRMethodGetAllPoolsReserves,
	}, []interface{}{&pools})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Error("aggregate request failed")
		return nil, err
	}

	return pools, nil
}
