package tricryptong

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// pool 0xf5f5b97624542d72a9e06f04804bf81baa15e2b4 at block 25192354 (timestamp 1779953387)
// A/gamma pinned to block-time interpolated values (A=756886) with FutureAGammaTime=0 (ramp done).
// D set to 13533423182336480804388507, the value newton_D(A,gamma,xp) produces during the ramp —
// this is the D our fix computes and what the on-chain views contract uses.
const poolF5JSON = `{
  "address":"0xf5f5b97624542d72a9e06f04804bf81baa15e2b4",
  "exchange":"curve-tricrypto-ng",
  "type":"curve-tricrypto-ng",
  "timestamp":1779953387,
  "reserves":["4332903906625","6010377250","2228069361366269473640"],
  "tokens":[
    {"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true},
    {"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true},
    {"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}
  ],
  "extra":"{\"InitialA\":\"756886\",\"InitialGamma\":\"175700099522697\",\"InitialAGammaTime\":0,\"FutureA\":\"756886\",\"FutureGamma\":\"175700099522697\",\"FutureAGammaTime\":0,\"D\":\"13533423182336480804388507\",\"PriceScale\":[\"75855801314196021500642\",\"2084347396023404453046\"],\"PriceOracle\":[\"73165887880271882853412\",\"1984580596191842865460\"],\"LastPrices\":[\"73166212202487361855289\",\"1984591878410942048545\"],\"FeeGamma\":\"400000000000000\",\"MidFee\":\"1000000\",\"OutFee\":\"140000000\",\"LpSupply\":\"7967512069557260391649\",\"XcpProfit\":\"1094111136024631831\",\"VirtualPrice\":\"1047072616644998080\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000\"}",
  "staticExtra":"{\"IsNativeCoins\":[false,false,false]}",
  "blockNumber":25192354
}`

func TestPoolF5f5_CalcAmountOut(t *testing.T) {
	t.Parallel()

	var poolEntity entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolF5JSON), &poolEntity))
	p, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// on-chain get_dy results at block 25192354
	testcases := []struct {
		in      string
		inAmt   string
		out     string
		onChain string
	}{
		// USDT->WBTC
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "1000000000", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "1351698"},
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "10000000000", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "13494837"},
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "100000000000", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "133055837"},
		// USDT->WETH
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "1000000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "498291832824029092"},
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "10000000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "4973357493860972412"},
		// WBTC->USDT
		{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "10000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", "7227080873"},
		{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "100000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", "71159485309"},
		// WBTC->WETH
		{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "10000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "3642938363107693855"},
		// WETH->USDT
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1000000000000000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", "1962645351"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "10000000000000000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", "19530569920"},
		// WETH->WBTC
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1000000000000000000", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "2682639"},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmt)},
					TokenOut:      tc.out,
				})
			})
			require.NoError(t, err)
			expected := bignumber.NewBig10(tc.onChain)
			got := out.TokenAmountOut.Amount
			t.Logf("expected=%s got=%s", expected, got)
			assert.Equal(t, expected, got)
		})
	}
}

func TestPoolF5f5_CalcAmountIn(t *testing.T) {
	t.Parallel()

	var poolEntity entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolF5JSON), &poolEntity))
	p, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// on-chain get_dx results at block 25192354
	testcases := []struct {
		tokenIn  string
		tokenOut string
		amtOut   string
		onChain  string
	}{
		// ?USDT -> 1000 WBTC units
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "1000000", "739774867"},
		// ?USDT -> 0.1 WBTC
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "10000000", "7406777006"},
		// ?USDT -> 1 WBTC
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "100000000", "74884850156"},
		// ?WBTC -> 1 WETH
		{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1000000000000000000", "2742767"},
		// ?WETH -> 1000 USDT
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xdac17f958d2ee523a2206206994597c13d831ec7", "1000000000", "509378498276154563"},
		// ?WETH -> 10000 USDT
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xdac17f958d2ee523a2206206994597c13d831ec7", "10000000000", "5106643144449848617"},
		// ?USDT -> 1 WETH
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1000000000000000000", "2007291490"},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: bignumber.NewBig10(tc.amtOut)},
					TokenIn:        tc.tokenIn,
				})
			})
			require.NoError(t, err)
			expected := bignumber.NewBig10(tc.onChain)
			got := res.TokenAmountIn.Amount
			t.Logf("expected=%s got=%s", expected, got)
			assert.Equal(t, expected, got)
		})
	}
}

func TestPoolF5f5_TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	var poolEntity entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolF5JSON), &poolEntity))
	p, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)
	testutil.TestCalcAmountIn(t, p)
}
