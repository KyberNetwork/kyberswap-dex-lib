package metricpropamm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	pastTs       = 1700000000 // quoteTimestamp keeps a past serverTs, so both paths stamp the same Timestamp
	quoteWithTs  = `{"bidAdj":"11","askAdj":"13","totalToken0Available":"5","totalToken1Available":"6","serverTs":1700000000,"depth":{"asks":[{"binIdx":0,"price":"1","cumulativeVolume":"1","priceImpactE6":"0"}],"bids":[{"binIdx":1,"price":"2","cumulativeVolume":"3","priceImpactE6":"4"}]},"priceProviderStatus":"ok"}`
	quoteFeedDwn = `{"bidAdj":"1","askAdj":"2","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[],"bids":[]},"priceProviderStatus":"feed_down"}`
)

// venue is a fake Metric API serving the same per-pool answers on GET /bid_ask and POST /bid_ask/batch.
type venue struct {
	quotes map[string]string // lowercase pool -> quote json
	errs   map[string]int    // lowercase pool -> http status
	meta   string
}

func (v venue) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/metadata"):
			_, _ = w.Write([]byte(v.meta))
		case strings.HasSuffix(r.URL.Path, "/bid_ask/batch"):
			var body struct{ Pools []string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			var rs []string
			for _, p := range body.Pools {
				if st, ok := v.errs[p]; ok {
					rs = append(rs, `{"poolAddress":"`+p+`","error":{"status":`+itoa(st)+`,"error":"x"}}`)
				} else if q, ok := v.quotes[p]; ok {
					rs = append(rs, `{"poolAddress":"`+p+`","quote":`+q+`}`)
				}
			}
			_, _ = w.Write([]byte(`{"results":[` + strings.Join(rs, ",") + `]}`))
		default: // /public/v1/evm/<chain>/<pool>/bid_ask
			parts := strings.Split(r.URL.Path, "/")
			p := parts[len(parts)-2]
			if st, ok := v.errs[p]; ok {
				w.WriteHeader(st)
				return
			}
			_, _ = w.Write([]byte(v.quotes[p]))
		}
	})
}

func itoa(i int) string { return strconv.Itoa(i) }

// For the same venue answers, the batch path must write exactly the pool state the per-pool path
// writes: pool-service saves what the tracker returns, so any drift changes prod quotes.
func TestGetNewPoolStatesEqualsSinglePathState(t *testing.T) {
	v := venue{
		quotes: map[string]string{"0xaa": quoteWithTs, "0xbb": quoteFeedDwn, "0xcc": quoteWithTs},
		errs:   map[string]int{"0xdd": 404},
		meta:   `{"data":[{"poolAddress":"0xAA","extensions":["0x1","0x2"]},{"poolAddress":"0xdd","extensions":["0x1"]}]}`,
	}
	srv := httptest.NewServer(v.handler())
	defer srv.Close()

	prev := `{"qA":true,"ext":9}`
	pools := []entity.Pool{
		{Address: "0xAA", Reserves: []string{"9", "9"}, StaticExtra: `{}`, Extra: prev},           // ok quote, extensions from /metadata
		{Address: "0xbb", Reserves: []string{"9", "9"}, StaticExtra: `{}`, Extra: prev},           // feed down -> unavailable, previous extensions kept
		{Address: "0xcc", Reserves: []string{"9", "9"}, StaticExtra: `{"swapWhitelisting":true}`}, // whitelisted -> unavailable, no API call
		{Address: "0xdd", Reserves: []string{"9", "9"}, StaticExtra: `{}`, Extra: prev},           // 4xx -> unavailable
	}
	single, batch := newBatchTracker(srv), newBatchTracker(srv)

	got, errs := batch.GetNewPoolStates(context.Background(), pools)
	for i, p := range pools {
		want, err := single.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NoError(t, errs[i], p.Address)
		require.Equal(t, want, got[i], p.Address)
	}
	require.Equal(t, int64(pastTs), got[0].Timestamp)
	require.True(t, quoteAvailable(t, got[0]))
	require.Contains(t, got[0].Extra, `"ext":2`, "count from /metadata")
	require.Contains(t, got[1].Extra, `"ext":9`, "pool absent from /metadata keeps its previous count")
	for _, i := range []int{1, 2, 3} {
		require.False(t, quoteAvailable(t, got[i]), pools[i].Address)
		require.Equal(t, entity.PoolReserves{"0", "0"}, got[i].Reserves)
	}
}

