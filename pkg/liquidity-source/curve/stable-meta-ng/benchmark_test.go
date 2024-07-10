package stablemetang

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/require"
)

var (
	benchBasePool1Redis = "{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710382680,\"reserves\":[\"21024903652839\",\"16240730126117\",\"37260809758180318203561662\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19430235}"

	benchMetaPool1Redis = "{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710382680,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000073979307112987\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19430235}"

	benchBasePool2Redis = "{\"address\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1710405854,\"reserves\":[\"74882317978601283428112533\",\"76066551886323\",\"32115318520985\",\"177637651221630809031052488\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"symbol\":\"DAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"2000\\\",\\\"InitialATime\\\":1653559305,\\\"FutureATime\\\":1654158027,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"1\\\",\\\"LpToken\\\":\\\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\\\",\\\"IsNativeCoin\\\":[false,false,false]}\",\"blockNumber\":19432140}"

	benchMetaPool2Redis = "{\"address\":\"0x2482dfb5a65d901d137742ab1095f26374509352\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710405853,\"reserves\":[\"4556837199510378636842480\",\"113547535917173130561003\",\"4650797641270672114959944\"],\"tokens\":[{\"address\":\"0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54\",\"symbol\":\"PUSd\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x6c3f90f043a72fa612cbac8115ee7e52bde6e490\",\"symbol\":\"3Crv\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1030506792713195533\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\\\"}\",\"blockNumber\":19432140}"
)

/*
BenchmarkGetDy-8                                 	  577413	      2083 ns/op	     240 B/op	       7 allocs/op
BenchmarkGetDyUnderlying_meta2base-8             	  237134	      4975 ns/op	    1064 B/op	      11 allocs/op
BenchmarkGetDyUnderlying_base2meta-8             	  210609	      5673 ns/op	    1064 B/op	      11 allocs/op
BenchmarkGetDy_plainbase-8                       	  521269	      2283 ns/op	     240 B/op	       7 allocs/op
BenchmarkGetDyUnderlying_plainbase_meta2base-8   	  251290	      4722 ns/op	    1064 B/op	      11 allocs/op
BenchmarkGetDyUnderlying_plainbase_base2meta-8   	  227568	      5134 ns/op	    1064 B/op	      11 allocs/op
*/

func doBench(b *testing.B, basepoolRedis, poolRedis, tokIn, tokOut, amount string) {
	var poolEntity entity.Pool
	require.Nil(b, json.Unmarshal([]byte(basepoolRedis), &poolEntity))
	var baseSim ICurveBasePool
	var err error
	if poolEntity.Exchange == stableng.DexType {
		baseSim, err = stableng.NewPoolSimulator(poolEntity)
		require.Nil(b, err)
	} else if poolEntity.Exchange == plain.DexType {
		baseSim, err = plain.NewPoolSimulator(poolEntity)
		require.Nil(b, err)
	}

	require.Nil(b, json.Unmarshal([]byte(poolRedis), &poolEntity))
	sim, err := NewPoolSimulator(poolEntity, baseSim)
	require.Nil(b, err)

	amt := bignumber.NewBig10(amount)

	for i := 0; i < b.N; i++ {
		_, _ = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokIn, Amount: amt},
			TokenOut:      tokOut,
			Limit:         nil,
		})
	}
}

func BenchmarkGetDy(b *testing.B) {
	doBench(b, benchBasePool1Redis, benchMetaPool1Redis, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "0x383e6b4437b59fff47b619cba855ca29342a8559", "5000000000000000000")
}

func BenchmarkGetDyUnderlying_meta2base(b *testing.B) {
	doBench(b, benchBasePool1Redis, benchMetaPool1Redis, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000000000000000")
}

func BenchmarkGetDyUnderlying_base2meta(b *testing.B) {
	doBench(b, benchBasePool1Redis, benchMetaPool1Redis, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000")
}

func BenchmarkGetDy_plainbase(b *testing.B) {
	doBench(b, benchBasePool2Redis, benchMetaPool2Redis, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "5000000000000000000")
}

func BenchmarkGetDyUnderlying_plainbase_meta2base(b *testing.B) {
	doBench(b, benchBasePool2Redis, benchMetaPool2Redis, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "0x6b175474e89094c44da98b954eedeac495271d0f", "5000000000000000000")
}

func BenchmarkGetDyUnderlying_plainbase_base2meta(b *testing.B) {
	doBench(b, benchBasePool2Redis, benchMetaPool2Redis, "0x6b175474e89094c44da98b954eedeac495271d0f", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000")
}
