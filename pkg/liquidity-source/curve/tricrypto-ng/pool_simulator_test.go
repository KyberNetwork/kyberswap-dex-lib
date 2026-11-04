package tricryptong

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	pools := []string{
		// https://etherscan.io/address/0x2889302a794da87fbf1d6db415c1492194663d13#events
		"{\"address\":\"0x2889302a794da87fbf1d6db415c1492194663d13\",\"exchange\":\"curve-tricrypto-ng\",\"type\":\"curve-tricrypto-ng\",\"timestamp\":1710842900,\"reserves\":[\"3848079508071253519125552\",\"60997386412794855327\",\"1028200997183081004168\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x18084fba666a33d37592fa2633fd49a74dd93a88\",\"symbol\":\"tBTC\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1707629\\\",\\\"InitialGamma\\\":\\\"11809167828997\\\",\\\"InitialAGammaTime\\\":1705051559,\\\"FutureA\\\":\\\"540000\\\",\\\"FutureGamma\\\":\\\"80500000000000\\\",\\\"FutureAGammaTime\\\":1705537322,\\\"D\\\":\\\"11990883592127090140834712\\\",\\\"PriceScale\\\":[\\\"66313464177401058702341\\\",\\\"3988288337309167729564\\\"],\\\"PriceOracle\\\":[\\\"63612706012126486095056\\\",\\\"3782761569503404058823\\\"],\\\"LastPrices\\\":[\\\"63608488224235038716789\\\",\\\"3782322291001686876800\\\"],\\\"LastPricesTimestamp\\\":1710838775,\\\"FeeGamma\\\":\\\"400000000000000\\\",\\\"MidFee\\\":\\\"1000000\\\",\\\"OutFee\\\":\\\"140000000\\\",\\\"LpSupply\\\":\\\"6209561906175920711602\\\",\\\"XcpProfit\\\":\\\"1005532234158713186\\\",\\\"VirtualPrice\\\":\\\"1002781276086899355\\\",\\\"AllowedExtraProfit\\\":\\\"100000000\\\",\\\"AdjustmentStep\\\":\\\"100000000000\\\",\\\"MaTime\\\":\\\"601\\\"}\",\"staticExtra\":\"{\\\"IsNativeCoins\\\":[false,false,false]}\",\"blockNumber\":19468099}",
	}

	testcases := []struct {
		poolIdx    int
		in         string
		inAmount   string
		out        string
		outOrError any
	}{
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "777940997580"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000000000001", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "7779409210946"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "5000000000000000012", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "77794015730818"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000123", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "777932520003489"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000000000001234", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "7778561960860400"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "5000000000000000012345", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "77709786016695971"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "13082875266807"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000000000001", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "130828739090584"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "5000000000000000012", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "1308286035086453"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "50000000000000000123", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "13082724777147855"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000000000001234", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "130813699671187605"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "5000000000000000012345", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "1306791631101999948"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "50000000000000000", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "2942298304726216"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "500000000000000001", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "29411687945522080"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "5000000000000000012", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "292970319746068264"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "50000000000000000123", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "2809513964503774599"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "500000000000000001234", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "19769566246680798724"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "5000000000000000012345", "0x18084fba666a33d37592fa2633fd49a74dd93a88", "49961193949748966896"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "500000000000000000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1247392878908745428005336"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "5000000000000000000001", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "3151867100859420032898396"},
		{0, "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "50000000000000000000012", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", ErrUnsafeY},
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
	t.Parallel()
	pools := []string{
		// https://etherscan.io/address/0x2889302a794da87fbf1d6db415c1492194663d13#readContract
		"{\"address\":\"0x2889302a794da87fbf1d6db415c1492194663d13\",\"reserveUsd\":9528657.094819583,\"amplifiedTvl\":9528657.094819583,\"exchange\":\"curve-tricrypto-ng\",\"type\":\"curve-tricrypto-ng\",\"timestamp\":1714975165,\"reserves\":[\"2947201605123522350748728\",\"45611346320331519581\",\"788479732384942283053\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x18084fba666a33d37592fa2633fd49a74dd93a88\",\"symbol\":\"tBTC\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1707629\\\",\\\"InitialGamma\\\":\\\"11809167828997\\\",\\\"InitialAGammaTime\\\":1705051559,\\\"FutureA\\\":\\\"540000\\\",\\\"FutureGamma\\\":\\\"80500000000000\\\",\\\"FutureAGammaTime\\\":1705537322,\\\"D\\\":\\\"8754450085519836953184450\\\",\\\"PriceScale\\\":[\\\"63936461273794516756888\\\",\\\"3666635369668832599935\\\"],\\\"PriceOracle\\\":[\\\"64075375610827630797332\\\",\\\"3681151306766592332262\\\"],\\\"LastPrices\\\":[\\\"64129534522750421957793\\\",\\\"3686896248129881507013\\\"],\\\"LastPricesTimestamp\\\":1714974575,\\\"FeeGamma\\\":\\\"400000000000000\\\",\\\"MidFee\\\":\\\"1000000\\\",\\\"OutFee\\\":\\\"140000000\\\",\\\"LpSupply\\\":\\\"4703464587192803610456\\\",\\\"XcpProfit\\\":\\\"1010482237832981057\\\",\\\"VirtualPrice\\\":\\\"1006199965234185124\\\",\\\"AllowedExtraProfit\\\":\\\"100000000\\\",\\\"AdjustmentStep\\\":\\\"100000000000\\\"}\",\"staticExtra\":\"{\\\"IsNativeCoins\\\":[false,false,false]}\",\"blockNumber\":19809115}",
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
		// ? crvUSD -> 1 tBTC
		{
			0,
			"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			"0x18084fba666a33d37592fa2633fd49a74dd93a88",
			bignumber.NewBig10("1000000000000000000"),
			bignumber.NewBig10("65933872547199101612245"),
			bignumber.NewBig10("9551219516814916"),
			nil,
		},

		// ? crvUSD -> 10 tBTC
		{
			0,
			"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			"0x18084fba666a33d37592fa2633fd49a74dd93a88",
			bignumber.NewBig10("10000000000000000000"),
			bignumber.NewBig10("835698563324567662123694"),
			bignumber.NewBig10("141090623800609154"),
			nil,
		},

		// ? tBTC -> 10 wstETH
		{
			0,
			"0x18084fba666a33d37592fa2633fd49a74dd93a88",
			"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			bignumber.NewBig10("10000000000000000000"),
			bignumber.NewBig10("583696141819846118"),
			bignumber.NewBig10("67891482142363740"),
			nil,
		},

		// ? wstETH -> 10000 crvUSD
		{
			0,
			"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			bignumber.NewBig10("10000000000000000000000"),
			bignumber.NewBig10("2721164077515907441"),
			bignumber.NewBig10("13265702572569133085"),
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
			amountIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
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

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	pools := []string{
		// https://etherscan.io/address/0x2889302a794da87fbf1d6db415c1492194663d13#events
		`{"address":"0x4ebdf703948ddcea3b11f675b4d1fba9d2414a14","amplifiedTvl":9342623.983114064,"exchange":"curve-tricrypto-ng","type":"curve-tricrypto-ng","timestamp":1747387794,"reserves":["2861820037467305203466191","1093506849144527022340","3982280374395661297312344"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","name":"","symbol":"crvUSD","decimals":18,"weight":0,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true},{"address":"0xd533a949740bb3306d119cc777fa900ba034cd52","name":"","symbol":"CRV","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"2700000\",\"InitialGamma\":\"1300000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"2700000\",\"FutureGamma\":\"1300000000000\",\"FutureAGammaTime\":0,\"D\":\"8532048735922426944154787\",\"PriceScale\":[\"2593891046098680504189\",\"711612014969964544\"],\"PriceOracle\":[\"2597753463916981281317\",\"712667936366669756\"],\"LastPrices\":[\"2612188986152217640529\",\"717156747966546519\"],\"LastPricesTimestamp\":1747387787,\"FeeGamma\":\"350000000000000\",\"MidFee\":\"2999999\",\"OutFee\":\"80000000\",\"LpSupply\":\"200731208332421373598995\",\"XcpProfit\":\"1133157971398689394\",\"VirtualPrice\":\"1155009367459460589\",\"AllowedExtraProfit\":\"100000000000\",\"AdjustmentStep\":\"100000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false,false]}"}`,
	}

	testcases := []struct {
		poolIdx    int
		in         string
		inAmount   string
		out        string
		outOrError any
	}{
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304718298868"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304675974788"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304906600525870"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304912860512857"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1819369083371"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1819713988051"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1819890952200330"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1819912010671452"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "599088140"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "83966477"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "696670096424"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "696617191992"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "61379"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "104742"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "191147306"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "191237005"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "102047"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "115007"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "136987980"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "137061149"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", ErrExchange0Coins},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", ErrExchange0Coins},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "357723621952"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "357762042492"},
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
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			if expErr, ok := tc.outOrError.(error); ok {
				assert.ErrorIs(t, err, expErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.outOrError.(string)), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			assert.NotPanics(t, func() {
				p.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
					TokenAmountOut: *out.TokenAmountOut,
					Fee:            *out.Fee,
					SwapInfo:       out.SwapInfo,
					SwapLimit:      nil,
				})
			})
			fmt.Println("balances", p.Reserves[0].Dec(), p.Reserves[1].Dec(), p.Reserves[2].Dec())
			fmt.Println("PriceOracle", p.Extra.PriceOracle[0].Dec(), p.Extra.PriceOracle[1].Dec())
			fmt.Println("PriceScale", p.Extra.PriceScale[0].Dec(), p.Extra.PriceScale[1].Dec())
			fmt.Println("LastPrices", p.Extra.LastPrices[0].Dec(), p.Extra.LastPrices[1].Dec())
			fmt.Println("D", p.Extra.D.Dec())
		})
	}
}

