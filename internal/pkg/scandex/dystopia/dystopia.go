package dystopia

import (
	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/velodrome"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
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
