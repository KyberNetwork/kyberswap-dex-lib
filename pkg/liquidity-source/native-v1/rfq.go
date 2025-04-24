package nativev1

import (
	"context"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/logger"
	"github.com/mitchellh/mapstructure"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	HTTP  HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	Quote(ctx context.Context, params QuoteParams) (QuoteResult, error)
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
	logger.Debugf("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)

	result, err := h.client.Quote(ctx, QuoteParams{
		Chain:              ChainById(valueobject.ChainID(params.NetworkID)),
		TokenIn:            swapInfo.BaseToken,
		TokenOut:           swapInfo.QuoteToken,
		AmountWei:          swapInfo.BaseTokenAmount,
		FromAddress:        params.RFQSender,
		BeneficiaryAddress: params.Sender,
		ToAddress:          params.RFQRecipient,
		ExpiryTime:         strconv.Itoa(int(swapInfo.ExpirySecs)),
		Slippage:           strconv.FormatFloat(float64(params.Slippage)/100, 'f', 2, 64),
	})
	if err != nil {
		return nil, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.AmountOut, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
