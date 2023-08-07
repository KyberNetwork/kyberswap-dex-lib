package traderjoev20

import (
	"context"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoecommon"
)

type PoolsListUpdater struct {
	*traderjoecommon.PoolsListUpdater
}

func NewPoolsListUpdater(
	cfg *traderjoecommon.Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		PoolsListUpdater: &traderjoecommon.PoolsListUpdater{
			Config:                     cfg,
			EthrpcClient:               ethrpcClient,
			FactoryABI:                 factoryABI,
			FactoryNumberOfPairsMethod: factoryNumberOfPairsMethod,
			FactoryGetPairMethod:       factoryGetPairMethod,
			PairABI:                    pairABI,
			PairTokenXMethod:           pairTokenXMethod,
			PairTokenYMethod:           pairTokenYMethod,
			DexType:                    DexTypeTraderJoeV20,
			DefaultTokenWeight:         defaultTokenWeight,
		},
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	return d.PoolsListUpdater.GetNewPools(ctx, metadataBytes)
}
