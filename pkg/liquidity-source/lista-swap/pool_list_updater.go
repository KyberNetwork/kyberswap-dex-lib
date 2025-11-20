package listaswap

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	pancakestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/stable"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
)

type PoolsListUpdater struct {
	stablePoolListsUpdater *pancakestable.PoolsListUpdater
	config                 *pancakestable.Config
	ethrpcClient           *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *pancakestable.Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		stablePoolListsUpdater: pancakestable.NewPoolsListUpdater(cfg, ethrpcClient),
		config:                 cfg,
		ethrpcClient:           ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	dexID := u.config.DexID

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	pools, newMetadataBytes, err := u.stablePoolListsUpdater.GetNewPools(ctx, metadataBytes)

	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("GetNewPools from stablePoolListsUpdater failed")

		return nil, metadataBytes, err
	}

	// Since ListaSwap supports swap with native tokens, we need to adjust the pool info accordingly.
	// Loop through the pools and replace native token addresses with wrapped native token addresses.
	// Also update staticExtra to indicate that the pool supports native token swap.
	for i := range pools {
		pool := pools[i]

		isNativeCoins := make([]bool, len(pool.Tokens))
		for j := range pool.Tokens {
			token := pool.Tokens[j]

			if valueobject.IsNative(token.Address) {
				pools[i].Tokens[j].Address = valueobject.WrapNativeLower(token.Address, valueobject.ChainID(u.config.ChainID))
				isNativeCoins[j] = true
			}
		}

		var staticExtra StaticExtra
		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			return nil, metadataBytes, err
		}

		staticExtra.APrecision = "100"
		staticExtra.IsNativeCoins = isNativeCoins

		extraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, metadataBytes, err
		}
		pools[i].StaticExtra = string(extraBytes)
		pools[i].Type = DexType
	}

	return pools, newMetadataBytes, nil
}
