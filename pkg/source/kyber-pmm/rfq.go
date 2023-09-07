package kyberpmm

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/logger"
)

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

func (h *RFQHandler) RFQ(ctx context.Context, recipient string, params any) (any, error) {
	paramsByteData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	var swapExtra SwapExtra
	if err = json.Unmarshal(paramsByteData, &swapExtra); err != nil {
		return nil, ErrInvalidFirmQuoteParams
	}

	result, err := h.client.Firm(ctx,
		FirmRequestParams{
			MakerAsset:  swapExtra.MakerAsset,
			TakerAsset:  swapExtra.TakerAsset,
			MakerAmount: swapExtra.MakingAmount,
			TakerAmount: swapExtra.TakingAmount,
			UserAddress: recipient,
		})
	if err != nil {
		logger.WithFields(logger.Fields{
			"params": params,
			"error":  err,
		}).Errorf("failed to get firm quote")
		return nil, err
	}

	return RFQExtra{
		RFQContractAddress: h.config.RFQContractAddress,
		Info:               result.Order.Info,
		Expiry:             result.Order.Expiry,
		MakerAsset:         result.Order.MakerAsset,
		TakerAsset:         result.Order.TakerAsset,
		Maker:              result.Order.Maker,
		Taker:              result.Order.Taker,
		MakerAmount:        result.Order.MakerAmount,
		TakerAmount:        result.Order.TakerAmount,
		Signature:          result.Order.Signature,
		Recipient:          recipient,
	}, nil
}
