package metricpropamm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
)

// Metric serves key-gated chains only to authenticated callers, so pool discovery
// must send the API key too, not just the bid_ask tracker.
func TestGetNewPoolsSendsAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, `{"error":"Unknown chainId: 143"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"poolAddress":"0xAB","token0":"0x01","token1":"0x02"}],"nextOffset":null}`))
	}))
	defer srv.Close()

	cfg := &axima.Config{DexID: DexType, ChainID: 143, HTTPConfig: axima.HTTPConfig{BaseURL: srv.URL, APIKey: "k"}}
	pools, _, err := NewPoolsListUpdater(cfg).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, "0xab", pools[0].Address)
}

// The tracker can only check the on-chain feed if discovery persists the provider
// address, so a pool published without it silently loses the staleness guard. Pools
// listed before this field existed carry no address, and must still be published —
// they simply fall back to the off-chain signals.
func TestGetNewPoolsPersistsPriceProvider(t *testing.T) {
	for name, tc := range map[string]struct {
		pair string
		want string
	}{
		"provider persisted lowercased": {`{"poolAddress":"0xAB","token0":"0x01","token1":"0x02","priceProviderAddress":"0xCDEF"}`, "0xcdef"},
		"absent provider still listed":  {`{"poolAddress":"0xAB","token0":"0x01","token1":"0x02"}`, ""},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[` + tc.pair + `],"nextOffset":null}`))
			}))
			defer srv.Close()

			cfg := &axima.Config{DexID: DexType, ChainID: 1, HTTPConfig: axima.HTTPConfig{BaseURL: srv.URL}}
			pools, _, err := NewPoolsListUpdater(cfg).GetNewPools(context.Background(), nil)
			require.NoError(t, err)
			require.Len(t, pools, 1)

			var se axima.StaticExtra
			require.NoError(t, json.Unmarshal([]byte(pools[0].StaticExtra), &se))
			require.Equal(t, tc.want, se.PriceProviderAddress)
		})
	}
}
