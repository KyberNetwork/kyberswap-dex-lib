package limitorder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

// defaultHTTPTimeout exists only to bound a hung backend, so it is deliberately far
// looser than any caller's SLO. Latency-sensitive callers set Config.HTTPTimeout.
const defaultHTTPTimeout = 30 * time.Second

type httpClient struct {
	client *resty.Client
	// opSigCache is nil when caching is disabled; every method on it tolerates that.
	opSigCache *opSignatureCache
}

// Deprecated: use NewHTTPClientWithConfig, which also supports a timeout, retries and a
// caller-supplied Transport.
func NewHTTPClient(baseURL string) *httpClient {
	return NewHTTPClientWithConfig(&Config{LimitOrderHTTPUrl: baseURL})
}

// Deprecated: use NewHTTPClientWithConfig and set Config.HTTPClient instead.
func NewHTTPClientWithRestyClient(baseURL string, client *resty.Client) *httpClient {
	client.SetBaseURL(baseURL)
	// bound only a client that has none of its own; overwriting would silently retune
	// a caller that already chose a timeout
	if client.GetClient().Timeout <= 0 {
		client.SetTimeout(defaultHTTPTimeout)
	}

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
		client:     client,
		opSigCache: newOpSignatureCache(cfg.OpSignatureCacheTTL.Duration, cfg.OpSignatureValidityMargin.Duration),
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
	cached, uncached := c.opSigCache.reusable(orderIds)
	if len(uncached) == 0 {
		return cached, nil
	}

	fetched, err := c.fetchOpSignatures(ctx, chainId, uncached)
	if err != nil {
		return nil, err
	}
	c.opSigCache.store(fetched)

	if len(cached) == 0 {
		return fetched, nil
	}
	return append(cached, fetched...), nil
}

func (c *httpClient) fetchOpSignatures(
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

// The operator signs (orderHash, opExpireTime) only — never amount, taker or recipient — and
// the backend mints a new expiry only once the current one has less than its minimum valid
// time left. Repeated calls for the same order inside that gap hand back a byte-identical
// signature, so a cache hit is equivalent to a live call rather than an approximation of one.
type opSignatureCache struct {
	mu             sync.Mutex
	entries        map[int64]opSignatureCacheEntry
	ttl            time.Duration
	validityMargin time.Duration
}

type opSignatureCacheEntry struct {
	sig     operatorSignatures
	staleAt time.Time
}

// defaultOpSignatureValidityMargin approximates "still valid when the tx executes": dex-lib
// cannot see the target block time, so it requires the signature to outlive two Ethereum
// blocks instead.
const defaultOpSignatureValidityMargin = 24 * time.Second

// maxCachedOpSignatures is the size past which unusable entries are swept before new ones are
// stored. No entry survives its TTL, so a sweep always reclaims everything older than that.
const maxCachedOpSignatures = 16384

func newOpSignatureCache(ttl, validityMargin time.Duration) *opSignatureCache {
	if ttl <= 0 {
		return nil
	}
	if validityMargin <= 0 {
		validityMargin = defaultOpSignatureValidityMargin
	}

	return &opSignatureCache{
		entries:        make(map[int64]opSignatureCacheEntry),
		ttl:            ttl,
		validityMargin: validityMargin,
	}
}

// reusable splits orderIds into the signatures that can be served from memory and the ids that
// still have to be fetched.
func (c *opSignatureCache) reusable(orderIds []int64) (cached []*operatorSignatures, uncached []int64) {
	if c == nil {
		return nil, orderIds
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range orderIds {
		entry, ok := c.entries[id]
		if !ok || !entry.usableAt(now, c.validityMargin) {
			uncached = append(uncached, id)
			continue
		}
		sig := entry.sig
		cached = append(cached, &sig)
	}

	return cached, uncached
}

func (c *opSignatureCache) store(sigs []*operatorSignatures) {
	if c == nil || len(sigs) == 0 {
		return
	}

	now := time.Now()
	staleAt := now.Add(c.ttl)

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) > maxCachedOpSignatures {
		for id, entry := range c.entries {
			if !entry.usableAt(now, c.validityMargin) {
				delete(c.entries, id)
			}
		}
	}
	for _, sig := range sigs {
		if sig == nil {
			continue
		}
		c.entries[sig.ID] = opSignatureCacheEntry{sig: *sig, staleAt: staleAt}
	}
}

func (e opSignatureCacheEntry) usableAt(now time.Time, validityMargin time.Duration) bool {
	return now.Before(e.staleAt) && e.sig.OperatorSignatureExpiredAt > now.Add(validityMargin).Unix()
}
