package synapse

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/saddle"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return saddle.NewWithFunc(scanDexCfg, scanService, saddle.Option{})
}
