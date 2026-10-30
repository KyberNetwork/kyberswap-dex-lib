package metricpropamm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// A pool whose price feed is down must never be priced: Metric answers 200 with empty depth,
// and the tracker has to publish zero reserves and qA=false instead of the stale/empty quote.
func TestTrackerMarksFeedDownPoolUnquotable(t *testing.T) {
	const level = `{"binIdx":0,"price":"1","cumulativeVolume":"1","priceImpactE6":"0"}`
	for name, tc := range map[string]struct {
		body     string
		quotable bool
	}{
		"feed down":   {`{"bidAdj":"0","askAdj":"1","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[],"bids":[]},"priceProviderStatus":"feed_down"}`, false},
		"empty depth": {`{"bidAdj":"0","askAdj":"1","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[],"bids":[]}}`, false},
		"healthy":     {`{"bidAdj":"1","askAdj":"2","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[` + level + `],"bids":[]},"priceProviderStatus":"ok"}`, true},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			cfg := &Config{DexID: DexType, ChainID: 8453, HTTPConfig: HTTPConfig{BaseURL: srv.URL}}
			p, err := NewPoolTracker(cfg, nil).GetNewPoolState(context.Background(),
				entity.Pool{Address: "0xab", Reserves: []string{"9", "9"}, StaticExtra: `{}`}, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)

			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
			require.Equal(t, tc.quotable, extra.QuoteAvailable)
			if tc.quotable {
				require.Equal(t, entity.PoolReserves{"5", "6"}, p.Reserves)
			} else {
				require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
			}
		})
	}
}

// Each extension adds swap gas, so the tracker must publish the pool's extension count from /metadata
// (matched case-insensitively), and keep the last count when /metadata fails.
func TestTrackerPublishesExtensionCount(t *testing.T) {
	const level = `{"binIdx":0,"price":"1","cumulativeVolume":"1","priceImpactE6":"0"}`
	const bidAsk = `{"bidAdj":"1","askAdj":"2","totalToken0Available":"5","totalToken1Available":"6","depth":{"asks":[` + level + `],"bids":[]},"priceProviderStatus":"ok"}`
	metadataOK := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/metadata") && !metadataOK:
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasSuffix(r.URL.Path, "/metadata"):
			_, _ = w.Write([]byte(`{"data":[{"poolAddress":"0xABcd","extensions":["0x1","0x2","0x3"]},{"poolAddress":"0xother","extensions":["0x1"]}]}`))
		default:
			_, _ = w.Write([]byte(bidAsk))
		}
	}))
	defer srv.Close()

	cfg := &Config{DexID: DexType, ChainID: 8453, HTTPConfig: HTTPConfig{BaseURL: srv.URL}}
	tracker := NewPoolTracker(cfg, nil)
	extensions := func(p entity.Pool) int {
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		return extra.Extensions
	}

	p, err := tracker.GetNewPoolState(context.Background(),
		entity.Pool{Address: "0xabCD", StaticExtra: `{}`}, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, 3, extensions(p))

	// Cache expired and /metadata down: the previous count survives.
	metadataOK = false
	tracker.extExpires = time.Time{}
	p, err = tracker.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, 3, extensions(p))
}

func TestQuoteTimestamp(t *testing.T) {
	const now = int64(1_700_000_100)
	for name, tc := range map[string]struct{ serverTs, want int64 }{
		"server timestamp is used":                            {now - 7, now - 7},
		"server timestamp equal to now":                       {now, now},
		"missing server timestamp falls back to now":          {0, now},
		"negative server timestamp falls back to now":         {-1, now},
		"server clock ahead is capped at now":                 {now + 90, now},
		"stale server timestamp is kept so the pool ages out": {now - 3600, now - 3600},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, quoteTimestamp(tc.serverTs, now))
		})
	}
}