func TestUpdateBalanceUpdatesStateOnEverySwap(t *testing.T) {
	t.Parallel()

	poolRedis := `{"address":"0x4ebdf703948ddcea3b11f675b4d1fba9d2414a14","exchange":"curve-tricrypto-ng","type":"curve-tricrypto-ng","timestamp":1747387794,"reserves":["2861820037467305203466191","1093506849144527022340","3982280374395661297312344"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","name":"","symbol":"crvUSD","decimals":18,"weight":0,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true},{"address":"0xd533a949740bb3306d119cc777fa900ba034cd52","name":"","symbol":"CRV","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"2700000\",\"InitialGamma\":\"1300000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"2700000\",\"FutureGamma\":\"1300000000000\",\"FutureAGammaTime\":0,\"D\":\"8532048735922426944154787\",\"PriceScale\":[\"2593891046098680504189\",\"711612014969964544\"],\"PriceOracle\":[\"2597753463916981281317\",\"712667936366669756\"],\"LastPrices\":[\"2612188986152217640529\",\"717156747966546519\"],\"LastPricesTimestamp\":1747387787,\"FeeGamma\":\"350000000000000\",\"MidFee\":\"2999999\",\"OutFee\":\"80000000\",\"LpSupply\":\"200731208332421373598995\",\"XcpProfit\":\"1133157971398689394\",\"VirtualPrice\":\"1155009367459460589\",\"AllowedExtraProfit\":\"100000000000\",\"AdjustmentStep\":\"100000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false,false]}"}`

	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	weth := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	crvUsd := "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"

	swap := func(amountIn string) {
		out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: bignumber.NewBig10(amountIn)},
			TokenOut:      crvUsd,
		})
		require.NoError(t, err)
		p.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: bignumber.NewBig10(amountIn)},
			TokenAmountOut: *out.TokenAmountOut,
			Fee:            *out.Fee,
			SwapInfo:       out.SwapInfo,
		})
	}

	swap("500000000000")
	dAfterFirstSwap := p.Extra.D.Clone()

	swap("500000000000")
	dAfterSecondSwap := p.Extra.D.Clone()

	// D must be recalculated from the new balances on every swap, matching the
	// on-chain contract, which updates self.D unconditionally in tweak_price.
	assert.NotEqual(t, dAfterFirstSwap, dAfterSecondSwap,
		"D is frozen after the first swap instead of being recalculated on every UpdateBalance")
}

