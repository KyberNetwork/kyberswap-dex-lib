package bootstrap

import (
	"context"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	kyberpmmclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
)

type NoopRFQHandler struct{}

func NewNoopRFQHandler() *NoopRFQHandler {
	return &NoopRFQHandler{}
}

func (h *NoopRFQHandler) RFQ(ctx context.Context, recipient string, params any) (pool.RFQResult, error) {
	return pool.RFQResult{}, nil
}

func NewRFQHandler(
	rfqCfg usecase.RFQConfig,
) (pool.IPoolRFQ, error) {
	switch rfqCfg.Handler {
	case kyberpmm.DexTypeKyberPMM:
		var cfg kyberpmm.Config
		err := PropertiesToStruct(rfqCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = rfqCfg.Id

		httpClient := kyberpmmclient.NewHTTPClient(&cfg.HTTP)

		return kyberpmm.NewRFQHandler(&cfg, httpClient), nil
	case limitorder.DexTypeLimitOrder:
		var cfg limitorder.Config
		err := PropertiesToStruct(rfqCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = rfqCfg.Id

		return limitorder.NewRFQHandler(&cfg), nil
	default:
		return NewNoopRFQHandler(), nil
	}
}
