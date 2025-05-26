package nativev1

import (
	"context"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type Config struct {
	HTTP HTTPClientConfig `mapstructure:"http" json:"http"`
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
	swapInfo, err := util.AnyToStruct[SwapInfo](params.SwapInfo)
	if err != nil {
		return nil, err
	}
	logger.Debugf("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)

	chainName := ChainById(params.NetworkID)
	result, err := h.client.Quote(ctx, QuoteParams{
		SrcChain:           chainName,
		DstChain:           chainName,
		TokenIn:            swapInfo.BaseToken,
		TokenOut:           swapInfo.QuoteToken,
		AmountWei:          swapInfo.BaseTokenAmount,
		FromAddress:        params.RFQSender,
		BeneficiaryAddress: params.Sender,
		ToAddress:          params.RFQRecipient,
		ExpiryTime:         swapInfo.ExpirySecs,
		Slippage:           strconv.FormatFloat(float64(params.Slippage)/100, 'f', 2, 64),
	})
	if err != nil {
		return nil, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.AmountOut, 10)

	result.ApprovalAddress = result.TxRequest.Target

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
