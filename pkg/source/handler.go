package source

import (
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velodromev2"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/algebrav1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer"
	balancerComposableStable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer-composable-stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/biswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/camelot"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dodo"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dystopia"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/elastic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fraxswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ironstable"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	kyberpmmclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido"
	lido_steth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido-steth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/madmex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/makerpsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/muteswitch"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nerve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/oneswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pancakev3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pearl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/platypus"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/polydex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramses"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoecommon"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoev20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoev21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocimeter"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velodrome"
)

func NewPoolsListUpdaterHandler(
	scanDexCfg *ScanDex,
	ethrpcClient *ethrpc.Client,
) (pool.IPoolsListUpdater, error) {
	switch scanDexCfg.Handler {
	case uniswap.DexTypeUniswap:
		var cfg uniswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return uniswap.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case uniswapv3.DexTypeUniswapV3:
		var cfg uniswapv3.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return uniswapv3.NewPoolsListUpdater(&cfg), nil
	case algebrav1.DexTypeAlgebraV1:
		var cfg algebrav1.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return algebrav1.NewPoolsListUpdater(&cfg), nil
	case dmm.DexTypeDMM:
		var cfg dmm.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return dmm.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case elastic.DexTypeElastic:
		var cfg elastic.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return elastic.NewPoolsListUpdater(&cfg), nil
	case balancer.DexTypeBalancer:
		var cfg balancer.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return balancer.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case balancerComposableStable.DexTypeBalancerComposableStableExchange:
		var cfg balancerComposableStable.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return balancerComposableStable.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case velodrome.DexTypeVelodrome:
		var cfg velodrome.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velodrome.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case velodromev2.DexTypeVelodromeV2:
		var cfg velodromev2.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velodromev2.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case velocimeter.DexTypeVelocimeter:
		var cfg velocimeter.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velocimeter.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case muteswitch.DexTypeMuteSwitch:
		var cfg muteswitch.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return muteswitch.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case ramses.DexTypeRamses:
		var cfg ramses.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return ramses.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case platypus.DexTypePlatypus:
		var cfg platypus.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return platypus.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case biswap.DexTypeBiswap:
		var cfg biswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return biswap.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case makerpsm.DexTypeMakerPSM:
		var cfg makerpsm.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return makerpsm.NewPoolsListUpdater(&cfg), nil
	case curve.DexTypeCurve:
		var cfg curve.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return curve.NewPoolsListUpdater(&cfg, ethrpcClient)
	case oneswap.DexTypeOneSwap:
		var cfg oneswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return oneswap.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case saddle.DexTypeSaddle:
		var cfg saddle.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return saddle.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case dodo.DexTypeDodo:
		var cfg dodo.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return dodo.NewPoolsListUpdater(&cfg), nil
	case nerve.DexTypeNerve:
		var cfg nerve.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return nerve.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case synthetix.DexTypeSynthetix:
		var cfg synthetix.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return synthetix.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case dystopia.DexTypeDystopia:
		var cfg dystopia.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return dystopia.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case metavault.DexTypeMetavault:
		var cfg metavault.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return metavault.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case camelot.DexTypeCamelot:
		var cfg camelot.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return camelot.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case lido.DexTypeLido:
		var cfg lido.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return lido.NewPoolsListUpdater(&cfg), nil
	case lido_steth.DexTypeLidoStETH:
		var cfg lido_steth.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return lido_steth.NewPoolsListUpdater(&cfg), nil
	case gmx.DexTypeGmx:
		var cfg gmx.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return gmx.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case fraxswap.DexTypeFraxswap:
		var cfg fraxswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return fraxswap.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case madmex.DexTypeMadmex:
		var cfg madmex.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return madmex.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case polydex.DexTypePolydex:
		var cfg polydex.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return polydex.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case ironstable.DexTypeIronStable:
		var cfg ironstable.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return ironstable.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case limitorder.DexTypeLimitOrder:
		var cfg limitorder.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return limitorder.NewPoolsListUpdater(&cfg)
	case syncswap.DexTypeSyncSwap:
		var cfg syncswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return syncswap.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case pancakev3.DexTypePancakeV3:
		var cfg pancakev3.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return pancakev3.NewPoolsListUpdater(&cfg), nil
	case maverickv1.DexTypeMaverickV1:
		var cfg maverickv1.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return maverickv1.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case pearl.DexTypePearl:
		var cfg pearl.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return pearl.NewPoolListUpdater(&cfg, ethrpcClient), nil
	case traderjoev20.DexTypeTraderJoeV20:
		var cfg traderjoecommon.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return traderjoev20.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case traderjoev21.DexTypeTraderJoeV21:
		var cfg traderjoecommon.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return traderjoev21.NewPoolsListUpdater(&cfg, ethrpcClient), nil
	case kyberpmm.DexTypeKyberPMM:
		var cfg kyberpmm.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		httpClient := kyberpmmclient.NewHTTPClient(&cfg.HTTP)
		memoryCacheClient := kyberpmmclient.NewMemoryCacheClient(&cfg.MemoryCache, httpClient)

		return kyberpmm.NewPoolsListUpdater(cfg, memoryCacheClient), nil
	}

	return nil, fmt.Errorf("can not find pools list updater handler: %s", scanDexCfg.Handler)
}

