package kyberpmm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/account"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

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
	results, err := h.BatchRFQ(ctx, []pool.RFQParams{params})
	if err != nil {
		return nil, err
	}

	return results[0], nil
}

func (h *RFQHandler) BatchRFQ(ctx context.Context, paramsList []pool.RFQParams) ([]*pool.RFQResult, error) {
	if len(paramsList) == 0 {
		return nil, errors.New("empty batch params")
	}

	var orders = make([]Order, 0, len(paramsList))

	for i, params := range paramsList {
		swapExtraBytes, err := json.Marshal(params.SwapInfo)
		if err != nil {
			return nil, fmt.Errorf("Order %d error : %v", i, err)
		}

		var swapExtra SwapExtra
		if err = json.Unmarshal(swapExtraBytes, &swapExtra); err != nil {
			return nil, fmt.Errorf("Order %d error : %v", i, ErrInvalidFirmQuoteParams)
		}

		if swapExtra.MakingAmount == "" || swapExtra.TakingAmount == "" {
			return nil, fmt.Errorf("Order %d error : %v", i, ErrInvalidFirmQuoteParams)
		}

		if !account.IsValidAddress(swapExtra.MakerAsset) || !account.IsValidAddress(swapExtra.TakerAsset) {
			return nil, fmt.Errorf("Order %d error : %v", i, ErrInvalidFirmQuoteParams)
		}

		expectedMakerAmount, _ := new(big.Int).SetString(swapExtra.MakingAmount, 10)
		if params.AlphaFee != "" {
			alphaFee, _ := new(big.Int).SetString(params.AlphaFee, 10)
			expectedMakerAmount.Sub(expectedMakerAmount, alphaFee)
		}

		minMakerAmount := estimateMinMakerAmount(expectedMakerAmount, params.Slippage)

		orders = append(orders, Order{
			MakerAsset:          swapExtra.MakerAsset,
			TakerAsset:          swapExtra.TakerAsset,
			TakerAmount:         swapExtra.TakingAmount,
			ExpectedMakerAmount: expectedMakerAmount.String(),
			MinMakerAmount:      minMakerAmount.String(),
		})
	}

	result, err := h.client.MultiFirm(ctx,
		MultiFirmRequestParams{
			RequestID:   paramsList[0].RequestID,
			UserAddress: paramsList[0].Recipient,
			RFQSender:   paramsList[0].RFQSender,
			Partner:     paramsList[0].Source,
			Orders:      orders,
		})
	if err != nil {
		logger.WithFields(logger.Fields{
			"paramsList": paramsList,
			"error":      err,
		}).Errorf("failed to get multiFirm quote")
		return nil, err
	}

	if len(result.Orders) == 0 {
		return nil, ErrEmptyOrderList
	}

	var rfqResult = make([]*pool.RFQResult, 0, len(result.Orders))

	for i, order := range result.Orders {
		if order.Error != "" {
			logger.WithFields(logger.Fields{
				"paramsList": paramsList,
			}).Errorf("failed to get multiFirm quote: %s", order.Error)

			return nil, fmt.Errorf("Order %d error: %s", i, order.Error)
		}

		actualMakerAmount, _ := new(big.Int).SetString(order.MakerAmount, 10)
		minMakerAmount, _ := new(big.Int).SetString(orders[i].MinMakerAmount, 10)

		if actualMakerAmount.Cmp(minMakerAmount) < 0 {
			logger.WithFields(logger.Fields{
				"paramsList": paramsList,
				"error":      ErrMakerAmountTooLow,
			}).Error("failed to get multiFirm quote")

			return nil, ErrMakerAmountTooLow
		}

		alphaFee, ok := new(big.Int).SetString(order.FeeAmount, 10)
		if !ok {
			logger.WithFields(logger.Fields{
				"paramsList": paramsList,
				"error":      ErrInvalidFeeAmount,
			}).Error("failed to get multiFirm quote")

			return nil, ErrInvalidFeeAmount
		}

		rfqResult = append(rfqResult, &pool.RFQResult{
			NewAmountOut:  actualMakerAmount,
			AlphaFee:      alphaFee,
			AlphaFeeAsset: order.MakerAsset,
			Extra: RFQExtra{
				RFQContractAddress: h.config.RFQContractAddress,
				Info:               order.Info,
				Expiry:             result.Expiry,
				MakerAsset:         order.MakerAsset,
				TakerAsset:         order.TakerAsset,
				Maker:              result.Maker,
				Taker:              result.Taker,
				MakerAmount:        order.MakerAmount,
				TakerAmount:        order.TakerAmount,
				Signature:          order.Signature,
				Recipient:          paramsList[i].Recipient,
				AllowedSender:      result.AllowedSender,
				Partner:            paramsList[i].Source,
				QuoteTimestamp:     time.Now().Unix(),
			},
		})
	}

	return rfqResult, nil
}

func (p *RFQHandler) SupportBatch() bool {
	return true
}

func estimateMinMakerAmount(expectedMakerAmount *big.Int, slippage int64) *big.Int {
	minMakerAmount := new(big.Int).Set(expectedMakerAmount)
	minMakerAmount.Mul(minMakerAmount, big.NewInt(10000-slippage))
	minMakerAmount.Div(minMakerAmount, valueobject.BasisPoint)

	return minMakerAmount
}
