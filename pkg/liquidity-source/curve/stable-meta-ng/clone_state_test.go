package stablemetang

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// cloneCases are two meta pools from TestUpdateBalance: one on a curve-stable-ng base pool, one on a
// curve-stable-plain base pool, with a swap from the meta coin to a base coin (the meta <-> base path,
// which goes through the base pool's ApplyRemoveLiquidityOneCoinU256).
var cloneCases = []struct {
	name, base, meta string
	in, out, amount  string
}{
	{
		name:   "stable-ng base",
		base:   "{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710382680,\"reserves\":[\"21024903652839\",\"16240730126117\",\"37260809758180318203561662\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19430235}",
		meta:   "{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710382680,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000073979307112987\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19430235}",
		in:     "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", // mkUSD
		out:    "0x6c3ea9036406852006290770bedfcaba0e23a0e8", // PYUSD, a base coin
		amount: "50000000000000000000000",
	},
	{
		name:   "plain base",
		base:   "{\"address\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1710405854,\"reserves\":[\"74882317978601283428112533\",\"76066551886323\",\"32115318520985\",\"177637651221630809031052488\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"symbol\":\"DAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"2000\\\",\\\"InitialATime\\\":1653559305,\\\"FutureATime\\\":1654158027,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"1\\\",\\\"LpToken\\\":\\\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\\\",\\\"IsNativeCoin\\\":[false,false,false]}\",\"blockNumber\":19432140}",
		meta:   "{\"address\":\"0x2482dfb5a65d901d137742ab1095f26374509352\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710405853,\"reserves\":[\"4556837199510378636842480\",\"113547535917173130561003\",\"4650797641270672114959944\"],\"tokens\":[{\"address\":\"0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54\",\"symbol\":\"PUSd\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x6c3f90f043a72fa612cbac8115ee7e52bde6e490\",\"symbol\":\"3Crv\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1030506792713195533\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\\\"}\",\"blockNumber\":19432140}",
		in:     "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", // PUSd
		out:    "0x6b175474e89094c44da98b954eedeac495271d0f", // DAI, a base coin
		amount: "50000000000000000000000",
	},
}

// newMetaAndBase builds the base pool and a meta pool on it, as a pool manager does.
func newMetaAndBase(t *testing.T, base, meta string) (*PoolSimulator, pool.IPoolSimulator) {
	t.Helper()
	var be entity.Pool
	require.NoError(t, json.Unmarshal([]byte(base), &be))
	var baseSim pool.IPoolSimulator
	var err error
	switch be.Exchange {
	case stableng.DexType:
		baseSim, err = stableng.NewPoolSimulator(be)
	case plain.DexType:
		baseSim, err = plain.NewPoolSimulator(be)
	}
	require.NoError(t, err)
	var me entity.Pool
	require.NoError(t, json.Unmarshal([]byte(meta), &me))
	metaSim, err := NewPoolSimulator(me, map[string]pool.IPoolSimulator{be.Address: baseSim})
	require.NoError(t, err)
	return metaSim, baseSim
}

// A meta <-> base swap on a clone of a meta pool changes only the clone: the original meta pool and the base pool it
// shares with other meta pools keep their reserves, also while clones swap on several goroutines.
func TestUpdateBalanceOnACloneLeavesTheOriginal(t *testing.T) {
	for _, tc := range cloneCases {
		t.Run(tc.name, func(t *testing.T) {
			meta, base := newMetaAndBase(t, tc.base, tc.meta)
			testutil.TestCloneIsolation(t, meta, map[string]testutil.CloneOp{
				"meta to base coin": testutil.SwapOp(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.amount)}, TokenOut: tc.out}),
			}, base)
		})
	}
}
