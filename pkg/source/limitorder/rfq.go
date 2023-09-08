package limitorder

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/logger"
	"github.com/samber/lo"
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

func (h *RFQHandler) RFQ(ctx context.Context, recipient string, params any) (any, error) {
	paramsByteData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	var swapInfo SwapInfo
	if err = json.Unmarshal(paramsByteData, &swapInfo); err != nil {
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

	return OpSignatureExtra{
		SwapInfo:               swapInfo,
		OperatorSignaturesById: lo.SliceToMap(result, func(sig *operatorSignatures) (int64, *operatorSignatures) { return sig.ID, sig }),
	}, nil
}
