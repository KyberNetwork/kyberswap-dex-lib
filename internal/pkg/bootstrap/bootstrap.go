package bootstrap

import (
	"context"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	onebit "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/one-bit"

	onebitclient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/one-bit/client"

	kyberpmmclient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm/client"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	bebopclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	clipperclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dexalotclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot/client"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	hashflowv3client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3/client"
	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mx-trading"
	mxtradingclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mx-trading/client"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
	nativev1client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	swaapv2client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
)

type NoopRFQHandler struct{}

func NewNoopRFQHandler() *NoopRFQHandler {
	return &NoopRFQHandler{}
}

func (h *NoopRFQHandler) RFQ(context.Context, pool.RFQParams) (*pool.RFQResult, error) {
	return &pool.RFQResult{}, nil
}

func (h *NoopRFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}

func (h *NoopRFQHandler) SupportBatch() bool {
	return false
}

func NewRFQHandler(
	dexId string,
	rfqCfg buildroute.RFQConfig,
) (pool.IPoolRFQ, error) {
	switch rfqCfg.Handler {
	case kyberpmm.DexTypeKyberPMM:
		var cfg kyberpmm.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := kyberpmmclient.NewHTTPClient(&cfg.HTTP)

		return kyberpmm.NewRFQHandler(&cfg, httpClient), nil

	case limitorder.DexTypeLimitOrder:
		var cfg limitorder.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		return limitorder.NewRFQHandler(&cfg), nil

	case swaapv2.DexType:
		var cfg swaapv2.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := swaapv2client.NewHTTPClient(&cfg.HTTP)

		return swaapv2.NewRFQHandler(&cfg, httpClient), nil

	case hashflowv3.DexType:
		var cfg hashflowv3.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := hashflowv3client.NewHTTPClient(&cfg.HTTP)

		return hashflowv3.NewRFQHandler(&cfg, httpClient), nil

	case nativev1.DexType:
		var cfg nativev1.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := nativev1client.NewHTTPClient(&cfg.HTTP)

		return nativev1.NewRFQHandler(&cfg, httpClient), nil

	case bebop.DexType:
		var cfg bebop.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := bebopclient.NewHTTPClient(&cfg.HTTP)

		return bebop.NewRFQHandler(&cfg, httpClient), nil

	case clipper.DexType:
		var cfg clipper.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := clipperclient.NewHTTPClient(cfg.HTTP)

		return clipper.NewRFQHandler(&cfg, httpClient), nil

	case dexalot.DexType:
		var cfg dexalot.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := dexalotclient.NewHTTPClient(&cfg.HTTP)

		return dexalot.NewRFQHandler(&cfg, httpClient), nil

	case mxtrading.DexType:
		var cfg mxtrading.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := mxtradingclient.NewHTTPClient(&cfg.HTTP)

		return mxtrading.NewRFQHandler(&cfg, httpClient), nil

	case uniswapv4.DexType:
		var cfg uniswapv4.RFQConfig
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}
		return uniswapv4.NewRFQHandler(&cfg), nil

	case onebit.DexType:
		var cfg onebit.Config
		if err := PropertiesToStruct(rfqCfg.Properties, &cfg); err != nil {
			return nil, err
		}

		cfg.DexID = dexId
		httpClient := onebitclient.NewHTTPClient(&cfg.HTTP)

		return onebit.NewRFQHandler(&cfg, httpClient), nil

	default:
		return NewNoopRFQHandler(), nil
	}
}