func BenchmarkCalcAmountOut(b *testing.B) {
	benchPoolRedis := "{\"address\":\"0x2889302a794da87fbf1d6db415c1492194663d13\",\"exchange\":\"curve-tricrypto-ng\",\"type\":\"curve-tricrypto-ng\",\"timestamp\":1710842900,\"reserves\":[\"3848079508071253519125552\",\"60997386412794855327\",\"1028200997183081004168\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x18084fba666a33d37592fa2633fd49a74dd93a88\",\"symbol\":\"tBTC\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1707629\\\",\\\"InitialGamma\\\":\\\"11809167828997\\\",\\\"InitialAGammaTime\\\":1705051559,\\\"FutureA\\\":\\\"540000\\\",\\\"FutureGamma\\\":\\\"80500000000000\\\",\\\"FutureAGammaTime\\\":1705537322,\\\"D\\\":\\\"11990883592127090140834712\\\",\\\"PriceScale\\\":[\\\"66313464177401058702341\\\",\\\"3988288337309167729564\\\"],\\\"PriceOracle\\\":[\\\"63612706012126486095056\\\",\\\"3782761569503404058823\\\"],\\\"LastPrices\\\":[\\\"63608488224235038716789\\\",\\\"3782322291001686876800\\\"],\\\"LastPricesTimestamp\\\":1710838775,\\\"FeeGamma\\\":\\\"400000000000000\\\",\\\"MidFee\\\":\\\"1000000\\\",\\\"OutFee\\\":\\\"140000000\\\",\\\"LpSupply\\\":\\\"6209561906175920711602\\\",\\\"XcpProfit\\\":\\\"1005532234158713186\\\",\\\"VirtualPrice\\\":\\\"1002781276086899355\\\",\\\"AllowedExtraProfit\\\":\\\"100000000\\\",\\\"AdjustmentStep\\\":\\\"100000000000\\\",\\\"MaTime\\\":\\\"866\\\"}\",\"staticExtra\":\"{\\\"IsNativeCoins\\\":[false,false,false]}\",\"blockNumber\":19468099}"

	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(benchPoolRedis), &poolEntity)
	require.Nil(b, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(b, err)

	ain := bignumber.NewBig10("50000000000000000123")

	for i := 0; i < b.N; i++ {
		_, _ = p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", Amount: ain},
			TokenOut:      "0x18084fba666a33d37592fa2633fd49a74dd93a88",
			Limit:         nil,
		})
	}
}

