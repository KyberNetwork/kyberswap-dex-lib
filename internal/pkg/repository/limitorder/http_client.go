package limitorder

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	listOrdersEndpoint   = "/read-partner/api/v1/orders"
	listAllPairsEndpoint = "/read-partner/api/v1/orders/pairs"

	ContentTypeHeaderKey = "Content-Type"
)

type (
	httpClient struct {
		client *resty.Client
	}

	listOrdersResult struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    *listOrdersData `json:"data"`
	}

	listAllPairsResult struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    *listAllPairsData `json:"data"`
	}

	listAllPairsData struct {
		Pairs []*valueobject.LimitOrderPair `json:"pairs"`
	}

	listOrdersData struct {
		Orders []*order `json:"orders"`
	}

	order struct {
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
)

func NewHTTPClient(baseURL string) *httpClient {
	client := resty.New()
	client.SetBaseURL(baseURL)
	return &httpClient{
		client,
	}
}

func (c *httpClient) ListOrders(ctx context.Context, filter service.ListOrdersFilter) ([]*valueobject.Order, error) {
	req := c.buildListOrdersRequest(filter)
	var result listOrdersResult
	resp, err := req.SetResult(&result).Get(listOrdersEndpoint)

	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("error when performing ListOrders with response=%v", resp)
	}
	if result.Data == nil {
		return nil, nil
	}
	orders := result.Data.Orders
	if filter.ExcludeExpiredOrder {
		orders = c.pruneExpiredOrders(orders)
	}
	return toOrder(orders)
}

func (c *httpClient) pruneExpiredOrders(orders []*order) []*order {
	timeNow := time.NowFunc
	result := make([]*order, 0, len(orders))
	for _, o := range orders {
		if timeNow().Unix() > o.ExpiredAt {
			continue
		}
		result = append(result, o)
	}
	return result
}

func (c *httpClient) ListAllPairs(ctx context.Context, chainID valueobject.ChainID) ([]*valueobject.LimitOrderPair, error) {
	req := c.client.R().
		SetHeader(ContentTypeHeaderKey, "application/json").
		SetQueryParams(map[string]string{
			"chainId": strconv.Itoa(int(chainID)),
		})
	var result listAllPairsResult
	resp, err := req.SetResult(&result).Get(listAllPairsEndpoint)

	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("error when performing ListAllPairs with response=%v", resp)
	}

	return result.Data.Pairs, nil
}

func (c *httpClient) buildListOrdersRequest(filter service.ListOrdersFilter) *resty.Request {
	req := c.client.R().
		SetHeader(ContentTypeHeaderKey, "application/json").
		SetQueryParams(map[string]string{
			"takerAsset": filter.TakerAsset,
			"makerAsset": filter.MakerAsset,
			"chainId":    strconv.Itoa(int(filter.ChainID)),
		})
	return req
}

func toOrder(orders []*order) ([]*valueobject.Order, error) {
	result := make([]*valueobject.Order, len(orders))
	for i, o := range orders {
		result[i] = &valueobject.Order{
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
