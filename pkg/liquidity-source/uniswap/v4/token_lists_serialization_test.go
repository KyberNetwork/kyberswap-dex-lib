package uniswapv4_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The token-list caches are not serialized, so a decoded simulator computes the lists per call. It
// must still answer like the original, wrapped tokens included: an fwWETH/fwUNI pool (FEW v1).
func TestTokenListsAfterMsgpackRoundTrip(t *testing.T) {
	const (
		weth, fwWETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xa250cc729bb3323e7933022a67b52200fe354767"
		uni, fwUNI   = "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", "0xe8e1f50392bd61d0f8f48e8e7af51d3b8a52090a"
	)
	p := entity.Pool{
		Address:     "uniswapV4PoolAddress",
		Exchange:    uniswapv4.DexType,
		Type:        uniswapv4.DexType,
		Reserves:    entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens:      []*entity.PoolToken{{Address: fwWETH, Swappable: true}, {Address: fwUNI, Swappable: true}},
		StaticExtra: `{"0x0":[false,false],"fee":100,"tS":1,"hooks":"0x0000000000000000000000000000000000000000"}`,
		Extra: `{"liquidity":1000000000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":1,"tick":0,` +
			`"ticks":[{"index":-887272,"liquidityGross":1000000000000000000,"liquidityNet":1000000000000000000},` +
			`{"index":887272,"liquidityGross":1000000000000000000,"liquidityNet":-1000000000000000000}]}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	sims, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	decoded := sims[p.Address]

	require.ElementsMatch(t, []string{fwWETH, weth, fwUNI, uni}, decoded.GetTokens())
	require.ElementsMatch(t, sim.GetTokens(), decoded.GetTokens())
	for _, token := range []string{fwWETH, weth, fwUNI, uni, "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202"} {
		require.ElementsMatch(t, sim.CanSwapTo(token), decoded.CanSwapTo(token), token)
		require.ElementsMatch(t, sim.CanSwapFrom(token), decoded.CanSwapFrom(token), token)
	}
	require.ElementsMatch(t, []string{fwUNI, uni}, decoded.CanSwapFrom(weth))
}
