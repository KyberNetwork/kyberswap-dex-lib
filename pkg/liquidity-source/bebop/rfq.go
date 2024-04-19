package bebop

import (
	"context"
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
	Quote(ctx context.Context, params QuoteParams) (QuoteResult, error)
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
	result, err := h.client.Quote(ctx, QuoteParams{
		SellTokens:      swapInfo.BaseToken,
		BuyTokens:       swapInfo.QuoteToken,
		SellAmounts:     swapInfo.BaseTokenAmount,
		BuyAmounts:      swapInfo.QuoteTokenAmount,
		TakerAddress:    params.RFQSender,
		ReceiverAddress: params.RFQRecipient,
	})
	if err != nil {
		return nil, err
	}

	newTotalAmountOut := big.NewInt(0)
	for _, mA := range result.ToSign.MakerAmounts {
		for _, amount := range mA {
			newAmountOut, _ := new(big.Int).SetString(amount, 10)
			newTotalAmountOut = new(big.Int).Add(newTotalAmountOut, newAmountOut)
		}
	}

	return &pool.RFQResult{
		NewAmountOut: newTotalAmountOut,
		Extra:        result,
	}, nil
}