// Chain quotes below come from an anvil fork of Ethereum at block 26307080: get_dy(probe), then a real
// exchange(tiny), then get_dy(probe) again. They pin what the pool quotes after a trade, which depends on
// price_oracle as the contract computes it at the trade's time.

// XBEN/WETH/USDT last traded days before the fork block, so price_oracle() has fully converged on
// last_prices while the state read four days earlier still holds an older value.
const idlePoolEntity = `{"address":"0x181a7464554e68333193a71719eb164e0db7c038","exchange":"curve-tricrypto-ng","type":"curve-tricrypto-ng","timestamp":1793002214,"reserves":["65030304283067395006853","101233597497079","235078"],"tokens":[{"address":"0xa49bf1560119e2df3316885d7ddb4a248b773e34","symbol":"XBEN","decimals":18,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"2700000\",\"InitialGamma\":\"1300000000000\",\"InitialAGammaTime\":0,\"FutureA\":\"2700000\",\"FutureGamma\":\"1300000000000\",\"FutureAGammaTime\":0,\"D\":\"195139619610630919631441\",\"PriceScale\":[\"664587765160418965337897373\",\"267574311111688644481757\"],\"PriceOracle\":[\"656978986867205835049796140\",\"270530966474470559802612\"],\"LastPrices\":[\"642738502607373715579967548\",\"276486663100919303243535\"],\"PriceOracleTimestamp\":1793002214,\"MaTime\":\"865\",\"FeeGamma\":\"350000000000000\",\"MidFee\":\"2999999\",\"OutFee\":\"80000000\",\"LpSupply\":\"1132717387292003582\",\"XcpProfit\":\"1021424979903121036\",\"VirtualPrice\":\"1021183321544013788\",\"AllowedExtraProfit\":\"100000000000\",\"AdjustmentStep\":\"100000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false,false]}","blockNumber":26275577}`

