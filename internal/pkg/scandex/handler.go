package scandex

import (
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/balancer"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/biswap"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/camelot"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/curve"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/dmm"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/dodo"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/dystopia"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/firebird"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/fraxswap"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/gmx"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/ironstable"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/lido"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/limitorder"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/madmex"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/makerpsm"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/metavault"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/nerve"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/oneswap"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/platypus"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/polydex"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/promm"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/saddle"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/synapse"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswapv3"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/velodrome"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

func NewScanDexHandler(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	switch scanDexCfg.Handler {
	case "uniswap":
		return uniswap.New(scanDexCfg, scanService)
	case "uniswapv3":
		return uniswapv3.New(scanDexCfg, scanService)
	case "polydex":
		return polydex.New(scanDexCfg, scanService)
	case "dmm":
		return dmm.New(scanDexCfg, scanService)
	case "promm":
		return promm.New(scanDexCfg, scanService)
	case "firebird":
		return firebird.New(scanDexCfg, scanService)
	case "oneswap":
		return oneswap.New(scanDexCfg, scanService)
	case "saddle":
		return saddle.New(scanDexCfg, scanService)
	case "iron-stable":
		return ironstable.New(scanDexCfg, scanService)
	case "curve":
		return curve.New(scanDexCfg, scanService)
	case "nerve":
		return nerve.New(scanDexCfg, scanService)
	case "biswap":
		return biswap.New(scanDexCfg, scanService)
	case "balancer":
		return balancer.New(scanDexCfg, scanService)
	case "synapse":
		return synapse.New(scanDexCfg, scanService)
	case "dodo":
		return dodo.New(scanDexCfg, scanService)
	case "velodrome":
		return velodrome.New(scanDexCfg, scanService)
	case "platypus":
		return platypus.New(scanDexCfg, scanService)
	case "dystopia":
		return dystopia.New(scanDexCfg, scanService)
	case "gmx":
		return gmx.New(scanDexCfg, scanService)
	case "maker-psm":
		return makerpsm.New(scanDexCfg, scanService)
	case "synthetix":
		return synthetix.New(scanDexCfg, scanService)
	case "madmex":
		return madmex.New(scanDexCfg, scanService)
	case "metavault":
		return metavault.New(scanDexCfg, scanService)
	case "lido":
		return lido.New(scanDexCfg, scanService)
	case "fraxswap":
		return fraxswap.New(scanDexCfg, scanService)
	case "camelot":
		return camelot.New(scanDexCfg, scanService)
	case "limit-order":
		return limitorder.New(scanDexCfg, scanService)
	}
	return nil, fmt.Errorf("can not find dex handler: %s", scanDexCfg.Handler)
}
