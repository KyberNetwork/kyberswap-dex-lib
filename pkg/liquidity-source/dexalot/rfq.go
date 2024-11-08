package dexalot

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	DexID          string           `json:"dexId"`
	HTTP           HTTPClientConfig `mapstructure:"http" json:"http"`
	UpscalePercent int              `mapstructure:"upscale_percent" json:"upscale_percent"`
}

type IClient interface {
	Quote(ctx context.Context, params FirmQuoteParams, upscalePercent int) (FirmQuoteResult, error)
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
	var swapInfo SwapInfo
	if err := mapstructure.WeakDecode(params.SwapInfo, &swapInfo); err != nil {
		return nil, err
	}
	logger.Infof("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)

	upscaledTakerAmount := bignumber.NewBig(swapInfo.BaseTokenAmount)
	upscaledTakerAmount.Mul(
		upscaledTakerAmount,
		big.NewInt(int64(100+h.config.UpscalePercent)),
	).Div(
		upscaledTakerAmount,
		big.NewInt(100),
	)

	maxAmount := bignumber.NewBig(swapInfo.BaseTokenReserve)
	if upscaledTakerAmount.Cmp(bignumber.NewBig(swapInfo.BaseTokenReserve)) > 0 {
		upscaledTakerAmount = bignumber.NewBig(swapInfo.BaseTokenAmount)
		upscaledTakerAmount = upscaledTakerAmount.Add(upscaledTakerAmount, maxAmount).Div(upscaledTakerAmount, bignumber.Two)
	}
	p := FirmQuoteParams{
		ChainID:     int(params.NetworkID),
		TakerAsset:  swapInfo.BaseTokenOriginal,
		MakerAsset:  swapInfo.QuoteTokenOriginal,
		TakerAmount: upscaledTakerAmount.String(),
		UserAddress: params.RFQSender,
		Executor:    params.RFQRecipient,
	}
	result, err := h.client.Quote(ctx, p, h.config.UpscalePercent)
	if err != nil {
		return nil, fmt.Errorf("quote single order result: %w", err)
	}

	newAmountOut, _ := new(big.Int).SetString(result.Order.MakerAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
