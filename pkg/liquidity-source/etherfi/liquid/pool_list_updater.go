package liquid

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	return lo.MapToSlice(pools, func(poolAddress string, poolInfo PoolInfo) entity.Pool {
		tokens := make([]*entity.PoolToken, 0, len(poolInfo.SupportedDepositAssets)+1)

		tokens = append(tokens, &entity.PoolToken{Address: poolAddress})
		for _, addr := range poolInfo.SupportedDepositAssets {
			tokens = append(tokens, &entity.PoolToken{Address: addr})
		}

		extraBytes, _ := json.Marshal(Extra{
			SupportedWithdraw: poolInfo.SupportedWithdrawAssets,
		})

		return entity.Pool{
			Address:   poolAddress,
			Exchange:  string(valueobject.ExchangeEtherfiLiquid),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{unlimitedReserve, unlimitedReserve},
			Tokens:    tokens,
			Extra:     string(extraBytes),
		}
	}), nil, nil
}
