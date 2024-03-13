package stablemetang

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	basePools := []string{
		// base pool is NG https://etherscan.io/address/0x383e6b4437b59fff47b619cba855ca29342a8559
		"{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710325214,\"reserves\":[\"20645714947000\",\"16619279610257\",\"37260809758180318203561662\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19425514}",
	}

	pools := []string{
		// https://etherscan.io/address/0x9e10f9fb6f0d32b350cee2618662243d4f24c64a
		"{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710325225,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000073197173325044\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19425514}",
	}

	testcases := []struct {
		poolIdx           int
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		// meta pool swap
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183840532051551"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "491838395529404280"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "4918382977155694248"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183731951271033406"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781231606587338678"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "5000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5078133338462932470"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "500000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507813435641987387"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781344581152143"},

		// meta -> base
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49219"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492193"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "4921930"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49219208"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492182271"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49146"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491467"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "4914673"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49146640"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491456594"},

		// base -> meta
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5074020539796174052"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50740089393727228165"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507390727614246026441"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5072902983968984217079"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50639540695688204241877"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5081403477635882869"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50813917368257894387"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "508128962320265975010"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5080281497249917498225"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50712988541587113680160"},
	}

	baseSimsByAddress := make(map[string]ICurveBasePool, len(basePools))
	for _, basePool := range basePools {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(basePool), &poolEntity)
		require.Nil(t, err)

		p, err := stableng.NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		baseSimsByAddress[poolEntity.Address] = p
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		var e StaticExtra
		err = json.Unmarshal([]byte(poolEntity.StaticExtra), &e)
		require.Nil(t, err)

		baseSim := baseSimsByAddress[e.BasePool]
		p, err := NewPoolSimulator(poolEntity, baseSim)
		require.Nil(t, err)

		// 1st meta token can be swapped to anything
		assert.Equal(t, append([]string{p.Info.Tokens[1]}, baseSim.GetInfo().Tokens...), p.CanSwapTo(p.Info.Tokens[0]))

		// last meta token can't be swapped to anything other than the 1st one
		assert.Equal(t, []string{p.Info.Tokens[0]}, p.CanSwapTo(p.Info.Tokens[1]))

		// base token can be swapped to anything other than the last meta token and itself
		for i, baseToken := range baseSim.GetInfo().Tokens {
			baseExcluded := lo.Filter(baseSim.GetInfo().Tokens, func(_ string, ii int) bool { return ii != i })
			assert.Equal(t, append([]string{p.Info.Tokens[0]}, baseExcluded...), p.CanSwapTo(baseToken))
		}

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
	basePools := []string{
		// base pool is NG https://etherscan.io/address/0x383e6b4437b59fff47b619cba855ca29342a8559
		"{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710229593,\"reserves\":[\"22600707535418\",\"16842923284257\",\"39438046669577509318174297\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19417597}",
	}

	pools := []string{
		// https://etherscan.io/address/0x9e10f9fb6f0d32b350cee2618662243d4f24c64a
		"{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710229592,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000069503434122144\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19417597}",
	}

	testcases := []struct {
		poolIdx          int
		in               string
		inAmount         string
		out              string
		errorOrAmountOut interface{}
	}{
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49184017234162462"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "491840160385895998"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "4918400408762591035"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183884527960843832"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781296416670560503"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "5000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5078117199758934580"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "500000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507811595528318123"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781158308881341"},
	}

	baseSimsByAddress := make(map[string]ICurveBasePool, len(basePools))
	for _, basePool := range basePools {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(basePool), &poolEntity)
		require.Nil(t, err)

		p, err := stableng.NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		baseSimsByAddress[poolEntity.Address] = p
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		var e StaticExtra
		err = json.Unmarshal([]byte(poolEntity.StaticExtra), &e)
		require.Nil(t, err)

		p, err := NewPoolSimulator(poolEntity, baseSimsByAddress[e.BasePool])
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
			if expErr, ok := tc.errorOrAmountOut.(error); ok {
				require.Equal(t, expErr, err)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.errorOrAmountOut.(string)), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
				SwapLimit:      nil,
			})
		})
	}
}

func BenchmarkGetDyUnderlying(b *testing.B) {
	// {"Am", 1000, "A", 31},
	// base, err := base.NewPoolSimulator(entity.Pool{
	// 	Exchange:    "",
	// 	Type:        "",
	// 	Reserves:    entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243", "352290453972395231054279357"},
	// 	Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
	// 	Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
	// 	StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\",\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}",
	// })
	// require.Nil(b, err)

	// p, err := NewPoolSimulator(entity.Pool{
	// 	Exchange:    "",
	// 	Type:        "",
	// 	Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
	// 	Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
	// 	Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
	// 	StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
	// }, base)
	// require.Nil(b, err)

	// for i := 0; i < b.N; i++ {
	// 	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
	// 		TokenAmountIn: pool.TokenAmount{Token: "B", Amount: big.NewInt(10)},
	// 		TokenOut:      "A",
	// 		Limit:         nil,
	// 	})
	// 	require.Nil(b, err)
	// }
}