// crvUSD/tBTC/wstETH as the tracker reads it at the fork block, with price_oracle still on its way to
// last_prices. MaTime 866 is packed_rebalancing_params; the ma_time() view says 601.
const convergingPoolEntity = `{"address":"0x2889302a794da87fbf1d6db415c1492194663d13","exchange":"curve-tricrypto-ng","type":"curve-tricrypto-ng","timestamp":1793585990,"reserves":["1426847083933437585734103","18455901695458957243","504486581918630834702"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0x18084fba666a33d37592fa2633fd49a74dd93a88","symbol":"tBTC","decimals":18,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","symbol":"wstETH","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"1707629\",\"InitialGamma\":\"11809167828997\",\"InitialAGammaTime\":1705051559,\"FutureA\":\"540000\",\"FutureGamma\":\"80500000000000\",\"FutureAGammaTime\":1705537322,\"D\":\"4262998684971501993579040\",\"PriceScale\":[\"77483838002402501097865\",\"2787263116562429934396\"],\"PriceOracle\":[\"77451240317304208904803\",\"2795358358963973264794\"],\"LastPrices\":[\"77437919521269141718975\",\"2798021746139515972416\"],\"PriceOracleTimestamp\":1793381475,\"MaTime\":\"866\",\"FeeGamma\":\"400000000000000\",\"MidFee\":\"1000000\",\"OutFee\":\"140000000\",\"LpSupply\":\"2268022205913350776207\",\"XcpProfit\":\"1078985732841659025\",\"VirtualPrice\":\"1044279960688415810\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false,false]}","blockNumber":26307082}`

func quoteAfterTrade(t *testing.T, p entity.Pool, in, out int, probe, tiny string) (pre, post *big.Int) {
	t.Helper()
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	tokenIn, tokenOut := sim.Info.Tokens[in], sim.Info.Tokens[out]
	quote := func(s *PoolSimulator, amount *big.Int) *pool.CalcAmountOutResult {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		return res
	}
	probeAmount, _ := new(big.Int).SetString(probe, 10)
	tinyAmount, _ := new(big.Int).SetString(tiny, 10)

	pre = quote(sim, probeAmount).TokenAmountOut.Amount
	clone := sim.CloneState().(*PoolSimulator)
	trade := quote(clone, tinyAmount)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: tinyAmount},
		TokenAmountOut: *trade.TokenAmountOut,
		SwapInfo:       trade.SwapInfo,
	})
	post = quote(clone, probeAmount).TokenAmountOut.Amount

	assert.Equal(t, pre.String(), quote(sim, probeAmount).TokenAmountOut.Amount.String(),
		"the trade on the clone must not move the original")
	return pre, post
}

