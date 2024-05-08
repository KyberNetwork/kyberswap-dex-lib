package iziswap

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0xee45cffbfafe97691b8ef068c8d55163086a3431
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "2000000000000000000", "B", "18037620383221447462"},
		{"B", "2000000000000000000", "A", "110876282573252914"},
	}

	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xee45cffbfafe97691b8ef068c8d55163086a3431",
		Exchange: "iziswap",
		Type:     "iziswap",
		SwapFee:  400,
		Reserves: entity.PoolReserves{"1167087113545385273", "18037620383221447465"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
		Extra:    "{\"CurrentPoint\":28912,\"PointDelta\":8,\"LeftMostPt\":-800000,\"RightMostPt\":800000,\"Fee\":400,\"Liquidity\":23123688144702854,\"LiquidityX\":8210612878032008,\"Liquidities\":[{\"LiqudityDelta\":23123688144702854,\"Point\":28728},{\"LiqudityDelta\":-23123688144702854,\"Point\":29128}],\"LimitOrders\":[]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: amountIn,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	pools := []string{
		// Pool data at block https://lineascan.build/block/4315530

		// To get expected value, use Quoter contract to simulate the swap with Tenderly: https://lineascan.build/address/0xe6805638db944eA605e774e72c6F0D15Fb6a1347#writeContract
		// Example: https://dashboard.tenderly.co/tenderly_kyber/h8h/simulator/8004cae0-e467-43e7-9093-07b514a00490

		// https://lineascan.build/address/0x0d0ff66b77cfb8ff045ae22332c6a8497d774af4#readContract
		"{\"address\":\"0x0d0ff66b77cfb8ff045ae22332c6a8497d774af4\",\"reserveUsd\":712.3755125551611,\"amplifiedTvl\":4.0791184520273364e+45,\"swapFee\":10000,\"exchange\":\"iziswap\",\"type\":\"iziswap\",\"timestamp\":1714990434,\"reserves\":[\"505648343\",\"55398256814263496\"],\"tokens\":[{\"address\":\"0xa219439258ca9da29e9cc4ce5596924745e12b93\",\"name\":\"USDT\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f\",\"name\":\"wstETH\",\"symbol\":\"wstETH\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"CurrentPoint\\\":194210,\\\"PointDelta\\\":200,\\\"LeftMostPt\\\":-800000,\\\"RightMostPt\\\":800000,\\\"Fee\\\":10000,\\\"Liquidity\\\":837104264,\\\"LiquidityX\\\":470358777,\\\"Liquidities\\\":[{\\\"LiqudityDelta\\\":153320917,\\\"Point\\\":195400}],\\\"LimitOrders\\\":[]}\"}",

		// https://lineascan.build/address/0xe14f01667e2c2955b41c50a2ac39680a66f2bdeb USDT-USDC.e
		"{\"address\":\"0xe14f01667e2c2955b41c50a2ac39680a66f2bdeb\",\"reserveUsd\":907030.164492,\"amplifiedTvl\":2.5378208782795067e+39,\"swapFee\":500,\"exchange\":\"iziswap\",\"type\":\"iziswap\",\"timestamp\":1714990434,\"reserves\":[\"157437576324\",\"749592588168\"],\"tokens\":[{\"address\":\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\",\"name\":\"USDC\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xa219439258ca9da29e9cc4ce5596924745e12b93\",\"name\":\"USDT\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"CurrentPoint\\\":6,\\\"PointDelta\\\":10,\\\"LeftMostPt\\\":-800000,\\\"RightMostPt\\\":800000,\\\"Fee\\\":500,\\\"Liquidity\\\":32022195186,\\\"LiquidityX\\\":11612321398,\\\"Liquidities\\\":[{\\\"LiqudityDelta\\\":4002,\\\"Point\\\":-1840},{\\\"LiqudityDelta\\\":48374,\\\"Point\\\":-1830},{\\\"LiqudityDelta\\\":772,\\\"Point\\\":-1820},{\\\"LiqudityDelta\\\":21299,\\\"Point\\\":-970},{\\\"LiqudityDelta\\\":604361,\\\"Point\\\":-960},{\\\"LiqudityDelta\\\":34782,\\\"Point\\\":-950},{\\\"LiqudityDelta\\\":8920,\\\"Point\\\":-730},{\\\"LiqudityDelta\\\":1470,\\\"Point\\\":-720},{\\\"LiqudityDelta\\\":39496,\\\"Point\\\":-410},{\\\"LiqudityDelta\\\":278819,\\\"Point\\\":-400},{\\\"LiqudityDelta\\\":3834,\\\"Point\\\":-390},{\\\"LiqudityDelta\\\":59651,\\\"Point\\\":-270},{\\\"LiqudityDelta\\\":16096,\\\"Point\\\":-220},{\\\"LiqudityDelta\\\":286205,\\\"Point\\\":-210},{\\\"LiqudityDelta\\\":11011852,\\\"Point\\\":-200},{\\\"LiqudityDelta\\\":70032,\\\"Point\\\":-190},{\\\"LiqudityDelta\\\":4016576,\\\"Point\\\":-110},{\\\"LiqudityDelta\\\":24814,\\\"Point\\\":-100},{\\\"LiqudityDelta\\\":5033369191,\\\"Point\\\":-50},{\\\"LiqudityDelta\\\":8635935,\\\"Point\\\":-40},{\\\"LiqudityDelta\\\":21343485,\\\"Point\\\":-30},{\\\"LiqudityDelta\\\":256394541,\\\"Point\\\":-20},{\\\"LiqudityDelta\\\":26704514140,\\\"Point\\\":-10},{\\\"LiqudityDelta\\\":-19898800,\\\"Point\\\":0},{\\\"LiqudityDelta\\\":-28271110156,\\\"Point\\\":10},{\\\"LiqudityDelta\\\":-3717226535,\\\"Point\\\":20},{\\\"LiqudityDelta\\\":-16030721,\\\"Point\\\":30},{\\\"LiqudityDelta\\\":-4016576,\\\"Point\\\":100},{\\\"LiqudityDelta\\\":-24814,\\\"Point\\\":110},{\\\"LiqudityDelta\\\":-1251,\\\"Point\\\":180},{\\\"LiqudityDelta\\\":-182481,\\\"Point\\\":190},{\\\"LiqudityDelta\\\":-2326213,\\\"Point\\\":200},{\\\"LiqudityDelta\\\":-8840671,\\\"Point\\\":210},{\\\"LiqudityDelta\\\":-59651,\\\"Point\\\":240},{\\\"LiqudityDelta\\\":-33569,\\\"Point\\\":300},{\\\"LiqudityDelta\\\":-196572,\\\"Point\\\":390},{\\\"LiqudityDelta\\\":-125577,\\\"Point\\\":400},{\\\"LiqudityDelta\\\":-1470,\\\"Point\\\":690},{\\\"LiqudityDelta\\\":-29413,\\\"Point\\\":950},{\\\"LiqudityDelta\\\":-630761,\\\"Point\\\":960},{\\\"LiqudityDelta\\\":-268,\\\"Point\\\":970},{\\\"LiqudityDelta\\\":-4002,\\\"Point\\\":1820},{\\\"LiqudityDelta\\\":-48946,\\\"Point\\\":1830},{\\\"LiqudityDelta\\\":-200,\\\"Point\\\":1840}],\\\"LimitOrders\\\":[]}\"}",

		// https://lineascan.build/address/0xb808b1d37d6fcd42ca67603370c64d0139ba0e0c USDC.e-wstETH
		"{\"address\":\"0xb808b1d37d6fcd42ca67603370c64d0139ba0e0c\",\"reserveUsd\":838.2957800228867,\"amplifiedTvl\":3.607518027106924e+45,\"swapFee\":500,\"exchange\":\"iziswap\",\"type\":\"iziswap\",\"timestamp\":1715134272,\"reserves\":[\"679262009\",\"45073502131637908\"],\"tokens\":[{\"address\":\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\",\"name\":\"USDC\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f\",\"name\":\"wstETH\",\"symbol\":\"wstETH\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"CurrentPoint\\\":194637,\\\"PointDelta\\\":10,\\\"LeftMostPt\\\":-800000,\\\"RightMostPt\\\":800000,\\\"Fee\\\":500,\\\"Liquidity\\\":766449295,\\\"LiquidityX\\\":71450257,\\\"Liquidities\\\":[{\\\"LiqudityDelta\\\":1694531,\\\"Point\\\":193290},{\\\"LiqudityDelta\\\":299143528,\\\"Point\\\":196050},{\\\"LiqudityDelta\\\":146866441,\\\"Point\\\":196240}],\\\"LimitOrders\\\":[]}\"}",

		// Some more pools for testing if needed (they have good liquidity):

		// https://lineascan.build/address/0x564e52bbdf3adf10272f3f33b00d65b2ee48afff USDC.e-WETH

		// https://lineascan.build/address/0x5615a7b1619980f7d6b5e7f69f3dc093dfe0c95c USDC.e-WETH

		// https://lineascan.build/address/0x86cca1299cf2b91301e790ac6b5f2aed0591a66a WETH-wstETH

		// https://lineascan.build/address/0x3395adb05caffbb203e3b2f026b91f333cdae6e7 USDC.e-BUSD

		// https://lineascan.build/address/0x03ddd23943b3c698442c5f2841eae70058dbab8b wstETH-WETH

		// https://lineascan.build/address/0x9cfcc5322455f0951336ccb4474795924f564015 BUSD-USDC.e

		// https://lineascan.build/address/0x0d0ff66b77cfb8ff045ae22332c6a8497d774af4 USDT-wstETH

		// https://lineascan.build/address/0x10656d09115bbe6be89bf6a66d4949f833566a24 WETH-WBTC

		// https://lineascan.build/address/0xe1d9617c4dd72589733dd1d418854804d5c14437 WETH-USDC.e

		// https://lineascan.build/address/0x76fe3bf0234fc20e35a8cdc7687d87ce893b215d BUSD-WETH

		// https://lineascan.build/address/0x946a3205a9fa805c89ef8a29177d8d721c968eaa USDT-USDC.e

		// https://lineascan.build/address/0x1a9ae59aa2549e12d6ac966b4d5deee5e9102ec1 USDT-WETH

		// https://lineascan.build/address/0x836f200ed9bddb5fbb6103cfcc21aad37be12dac ezETH-WETH
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
		// ? USDT -> 0.01 wstETH (X2YDesireY)
		{
			0,
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f",
			bignumber.NewBig10("10000000000000000"),
			bignumber.NewBig10("38582137"),
			nil,
			nil,
		},

		// ? USDT -> 0.03 wstETH (X2YDesireY)
		{
			0,
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f",
			bignumber.NewBig10("30000000000000000"),
			bignumber.NewBig10("107935549"),
			nil,
			nil,
		},

		// ? wstETH -> 100 USDT (Y2XDesireX)
		{
			0,
			"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("30342728977393295"),
			nil,
			nil,
		},

		// ? wstETH -> 200 USDT (Y2XDesireX)
		{
			0,
			"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			bignumber.NewBig10("200000000"),
			bignumber.NewBig10("31546414529043889"),
			nil,
			nil,
		},

		// ? USDC.e -> 1000 USDT (X2YDesireY)
		{
			1,
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("999900161"),
			nil,
			nil,
		},

		// ? USDC.e -> 10000 USDT (X2YDesireY)
		{
			1,
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			bignumber.NewBig10("10000000000"),
			bignumber.NewBig10("9999001601"),
			nil,
			nil,
		},

		// ? USDC.e -> 100000 USDT (X2YDesireY)
		{
			1,
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			bignumber.NewBig10("100000000000"),
			bignumber.NewBig10("100004281668"),
			nil,
			nil,
		},

		// ? USDT -> 1000 USDC.e (Y2XDesireX)
		{
			1,
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("1001100703"),
			nil,
			nil,
		},

		// ? USDT -> 10000 USDC.e (Y2XDesireX)
		{
			1,
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			bignumber.NewBig10("10000000000"),
			bignumber.NewBig10("10011007006"),
			nil,
			nil,
		},

		// ? USDT -> 100000 USDC.e (Y2XDesireX)
		{
			1,
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			bignumber.NewBig10("100000000000"),
			bignumber.NewBig10("100127003921"),
			nil,
			nil,
		},

		// ? wstETH -> 500.100082 USDC.e (Y2XDesireX)
		// This case is weird, it seems that the swapY2XDesireX on smart contract is not working correctly, thus our Golang simulation is also incorrect
		// `swapY2XDesireX` Simulation: https://dashboard.tenderly.co/tenderly_kyber/h8h/simulator/31157ef8-34df-4da1-bbda-7b39a607e3e7

		// If we use the `swapY2X` function, the output of USDC.e is only 99339211 with amountIn = 31442298566542614 of wstETH
		// `swapY2X` Simulation: https://dashboard.tenderly.co/tenderly_kyber/h8h/simulator/e4298ae1-e32d-421a-af89-1c5d784a134f

		// Solution: we should implement blacklisted pools functionality to prevent this kind of pool from being used
		{
			2,
			"0xb5bedd42000b71fdde22d3ee8a79bd49a568fc8f",
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			bignumber.NewBig10("500100082"),
			bignumber.NewBig10("31442298566542614"),
			nil,
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
			assert.Equalf(t, tc.expectedFee, amountIn.Fee.Amount, "expected fee %v, got %v", tc.expectedFee, amountIn.Fee.Amount)
		})
	}
}
