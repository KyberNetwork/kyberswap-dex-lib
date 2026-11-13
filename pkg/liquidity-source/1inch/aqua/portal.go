package oneinchaqua

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

// Reasons a Portal sweep fails, the labels of the portal_fallback metric.
const (
	portalNoKey       = "no_key"
	portalNetwork     = "network"
	portalTimeoutFail = "timeout"
	portalAuth        = "unauthorized"
	portalRateLimit   = "rate_limited"
	portalServer      = "server_error"
	portalStatus      = "bad_status"
	portalMalformed   = "malformed"
	portalWrongChain  = "wrong_chain"
	portalPageCap     = "page_cap"
	// portalBadItem is an item the lister cannot read (bad address, hash or hex) or whose bytes the Portal
	// left out: the on-chain logs hold it, so the run falls back.
	portalBadItem = "bad_item"
)

const portalPath = "/aqua/v1.0/strategies/opened"

// portalItem is one open strategy of the Portal's /strategies/opened. The Portal also sends balances
// (a cached snapshot with no block number) and the open time; neither is read.
type portalItem struct {
	ChainId                    uint64  `json:"chainId"`
	Maker                      string  `json:"maker"`
	App                        string  `json:"app"`
	StrategyHash               string  `json:"strategyHash"`
	StrategyBytes              string  `json:"strategyBytes"`
	StrategyBytesOmittedReason *string `json:"strategyBytesOmittedReason"`
	Tokens                     []struct {
		Address string `json:"address"`
	} `json:"tokens"`
}

// shipped is the item in the form the log path admits, false when it cannot be read: bytes the Portal left
// out, a bad hex, hash, maker, app or token address.
func (it *portalItem) shipped() (*shipped, bool) {
	strategy, err := hexutil.Decode(it.StrategyBytes)
	hash, hashErr := hexutil.Decode(it.StrategyHash)
	if it.StrategyBytesOmittedReason != nil || len(strategy) == 0 || err != nil || hashErr != nil || len(hash) != 32 ||
		!common.IsHexAddress(it.Maker) || !common.IsHexAddress(it.App) {
		return nil, false
	}
	s := &shipped{maker: common.HexToAddress(it.Maker), app: common.HexToAddress(it.App),
		hash: common.BytesToHash(hash), strategy: strategy}
	for _, tk := range it.Tokens {
		if !common.IsHexAddress(tk.Address) {
			return nil, false
		}
		if a := common.HexToAddress(tk.Address); !lo.Contains(s.tokens, a) {
			s.tokens = append(s.tokens, a)
		}
	}
	return s, true
}

type portalPage struct {
	Items      []portalItem `json:"items"`
	NextCursor string       `json:"nextCursor"`
}

// portalError is a failed sweep: the reason is the metric label, err carries no secret.
type portalError struct {
	reason string
	err    error
}

func (e *portalError) Error() string { return "1inch-aqua portal: " + e.reason + ": " + e.err.Error() }
func (e *portalError) Unwrap() error { return e.err }

// portalClient pages through the open strategies of one chain. It holds no state between sweeps.
type portalClient struct {
	cfg  *portalSettings
	http *resty.Client
}

func newPortalClient(cfg *portalSettings) *portalClient {
	c := resty.New().
		SetBaseURL(cfg.baseURL).
		// A redirect would carry the Authorization header to wherever it points: refuse it.
		SetRedirectPolicy(resty.NoRedirectPolicy())
	return &portalClient{cfg: cfg, http: c}
}

// sweep reads every open strategy of the chain, at most maxPages pages. On a failure it returns the items
// of the pages read before it together with the error: they are valid, and the caller may use them.
func (c *portalClient) sweep(ctx context.Context) ([]portalItem, *portalError) {
	var (
		items  []portalItem
		cursor string
	)
	for page := 0; ; page++ {
		if page >= c.cfg.maxPages {
			return items, &portalError{portalPageCap, errors.New("more pages than portalMaxPages")}
		}
		p, perr := c.fetch(ctx, cursor)
		if perr != nil {
			return items, perr
		}
		for i := range p.Items {
			if p.Items[i].ChainId != c.cfg.chainId {
				return items, &portalError{portalWrongChain, errors.New("an item of another chain")}
			}
		}
		items = append(items, p.Items...)
		if p.NextCursor == "" || len(p.Items) == 0 {
			return items, nil
		}
		if p.NextCursor == cursor {
			return items, &portalError{portalMalformed, errors.New("the cursor did not advance")}
		}
		cursor = p.NextCursor
	}
}

// fetch reads one page. A 429 is retried up to the configured number of times with a growing wait; every other
// failure ends the sweep. Errors name the status or the failure kind and never the URL query, a header or the
// body: nothing the Portal or the HTTP client says is passed on unchecked.
func (c *portalClient) fetch(ctx context.Context, cursor string) (*portalPage, *portalError) {
	for attempt := 0; ; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, c.cfg.timeout)
		req := c.http.R().SetContext(reqCtx).
			SetAuthToken(c.cfg.key.value).
			SetHeader("Accept", "application/json").
			SetQueryParam("chainIds", strconv.FormatUint(c.cfg.chainId, 10)).
			SetQueryParam("limit", strconv.Itoa(c.cfg.pageLimit))
		if cursor != "" {
			req.SetQueryParam("cursor", cursor)
		}
		resp, err := req.Get(portalPath)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil, &portalError{portalNetwork, ctx.Err()}
			}
			var netErr net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
				return nil, &portalError{portalTimeoutFail, errors.New("request timed out")}
			}
			// A *url.Error names the request URL with its query; keep only the cause.
			if uerr := (*url.Error)(nil); errors.As(err, &uerr) {
				err = uerr.Err
			}
			return nil, &portalError{portalNetwork, errors.New(c.cfg.scrub(err.Error()))}
		}
		switch code := resp.StatusCode(); {
		case code == http.StatusOK:
			// A body without an items array ({}, null, an error object, another schema) decodes without error,
			// so it is checked: only a real list, possibly empty, is a page.
			var page portalPage
			if err := json.Unmarshal(resp.Body(), &page); err != nil || page.Items == nil {
				return nil, &portalError{portalMalformed, errors.New("the body is not a page of strategies")}
			}
			return &page, nil
		case code == http.StatusUnauthorized || code == http.StatusForbidden:
			return nil, &portalError{portalAuth, errors.New("status " + strconv.Itoa(code))}
		case code == http.StatusTooManyRequests:
			if attempt >= c.cfg.retries {
				return nil, &portalError{portalRateLimit, errors.New("status 429 after " + strconv.Itoa(attempt) + " retries")}
			}
			if err := sleepCtx(ctx, time.Duration(attempt+1)*c.cfg.backoff); err != nil {
				return nil, &portalError{portalNetwork, err}
			}
		case code >= 500:
			return nil, &portalError{portalServer, errors.New("status " + strconv.Itoa(code))}
		default:
			return nil, &portalError{portalStatus, errors.New("status " + strconv.Itoa(code))}
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
