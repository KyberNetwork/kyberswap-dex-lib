package bebop

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	DexID string           `json:"dexId"`
	HTTP  HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	QuoteSingleOrderResult(ctx context.Context, params QuoteParams) (QuoteSingleOrderResult, error)
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
	var swapInfo SwapInfo
	if err := mapstructure.WeakDecode(params.SwapInfo, &swapInfo); err != nil {
		return nil, err
	}
	logger.Infof("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)
	p := QuoteParams{
		SellTokens:      swapInfo.BaseToken,
		BuyTokens:       swapInfo.QuoteToken,
		SellAmounts:     swapInfo.BaseTokenAmount,
		BuyAmounts:      swapInfo.QuoteTokenAmount,
		TakerAddress:    params.RFQSender,
		ReceiverAddress: params.RFQRecipient,
	}
	result, err := h.client.QuoteSingleOrderResult(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("quote single order result: %w", err)
	}

	newAmountOut, _ := new(big.Int).SetString(result.ToSign.MakerAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}
