package bebop

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Config struct {
	DexID string           `json:"dexId"`
	HTTP  HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	QuoteSingleOrderResult(ctx context.Context, params QuoteParams) (QuoteSingleOrderResult, error)
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
	p := QuoteParams{
		SellTokens:      swapInfo.BaseToken,
		BuyTokens:       swapInfo.QuoteToken,
		SellAmounts:     swapInfo.BaseTokenAmount,
		TakerAddress:    params.RFQSender,
		ReceiverAddress: params.RFQRecipient,
		OriginAddress:   params.Sender,
		Source:          params.Source,
	}
	result, err := h.client.QuoteSingleOrderResult(ctx, p)
	if err != nil {
		return nil, errors.WithMessage(err, "quote failed")
	}

	buyToken, ok := result.BuyTokens[common.HexToAddress(swapInfo.QuoteToken).Hex()]
	if !ok {
		return nil, fmt.Errorf("quote result doesn't have buy token %s", swapInfo.QuoteToken)
	}

	newAmountOut, ok := new(big.Int).SetString(buyToken.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid buy token amount: %s", buyToken.Amount)
	}

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
