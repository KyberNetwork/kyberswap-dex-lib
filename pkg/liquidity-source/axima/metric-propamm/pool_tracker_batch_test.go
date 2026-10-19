package metricpropamm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
)

const batchQuote = `{"bidAdj":"1","askAdj":"2","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[{"binIdx":0,"price":"1","cumulativeVolume":"1","priceImpactE6":"0"}],"bids":[]},"priceProviderStatus":"ok"}`

func batchPools(addrs ...string) []entity.Pool {
	ps := make([]entity.Pool, len(addrs))
	for i, a := range addrs {
		ps[i] = entity.Pool{Address: a, Reserves: []string{"9", "9"}, StaticExtra: `{}`}
	}
	return ps
}

func newBatchTracker(srv *httptest.Server) *PoolTracker {
	cfg := &axima.Config{DexID: DexType, ChainID: 8453, HTTPConfig: axima.HTTPConfig{BaseURL: srv.URL}}
	return NewPoolTracker(cfg, nil)
}

func quoteAvailable(t *testing.T, p entity.Pool) bool {
	var e axima.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	return e.QuoteAvailable
}

// One request serves all pools; a per-pool error must only hit that pool: a 4xx marks it
// unavailable, a temporary 5xx or a missing entry keeps its stored state (error), the rest update.
func TestGetNewPoolStatesIsolatesPerPoolErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/bid_ask/batch") {
			w.WriteHeader(http.StatusNotFound) // /metadata: extension counts fall back to the previous value
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"poolAddress":"0xA1","quote":` + batchQuote + `},
			{"poolAddress":"0xa2","error":{"status":404,"error":"nf"}},
			{"poolAddress":"0xa3","error":{"status":502,"error":"up"}},
			{"poolAddress":"0xa5","quote":` + batchQuote + `}]}`))
	}))
	defer srv.Close()

	out, errs := newBatchTracker(srv).GetNewPoolStates(context.Background(),
		batchPools("0xa1", "0xa2", "0xa3", "0xa4", "0xa5"))
	require.EqualValues(t, 1, calls.Load())
	require.NoError(t, errs[0])
	require.True(t, quoteAvailable(t, out[0]))
	require.Equal(t, entity.PoolReserves{"5", "6"}, out[0].Reserves)
	require.NoError(t, errs[1])
	require.False(t, quoteAvailable(t, out[1]))
	require.Equal(t, entity.PoolReserves{"0", "0"}, out[1].Reserves)
	require.Error(t, errs[2])
	require.Error(t, errs[3])
	require.NoError(t, errs[4])
	require.True(t, quoteAvailable(t, out[4]))
}

// A 429 must not mark pools unavailable (errors only) and Retry-After must stop further requests.
func TestGetNewPoolStatesHonorsRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bid_ask/batch") {
			calls.Add(1)
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}))
	defer srv.Close()
	tr := newBatchTracker(srv)

	for range 2 {
		_, errs := tr.GetNewPoolStates(context.Background(), batchPools("0xa1", "0xa2"))
		require.Error(t, errs[0])
		require.Error(t, errs[1])
	}
	require.EqualValues(t, 1, calls.Load(), "second batch must not hit the API during Retry-After")
}

// A whole-batch 503 (>20s upstream) is retried once as smaller requests, so one slow pool group
// does not drop every pool.
func TestGetNewPoolStatesSplitsOn503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/bid_ask/batch") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		var body struct{ Pools []string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Pools) > 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var rs []string
		for _, p := range body.Pools {
			rs = append(rs, fmt.Sprintf(`{"poolAddress":%q,"quote":%s}`, p, batchQuote))
		}
		_, _ = w.Write([]byte(`{"results":[` + strings.Join(rs, ",") + `]}`))
	}))
	defer srv.Close()

	out, errs := newBatchTracker(srv).GetNewPoolStates(context.Background(),
		batchPools("0xa1", "0xa2", "0xa3", "0xa4", "0xa5", "0xa6", "0xa7", "0xa8"))
	require.EqualValues(t, 5, calls.Load()) // 1 failed + 4 chunks of 2
	for i := range out {
		require.NoError(t, errs[i])
		require.True(t, quoteAvailable(t, out[i]))
	}
}
