package hashflowv3

import (
	"context"
	"errors"
	"math/big"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const rfqDefaultChainType = "evm"

type Config struct {
	DexID string           `json:"dexId"`
	HTTP  HTTPClientConfig `mapstructure:"http" json:"http"`
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
	swapInfoBytes, err := json.Marshal(params.SwapInfo)
	if err != nil {
		return nil, err
	}

	var swapInfo SwapInfo
	if err = json.Unmarshal(swapInfoBytes, &swapInfo); err != nil {
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
				MarketMakers:    []string{swapInfo.MarketMaker},
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
