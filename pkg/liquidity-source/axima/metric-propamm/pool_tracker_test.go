package metricpropamm

import (
	"context"
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
