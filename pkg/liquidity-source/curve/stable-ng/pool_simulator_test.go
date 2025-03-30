package stableng

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
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
		`{"address":"0xdc40d14accd5629bbfa65d057f175871628d13c7","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709285278,"reserves":["50980","75958","100000000000000"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"20000000000\"}","blockNumber":185969597}`,

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		`{"address":"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287180,"reserves":["8994725349517509957774712","1568153728639","10550045569550900254909685"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"50000000000\"}","blockNumber":185977087}`,
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
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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

// TestCalcAmountIn how to test it? Go to the pool contract and use the `get_dx` function to get the expected amount in
// For example, https://etherscan.io/address/0x02950460E2b9529D0E00284A5fA2d7bDF3fA4d72#readContract
func TestCalcAmountIn(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
		`{"address":"0xdc40d14accd5629bbfa65d057f175871628d13c7","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709285278,"reserves":["66996","59934","100000000000000"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"20000000000\"}","blockNumber":207324939}`,

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		`{"address":"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287180,"reserves":["4275318662128184254659562","1213843323736","5478394789384650120577777"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"50000000000\"}","blockNumber":207324939}`,

		// https://etherscan.io/address/0x02950460E2b9529D0E00284A5fA2d7bDF3fA4d72#readContract
		`{"address":"0x02950460e2b9529d0e00284a5fa2d7bdf3fa4d72","reserveUsd":34412208.418654285,"amplifiedTvl":34412208.418654285,"exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1714709273,"reserves":["19563159037462843534173493","14855154831138","34253799909454399144096840"],"tokens":[{"address":"0x4c9edd5852cd905f086c759e8383e09bff1e68b3","symbol":"USDe","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"50000000000\",\"IsNativeCoins\":[false,false]}","blockNumber":19787999}`,
	}

	amountOutTest1, _ := new(big.Int).SetString("100000000", 10)
	expectedAmountInTest1, _ := new(big.Int).SetString("100151570619671643706", 10)

	amountOutTest2, _ := new(big.Int).SetString("1000000001", 10)
	expectedAmountInTest2, _ := new(big.Int).SetString("1001515998388117040232", 10)

	amountOutTest3, _ := new(big.Int).SetString("10000000012", 10)
	expectedAmountInTest3, _ := new(big.Int).SetString("10015189113903312626697", 10)

	amountOutTest4, _ := new(big.Int).SetString("100000000000000000000", 10)
	expectedAmountInTest4, _ := new(big.Int).SetString("99868941", 10)

	amountOutTest5, _ := new(big.Int).SetString("1000000000000000000001", 10)
	expectedAmountInTest5, _ := new(big.Int).SetString("998689704", 10)

	amountOutTest6, _ := new(big.Int).SetString("10000000000000000000012", 10)
	expectedAmountInTest6, _ := new(big.Int).SetString("9986926034", 10)

	amountOutTest7, _ := new(big.Int).SetString("59888", 10)
	expectedAmountInTest7, _ := new(big.Int).SetString("245510", 10)

	amountOutTest8, _ := new(big.Int).SetString("59900", 10)
	expectedAmountInTest8, _ := new(big.Int).SetString("359489", 10)

	amountOutTest9, _ := new(big.Int).SetString("59934", 10)

	amountOutTest10, _ := new(big.Int).SetString("59934", 10)
	expectedAmountInTest10, _ := new(big.Int).SetString("61139", 10)

	amountOutTest11, _ := new(big.Int).SetString("66900", 10)
	expectedAmountInTest11, _ := new(big.Int).SetString("153257", 10)

	amountOutTest12, _ := new(big.Int).SetString("66996", 10)

	amountOutTest13, _ := new(big.Int).SetString("100000000", 10)
	expectedAmountInTest13, _ := new(big.Int).SetString("100248005680952903976", 10)

	amountOutTest14, _ := new(big.Int).SetString("1000000001", 10)
	expectedAmountInTest14, _ := new(big.Int).SetString("1002481994891621945984", 10)

	amountOutTest15, _ := new(big.Int).SetString("10000000012", 10)
	expectedAmountInTest15, _ := new(big.Int).SetString("10025015205490539024214", 10)

	amountOutTest16, _ := new(big.Int).SetString("100000000000000000000", 10)
	expectedAmountInTest16, _ := new(big.Int).SetString("99779215", 10)

	amountOutTest17, _ := new(big.Int).SetString("1000000000000000000001", 10)
	expectedAmountInTest17, _ := new(big.Int).SetString("997794079", 10)

	amountOutTest18, _ := new(big.Int).SetString("10000000000000000000012", 10)
	expectedAmountInTest18, _ := new(big.Int).SetString("9978131310", 10)

	testcases := []struct {
		poolIdx          int
		tokenIn          string
		expectedAmountIn *big.Int
		tokenOut         string
		amountOut        *big.Int
		expectedError    error
	}{
		// Test 1 -> 6: Pool USDe + USDC
		// USDe -> USDC
		{2, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", expectedAmountInTest1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", amountOutTest1, nil},
		{2, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", expectedAmountInTest2, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", amountOutTest2, nil},
		{2, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", expectedAmountInTest3, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", amountOutTest3, nil},

		// USDC -> USDe
		{2, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", expectedAmountInTest4, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", amountOutTest4, nil},
		{2, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", expectedAmountInTest5, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", amountOutTest5, nil},
		{2, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", expectedAmountInTest6, "0x4c9edd5852cd905f086c759e8383e09bff1e68b3", amountOutTest6, nil},

		// Test 7 -> 12: Pool USDT + USDC.e
		// USDT -> USDC.e
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", expectedAmountInTest7, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest7, nil},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", expectedAmountInTest8, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest8, nil},
		{0, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", nil, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest9, ErrExecutionReverted},

		// USDC.e -> USDT
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", expectedAmountInTest10, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", amountOutTest10, nil},
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", expectedAmountInTest11, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", amountOutTest11, nil},
		{0, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", nil, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", amountOutTest12, ErrExecutionReverted},

		// Test 13 -> 18: Pool crvUSD + USDC.e
		// crvUSD -> USDC.e
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", expectedAmountInTest13, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest13, nil},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", expectedAmountInTest14, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest14, nil},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", expectedAmountInTest15, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", amountOutTest15, nil},

		// USDC.e -> crvUSD
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", expectedAmountInTest16, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", amountOutTest16, nil},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", expectedAmountInTest17, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", amountOutTest17, nil},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", expectedAmountInTest18, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", amountOutTest18, nil},
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
			amountIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: tc.amountOut},
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			})

			if err != nil {
				assert.ErrorIsf(t, err, tc.expectedError, "expected error %v, got %v", tc.expectedError, err)
				return
			}

			assert.Equal(t, tc.tokenIn, amountIn.TokenAmountIn.Token)
			assert.Equalf(t, tc.expectedAmountIn, amountIn.TokenAmountIn.Amount, "expected amount in %s, got %s", tc.expectedAmountIn.String(), amountIn.TokenAmountIn.Amount.String())
			fmt.Println("fee", amountIn.Fee.Amount)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
		`{"address":"0xdc40d14accd5629bbfa65d057f175871628d13c7","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709285278,"reserves":["50980","75958","100000000000000"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"20000000000\"}","blockNumber":185969597}`,

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		`{"address":"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287180,"reserves":["8262422587316288724376566","1219069890648","9468566906908624689768063"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"50000000000\"}","blockNumber":185977087}`,
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
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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
		`{"address":"0x9097065db449a59ce30bec522e1e077292c0d8fc","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287720,"reserves":["0","0","0"],"tokens":[{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xb88a5ac00917a02d82c7cd6cebd73e2852d43574","symbol":"SWEEP","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"10000\",\"FutureA\":\"10000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1023767000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"20000000000\"}","blockNumber":185979218}`,
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
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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
		Extra: fmt.Sprintf(`{"swapFee": "%v", "adminFee": "%v", "initialA": "%v", "futureA": "%v", "rateMultipliers": ["%v","%v"]}`,
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000,
			"1000000000000000000", "1000000000000000000",
		),
		StaticExtra: `{"lpToken": "LP", "aPrecision": "100", "OffpegFeeMultiplier":"20000000000"}`,
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
