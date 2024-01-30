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

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	swapExtraBytes, err := json.Marshal(params.SwapInfo)
	if err != nil {
		return nil, err
	}

	var swapExtra SwapExtra
	if err = json.Unmarshal(swapExtraBytes, &swapExtra); err != nil {
		return nil, ErrInvalidFirmQuoteParams
	}

	if swapExtra.MakingAmount == "" || swapExtra.TakingAmount == "" {
		return nil, ErrInvalidFirmQuoteParams
	}

	if !account.IsValidAddress(swapExtra.MakerAsset) || !account.IsValidAddress(swapExtra.TakerAsset) {
		return nil, ErrInvalidFirmQuoteParams
	}

	result, err := h.client.Firm(ctx,
		FirmRequestParams{
			MakerAsset:  swapExtra.MakerAsset,
			TakerAsset:  swapExtra.TakerAsset,
			MakerAmount: swapExtra.MakingAmount,
			TakerAmount: swapExtra.TakingAmount,
			UserAddress: params.Recipient,
		})
	if err != nil {
		logger.WithFields(logger.Fields{
			"params": params,
			"error":  err,
		}).Errorf("failed to get firm quote")
		return nil, err
	}

	newAmountOut, _ := new(big.Int).SetString(result.Order.MakerAmount, 10)

	return &pool.RFQResult{
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
			Recipient:          params.Recipient,
		},
	}, nil
}
