package stable_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lista/stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// A decoded simulator, after a swap and two msgpack round trips, quotes like the original: USDC/USDT
// (0x35c9a4dae1ff05788f24b5b32721d89340cbb636 on Ethereum).
func TestMsgpackRoundTrip(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{"address":"0x35c9a4dae1ff05788f24b5b32721d89340cbb636","exchange":"lista-stable","type":"lista-stable","reserves":["85777228845","324257429898","410020487529258290804668"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"initialA\":\"472311\",\"futureA\":\"1000000\",\"initialATime\":1780282307,\"futureATime\":1780386578,\"swapFee\":\"100000\",\"adminFee\":\"2000000000\",\"oraclePrices\":[999690000000000000,999360000000000000],\"priceDiffThreshold\":[50000000000000000,50000000000000000]}","staticExtra":"{\"lpToken\":\"0x923C67CeD114dd7341e65C0e2be47f3b0927d667\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"],\"isNativeCoins\":[false,false]}"}`), &p))
	sim, err := stable.NewPoolSimulator(p)
	require.NoError(t, err)
	usdc, usdt := sim.Info.Tokens[0], sim.Info.Tokens[1]
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(5_000e6)}, TokenOut: usdt}
	res, err := sim.CalcAmountOut(in)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee})

	sims := map[string]pool.IPoolSimulator{p.Address: sim}
	for range 2 {
		encoded, err := msgpack.EncodePoolSimulatorsMap(sims)
		require.NoError(t, err)
		sims, err = msgpack.DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
	}
	for _, q := range []pool.CalcAmountOutParams{
		{TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(100e6)}, TokenOut: usdt},
		{TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: big.NewInt(100e6)}, TokenOut: usdc},
	} {
		want, err := sim.CalcAmountOut(q)
		require.NoError(t, err)
		got, err := sims[p.Address].CalcAmountOut(q)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	require.Equal(t, sim.Info.Reserves, sims[p.Address].GetReserves())
}
