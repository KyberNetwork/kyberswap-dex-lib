package metricpropamm

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Captured from origin/main (before the axima package was folded into this one); never regenerate.
// Stored pools in redis hold exactly this JSON, so it must decode and re-encode byte-identically.
const (
	goldenExtra  = `{"initBid":18446744073709551616,"initAsk":18500000000000000000,"qA":true,"maxAge":30,"isV2":true,"ext":2,"asks":[{"bi":0,"p":18500000000000000001,"cv":1000000},{"bi":1,"p":18600000000000000000,"cv":5000000}],"bids":[{"bi":0,"p":18400000000000000000,"cv":2000000}]}`
	goldenStatic = `{"pair":"WETH-USDC","priceProvider":"0xabc","swapWhitelisting":true}`
)

func TestStoredJSONRoundTrip(t *testing.T) {
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(goldenExtra), &e))
	out, err := json.Marshal(e)
	require.NoError(t, err)
	require.Equal(t, goldenExtra, string(out))

	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(goldenStatic), &se))
	out, err = json.Marshal(se)
	require.NoError(t, err)
	require.Equal(t, goldenStatic, string(out))

	// the registered simulator still builds from the stored JSON and quotes
	weth, usdc := "0xa", "0xb"
	sim, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: entity.Pool{
		Address: "0x1", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"1000000000", "1000000000"},
		Extra: goldenExtra, StaticExtra: goldenStatic, Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{{Address: weth, Decimals: 6, Swappable: true}, {Address: usdc, Decimals: 6, Swappable: true}},
	}})
	require.NoError(t, err)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: bignumber.NewBig("1000")}, TokenOut: usdc})
	require.NoError(t, err)
	require.True(t, res.TokenAmountOut.Amount.Sign() > 0)
}
