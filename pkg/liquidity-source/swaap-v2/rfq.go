package swaapv2

import (
	"context"
	"math/big"
	"time"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Config struct {
	DexID string                  `json:"dexId"`
	HTTP  client.HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	Quote(ctx context.Context, params client.QuoteParams) (client.QuoteResult, error)
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

	result, err := h.client.Quote(ctx, client.QuoteParams{
		Origin:    params.Sender,
		Sender:    params.RFQSender,
		Recipient: params.RFQRecipient,
		Timestamp: time.Now().Unix(),
		OrderType: client.OrderTypeSell,
		TokenIn:   swapInfo.TokenIn,
		TokenOut:  swapInfo.TokenOut,
		Amount:    swapInfo.AmountIn,
		NetworkID: params.NetworkID,
	})
	if err != nil {
		return nil, err
	}

	amount, _ := new(big.Int).SetString(result.Amount, 10)

	return &pool.RFQResult{
		NewAmountOut: amount,
		Extra:        result,
	}, nil
}

func (h *RFQHandler) BatchRFQ(ctx context.Context, paramsSlice []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
