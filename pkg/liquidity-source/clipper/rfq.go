package clipper

import (
	"context"
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
	RFQ(ctx context.Context, params QuoteParams) (SignResponse, error)
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
		ChainID:           swapInfo.ChainID,
		TimeInSeconds:     swapInfo.TimeInSeconds,
		InputAmount:       swapInfo.InputAmount.String(),
		InputAssetSymbol:  swapInfo.InputAssetSymbol,
		OutputAssetSymbol: swapInfo.OutputAssetSymbol,

		DestinationAddress: params.RFQRecipient,
		SenderAddress:      params.Sender,
	})
	if err != nil {
		return nil, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.OutputAmount, 10)

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result.Signature,
	}, nil
}
