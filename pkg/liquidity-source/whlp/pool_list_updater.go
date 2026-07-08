package whlp

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	hasInitialized bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(_ *ethrpc.Client) *PoolListUpdater {
	return &PoolListUpdater{}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	staticExtraBytes, _ := json.Marshal(StaticExtra{
		Accountant:    accountantAddress,
		Depositor:     depositorAddress,
		CommunityCode: "kyberswap",
	})

	return lo.MapToSlice(pools, func(poolAddress string, _ struct{}) entity.Pool {
		return entity.Pool{
			Address:  poolAddress,
			Exchange: string(valueobject.ExchangeWHLP),
			Type:     DexType,
			Timestamp: time.Now().Unix(),
			Reserves: []string{unlimitedReserve, unlimitedReserve},
			Tokens: []*entity.PoolToken{
				{Address: poolAddress},
				{Address: usdt0Address.Hex()},
			},
			StaticExtra: string(staticExtraBytes),
			Extra:       "{}",
		}
	}), nil, nil
}
