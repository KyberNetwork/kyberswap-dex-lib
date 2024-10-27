package hashflowv3

import (
	"context"
	"errors"
	"math/big"

	"github.com/bytedance/sonic"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const rfqDefaultChainType = "evm"

type Config struct {
	DexID               string           `json:"dexId"`
	ExcludeMarketMakers []string         `mapstructure:"excludeMarketMakers" json:"excludeMarketMakers"`
	HTTP                HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	RFQ(ctx context.Context, params QuoteParams) (QuoteResult, error)
}

type RFQHandler struct {
	config *Config
	client IClient
}

func NewRFQHandler(config *Config, client IClient) *RFQHandler {
	return &RFQHandler{
		config: config,
		client: client,
	}
}

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	swapInfoBytes, err := sonic.Marshal(params.SwapInfo)
	if err != nil {
		return nil, err
	}

	var swapInfo SwapInfo
	if err = sonic.Unmarshal(swapInfoBytes, &swapInfo); err != nil {
		return nil, err
	}

	result, err := h.client.RFQ(ctx, QuoteParams{
		BaseChain: Chain{
			ChainType: rfqDefaultChainType,
			ChainId:   params.NetworkID,
		},
		QuoteChain: Chain{
			ChainType: rfqDefaultChainType,
			ChainId:   params.NetworkID,
		},
		RFQs: []RFQ{
			{
				BaseToken:       swapInfo.BaseToken,
				QuoteToken:      swapInfo.QuoteToken,
				BaseTokenAmount: swapInfo.BaseTokenAmount,

				Trader:          params.RFQRecipient,
				EffectiveTrader: params.Recipient,

				// Intentionally not specific marketMakers field to have higher chance to successfully RFQ
				// MarketMakers: []string{swapInfo.MarketMaker},

				ExcludeMarketMakers: h.config.ExcludeMarketMakers,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Quotes) != 1 {
		return nil, errors.New("mismatch quotes length")
	}

	newAmountOut, _ := new(big.Int).SetString(result.Quotes[0].QuoteData.QuoteTokenAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result.Quotes[0],
	}, nil
}
