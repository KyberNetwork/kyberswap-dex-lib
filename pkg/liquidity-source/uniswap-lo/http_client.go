package uniswaplo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	dutchV2OrderLimit      = 10000
	retrieveLimitOrderPath = "/limit-orders"
	defaultTimeout         = 10 * time.Second
)

type UniSwapXClient struct {
	client *resty.Client
}

func NewUniSwapXClient(baseURL string) *UniSwapXClient {
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetTimeout(defaultTimeout)

	return &UniSwapXClient{
		client: client,
	}
}

func (c *UniSwapXClient) FetchDutchOrders(ctx context.Context, q DutchOrderQuery) (DutchOrdersResponse, error) {
	var response DutchOrdersResponse

	// Create query params, excluding empty values
	queryParams := make(map[string]string)
	if q.Limit > 0 {
		queryParams["limit"] = fmt.Sprintf("%d", q.Limit)
	}
	if q.OrderStatus != "" {
		queryParams["orderStatus"] = string(q.OrderStatus)
	}
	if q.OrderType != "" {
		queryParams["orderType"] = string(q.OrderType)
	}
	if q.OrderHash != "" {
		queryParams["orderHash"] = q.OrderHash
	}
	if q.Swapper != "" {
		queryParams["swapper"] = q.Swapper
	}
	if q.Filler != "" {
		queryParams["filler"] = q.Filler
	}
	if q.Cursor != "" {
		queryParams["cursor"] = q.Cursor
	}
	queryParams["chainId"] = fmt.Sprintf("%d", q.ChainID)
	if q.SortKey != "" {
		queryParams["sortKey"] = string(q.SortKey)
	}
	if q.Sort != "" {
		queryParams["sort"] = q.Sort
	}

	request := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryParams).
		SetResult(&response)

	// Enable debug mode to print request details
	c.client.SetDebug(true)

	_, err := request.Get(retrieveLimitOrderPath)

	if err != nil {
		return DutchOrdersResponse{}, fmt.Errorf("failed to fetch dutch orders: %w", err)
	}

	// fmt.Printf("Request URL: %s\n", resp.Request.URL)
	// fmt.Printf("Response Status: %s\n", resp.Status())
	// fmt.Printf("Response Body: %s\n", resp.String())

	return response, nil
}
