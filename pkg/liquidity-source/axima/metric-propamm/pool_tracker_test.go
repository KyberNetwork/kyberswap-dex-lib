package metricpropamm

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
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

			cfg := &axima.Config{DexID: DexType, ChainID: 8453, HTTPConfig: axima.HTTPConfig{BaseURL: srv.URL}}
			p, err := NewPoolTracker(cfg, nil).GetNewPoolState(context.Background(),
				entity.Pool{Address: "0xab", Reserves: []string{"9", "9"}, StaticExtra: `{}`}, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)

			var extra axima.Extra
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

// The incident this guards: Metric's off-chain bid_ask answered with a usable quote
// and real depth while the on-chain feed its quoter extension reads had gone stale,
// so every routed swap reverted FeedStalled() inside the extension. Staleness has to
// take the pool out of routing. The fail-open cases matter just as much: a feed read
// that tells us nothing (absent, zero, future-dated, unconfigured) must leave a
// healthy pool routable, because treating "unknown" as stale would drop every
// Metric pool at once.
func TestFeedStalled(t *testing.T) {
	const now, maxAge = 1_000_000, 60
	for name, tc := range map[string]struct {
		updatedAt *big.Int
		maxAge    int64
		stalled   bool
	}{
		"fresh feed stays routable":     {big.NewInt(now - 10), maxAge, false},
		"exactly at the limit is fresh": {big.NewInt(now - maxAge), maxAge, false},
		"past the limit is stale":       {big.NewInt(now - maxAge - 1), maxAge, true},
		"long-dead feed is stale":       {big.NewInt(now - 86_400), maxAge, true},
		"check disabled by zero maxAge": {big.NewInt(now - 86_400), 0, false},
		"absent reading is not stale":   {nil, maxAge, false},
		"zero reading is not stale":     {big.NewInt(0), maxAge, false},
		"future reading is not stale":   {big.NewInt(now + 60), maxAge, false},
		"unrepresentable is not stale":  {new(big.Int).Lsh(big.NewInt(1), 200), maxAge, false},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.stalled, feedStalled(tc.updatedAt, tc.maxAge, now))
		})
	}
}
