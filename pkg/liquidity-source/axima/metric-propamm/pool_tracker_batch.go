package metricpropamm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ poolpkg.IBatchPoolStateTracker = (*PoolTracker)(nil)

type batchResult struct {
	PoolAddress string  `json:"poolAddress"`
	Quote       *bidAsk `json:"quote"`
	Error       *struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
	} `json:"error"`
}

// batchStatusError is a whole-request failure with its HTTP status.
type batchStatusError struct {
	status int
	body   string
}

func (e *batchStatusError) Error() string {
	return fmt.Sprintf("bid_ask batch API error %d: %s", e.status, e.body)
}

// quoteEntry is one pool's batch outcome. A nil err with a nil ba never occurs.
// unavailable marks a definitive per-pool answer (4xx): the pool is published as not quotable.
// Any other err keeps the stored state.
type quoteEntry struct {
	ba          *bidAsk
	err         error
	unavailable bool
}

// GetNewPoolStates refreshes all pools with one POST /bid_ask/batch per 100 pools.
// A whole-request failure (429, 5xx, timeout) keeps the stored state of the pools it covers;
// quotes expire on their own via MaxAge. Only a per-pool 4xx marks that pool unavailable.
func (t *PoolTracker) GetNewPoolStates(ctx context.Context, pools []entity.Pool) ([]entity.Pool, []error) {
	out := make([]entity.Pool, len(pools))
	errs := make([]error, len(pools))
	unswappable := t.unswappablePools(ctx, pools)

	addrs := make([]string, 0, len(pools))
	for i, p := range pools {
		if !unswappable[i] {
			addrs = append(addrs, strings.ToLower(p.Address))
		}
	}
	quotes := make(map[string]quoteEntry, len(addrs))
	for _, chunk := range lo.Chunk(addrs, batchSize) {
		t.fetchQuotes(ctx, chunk, quotes, true)
	}

	for i, p := range pools {
		poolAddr := strings.ToLower(p.Address)
		var prev axima.Extra
		_ = json.Unmarshal([]byte(p.Extra), &prev)
		extensions, ok := t.extensionCount(ctx, poolAddr)
		if !ok {
			extensions = prev.Extensions
		}

		var ba *bidAsk
		var fetchErr error
		if !unswappable[i] {
			q := quotes[poolAddr]
			switch {
			case q.ba != nil:
				ba = q.ba
			case q.unavailable:
				fetchErr = q.err
			default:
				errs[i] = lo.Ternary(q.err != nil, q.err, fmt.Errorf("pool missing from bid_ask batch response"))
				continue
			}
		}
		out[i], errs[i] = t.buildPool(p, extensions, unswappable[i], ba, fetchErr)
	}
	return out, errs
}

// unswappablePools mirrors the single-pool checks: whitelisted pools and pools paused on-chain
// (one multicall for all pools). A failed read leaves the pool swappable.
func (t *PoolTracker) unswappablePools(ctx context.Context, pools []entity.Pool) []bool {
	res := make([]bool, len(pools))
	if t.ethrpcClient == nil {
		for i, p := range pools {
			res[i] = whitelisted(p)
		}
		return res
	}
	slots := make([][32]byte, len(pools))
	var idx []int
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i, p := range pools {
		if res[i] = whitelisted(p); res[i] {
			continue
		}
		idx = append(idx, i)
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "extsload",
			Params: []any{[32]byte{}}, // storage slot 0; low byte = pause level (0 = active)
		}, []any{&slots[i]})
	}
	if len(idx) == 0 {
		return res
	}
	resp, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType}).Warnf("failed to read on-chain pause state: %v", err)
		return res
	}
	for k, i := range idx {
		if resp.Result[k] && slots[i][31] != 0 {
			res[i] = true
		}
	}
	return res
}

func whitelisted(p entity.Pool) bool {
	var se axima.StaticExtra
	return json.Unmarshal([]byte(p.StaticExtra), &se) == nil && se.SwapWhitelistingEnabled
}

// fetchQuotes fills out for addrs. On a 503 (batch took >20s server side) it retries once as
// quarter-size requests (split=false stops further splitting).
func (t *PoolTracker) fetchQuotes(ctx context.Context, addrs []string, out map[string]quoteEntry, split bool) {
	results, err := t.postBatch(ctx, addrs)
	if err != nil {
		if se, ok := err.(*batchStatusError); ok && se.status == http.StatusServiceUnavailable && split && len(addrs) > 1 {
			for _, c := range lo.Chunk(addrs, (len(addrs)+3)/4) {
				t.fetchQuotes(ctx, c, out, false)
			}
			return
		}
		for _, a := range addrs {
			out[a] = quoteEntry{err: err}
		}
		return
	}
	for _, r := range results {
		a := strings.ToLower(r.PoolAddress)
		switch {
		case r.Quote != nil:
			out[a] = quoteEntry{ba: r.Quote}
		case r.Error != nil:
			out[a] = quoteEntry{
				err:         fmt.Errorf("bid_ask error %d: %s", r.Error.Status, r.Error.Error),
				unavailable: r.Error.Status >= 400 && r.Error.Status < 500,
			}
		}
	}
}

func (t *PoolTracker) postBatch(ctx context.Context, addrs []string) ([]batchResult, error) {
	if until := time.Unix(0, t.blockedUntil.Load()); time.Now().Before(until) {
		return nil, &batchStatusError{status: http.StatusTooManyRequests, body: "backing off until " + until.Format(time.RFC3339)}
	}
	var resp struct {
		Results []batchResult `json:"results"`
	}
	res, err := t.client.R().SetContext(ctx).SetResult(&resp).
		SetBody(map[string]any{"pools": addrs}).
		Post(fmt.Sprintf("/public/v1/evm/%d/bid_ask/batch", t.config.ChainID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() == http.StatusTooManyRequests {
		// Honor Retry-After for every task on this pod; the stored state is kept meanwhile.
		wait := retryAfterDefault
		if s, err := strconv.Atoi(res.Header().Get("Retry-After")); err == nil && s > 0 {
			wait = min(time.Duration(s)*time.Second, retryAfterMax)
		}
		t.blockedUntil.Store(time.Now().Add(wait).UnixNano())
	}
	if res.IsError() {
		return nil, &batchStatusError{status: res.StatusCode(), body: res.String()}
	}
	return resp.Results, nil
}
