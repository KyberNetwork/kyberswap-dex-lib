package tickermemev4_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	ticker "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/ticker-meme-v4"
	cache "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestRegistrationUsesPons(t *testing.T) {
	hook, ok := uniswapv4.GetHook(ticker.HookAddress, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"r":true,"f":200}`)})
	require.True(t, ok)
	require.Equal(t, "uniswap-v4-ticker-meme", hook.GetExchange())
	require.True(t, hook.CanBeforeSwap(ticker.HookAddress), "simulator-only: runs the exact-out gross-up")
	require.True(t, hook.CanAfterSwap(ticker.HookAddress))
	params := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountOut: big.NewInt(199)}
	got, err := hook.AfterSwap(params)
	require.NoError(t, err)
	want, err := (&ponsv2.Extra{Registered: true, FeeBps: 200}).AfterSwap(params)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func memeSim(t *testing.T, extra string) *uniswapv4.PoolSimulator {
	t.Helper()
	static, err := json.Marshal(map[string]any{"0x0": [2]bool{true, false}, "fee": 2000, "tS": 60, "hooks": ticker.HookAddress})
	require.NoError(t, err)
	dynamic := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":` + extra + `}`
	p, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address: "0x1111111111111111111111111111111111111111111111111111111111111111",
		Type:    uniswapv4.DexType, Exchange: ticker.DexType, SwapFee: 2000,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens:   []*entity.PoolToken{{Address: valueobject.LowerWrapped(valueobject.ChainIDMonad), Decimals: 18, Swappable: true}, {Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true}},
		Extra:    dynamic, StaticExtra: string(static), BlockNumber: 112361144,
	}, valueobject.ChainIDMonad)
	require.NoError(t, err)
	return p
}

func TestProductionCache(t *testing.T) {
	for _, extra := range []string{`{"r":true,"f":233}`, `{"r":true,"f":"bad"}`} {
		p := memeSim(t, extra)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Info.Tokens[0], Amount: big.NewInt(1000000)}, TokenOut: p.Info.Tokens[1]}
		before, err := p.CalcAmountOut(params)
		require.NoError(t, err)
		encoded, err := cache.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.GetAddress(): p})
		require.NoError(t, err)
		decoded, err := cache.DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
		for _, restored := range []pool.IPoolSimulator{decoded[p.GetAddress()], p.CloneState()} {
			after, err := restored.CalcAmountOut(params)
			require.NoError(t, err)
			require.Equal(t, before.TokenAmountOut, after.TokenAmountOut)
			require.Equal(t, ticker.DexType, restored.GetExchange())
		}
	}
}

// CalcAmountIn reverses an exact-in swap whose 233 bps fee comes out of the realized
// output, so the quoted input must be the smallest one that still delivers `want`.
func TestExactOutIsSmallestInputSurvivingOutputFee(t *testing.T) {
	p := memeSim(t, `{"r":true,"f":233}`)
	free := memeSim(t, `{"r":true}`)
	tokens := p.Info.Tokens
	want := big.NewInt(1000000)
	in := func(sim *uniswapv4.PoolSimulator) *big.Int {
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: tokens[1], Amount: want}, TokenIn: tokens[0]})
		require.NoError(t, err)
		return res.TokenAmountIn.Amount
	}
	out := func(amount *big.Int) *big.Int {
		res, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: amount}, TokenOut: tokens[1]})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount
	}
	got := in(p)
	require.Greater(t, got.Cmp(in(free)), 0, "the output fee must raise the required input")
	require.GreaterOrEqual(t, out(got).Cmp(want), 0)
	require.Less(t, out(new(big.Int).Sub(got, big.NewInt(1))).Cmp(want), 0)
}
