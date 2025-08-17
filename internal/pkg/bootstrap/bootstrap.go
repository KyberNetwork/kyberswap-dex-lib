package bootstrap

import (
	"context"

	_ "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	kyberpmmclient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm/client"
	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/mx-trading"
	mxtradingclient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/mx-trading/client"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/onebit"
	onebitclient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/onebit/client"
	pcsfairflow "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/pancake-infinity/hooks/fairflow"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/uniswap-v4/hooks/fairflow"
	kem "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/uniswap-v4/hooks/kyber-exclusive-amm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	bebopclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	clipperclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dexalotclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot/client"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	hashflowv3client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3/client"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1"
	nativev1client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	swaapv2client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
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

func NewRFQHandler(rfqCfg buildroute.RFQConfig, poolManager *poolmanager.PointerSwapPoolManager,
	customFuncs entity.ICustomFuncs) (pool.IPoolRFQ, error) {
	switch rfqCfg.Handler {
	case kyberpmm.DexTypeKyberPMM:
		cfg, err := util.AnyToStruct[kyberpmm.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := kyberpmmclient.NewHTTPClient(&cfg.HTTP)
		return kyberpmm.NewRFQHandler(cfg, httpClient), nil

	case limitorder.DexTypeLimitOrder:
		cfg, err := util.AnyToStruct[limitorder.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		return limitorder.NewRFQHandler(cfg), nil

	case swaapv2.DexType:
		cfg, err := util.AnyToStruct[swaapv2.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := swaapv2client.NewHTTPClient(&cfg.HTTP)
		return swaapv2.NewRFQHandler(cfg, httpClient), nil

	case hashflowv3.DexType:
		cfg, err := util.AnyToStruct[hashflowv3.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := hashflowv3client.NewHTTPClient(&cfg.HTTP)
		return hashflowv3.NewRFQHandler(cfg, httpClient), nil

	case nativev1.DexType:
		cfg, err := util.AnyToStruct[nativev1.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := nativev1client.NewHTTPClient(&cfg.HTTP)
		return nativev1.NewRFQHandler(cfg, httpClient), nil

	case bebop.DexType:
		cfg, err := util.AnyToStruct[bebop.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := bebopclient.NewHTTPClient(&cfg.HTTP)
		return bebop.NewRFQHandler(cfg, httpClient), nil

	case clipper.DexType:
		cfg, err := util.AnyToStruct[clipper.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := clipperclient.NewHTTPClient(cfg.HTTP)
		return clipper.NewRFQHandler(cfg, httpClient), nil

	case dexalot.DexType:
		cfg, err := util.AnyToStruct[dexalot.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := dexalotclient.NewHTTPClient(&cfg.HTTP)
		return dexalot.NewRFQHandler(cfg, httpClient), nil

	case mxtrading.Handler:
		cfg, err := util.AnyToStruct[mxtrading.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := mxtradingclient.NewHTTPClient(&cfg.HTTP)
		return mxtrading.NewRFQHandler(cfg, httpClient), nil

	case onebit.Handler:
		cfg, err := util.AnyToStruct[onebit.Config](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		httpClient := onebitclient.NewHTTPClient(&cfg.HTTP)
		return onebit.NewRFQHandler(cfg, httpClient), nil

	case kem.Handler:
		cfg, err := util.AnyToStruct[fairflow.RFQConfig](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		return kem.NewRFQHandler(cfg, poolManager, customFuncs), nil

	case fairflow.Handler:
		cfg, err := util.AnyToStruct[fairflow.RFQConfig](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		return fairflow.NewRFQHandler(cfg, poolManager, customFuncs), nil

	case pcsfairflow.Handler:
		cfg, err := util.AnyToStruct[fairflow.RFQConfig](rfqCfg.Properties)
		if err != nil {
			return nil, err
		}
		return pcsfairflow.NewRFQHandler(cfg, poolManager, customFuncs), nil

	default:
		return NewNoopRFQHandler(), nil
	}
}
