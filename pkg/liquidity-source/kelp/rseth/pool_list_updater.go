package rseth

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	// Step 1:
	// - call LRTConfig.getSupportedAssetList to get supported assets
	// Step 2:
	// With each asset:
	// - call LRTConfig.depositLimitByAsset to get depositLimitByAsset
	// - call LRTDepositPool.getTotalAssetDeposits to get totalDepositByAsset
	// - call LRTOracle.getAssetPrice to get priceByAsset
	// Step 3:
	// - call LRTDepositPool.minAmountToDeposit to get minAmountToDeposit
	// - call LRTOracle.rsETHPrice to get LRTOracle
	// Step 4:
	// - combine data from 3 steps above, remember to convert from ETH to WETH and build pool. The first token have to be rsETH

	return []entity.Pool{
		{
			Address:   strings.ToLower(common.LRTDepositPool),
			Exchange:  string(valueobject.ExchangeKelpRSETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(common.RSETH),
					Symbol:    "rsETH",
					Decimals:  18,
					Name:      "rsETH",
					Swappable: true,
				},
			},
		},
	}, nil, nil
}
