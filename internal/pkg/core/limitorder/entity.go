package limitorder

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type (
	Extra struct {
		SellOrders []*valueobject.Order `json:"sellOrders"`
		BuyOrders  []*valueobject.Order `json:"buyOrders"`
	}

	SwapInfo struct {
		AmountIn     *big.Int           `json:"amountIn"`
		SwapSide     SwapSide           `json:"swapSide"`
		FilledOrders []*FilledOrderInfo `json:"filledOrders"`
	}

	FilledOrderInfo struct {
		OrderID              int64    `json:"orderId"`
		FilledTakingAmount   *big.Int `json:"filledTakingAmount"`
		FilledMakingAmount   *big.Int `json:"filledMakingAmount"`
		FeeAmount            *big.Int `json:"feeAmount"`
		TakingAmount         *big.Int `json:"takingAmount"`
		MakingAmount         *big.Int `json:"makingAmount"`
		Salt                 string   `json:"salt"`
		MakerAsset           string   `json:"makerAsset"`
		TakerAsset           string   `json:"takerAsset"`
		Maker                string   `json:"maker"`
		Receiver             string   `json:"receiver"`
		AllowedSenders       string   `json:"allowedSenders"`
		GetMakerAmount       string   `json:"getMakerAmount"`
		GetTakerAmount       string   `json:"getTakerAmount"`
		FeeRecipient         string   `json:"feeRecipient"`
		MakerTokenFeePercent uint32   `json:"makerTokenFeePercent"`
		MakerAssetData       string   `json:"makerAssetData"`
		TakerAssetData       string   `json:"takerAssetData"`
		Predicate            string   `json:"predicate"`
		Permit               string   `json:"permit"`
		Interaction          string   `json:"interaction"`
		Signature            string   `json:"signature"`
		IsFallBack           bool     `json:"isFallback"`
	}
)

func newFilledOrderInfo(order *valueobject.Order, filledTakingAmount, filledMakingAmount *big.Int, feeAmount *big.Int) *FilledOrderInfo {
	return &FilledOrderInfo{
		OrderID:              order.ID,
		FilledTakingAmount:   filledTakingAmount,
		FilledMakingAmount:   filledMakingAmount,
		TakingAmount:         order.TakingAmount,
		MakingAmount:         order.MakingAmount,
		Salt:                 order.Salt,
		MakerAsset:           order.MakerAsset,
		TakerAsset:           order.TakerAsset,
		Maker:                order.Maker,
		Receiver:             order.Receiver,
		AllowedSenders:       order.AllowedSenders,
		GetMakerAmount:       order.GetMakerAmount,
		GetTakerAmount:       order.GetTakerAmount,
		FeeRecipient:         order.FeeRecipient,
		MakerAssetData:       order.MakerAssetData,
		MakerTokenFeePercent: order.MakerTokenFeePercent,
		TakerAssetData:       order.TakerAssetData,
		Predicate:            order.Predicate,
		Permit:               order.Permit,
		Interaction:          order.Interaction,
		Signature:            order.Signature,
		FeeAmount:            feeAmount,
	}
}
