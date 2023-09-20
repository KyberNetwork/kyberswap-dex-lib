package client

import (
	"context"
	"strconv"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

type httpClient struct {
	client *resty.Client
	config *iziswap.HTTPConfig
}

const (
	listPoolsEndpoint = "/api/v1/izi_swap/meta_record"

	POOL_LIST_LIMIT = 1000
	POOL_TYPE_VALUE = "10"
)

func NewHTTPClient(config *iziswap.HTTPConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &httpClient{
		client: client,
		config: config,
	}
}

// ListPools example params="chain_id=324&type=10&version=v2&time_start=2023-06-02T13:53:13&page=1&page_size=10&order_by=time"
func (c *httpClient) ListPools(ctx context.Context, params iziswap.ListPoolsParams) ([]iziswap.PoolInfo, error) {
	var result iziswap.ListPoolsResponse
	resp, err := c.client.R().
		SetQueryParams(map[string]string{
			"chain_id":   strconv.Itoa(params.ChainId),
			"type":       POOL_TYPE_VALUE,
			"version":    params.Version,
			"time_start": time.Unix(int64(params.TimeStart), 0).Format("2006-01-02T15:04:05"),
			"format":     "json",
			"order_by":   "time",
			"page_size":  strconv.Itoa(params.Limit),
		}).
		SetContext(ctx).
		SetResult(&result).
		Get(listPoolsEndpoint)

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, errors.Wrapf(ErrListPoolsFailed, "response status: %v, response error %v", resp.Status(), resp.Error())
	}

	return result.Data, nil
}
