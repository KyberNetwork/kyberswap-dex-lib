package limitorder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type httpClient struct {
	client *resty.Client
}

func NewHTTPClient(baseURL string) *httpClient {
	// Override MaxConnsPerHost, MaxIdleConnsPerHost
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 100
	transport.MaxIdleConnsPerHost = 100

	client := resty.New()
	client.SetBaseURL(baseURL)

	client.SetTimeout(APITimeout)
	client.SetTransport(transport)

	return &httpClient{
		client: client,
	}
}

func (c *httpClient) ListAllPairs(
	ctx context.Context,
	chainID ChainID,
) ([]*limitOrderPair, error) {
	req := c.client.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
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
		return nil, fmt.Errorf("when performing ListAllPairs with response %v", resp)
	}

	return result.Data.Pairs, nil
}

func (c *httpClient) ListOrders(
	ctx context.Context,
	filter listOrdersFilter,
) ([]*order, error) {
	req := c.client.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			"takerAsset": filter.TakerAsset,
			"makerAsset": filter.MakerAsset,
			"chainId":    strconv.Itoa(int(filter.ChainID)),
		})
	var result listOrdersResult
	resp, err := req.SetResult(&result).Get(listOrdersEndpoint)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("error when performing ListOrders with response %v", result)
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

func (c *httpClient) pruneExpiredOrders(orders []*orderData) []*orderData {
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