func TestPriceOracleCarriedForward_IdlePool(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(idlePoolEntity), &p))
	for _, tc := range []struct {
		name        string
		in, out     int
		probe, tiny string
		pre, post   string
	}{
		{"XBEN to WETH", 0, 1, "650303042830673950", "650303042830673", "1005491386", "1006128262"},
		{"WETH to XBEN", 1, 0, "1012335974", "1012335", "646648899036478407", "647118633085104470"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pre, post := quoteAfterTrade(t, p, tc.in, tc.out, tc.probe, tc.tiny)
			assert.Equal(t, tc.pre, pre.String(), "get_dy before the trade")
			assert.Equal(t, tc.post, post.String(), "get_dy after the trade")
		})
	}
}

// The trade lands 300 s after the read, about a third of the EMA's time constant, so the simulator has to
// carry the oracle part of the way with the pool's true constant.
func TestPriceOracleCarriedForward_PartWay(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(convergingPoolEntity), &p))
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	nowUnix = func() int64 { return 1793381775 }

	pre, post := quoteAfterTrade(t, p, 0, 1, "14268470839334375857", "14268470839334375")
	assert.Equal(t, "183797572098291", pre.String(), "get_dy before the trade")
	assert.Equal(t, "183834889757060", post.String(), "get_dy after the trade")
}

// A state written before this change has neither field and keeps quoting from the stored oracle.
func TestPriceOracleNotCarriedForward_OldState(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(idlePoolEntity), &p))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra.MaTime, extra.PriceOracleTimestamp = nil, 0
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)

	_, post := quoteAfterTrade(t, p, 0, 1, "650303042830673950", "650303042830673")
	assert.Equal(t, "1005679663", post.String())
}

func TestPriceOracleAt(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(convergingPoolEntity), &p))
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	readAt := sim.Extra.PriceOracleTimestamp
	stored, last := sim.Extra.PriceOracle, sim.Extra.LastPrices

	at := func(s *PoolSimulator, timestamp int64) ([]uint256.Int, int64) {
		out := make([]uint256.Int, NumTokens-1)
		got, err := s.priceOracleAt(timestamp, out)
		require.NoError(t, err)
		return out, got
	}

	t.Run("at or before the read, the stored value", func(t *testing.T) {
		for _, timestamp := range []int64{readAt - 100, readAt} {
			out, got := at(sim, timestamp)
			assert.Equal(t, readAt, got)
			assert.Equal(t, stored, out)
		}
	})
	t.Run("after the read, between the stored value and last_prices", func(t *testing.T) {
		out, got := at(sim, readAt+300)
		assert.Equal(t, readAt+300, got)
		assert.True(t, out[0].Lt(&stored[0]) && out[0].Gt(&last[0]), "oracle 0 moves down toward last_prices")
		assert.True(t, out[1].Gt(&stored[1]) && out[1].Lt(&last[1]), "oracle 1 moves up toward last_prices")
	})
	t.Run("long after the read, last_prices", func(t *testing.T) {
		out, _ := at(sim, readAt+100*866)
		assert.Equal(t, last, out)
	})
	t.Run("does not write to the stored value", func(t *testing.T) {
		before := append([]uint256.Int(nil), sim.Extra.PriceOracle...)
		at(sim, readAt+300)
		assert.Equal(t, before, sim.Extra.PriceOracle)
	})
}

func TestMaTimeFromRebalancingParams(t *testing.T) {
	allowedExtraProfit, adjustmentStep := big.NewInt(100000000), big.NewInt(100000000000)
	packed := new(big.Int).Lsh(allowedExtraProfit, 128)
	packed.Or(packed, new(big.Int).Lsh(adjustmentStep, 64))
	packed.Or(packed, big.NewInt(866))
	assert.Equal(t, uint64(866), MaTimeFromRebalancingParams(packed).Uint64())
	assert.True(t, MaTimeFromRebalancingParams(new(big.Int)).IsZero())
}
