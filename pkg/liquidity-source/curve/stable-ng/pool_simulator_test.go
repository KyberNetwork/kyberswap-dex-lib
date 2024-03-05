package stableng

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
		"{\"address\":\"0xdc40d14accd5629bbfa65d057f175871628d13c7\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709285278,\"reserves\":[\"50980\",\"75958\",\"100000000000000\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\"}\",\"blockNumber\":185969597}",

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		"{\"address\":\"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709287180,\"reserves\":[\"8994725349517509957774712\",\"1568153728639\",\"10550045569550900254909685\"],\"tokens\":[{\"address\":\"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\"}\",\"blockNumber\":185977087}",
	}

	testcases := []struct {
		poolIdx           int
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 5000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 75900},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 75897},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 500000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 75897},
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 500000012, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50939},
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 50000001, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50939},
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000000, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50940},

		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 50000000000000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 49719},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 500000000000000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 497190},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 5000000000000000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 4971903},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000001, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 5026592848229394394},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 500000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 502659192342457080},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 50000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 50265919315764975},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 5026591932391690},
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
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			fmt.Println("fee", out.Fee.Amount)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
		"{\"address\":\"0xdc40d14accd5629bbfa65d057f175871628d13c7\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709285278,\"reserves\":[\"50980\",\"75958\",\"100000000000000\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\"}\",\"blockNumber\":185969597}",

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		"{\"address\":\"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709287180,\"reserves\":[\"8262422587316288724376566\",\"1219069890648\",\"9468566906908624689768063\"],\"tokens\":[{\"address\":\"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\"}\",\"blockNumber\":185977087}",
	}

	testcases := []struct {
		poolIdx          int
		in               string
		inAmount         int64
		out              string
		errorOrAmountOut interface{}
	}{
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 5000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(75900)},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 50000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(29)},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", 500000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", ErrDDoesNotConverge},

		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 50000000000000000, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(49625)},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 500000000000000001, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(496251)},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", 5000000000000000012, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", big.NewInt(4962511)},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000001, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(5035968519257369998)},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 500000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(503596734266597847)},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 50000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(50359673256255325)},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 5000, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", big.NewInt(5035967323921488)},
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
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			if expErr, ok := tc.errorOrAmountOut.(error); ok {
				require.Equal(t, expErr, err)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, tc.errorOrAmountOut, out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
				SwapLimit:      nil,
			})
		})
	}
}

func TestCalcAmountOutError(t *testing.T) {
	pools := []string{
		// zero balance: https://arbiscan.io/address/0x9097065db449a59ce30bec522e1e077292c0d8fc#readContract
		"{\"address\":\"0x9097065db449a59ce30bec522e1e077292c0d8fc\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709287720,\"reserves\":[\"0\",\"0\",\"0\"],\"tokens\":[{\"address\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xb88a5ac00917a02d82c7cd6cebd73e2852d43574\",\"symbol\":\"SWEEP\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"10000\\\",\\\"FutureA\\\":\\\"10000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1023767000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\"}\",\"blockNumber\":185979218}",
	}

	testcases := []struct {
		poolIdx  int
		in       string
		inAmount int64
		out      string
	}{
		{0, "0xaf88d065e77c8cc2239327c5edb3a432268e5831", 1000000, "0xb88a5ac00917a02d82c7cd6cebd73e2852d43574"},
		{0, "0xb88a5ac00917a02d82c7cd6cebd73e2852d43574", 1000000, "0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
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
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			if out != nil && out.TokenAmountOut != nil {
				fmt.Println(out.TokenAmountOut.Amount)
			}
			require.NotNil(t, err)
		})
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"rateMultipliers\": [\"%v\",\"%v\"]}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000,
			"1000000000000000000", "1000000000000000000",
		),
		StaticExtra: "{\"lpToken\": \"LP\", \"aPrecision\": \"100\", \"OffpegFeeMultiplier\":\"20000000000\"}",
	})
	require.Nil(b, err)
	ain := big.NewInt(5000)

	for i := 0; i < b.N; i++ {
		_, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: ain},
			TokenOut:      "B",
			Limit:         nil,
		})
		require.Nil(b, err)
	}
}
