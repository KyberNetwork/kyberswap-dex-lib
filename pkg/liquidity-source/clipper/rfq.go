package clipper

import (
	"context"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Config struct {
	HTTP HTTPClientConfig `mapstructure:"http" json:"http"`
}

type IClient interface {
	RFQ(ctx context.Context, params QuoteParams) (SignResponse, error)
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

	result, err := h.client.RFQ(ctx, QuoteParams{
		ChainID:           swapInfo.ChainID,
		TimeInSeconds:     swapInfo.TimeInSeconds,
		InputAmount:       swapInfo.InputAmount,
		InputAssetSymbol:  swapInfo.InputAssetSymbol,
		OutputAssetSymbol: swapInfo.OutputAssetSymbol,

		DestinationAddress: params.RFQRecipient,
		SenderAddress:      params.Sender,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "quote failed")
	}

	newAmountOut, _ := new(big.Int).SetString(result.OutputAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra: RFQExtra{
			V:         result.Signature.V,
			R:         result.Signature.R,
			S:         result.Signature.S,
			GoodUntil: result.GoodUntil,
		},
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
