package dystopia

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/velodrome"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	option := uniswap.Option{
		UpdateReserveFunc:          uniswap.UpdateReservesFunc,
		UpdateNewPoolFunc:          velodrome.UpdateNewPoolFunc,
		DexType:                    constant.PoolTypes.Velodrome,
		FactoryAbi:                 abis.BiswapFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	}

	return uniswap.NewWithFunc(scanDexCfg, scanService, option)
}
