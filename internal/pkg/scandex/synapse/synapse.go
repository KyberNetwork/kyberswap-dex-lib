package synapse

import (
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/saddle"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return saddle.NewWithFunc(scanDexCfg, scanService, saddle.Option{})
}
