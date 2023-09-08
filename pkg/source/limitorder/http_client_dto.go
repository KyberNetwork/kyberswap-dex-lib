package limitorder

import (
	"fmt"
	"math/big"
	"strconv"
)

const (
	listOrdersEndpoint      = "/read-partner/api/v1/orders"
	listAllPairsEndpoint    = "/read-partner/api/v1/orders/pairs"
	getOpSignaturesEndpoint = "/read-partner/api/v1/orders/operator-signature"
)

type (
	listOrdersResult struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    *listOrdersData `json:"data"`
	}

	getOpSignaturesResult struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    *struct {
			OperatorSignatures []*operatorSignatures `json:"orders"`
		} `json:"data"`
	}

	listAllPairsResult struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    *listAllPairsData `json:"data"`
	}

	listAllPairsData struct {
		Pairs []*limitOrderPair `json:"pairs"`
	}

	limitOrderPair struct {
		MakerAsset      string `json:"makerAsset"`
		TakeAsset       string `json:"takerAsset"`
		ContractAddress string `json:"contractAddress"`
	}

	listOrdersData struct {
		Orders []*orderData `json:"orders"`
	}

	orderData struct {
		ID                   int64  `json:"id"`
		ChainID              string `json:"chainId"`
		Salt                 string `json:"salt"`
		Signature            string `json:"signature"`
		MakerAsset           string `json:"makerAsset"`
		TakerAsset           string `json:"takerAsset"`
		Maker                string `json:"maker"`
		Receiver             string `json:"receiver"`
		AllowedSenders       string `json:"allowedSenders"`
		MakingAmount         string `json:"makingAmount"`
		TakingAmount         string `json:"takingAmount"`
		FilledMakingAmount   string `json:"filledMakingAmount"`
		FilledTakingAmount   string `json:"filledTakingAmount"`
		FeeConfig            string `json:"feeConfig"`
		FeeRecipient         string `json:"feeRecipient"`
		MakerTokenFeePercent string `json:"makerTokenFeePercent"`
		MakerAssetData       string `json:"makerAssetData"`
		TakerAssetData       string `json:"takerAssetData"`
		GetMakerAmount       string `json:"getMakerAmount"`
		GetTakerAmount       string `json:"getTakerAmount"`
		Predicate            string `json:"predicate"`
		Permit               string `json:"permit"`
		Interaction          string `json:"interaction"`
		ExpiredAt            int64  `json:"expiredAt"`
	}

	listOrdersFilter struct {
		ChainID             ChainID
		MakerAsset          string
		TakerAsset          string
		ContractAddress     string
		ExcludeExpiredOrder bool
	}

	order struct {
		ID                   int64    `json:"id"`
		ChainID              string   `json:"chainId"`
		Salt                 string   `json:"salt"`
		Signature            string   `json:"signature"`
		MakerAsset           string   `json:"makerAsset"`
		TakerAsset           string   `json:"takerAsset"`
		Maker                string   `json:"maker"`
		Receiver             string   `json:"receiver"`
		AllowedSenders       string   `json:"allowedSenders"`
		MakingAmount         *big.Int `json:"makingAmount"`
		TakingAmount         *big.Int `json:"takingAmount"`
		FeeConfig            *big.Int `json:"feeConfig"`
		FeeRecipient         string   `json:"feeRecipient"`
		FilledMakingAmount   *big.Int `json:"filledMakingAmount"`
		FilledTakingAmount   *big.Int `json:"filledTakingAmount"`
		MakerTokenFeePercent uint32   `json:"makerTokenFeePercent"`
		MakerAssetData       string   `json:"makerAssetData"`
		TakerAssetData       string   `json:"takerAssetData"`
		GetMakerAmount       string   `json:"getMakerAmount"`
		GetTakerAmount       string   `json:"getTakerAmount"`
		Predicate            string   `json:"predicate"`
		Permit               string   `json:"permit"`
		Interaction          string   `json:"interaction"`
		ExpiredAt            int64    `json:"expiredAt"`
	}

	operatorSignatures struct {
		ID                         int64  `json:"id"`
		ChainID                    string `json:"chainId"`
		OperatorSignature          string `json:"operatorSignature"`
		OperatorSignatureExpiredAt int64  `json:"operatorSignatureExpiredAt"`
	}
)

func toOrder(ordersData []*orderData) ([]*order, error) {
	result := make([]*order, len(ordersData))
	for i, o := range ordersData {
		result[i] = &order{
			ID:             o.ID,
			Salt:           o.Salt,
			ChainID:        o.ChainID,
			Signature:      o.Signature,
			MakerAsset:     o.MakerAsset,
			TakerAsset:     o.TakerAsset,
			Maker:          o.Maker,
			Receiver:       o.Receiver,
			AllowedSenders: o.AllowedSenders,
			FeeRecipient:   o.FeeRecipient,
			MakerAssetData: o.MakerAssetData,
			TakerAssetData: o.TakerAssetData,
			GetMakerAmount: o.GetMakerAmount,
			GetTakerAmount: o.GetTakerAmount,
			Predicate:      o.Predicate,
			Permit:         o.Permit,
			Interaction:    o.Interaction,
			ExpiredAt:      o.ExpiredAt,
		}
		makerTokenFeePercent, err := strconv.ParseInt(o.MakerTokenFeePercent, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing makerTokenFeePercent error by %s", err.Error())
		}
		result[i].MakerTokenFeePercent = uint32(makerTokenFeePercent)
		takingAmount, ok := new(big.Int).SetString(o.TakingAmount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid takingAmount")
		}
		makingAmount, ok := new(big.Int).SetString(o.MakingAmount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid makingAmount")
		}
		if len(o.FeeConfig) > 0 {
			feeConfig, ok := new(big.Int).SetString(o.FeeConfig, 10)
			if !ok {
				return nil, fmt.Errorf("invalid feeConfig %v", o.FeeConfig)
			}
			result[i].FeeConfig = feeConfig
		}
		if len(o.FilledTakingAmount) > 0 {
			filledTakingAmount, ok := new(big.Int).SetString(o.FilledTakingAmount, 10)
			if !ok {
				return nil, fmt.Errorf("parsing filledTakingAmount error")
			}
			result[i].FilledTakingAmount = filledTakingAmount
		}
		if len(o.FilledMakingAmount) > 0 {
			filledMakingAmount, ok := new(big.Int).SetString(o.FilledMakingAmount, 10)
			if !ok {
				return nil, fmt.Errorf("invalid filledMakingAmount")
			}
			result[i].FilledMakingAmount = filledMakingAmount
		}
		result[i].TakingAmount = takingAmount
		result[i].MakingAmount = makingAmount
	}
	return result, nil
}
