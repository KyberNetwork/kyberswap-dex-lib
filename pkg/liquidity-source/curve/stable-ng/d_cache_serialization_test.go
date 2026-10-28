package stableng_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The D cache is not serialized: a decoded simulator solves for D per quote and must quote the same.
func TestDCacheAfterMsgpackRoundTrip(t *testing.T) {
	const poolJSON = `{"address":"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287180,"reserves":["8994725349517509957774712","1568153728639","10550045569550900254909685"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"50000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\"}","blockNumber":185977087}`
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolJSON), &p))
	sim, err := stableng.NewPoolSimulator(p)
	require.NoError(t, err)
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	sims, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)

	for _, amount := range []int64{1e15, 1e17, 1e18, 9e18} {
		in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(amount)},
			TokenOut: sim.Info.Tokens[1]}
		want, err := sim.CalcAmountOut(in)
		require.NoError(t, err)
		got, err := sims[p.Address].CalcAmountOut(in)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
		require.Equal(t, want.Fee.Amount, got.Fee.Amount)
	}
}
