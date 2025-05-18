package tricryptong

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
		outOrError interface{}
	}{
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304718298868"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304675974788"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1304898571422510"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "1306024453847454"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "3393536839049"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1822607809867"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1819677288075988"},
		{0, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1821248242501378"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "1574126861158"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xd533a949740bb3306d119cc777fa900ba034cd52", "2056531723"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "696648048123"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xd533a949740bb3306d119cc777fa900ba034cd52", "697248127714"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "362851"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "191462"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "191255367"},
		{0, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "191420171"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "302552"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "137440"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "137149910"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "137268281"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "444235376"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "358363928"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "358242922283"},
		{0, "0xd533a949740bb3306d119cc777fa900ba034cd52", "500000000123", "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", "358551652118"},
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
