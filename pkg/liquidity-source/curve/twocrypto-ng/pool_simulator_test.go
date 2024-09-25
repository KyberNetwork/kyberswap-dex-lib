package twocryptong

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0x1Fb84Fa6D252762e8367eA607A6586E09dceBe3D
		`{"address":"0x1fb84fa6d252762e8367ea607a6586e09dcebe3d","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["968569777414549410834","1045106588251996643768"],"tokens":[{"address":"0x18c14c2d707b2212e17d1579789fc06010cfca23","name":"","symbol":"ETH+","decimals":18,"weight":0,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"20000000\",\"InitialGamma\":\"20000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"20000000\",\"FutureGamma\":\"20000000000000000\",\"FutureAGammaTime\":0,\"D\":\"1996236386986675947911\",\"PriceScale\":[\"983313638977093334\"],\"PriceOracle\":[\"983239528662393033\"],\"LastPrices\":[\"983244856693732906\"],\"LastPricesTimestamp\":1726463246,\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"500000\",\"OutFee\":\"8000000\",\"LpSupply\":\"1006167834136870835627\",\"XcpProfit\":\"1000760564011364559\",\"VirtualPrice\":\"1000381175737496082\",\"AllowedExtraProfit\":\"1000000000000\",\"AdjustmentStep\":\"25000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
		// https://arbiscan.io/address/0xE34B3a4cEDB077b53cc813Df6fe34a85749fcecC
		`{"address":"0xe34b3a4cedb077b53cc813df6fe34a85749fcecc","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["4048585006552861060153","399999"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","name":"","symbol":"crvUSD","decimals":18,"weight":0,"swappable":true},{"address":"0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3","name":"","symbol":"EURS","decimals":2,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"1880000\",\"InitialGamma\":\"199000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"1880000\",\"FutureGamma\":\"199000000000000000\",\"FutureAGammaTime\":0,\"D\":\"8383386641969295080730\",\"PriceScale\":[\"1083716143157454024\"],\"PriceOracle\":[\"1083039505855158959\"],\"LastPrices\":[\"1083039505855158959\"],\"LastPricesTimestamp\":1721804004,\"FeeGamma\":\"12300000000000000\",\"MidFee\":\"4000000\",\"OutFee\":\"30000000\",\"LpSupply\":\"4026454270358976869472\",\"XcpProfit\":\"1000023302885130528\",\"VirtualPrice\":\"1000020627288204984\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
	}

	testcases := []struct {
		poolIdx    int
		in         string
		inAmount   string
		out        string
		outOrError any
	}{
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "50000000000000000", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "49158730955589571"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "500000000000000001", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "491586661400038591"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "5000000000000000012", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "4915798138812167060"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "50000000000000000123", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "49147140999173741771"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "500000000000000001234", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "453553432480766034585"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "5000000000000000012345", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "923692799321975179979"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0", "0x18c14c2d707b2212e17d1579789fc06010cfca23", ErrExchange0Coins},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "10918181", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "10734905"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "2480561515956081681", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "2438807364673706750"},
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "1017116218018521399", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "1000000000000000000"},
		{0, "0x18c14c2d707b2212e17d1579789fc06010cfca23", "1000000000000000000", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "1016967986681031868"},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "47322677960241", "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", ErrZero},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "4732267796024191790852", "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "327494"},
		{1, "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "1", "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "10822405757804743"},
		{1, "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "279108", "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "2680342795273391100498"},
		{1, "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "327494", "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "2949081412746367458511"},
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
			if e, ok := tc.outOrError.(error); ok {
				assert.Equal(t, err, e)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.outOrError.(string)), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			fmt.Println("fee", out.Fee.Amount)
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0x1Fb84Fa6D252762e8367eA607A6586E09dceBe3D
		`{"address":"0x1fb84fa6d252762e8367ea607a6586e09dcebe3d","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["968569777414549410834","1045106588251996643768"],"tokens":[{"address":"0x18c14c2d707b2212e17d1579789fc06010cfca23","name":"","symbol":"ETH+","decimals":18,"weight":0,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"20000000\",\"InitialGamma\":\"20000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"20000000\",\"FutureGamma\":\"20000000000000000\",\"FutureAGammaTime\":0,\"D\":\"1996236386986675947911\",\"PriceScale\":[\"983313638977093334\"],\"PriceOracle\":[\"983239528662393033\"],\"LastPrices\":[\"983244856693732906\"],\"LastPricesTimestamp\":1726463246,\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"500000\",\"OutFee\":\"8000000\",\"LpSupply\":\"1006167834136870835627\",\"XcpProfit\":\"1000760564011364559\",\"VirtualPrice\":\"1000381175737496082\",\"AllowedExtraProfit\":\"1000000000000\",\"AdjustmentStep\":\"25000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
		// https://arbiscan.io/address/0xE34B3a4cEDB077b53cc813Df6fe34a85749fcecC
		`{"address":"0xe34b3a4cedb077b53cc813df6fe34a85749fcecc","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["4048585006552861060153","399999"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","name":"","symbol":"crvUSD","decimals":18,"weight":0,"swappable":true},{"address":"0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3","name":"","symbol":"EURS","decimals":2,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"1880000\",\"InitialGamma\":\"199000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"1880000\",\"FutureGamma\":\"199000000000000000\",\"FutureAGammaTime\":0,\"D\":\"8383386641969295080730\",\"PriceScale\":[\"1083716143157454024\"],\"PriceOracle\":[\"1083039505855158959\"],\"LastPrices\":[\"1083039505855158959\"],\"LastPricesTimestamp\":1721804004,\"FeeGamma\":\"12300000000000000\",\"MidFee\":\"4000000\",\"OutFee\":\"30000000\",\"LpSupply\":\"4026454270358976869472\",\"XcpProfit\":\"1000023302885130528\",\"VirtualPrice\":\"1000020627288204984\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
	}

	testcases := []struct {
		poolIdx          int
		tokenIn          string
		tokenOut         string
		amountOut        *big.Int
		expectedAmountIn *big.Int
		expectedFee      *big.Int
		expectedErr      error
	}{
		// ? ETH+ -> 1 WETH
		{
			0,
			"0x18c14c2d707b2212e17d1579789fc06010cfca23",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("1000000000000000000"),
			bignumber.NewBig10("983315119241154082"),
			bignumber.NewBig10("69937390896588"),
			nil,
		},

		// ? WETH -> 1 ETH+
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0x18c14c2d707b2212e17d1579789fc06010cfca23",
			bignumber.NewBig10("1000000000000000000"),
			bignumber.NewBig10("1017116218018521399"),
			bignumber.NewBig10("72715187113983"),
			nil,
		},

		// ? crvUSD -> 1 EURS
		{
			1,
			"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5",
			"0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3",
			bignumber.NewBig10("100"),
			bignumber.NewBig10("1093750017449472686"),
			bignumber.NewBig10("0"),
			nil,
		},

		// ? EURS -> 1 crvUSD
		{
			1,
			"0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3",
			"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5",
			bignumber.NewBig10("1000000000000000000"),
			bignumber.NewBig10("92"),
			bignumber.NewBig10("628409650686775"),
			nil,
		},
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
			amountIn, err := testutil.MustConcurrentSafe[*pool.CalcAmountInResult](t, func() (any, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{
						Token:  tc.tokenOut,
						Amount: tc.amountOut,
					},
					TokenIn: tc.tokenIn,
					Limit:   nil,
				})
			})

			if err != nil {
				assert.ErrorIsf(t, err, tc.expectedErr, "expected error %v, got %v", tc.expectedErr, err)
				return
			}

			assert.Equal(t, tc.tokenIn, amountIn.TokenAmountIn.Token)
			assert.Equal(t, tc.expectedAmountIn, amountIn.TokenAmountIn.Amount)
			assert.Equalf(t, tc.expectedFee.String(), amountIn.Fee.Amount.String(),
				"expected fee %v, got %v", tc.expectedFee, amountIn.Fee.Amount)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0x1Fb84Fa6D252762e8367eA607A6586E09dceBe3D
		`{"address":"0x1fb84fa6d252762e8367ea607a6586e09dcebe3d","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["968569777414549410834","1045106588251996643768"],"tokens":[{"address":"0x18c14c2d707b2212e17d1579789fc06010cfca23","name":"","symbol":"ETH+","decimals":18,"weight":0,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"20000000\",\"InitialGamma\":\"20000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"20000000\",\"FutureGamma\":\"20000000000000000\",\"FutureAGammaTime\":0,\"D\":\"1996236386986675947911\",\"PriceScale\":[\"983313638977093334\"],\"PriceOracle\":[\"983239528662393033\"],\"LastPrices\":[\"983244856693732906\"],\"LastPricesTimestamp\":1726463246,\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"500000\",\"OutFee\":\"8000000\",\"LpSupply\":\"1006167834136870835627\",\"XcpProfit\":\"1000760564011364559\",\"VirtualPrice\":\"1000381175737496082\",\"AllowedExtraProfit\":\"1000000000000\",\"AdjustmentStep\":\"25000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
		// https://arbiscan.io/address/0xE34B3a4cEDB077b53cc813Df6fe34a85749fcecC
		`{"address":"0xe34b3a4cedb077b53cc813df6fe34a85749fcecc","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["4048585006552861060153","399999"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","name":"","symbol":"crvUSD","decimals":18,"weight":0,"swappable":true},{"address":"0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3","name":"","symbol":"EURS","decimals":2,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"1880000\",\"InitialGamma\":\"199000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"1880000\",\"FutureGamma\":\"199000000000000000\",\"FutureAGammaTime\":0,\"D\":\"8383386641969295080730\",\"PriceScale\":[\"1083716143157454024\"],\"PriceOracle\":[\"1083039505855158959\"],\"LastPrices\":[\"1083039505855158959\"],\"LastPricesTimestamp\":1721804004,\"FeeGamma\":\"12300000000000000\",\"MidFee\":\"4000000\",\"OutFee\":\"30000000\",\"LpSupply\":\"4026454270358976869472\",\"XcpProfit\":\"1000023302885130528\",\"VirtualPrice\":\"1000020627288204984\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`,
	}

	testcases := []struct {
		poolIdx    int
		in         string
		inAmount   string
		out        string
		outOrError interface{}
	}{
		{0, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "1017116218018521399", "0x18c14c2d707b2212e17d1579789fc06010cfca23", "1000000000000000000"},
		{0, "0x18c14c2d707b2212e17d1579789fc06010cfca23", "1000000000000000000", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "1016969759771777253"},
		{1, "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "4732267796024191790852", "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "327494"},
		{1, "0x5d8c5293dabc2c861d2f6dbd4bb0600889fdadf3", "327494", "0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5", "4701664153184618167557"},
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
			if expErr, ok := tc.outOrError.(error); ok {
				require.Equal(t, expErr, err)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.outOrError.(string)), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
				SwapLimit:      nil,
			})
			fmt.Println("balances", p.Reserves[0].Dec(), p.Reserves[1].Dec())
			fmt.Println("PriceOracle", p.Extra.PriceOracle[0].Dec())
			fmt.Println("PriceScale", p.Extra.PriceScale[0].Dec())
			fmt.Println("LastPrices", p.Extra.LastPrices[0].Dec())
			fmt.Println("D", p.Extra.D.Dec())
		})
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	// https://arbiscan.io/address/0x1Fb84Fa6D252762e8367eA607A6586E09dceBe3D
	benchPoolRedis := `{"address":"0x1fb84fa6d252762e8367ea607a6586e09dcebe3d","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1726463373,"reserves":["968569777414549410834","1045106588251996643768"],"tokens":[{"address":"0x18c14c2d707b2212e17d1579789fc06010cfca23","name":"","symbol":"ETH+","decimals":18,"weight":0,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"20000000\",\"InitialGamma\":\"20000000000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"20000000\",\"FutureGamma\":\"20000000000000000\",\"FutureAGammaTime\":0,\"D\":\"1996236386986675947911\",\"PriceScale\":[\"983313638977093334\"],\"PriceOracle\":[\"983239528662393033\"],\"LastPrices\":[\"983244856693732906\"],\"LastPricesTimestamp\":1726463246,\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"500000\",\"OutFee\":\"8000000\",\"LpSupply\":\"1006167834136870835627\",\"XcpProfit\":\"1000760564011364559\",\"VirtualPrice\":\"1000381175737496082\",\"AllowedExtraProfit\":\"1000000000000\",\"AdjustmentStep\":\"25000000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}"}`

	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(benchPoolRedis), &poolEntity)
	require.Nil(b, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(b, err)

	ain := bignumber.NewBig10("50000000000000000123")

	for i := 0; i < b.N; i++ {
		_, _ = p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: ain},
			TokenOut:      "0x18c14c2d707b2212e17d1579789fc06010cfca23",
			Limit:         nil,
		})
	}
}
