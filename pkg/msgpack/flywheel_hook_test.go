package msgpack

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/flywheel"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestFlywheelHookRoundTrip(t *testing.T) {
	static, err := json.Marshal(v4.StaticExtra{HooksAddress: flywheel.HookAddress, TickSpacing: 200})
	require.NoError(t, err)
	sim, err := v4.NewPoolSimulator(entity.Pool{
		Address: "0x1111111111111111111111111111111111111111111111111111111111111111", Type: v4.DexType, BlockNumber: 87735162,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xe000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
			{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
		},
		Extra: `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":200,"tick":0,` +
			`"ticks":[{"index":-887200,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
			`{"index":887200,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}]}`,
		StaticExtra: string(static),
	}, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{sim.GetAddress(): sim})
	require.NoError(t, err)
	decoded, err := DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	restored := decoded[sim.GetAddress()].(*v4.PoolSimulator)
	require.Equal(t, valueobject.ExchangeUniswapV4Flywheel, restored.GetExchange())
	for _, direction := range []int{0, 1} {
		tokens := sim.GetTokens()
		outArgs := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[direction], Amount: big.NewInt(1000000)}, TokenOut: tokens[1-direction]}
		want, err := sim.CalcAmountOut(outArgs)
		require.NoError(t, err)
		got, err := restored.CalcAmountOut(outArgs)
		require.NoError(t, err)
		require.Equal(t, want, got)
		inArgs := pool.CalcAmountInParams{TokenAmountOut: *want.TokenAmountOut, TokenIn: tokens[direction]}
		wantIn, err := sim.CalcAmountIn(inArgs)
		require.NoError(t, err)
		gotIn, err := restored.CalcAmountIn(inArgs)
		require.NoError(t, err)
		require.Equal(t, wantIn, gotIn)
		require.Equal(t, sim.GetMetaInfo(tokens[direction], tokens[1-direction]), restored.GetMetaInfo(tokens[direction], tokens[1-direction]))
	}
}
