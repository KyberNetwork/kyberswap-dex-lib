package mxtrading

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
)

type Config struct {
	DexID  string           `json:"dexId"`
	Router string           `json:"router"`
	HTTP   HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	Quote(ctx context.Context, params OrderParams) (SignedOrderResult, error)
}

type RFQHandler struct {
	pool.RFQHandler
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
	logger.Debugf("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)

	result, err := h.client.Quote(ctx, OrderParams{
		BaseToken:  swapInfo.BaseToken,
		QuoteToken: swapInfo.QuoteToken,
		Amount:     swapInfo.BaseTokenAmount,
		Taker:      params.RFQSender,
		FeeBps:     0,
	})
	if err != nil {
		return nil, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.Order.MakingAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra: RFQExtra{
			Router:            h.config.Router,
			Recipient:         params.RFQRecipient,
			SignedOrderResult: result,
		},
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
