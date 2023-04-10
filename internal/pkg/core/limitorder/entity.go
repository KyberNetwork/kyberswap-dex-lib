package limitorder

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Extra struct {
		SellOrders []*valueobject.Order `json:"sellOrders"`
		BuyOrders  []*valueobject.Order `json:"buyOrders"`
	}

	SwapInfo struct {
		AmountIn     string             `json:"amountIn"`
		SwapSide     SwapSide           `json:"swapSide"`
		FilledOrders []*FilledOrderInfo `json:"filledOrders"`
	}

	FilledOrderInfo struct {
		OrderID              int64  `json:"orderId"`
		FilledTakingAmount   string `json:"filledTakingAmount"`
		FilledMakingAmount   string `json:"filledMakingAmount"`
		FeeAmount            string `json:"feeAmount"`
		TakingAmount         string `json:"takingAmount"`
		MakingAmount         string `json:"makingAmount"`
		Salt                 string `json:"salt"`
		MakerAsset           string `json:"makerAsset"`
		TakerAsset           string `json:"takerAsset"`
		Maker                string `json:"maker"`
		Receiver             string `json:"receiver"`
		AllowedSenders       string `json:"allowedSenders"`
		GetMakerAmount       string `json:"getMakerAmount"`
		GetTakerAmount       string `json:"getTakerAmount"`
		FeeRecipient         string `json:"feeRecipient"`
		MakerTokenFeePercent uint32 `json:"makerTokenFeePercent"`
		MakerAssetData       string `json:"makerAssetData"`
		TakerAssetData       string `json:"takerAssetData"`
		Predicate            string `json:"predicate"`
		Permit               string `json:"permit"`
		Interaction          string `json:"interaction"`
		Signature            string `json:"signature"`
		IsFallBack           bool   `json:"isFallback"`
	}
)

func newFilledOrderInfo(order *valueobject.Order, filledTakingAmount, filledMakingAmount string, feeAmount string) *FilledOrderInfo {
	return &FilledOrderInfo{
		OrderID:              order.ID,
		FilledTakingAmount:   filledTakingAmount,
		FilledMakingAmount:   filledMakingAmount,
		TakingAmount:         order.TakingAmount.String(),
		MakingAmount:         order.MakingAmount.String(),
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
