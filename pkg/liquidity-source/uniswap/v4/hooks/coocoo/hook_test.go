package coocoo

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Every CooCoo hook address resolves to this package's handler under its own exchange, with
// the tracked Extra restored from HookExtra. The addresses' permission bits (…2044, …E044)
// enable afterSwap and afterSwapReturnDelta and never beforeSwap, like Pons' own hook.
func TestRegistered(t *testing.T) {
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{
			HookExtra: uniswapv4.HookExtra(`{"r":true,"f":150}`),
		})
		require.True(t, ok, addr)
		h, isCooCoo := hook.(*Hook)
		require.True(t, isCooCoo, addr)
		assert.Equal(t, string(valueobject.ExchangeUniswapV4CooCoo), h.GetExchange())
		assert.True(t, h.Registered)
		assert.Equal(t, int64(150), h.FeeBps)
		assert.True(t, uniswapv4.HasSwapPermissions(addr))
		assert.True(t, h.CanAfterSwap(addr))
		assert.True(t, h.CanBeforeSwap(addr), "simulator-only: runs the CalcAmountIn gross-up")
	}
}

// Same swap, same fee as hooks/pons-v2 on both sides: the deployments share
// PonsV2MemeHook's bytecode, so the fee is floor(unspecified * feeBps / 10_000) of the
// realized output on exact-in. CalcAmountIn reverses exact-in, so it charges no input fee.
func TestAfterSwap_MatchesPonsV2(t *testing.T) {
	ours := &Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 150}}
	theirs := &ponsv2.Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 150}}
	for _, calcOut := range []bool{true, false} {
		params := &uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: calcOut, AmountSpecified: big.NewInt(985_000)},
			AmountIn:         big.NewInt(1_000_000),
			AmountOut:        big.NewInt(2_000_000),
		}
		got, err := ours.AfterSwap(params)
		require.NoError(t, err)
		want, err := theirs.AfterSwap(params)
		require.NoError(t, err)
		assert.Equal(t, want.HookFee, got.HookFee)
		// exact-in: floor(2_000_000 * 150 / 10_000); CalcAmountIn: nothing
		assert.Equal(t, big.NewInt(map[bool]int64{true: 30_000, false: 0}[calcOut]), got.HookFee)

		gotBefore, err := ours.BeforeSwap(params.BeforeSwapParams)
		require.NoError(t, err)
		wantBefore, err := theirs.BeforeSwap(params.BeforeSwapParams)
		require.NoError(t, err)
		assert.Equal(t, wantBefore, gotBefore)
	}
}

// Executing the quoted input exact-in must deliver at least the requested output, and one
// wei less input must not. It failed when the fee was charged on the input in CalcAmountIn.
func TestCalcAmountIn_DeliversRequestedOutput(t *testing.T) {
	for _, addr := range HookAddresses {
		static, err := json.Marshal(map[string]any{"0x0": [2]bool{false, false}, "fee": 2000, "tS": 60, "hooks": addr})
		require.NoError(t, err)
		extra := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
			`"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
			`{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":{"r":true,"f":233}}`
		sim, err := uniswapv4.NewPoolSimulator(entity.Pool{
			Address:  "0x1111111111111111111111111111111111111111111111111111111111111111",
			Type:     uniswapv4.DexType,
			SwapFee:  2000,
			Reserves: []string{"1000000000000", "1000000000000"},
			Tokens: []*entity.PoolToken{
				{Address: "0xe000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
				{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
			},
			Extra: extra, StaticExtra: string(static),
		}, valueobject.ChainIDEthereum)
		require.NoError(t, err)

		tokens := sim.Info.Tokens
		want := big.NewInt(1_000_000)
		delivered := func(in *big.Int) *big.Int {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: in}, TokenOut: tokens[1]})
			require.NoError(t, err)
			return res.TokenAmountOut.Amount
		}
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokens[1], Amount: want}, TokenIn: tokens[0]})
		require.NoError(t, err)
		in := res.TokenAmountIn.Amount
		require.GreaterOrEqual(t, delivered(in).Cmp(want), 0, "%s: quoted input must deliver the output", addr)
		require.Less(t, delivered(new(big.Int).Sub(in, bignumber.One)).Cmp(want), 0, "%s: and be the smallest", addr)
	}
}

// A pool Track() never saw prices fee-free rather than erroring: PonsV2MemeHook._afterSwap
// itself returns 0 for an unregistered pool.
func TestAfterSwap_Unregistered_NoFee(t *testing.T) {
	result, err := (&Hook{}).AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

// What Track hands back is what the factory restores from HookExtra on the next load.
func TestExtraRoundTrip(t *testing.T) {
	h := &Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 100}}
	raw, err := json.Marshal(h)
	require.NoError(t, err)
	assert.JSONEq(t, `{"r":true,"f":100}`, string(raw))

	restored, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)})
	require.True(t, ok)
	assert.Equal(t, h.Extra, restored.(*Hook).Extra)
}
