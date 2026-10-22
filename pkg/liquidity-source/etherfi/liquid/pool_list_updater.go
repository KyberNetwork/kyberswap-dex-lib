package liquid

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	return lo.MapToSlice(poolsByChain[lo.Ternary(u.config.ChainID == 0, valueobject.ChainIDEthereum, u.config.ChainID)], func(poolAddress string, poolInfo PoolInfo) entity.Pool {
		tokens := make([]*entity.PoolToken, 0, len(poolInfo.SupportedDepositAssets)+1)

		tokens = append(tokens, &entity.PoolToken{Address: poolAddress})
		for _, addr := range poolInfo.SupportedDepositAssets {
			tokens = append(tokens, &entity.PoolToken{Address: addr})
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			LiquidRefer:   lo.Ternary(poolInfo.SimpleTeller, common.Address{}, liquidReferAddress),
			Teller:        common.HexToAddress(poolInfo.TellerContract),
			SimpleTeller:  poolInfo.SimpleTeller,
			Depositor:     common.HexToAddress(poolInfo.Depositor),
			Accountant:    common.HexToAddress(poolInfo.Accountant),
			CommunityCode: poolInfo.CommunityCode,
		})

		return entity.Pool{
			Address:     poolAddress,
			Exchange:    lo.Ternary(poolInfo.Exchange != "", poolInfo.Exchange, valueobject.ExchangeEtherfiLiquid),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{unlimitedReserve, unlimitedReserve},
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
			Extra:       "{}",
		}
	}), nil, nil
}
