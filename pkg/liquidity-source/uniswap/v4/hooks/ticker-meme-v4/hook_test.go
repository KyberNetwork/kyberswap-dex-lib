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
	require.False(t, hook.CanBeforeSwap(ticker.HookAddress))
	require.True(t, hook.CanAfterSwap(ticker.HookAddress))
	params := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountOut: big.NewInt(199)}
	got, err := hook.AfterSwap(params)
	require.NoError(t, err)
	want, err := (&ponsv2.Extra{Registered: true, FeeBps: 200}).AfterSwap(params)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestProductionCache(t *testing.T) {
	for _, extra := range []string{`{"r":true,"f":233}`, `{"r":true,"f":"bad"}`} {
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
