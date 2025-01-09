package limitorder

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

type client struct {
	restyClient *resty.Client
}

func NewClient(baseURL string) *client {
	restyClient := resty.New().SetBaseURL(baseURL)

	return &client{
		restyClient: restyClient,
	}
}

func (c *client) ListAllPairs(
	ctx context.Context,
	chainID ChainID,
	supportMultiSCs bool,
) ([]*limitOrderPair, error) {
	req := c.restyClient.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			"chainId":                    strconv.Itoa(int(chainID)),
			"hasDistinctContractAddress": strconv.FormatBool(supportMultiSCs),
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
		return nil, fmt.Errorf("when performing ListAllPairs with response %v", resp)
	}

	return result.Data.Pairs, nil
}

func (c *client) ListOrders(
	ctx context.Context,
	filter listOrdersFilter,
) ([]*order, error) {
	req := c.restyClient.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			"takerAsset":      filter.TakerAsset,
			"makerAsset":      filter.MakerAsset,
			"chainId":         strconv.Itoa(int(filter.ChainID)),
			"contractAddress": filter.ContractAddress,

			"includeInsufficientBalance": strconv.FormatBool(filter.IncludeInsufficientBalanceOrder),
		})
	var result listOrdersResult
	resp, err := req.SetResult(&result).Get(listOrdersEndpoint)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("error when ListOrders, url: %v, response: %v", resp.Request.URL, resp.String())
	}

	if result.Code != 0 {
		return nil, errors.New(result.Message)
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

func (c *client) pruneExpiredOrders(orders []*orderData) []*orderData {
	timeNow := time.Now().Unix()
	result := make([]*orderData, 0, len(orders))
	for _, o := range orders {
		if timeNow > o.ExpiredAt {
			continue
		}
		result = append(result, o)
	}
	return result
}

func (c *client) GetOpSignatures(
	ctx context.Context,
	chainId ChainID,
	orderIds []int64,
) ([]*operatorSignatures, error) {
	req := c.restyClient.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetQueryParam("chainId", strconv.Itoa(int(chainId))).
		SetQueryParamsFromValues(url.Values{
			"orderIds": lo.Map(orderIds, func(o int64, _ int) string { return strconv.FormatInt(o, 10) }),
		})
	var result getOpSignaturesResult
	resp, err := req.SetResult(&result).Get(getOpSignaturesEndpoint)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("error when getting Op Signatures, url: %v, response: %v", resp.Request.URL, resp.String())
	}

	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}

	if result.Data == nil {
		return nil, nil
	}

	return result.Data.OperatorSignatures, nil
}
