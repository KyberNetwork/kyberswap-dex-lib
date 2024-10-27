package dexalot

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
	Quote(ctx context.Context, params FirmQuoteParams) (FirmQuoteResult, error)
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

	p := FirmQuoteParams{
		ChainID:     int(params.NetworkID),
		TakerAsset:  swapInfo.BaseTokenOriginal,
		MakerAsset:  swapInfo.QuoteTokenOriginal,
		TakerAmount: swapInfo.BaseTokenAmount,
		UserAddress: params.RFQSender,
		Executor:    params.RFQRecipient,
	}
	result, err := h.client.Quote(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("quote single order result: %w", err)
	}

	newAmountOut, _ := new(big.Int).SetString(result.Order.MakerAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}
