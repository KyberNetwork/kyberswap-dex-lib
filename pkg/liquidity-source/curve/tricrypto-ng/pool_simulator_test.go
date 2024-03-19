package tricryptong

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	pools := []string{
		// https://etherscan.io/address/0x2889302a794da87fbf1d6db415c1492194663d13#events
		"{\"address\":\"0x2889302a794da87fbf1d6db415c1492194663d13\",\"exchange\":\"curve-tricrypto-ng\",\"type\":\"curve-tricrypto-ng\",\"timestamp\":1710842900,\"reserves\":[\"3848079508071253519125552\",\"60997386412794855327\",\"1028200997183081004168\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x18084fba666a33d37592fa2633fd49a74dd93a88\",\"symbol\":\"tBTC\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1707629\\\",\\\"InitialGamma\\\":\\\"11809167828997\\\",\\\"InitialAGammaTime\\\":1705051559,\\\"FutureA\\\":\\\"540000\\\",\\\"FutureGamma\\\":\\\"80500000000000\\\",\\\"FutureAGammaTime\\\":1705537322,\\\"D\\\":\\\"11990883592127090140834712\\\",\\\"PriceScale\\\":[\\\"66313464177401058702341\\\",\\\"3988288337309167729564\\\"],\\\"PriceOracle\\\":[\\\"63612706012126486095056\\\",\\\"3782761569503404058823\\\"],\\\"LastPrices\\\":[\\\"63608488224235038716789\\\",\\\"3782322291001686876800\\\"],\\\"LastPricesTimestamp\\\":1710838775,\\\"FeeGamma\\\":\\\"400000000000000\\\",\\\"MidFee\\\":\\\"1000000\\\",\\\"OutFee\\\":\\\"140000000\\\",\\\"LpSupply\\\":\\\"6209561906175920711602\\\",\\\"XcpProfit\\\":\\\"1005532234158713186\\\",\\\"VirtualPrice\\\":\\\"1002781276086899355\\\",\\\"AllowedExtraProfit\\\":\\\"100000000\\\",\\\"AdjustmentStep\\\":\\\"100000000000\\\",\\\"MaTime\\\":\\\"601\\\"}\",\"staticExtra\":\"{\\\"IsNativeCoins\\\":[false,false,false]}\",\"blockNumber\":19468099}",
	}

	testcases := []struct {
		poolIdx           int
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "777940997580"},
		// {0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "5000000000000000001", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "77302091201244"},
		// {0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000012", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "775767461853097"},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			fmt.Println("fee", out.Fee.Amount)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	// pools := []string{
	// 	// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
	// 	"{\"address\":\"0xdc40d14accd5629bbfa65d057f175871628d13c7\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709285278,\"reserves\":[\"50980\",\"75958\",\"100000000000000\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\"}\",\"blockNumber\":185969597}",

	// 	// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
	// 	"{\"address\":\"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709287180,\"reserves\":[\"8262422587316288724376566\",\"1219069890648\",\"9468566906908624689768063\"],\"tokens\":[{\"address\":\"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\"}\",\"blockNumber\":185977087}",
	// }

	// testcases := []struct {
	// 	poolIdx          int
	// 	in               string
	// 	inAmount         int64
	// 	out              string
	// 	errorOrAmountOut interface{}
	// }{
	// 	{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 5000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(75900)},
	// 	{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(29)},
	// 	{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 500000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", ErrDDoesNotConverge},

	// 	{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 50000000000000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(49625)},
	// 	{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 500000000000000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(496251)},
	// 	{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 5000000000000000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(4962511)},
	// 	{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000001, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(5035968519257369998)},
	// 	{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 500000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(503596734266597847)},
	// 	{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 50000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(50359673256255325)},
	// 	{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(5035967323921488)},
	// }

	// sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
	// 	var poolEntity entity.Pool
	// 	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	// 	require.Nil(t, err)
	// 	p, err := NewPoolSimulator(poolEntity)
	// 	require.Nil(t, err)
	// 	return p
	// })

	// for idx, tc := range testcases {
	// 	t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
	// 		p := sims[tc.poolIdx]
	// 		out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
	// 			return p.CalcAmountOut(pool.CalcAmountOutParams{
	// 				TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
	// 				TokenOut:      tc.out,
	// 				Limit:         nil,
	// 			})
	// 		})
	// 		if expErr, ok := tc.errorOrAmountOut.(error); ok {
	// 			require.Equal(t, expErr, err)
	// 			return
	// 		}

	// 		require.Nil(t, err)
	// 		assert.Equal(t, tc.errorOrAmountOut, out.TokenAmountOut.Amount)
	// 		assert.Equal(t, tc.out, out.TokenAmountOut.Token)

	// 		p.UpdateBalance(pool.UpdateBalanceParams{
	// 			TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
	// 			TokenAmountOut: *out.TokenAmountOut,
	// 			Fee:            *out.Fee,
	// 			SwapInfo:       out.SwapInfo,
	// 			SwapLimit:      nil,
	// 		})
	// 	})
	// }
}

func BenchmarkCalcAmountOut(b *testing.B) {
	// p, err := NewPoolSimulator(entity.Pool{
	// 	Exchange: "",
	// 	Type:     "",
	// 	Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
	// 	Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
	// 	Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"rateMultipliers\": [\"%v\",\"%v\"]}",
	// 		"3000000",    // 0.0003
	// 		"5000000000", // 0.5
	// 		150000, 150000,
	// 		"1000000000000000000", "1000000000000000000",
	// 	),
	// 	StaticExtra: "{\"lpToken\": \"LP\", \"aPrecision\": \"100\", \"OffpegFeeMultiplier\":\"20000000000\"}",
	// })
	// require.Nil(b, err)
	// ain := big.NewInt(5000)

	// for i := 0; i < b.N; i++ {
	// 	_, err := p.CalcAmountOut(pool.CalcAmountOutParams{
	// 		TokenAmountIn: pool.TokenAmount{Token: "A", Amount: ain},
	// 		TokenOut:      "B",
	// 		Limit:         nil,
	// 	})
	// 	require.Nil(b, err)
	// }
}