func NewPoolTrackerHandler(
	scanDexCfg *ScanDex,
	ethrpcClient *ethrpc.Client,
) (pool.IPoolTracker, error) {
	switch scanDexCfg.Handler {
	case uniswap.DexTypeUniswap:
		return uniswap.NewPoolTracker(ethrpcClient)
	case uniswapv3.DexTypeUniswapV3:
		var cfg uniswapv3.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return uniswapv3.NewPoolTracker(&cfg, ethrpcClient)
	case algebrav1.DexTypeAlgebraV1:
		var cfg algebrav1.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return algebrav1.NewPoolTracker(&cfg, ethrpcClient)
	case dmm.DexTypeDMM:
		return dmm.NewPoolTracker(ethrpcClient)
	case elastic.DexTypeElastic:
		var cfg elastic.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return elastic.NewPoolTracker(&cfg, ethrpcClient)
	case balancer.DexTypeBalancer:
		return balancer.NewPoolTracker(ethrpcClient)
	case balancerComposableStable.DexTypeBalancerComposableStableExchange:
		return balancerComposableStable.NewPoolTracker(ethrpcClient)
	case velodrome.DexTypeVelodrome:
		var cfg velodrome.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velodrome.NewPoolTracker(&cfg, ethrpcClient)
	case velodromev2.DexTypeVelodromeV2:
		var cfg velodromev2.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velodromev2.NewPoolTracker(&cfg, ethrpcClient)
	case velocimeter.DexTypeVelocimeter:
		var cfg velocimeter.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return velocimeter.NewPoolTracker(&cfg, ethrpcClient)
	case muteswitch.DexTypeMuteSwitch:
		var cfg muteswitch.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return muteswitch.NewPoolTracker(&cfg, ethrpcClient)
	case ramses.DexTypeRamses:
		var cfg ramses.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return ramses.NewPoolTracker(&cfg, ethrpcClient)
	case dodo.DexTypeDodo:
		var cfg dodo.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return dodo.NewPoolTracker(&cfg, ethrpcClient)
	case biswap.DexTypeBiswap:
		return biswap.NewPoolTracker(ethrpcClient)
	case platypus.DexTypePlatypus:
		return platypus.NewPoolTracker(ethrpcClient), nil
	case makerpsm.DexTypeMakerPSM:
		var cfg makerpsm.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return makerpsm.NewPoolTracker(&cfg, ethrpcClient), nil
	case curve.DexTypeCurve:
		var cfg curve.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return curve.NewPoolTracker(&cfg, ethrpcClient)
	case oneswap.DexTypeOneSwap:
		return oneswap.NewPoolTracker(ethrpcClient), nil
	case saddle.DexTypeSaddle:
		var cfg saddle.Config
		cfg.DexID = scanDexCfg.Id
		return saddle.NewPoolTracker(&cfg, ethrpcClient), nil
	case nerve.DexTypeNerve:
		var cfg nerve.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id
		return nerve.NewPoolTracker(&cfg, ethrpcClient)
	case dystopia.DexTypeDystopia:
		return dystopia.NewPoolTracker(ethrpcClient), nil
	case synthetix.DexTypeSynthetix:
		var cfg synthetix.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return synthetix.NewPoolTracker(&cfg, ethrpcClient), nil
	case metavault.DexTypeMetavault:
		var cfg metavault.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return metavault.NewPoolTracker(&cfg, ethrpcClient)
	case camelot.DexTypeCamelot:
		var cfg camelot.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return camelot.NewPoolTracker(&cfg, ethrpcClient), nil

	case lido.DexTypeLido:
		return lido.NewPoolTracker(ethrpcClient), nil
	case lido_steth.DexTypeLidoStETH:
		return lido_steth.NewPoolTracker(ethrpcClient), nil
	case gmx.DexTypeGmx:
		var cfg gmx.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return gmx.NewPoolTracker(&cfg, ethrpcClient)
	case fraxswap.DexTypeFraxswap:
		return fraxswap.NewPoolTracker(ethrpcClient), nil
	case madmex.DexTypeMadmex:
		var cfg madmex.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return madmex.NewPoolTracker(&cfg, ethrpcClient)
	case polydex.DexTypePolydex:
		return polydex.NewPoolTracker(ethrpcClient)
	case ironstable.DexTypeIronStable:
		var cfg ironstable.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return ironstable.NewPoolTracker(&cfg, ethrpcClient)
	case limitorder.DexTypeLimitOrder:
		var cfg limitorder.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return limitorder.NewPoolTracker(&cfg), nil
	case syncswap.DexTypeSyncSwap:
		var cfg syncswap.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return syncswap.NewPoolTracker(&cfg, ethrpcClient), nil
	case pancakev3.DexTypePancakeV3:
		var cfg pancakev3.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return pancakev3.NewPoolTracker(&cfg, ethrpcClient)
	case maverickv1.DexTypeMaverickV1:
		var cfg maverickv1.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return maverickv1.NewPoolTracker(&cfg, ethrpcClient), nil
	case pearl.DexTypePearl:
		var cfg pearl.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		return pearl.NewPoolTracker(&cfg, ethrpcClient)
	case traderjoev20.DexTypeTraderJoeV20:
		var cfg traderjoecommon.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id
		return traderjoev20.NewPoolTracker(ethrpcClient, &cfg)
	case traderjoev21.DexTypeTraderJoeV21:
		return traderjoev21.NewPoolTracker(ethrpcClient)
	case kyberpmm.DexTypeKyberPMM:
		var cfg kyberpmm.Config
		err := PropertiesToStruct(scanDexCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = scanDexCfg.Id

		httpClient := kyberpmmclient.NewHTTPClient(&cfg.HTTP)
		memoryCacheClient := kyberpmmclient.NewMemoryCacheClient(&cfg.MemoryCache, httpClient)

		return kyberpmm.NewPoolTracker(&cfg, memoryCacheClient), nil
	}

	return nil, fmt.Errorf("can not find pool tracker handler: %s", scanDexCfg.Handler)
}
