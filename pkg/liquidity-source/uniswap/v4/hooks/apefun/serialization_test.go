package apefun_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/apefun"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// A restored simulator keeps the hook's per-pool state (an unrestored one would refuse to quote): RH4 at
// Robinhood block 85021170, a 0.01 ETH buy matching V4Quoter, through two msgpack round trips.
func TestSimulatorSerialization(t *testing.T) {
	orig := apefun.NowFn
	apefun.NowFn = func() int64 { return 1791637725 }
	t.Cleanup(func() { apefun.NowFn = orig })

	const weth, rh4 = "0x0bd7d308f8e1639fab988df18a8011f41eacad73", "0x85a7c6615fc55d33c327641d9e87360c07a2b16f"
	p := entity.Pool{
		Address:  "0x331dfe75a9151db444783fef00f2a0a7b97bc5682211f2283f3e13db6626d813",
		Exchange: string(valueobject.ExchangeUniswapV4ApeFun),
		Type:     uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: weth, Swappable: true}, {Address: rh4, Swappable: true}},
		Reserves: entity.PoolReserves{"1000000000000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: `{"0x0":[false,false],"fee":0,"tS":1,"hooks":"0x6a87584119a16ef1df36c17a7d7c55e090da60cc",` +
			`"uR":"0x0000000000000000000000000000000000000000","pm2":"0x0000000000000000000000000000000000000000",` +
			`"mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`,
		Extra: `{"liquidity":40987439322977844309085,"sqrtPriceX96":1881323003042868532779145197824480,"tickSpacing":1,` +
			`"tick":201513,"ticks":[{"index":-887272,"liquidityGross":40987439322977844309085,"liquidityNet":40987439322977844309085},` +
			`{"index":202055,"liquidityGross":40987439322977844309085,"liquidityNet":-40987439322977844309085}],` +
			`"hX":{"v":true,"q0":true,"l":1791454166}}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	sims := map[string]pool.IPoolSimulator{p.Address: sim}
	for range 2 {
		encoded, err := msgpack.EncodePoolSimulatorsMap(sims)
		require.NoError(t, err)
		sims, err = msgpack.DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
	}
	res, err := sims[p.Address].CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1e16)}, TokenOut: rh4})
	require.NoError(t, err)
	require.Equal(t, "5550332439725505417402886", res.TokenAmountOut.Amount.String())
}
