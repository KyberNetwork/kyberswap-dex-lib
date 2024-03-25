package bootstrap

import (
	"context"

	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	hashflowv3client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	swaapv2client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	kyberpmmclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
)

type NoopRFQHandler struct{}

func NewNoopRFQHandler() *NoopRFQHandler {
	return &NoopRFQHandler{}
}

func (h *NoopRFQHandler) RFQ(context.Context, pool.RFQParams) (*pool.RFQResult, error) {
	return &pool.RFQResult{}, nil
}

func NewRFQHandler(
	rfqCfg buildroute.RFQConfig,
	commonCfg config.Common,
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
	case swaapv2.DexType:
		var cfg swaapv2.Config
		err := PropertiesToStruct(rfqCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = rfqCfg.Id
		cfg.HTTP.APIKey = commonCfg.SwaapAPIKey
		httpClient := swaapv2client.NewHTTPClient(&cfg.HTTP)

		return swaapv2.NewRFQHandler(&cfg, httpClient), nil

	case hashflowv3.DexType:
		var cfg hashflowv3.Config
		err := PropertiesToStruct(rfqCfg.Properties, &cfg)
		if err != nil {
			return nil, err
		}
		cfg.DexID = rfqCfg.Id
		cfg.HTTP.APIKey = commonCfg.HashflowAPIKey
		httpClient := hashflowv3client.NewHTTPClient(&cfg.HTTP)

		return hashflowv3.NewRFQHandler(&cfg, httpClient), nil

	default:
		return NewNoopRFQHandler(), nil
	}
}