// A pool whose answer is temporary-failed (per-pool 5xx, missing, whole-request 5xx) must return an
// error and leave the caller's old state alone; the single path would have overwritten it.
func TestGetNewPoolStatesKeepsOldStateOnFailure(t *testing.T) {
	v := venue{quotes: map[string]string{"0xaa": quoteWithTs}, errs: map[string]int{"0xbb": 502}}
	srv := httptest.NewServer(v.handler())
	defer srv.Close()
	old := entity.Pool{Address: "0xbb", Reserves: []string{"7", "7"}, StaticExtra: `{}`, Extra: `{"qA":true}`, Timestamp: 5}

	out, errs := newBatchTracker(srv).GetNewPoolStates(context.Background(),
		[]entity.Pool{{Address: "0xaa", StaticExtra: `{}`}, old, {Address: "0xcc", StaticExtra: `{}`}})
	require.NoError(t, errs[0])
	require.Error(t, errs[1])
	require.Error(t, errs[2], "missing from response")
	require.Equal(t, entity.Pool{}, out[1], "failed slot carries no state, pool-service must not save it")

	srv5 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer srv5.Close()
	_, errs = newBatchTracker(srv5).GetNewPoolStates(context.Background(), []entity.Pool{old})
	require.Error(t, errs[0])
}

// No pool needs the API (all whitelisted): zero HTTP requests.
func TestGetNewPoolStatesAllUnswappableMakesNoRequest(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bid_ask/batch") {
			hits.Add(1)
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	_, errs := newBatchTracker(srv).GetNewPoolStates(context.Background(),
		[]entity.Pool{{Address: "0xaa", StaticExtra: `{"swapWhitelisting":true}`}})
	require.NoError(t, errs[0])
	require.Zero(t, hits.Load())
}

// Many tasks share one tracker per pod. Run under -race: the pause flag and the /metadata cache
// are the only shared state; every call must see a consistent all-ok or all-paused answer.
func TestGetNewPoolStatesConcurrentSharedTracker(t *testing.T) {
	var batchHits atomic.Int32
	v := venue{quotes: map[string]string{"0xaa": quoteWithTs, "0xbb": quoteWithTs},
		meta: `{"data":[{"poolAddress":"0xaa","extensions":["0x1"]}]}`}
	h := v.handler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bid_ask/batch") && batchHits.Add(1) == 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()
	tr := newBatchTracker(srv)

	var wg sync.WaitGroup
	var paused, ok atomic.Int32
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 5 {
				_, errs := tr.GetNewPoolStates(context.Background(), batchPools("0xaa", "0xbb"))
				if errs[0] != nil || errs[1] != nil {
					require.Error(t, errs[0])
					require.Error(t, errs[1])
					paused.Add(1)
				} else {
					ok.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	require.NotZero(t, paused.Load(), "the 429 must pause later calls on this tracker")

	// After Retry-After the pod resumes.
	time.Sleep(1100 * time.Millisecond)
	_, errs := tr.GetNewPoolStates(context.Background(), batchPools("0xaa"))
	require.NoError(t, errs[0])
}

// Prod yaml sets retryCount 1. resty must not replay a batch on an HTTP error status (it would
// double the load right after a 5xx/429): one failed batch = one request.
func TestGetNewPoolStatesDoesNotRetryOnStatusError(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/bid_ask/batch") {
				hits.Add(1)
			}
			w.WriteHeader(status)
		}))
		cfg := &axima.Config{DexID: DexType, ChainID: 8453, HTTPConfig: axima.HTTPConfig{BaseURL: srv.URL, RetryCount: 1}}
		_, errs := NewPoolTracker(cfg, nil).GetNewPoolStates(context.Background(), batchPools("0xa1"))
		require.Error(t, errs[0])
		require.EqualValues(t, 1, hits.Load(), "status %d", status)
		srv.Close()
	}
}
