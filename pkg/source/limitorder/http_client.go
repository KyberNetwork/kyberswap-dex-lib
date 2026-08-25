package limitorder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

// defaultHTTPTimeout exists only to bound a hung backend, so it is deliberately far
// looser than any caller's SLO. Latency-sensitive callers set Config.HTTPTimeout.
const defaultHTTPTimeout = 30 * time.Second

type httpClient struct {
	client *resty.Client
}

// Deprecated: use NewHTTPClientWithConfig, which also supports a timeout, retries and a
// caller-supplied Transport.
func NewHTTPClient(baseURL string) *httpClient {
	return NewHTTPClientWithConfig(&Config{LimitOrderHTTPUrl: baseURL})
}

// Deprecated: use NewHTTPClientWithConfig and set Config.HTTPClient instead.
func NewHTTPClientWithRestyClient(baseURL string, client *resty.Client) *httpClient {
	client.SetBaseURL(baseURL).SetTimeout(defaultHTTPTimeout)

	return &httpClient{
		client: client,
	}
}

func NewHTTPClientWithConfig(cfg *Config) *httpClient {
	httpCli := cfg.HTTPClient
	if httpCli == nil {
		httpCli = http.DefaultClient
	}

	timeout := cfg.HTTPTimeout.Duration
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	// resty skips its single-shot path for any non-zero retry count, then runs
	// `for attempt := 0; attempt <= retries` — a negative count executes the request
	// zero times and hands back a nil response for the caller to dereference.
	retryCount := max(cfg.HTTPRetryCount, 0)

	// shallow-copy so SetTimeout writes a per-dex Timeout instead of mutating the
	// caller's client; the Transport pointer, and so the connection pool, is shared.
	client := resty.NewWithClient(lo.ToPtr(lo.FromPtr(httpCli))).
		SetBaseURL(cfg.LimitOrderHTTPUrl).
		SetTimeout(timeout).
		SetRetryCount(retryCount)

	return &httpClient{
		client: client,
	}
}

func (c *httpClient) ListAllPairs(
	ctx context.Context,
	chainID ChainID,
	supportMultiSCs bool,
) ([]*limitOrderPair, error) {
	req := c.client.R().SetContext(ctx).
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

func (c *httpClient) ListOrders(
	ctx context.Context,
	filter listOrdersFilter,
) ([]*order, error) {
	req := c.client.R().SetContext(ctx).
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

func (c *httpClient) GetOpSignatures(
	ctx context.Context,
	chainId ChainID,
	orderIds []int64,
) ([]*operatorSignatures, error) {
	req := c.client.R().SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetQueryParam("chainId", strconv.Itoa(int(chainId))).
		SetQueryParamsFromValues(url.Values{
			"orderIds": lo.Map(orderIds, func(o int64, _ int) string { return strconv.FormatInt(o, 10) }),
		})
	var result getOpSignaturesResult
	resp, err := req.SetResult(&result).Get(getOpSignaturesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOpSignaturesFailed, err)
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%w: error when getting Op Signatures, url: %v, response: %v", ErrGetOpSignaturesFailed, resp.Request.URL, resp.String())
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("%w: %s", ErrGetOpSignaturesFailed, result.Message)
	}

	if result.Data == nil {
		return nil, nil
	}

	return result.Data.OperatorSignatures, nil
}
