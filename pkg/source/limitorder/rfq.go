package limitorder

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/logger"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type RFQHandler struct {
	config *Config
	client *httpClient
}

func NewRFQHandler(config *Config) *RFQHandler {
	client := NewHTTPClient(config.LimitOrderHTTPUrl)
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
		return nil, InvalidSwapInfo
	}

	orderIds := lo.Map(swapInfo.FilledOrders, func(o *FilledOrderInfo, _ int) int64 { return o.OrderID })
	result, err := h.client.GetOpSignatures(ctx, ChainID(h.config.ChainID), orderIds)
	if err != nil {
		logger.WithFields(logger.Fields{
			"params": params,
			"error":  err,
		}).Errorf("failed to get operator signatures")
		return nil, err
	}

	return &pool.RFQResult{
		NewAmountOut: nil, // at the moment we don't use the new amount out of Limit Order, nil will ignore it
		Extra: OpSignatureExtra{
			SwapInfo:               swapInfo,
			OperatorSignaturesById: lo.SliceToMap(result, func(sig *operatorSignatures) (int64, *operatorSignatures) { return sig.ID, sig }),
		},
	}, nil
}
