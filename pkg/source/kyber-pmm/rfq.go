package kyberpmm

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/account"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

func (h *RFQHandler) RFQ(ctx context.Context, recipient string, params any) (pool.RFQResult, error) {
	paramsByteData, err := json.Marshal(params)
	if err != nil {
		return pool.RFQResult{}, err
	}

	var swapExtra SwapExtra
	if err = json.Unmarshal(paramsByteData, &swapExtra); err != nil {
		return pool.RFQResult{}, ErrInvalidFirmQuoteParams
	}

	if swapExtra.MakingAmount == "" || swapExtra.TakingAmount == "" {
		return pool.RFQResult{}, ErrInvalidFirmQuoteParams
	}

	if !account.IsValidAddress(swapExtra.MakerAsset) || !account.IsValidAddress(swapExtra.TakerAsset) {
		return pool.RFQResult{}, ErrInvalidFirmQuoteParams
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
		return pool.RFQResult{}, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.Order.MakerAmount, 10)

	return pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra: RFQExtra{
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
		},
	}, nil
}
