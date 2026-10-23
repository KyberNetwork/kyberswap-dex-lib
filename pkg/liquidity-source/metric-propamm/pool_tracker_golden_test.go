package metricpropamm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Golden from origin/main (the batch change moved the tail of getNewPoolState into buildPool):
// the single-path output for an ok quote and for a feed_down quote must stay byte-identical.
func TestGetNewPoolStateGoldenUnchangedBySplit(t *testing.T) {
	const ok = `{"bidAdj":"11","askAdj":"13","totalToken0Available":"5","totalToken1Available":"6","serverTs":1700000000,"depth":{"asks":[{"binIdx":0,"price":"1","cumulativeVolume":"1","priceImpactE6":"0"}],"bids":[]},"priceProviderStatus":"ok"}`
	const down = `{"bidAdj":"1","askAdj":"2","depth":{"asks":[],"bids":[]},"priceProviderStatus":"feed_down"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/metadata"):
			_, _ = w.Write([]byte(`{"data":[{"poolAddress":"0xAA","extensions":["0x1","0x2"]}]}`))
		case strings.Contains(r.URL.Path, "/0xaa/"):
			_, _ = w.Write([]byte(ok))
		default:
			_, _ = w.Write([]byte(down))
		}
	}))
	defer srv.Close()
	cfg := &Config{DexID: DexType, ChainID: 8453, MaxAge: 30, HTTPConfig: HTTPConfig{BaseURL: srv.URL}}
	tr := NewPoolTracker(cfg, nil)
	get := func(addr string) entity.Pool {
		p, err := tr.GetNewPoolState(context.Background(),
			entity.Pool{Address: addr, StaticExtra: `{}`, Extra: `{"ext":9}`}, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		return p
	}

	p := get("0xAA")
	require.Equal(t, `{"initBid":11,"initAsk":13,"qA":true,"maxAge":30,"isV2":true,"ext":2,"asks":[{"bi":0,"p":1,"cv":1}],"bids":[]}`, p.Extra)
	require.Equal(t, entity.PoolReserves{"5", "6"}, p.Reserves)
	require.EqualValues(t, 1700000000, p.Timestamp)

	p = get("0xbb")
	require.Equal(t, `{"initBid":null,"initAsk":null,"qA":false,"maxAge":30,"isV2":true,"ext":9,"asks":null,"bids":null}`, p.Extra)
	require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	require.Zero(t, p.Timestamp)
}
