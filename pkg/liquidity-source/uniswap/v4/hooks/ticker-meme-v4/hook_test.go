package tickermemev4

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func terms(fee int64) Extra {
	return Extra{Extra: ponsv2.Extra{Registered: true, FeeBps: fee}, ImplementationHash: ImplementationCodeHash}
}

func TestAfterSwapReusesPons(t *testing.T) {
	for _, extra := range []Extra{terms(0), terms(200), terms(233), {ImplementationHash: ImplementationCodeHash}} {
		for _, calcOut := range []bool{true, false} {
			for _, amount := range []int64{99, 199, 200, 1000000} {
				h := &Hook{Extra: extra}
				param := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: calcOut}, AmountIn: big.NewInt(amount), AmountOut: big.NewInt(amount)}
				want, err := extra.Extra.AfterSwap(param)
				require.NoError(t, err)
				got, err := h.AfterSwap(param)
				require.NoError(t, err)
				require.Equal(t, want.HookFee, got.HookFee)
				if got.HookFee.Sign() > 0 {
					require.EqualValues(t, afterSwapGas, got.Gas)
				} else {
					require.Zero(t, got.Gas)
				}
				require.EqualValues(t, amount, param.AmountIn.Int64())
				require.EqualValues(t, amount, param.AmountOut.Int64())
			}
		}
	}
}

func TestFactoryStateAndSerialization(t *testing.T) {
	extra := terms(233)
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, HookExtra: raw}
	hook, ok := uniswapv4.GetHook(HookAddress, param)
	require.True(t, ok)
	h := hook.(*Hook)
	require.NoError(t, h.validate())
	require.Equal(t, DexType, h.GetExchange())
	require.True(t, h.CanAfterSwap(HookAddress))
	require.False(t, h.CanBeforeSwap(HookAddress))
	encoded, err := msgpack.Marshal(h)
	require.NoError(t, err)
	var restored Hook
	require.NoError(t, msgpack.Unmarshal(encoded, &restored))
	require.NoError(t, restored.validate())
	require.Equal(t, extra, restored.Extra)
	clone := h.CloneState().(*Hook)
	clone.FeeBps = 1000
	require.EqualValues(t, 233, h.FeeBps)

	// The old f/t schema must be refreshed, never read as an already-combined fee.
	for _, invalid := range []string{
		`{}`, `{"impl":"` + ImplementationCodeHash.Hex() + `","r":true,"f":100,"t":133}`,
		`{"implementation":"` + ImplementationCodeHash.Hex() + `","r":true,"f":"bad"}`,
		`{"implementation":"` + ImplementationCodeHash.Hex() + `","r":true,"f":-1}`,
		`{"implementation":"` + ImplementationCodeHash.Hex() + `","r":true,"f":2001}`,
	} {
		param.HookExtra = uniswapv4.HookExtra(invalid)
		rejected := NewHook(param).(*Hook)
		require.Error(t, rejected.validate())
		encoded, err := msgpack.Marshal(rejected)
		require.NoError(t, err)
		var roundTrip Hook
		require.NoError(t, msgpack.Unmarshal(encoded, &roundTrip))
		require.Error(t, roundTrip.validate())
	}
	param.Cfg.ChainID = valueobject.ChainIDBase
	require.Error(t, NewHook(param).(*Hook).validate())
}

func TestV4ForwardQuote(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, index := range []int{0, 1} {
			p := newPool(t, terms(233), native)
			param := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Info.Tokens[index], Amount: big.NewInt(1000000)}, TokenOut: p.Info.Tokens[1-index]}
			first, err := p.CalcAmountOut(param)
			require.NoError(t, err)
			// This captured swap delivered 974747 on-chain; the shared Pons rounding
			// conservatively quotes one raw unit less. This PR preserves that model.
			require.Equal(t, "974746", first.TokenAmountOut.Amount.String())
			again, err := p.CalcAmountOut(param)
			require.NoError(t, err)
			require.Equal(t, first, again)
			require.Equal(t, DexType, p.GetExchange())
			require.EqualValues(t, snapshotBlock, p.GetMetaInfo(param.TokenAmountIn.Token, param.TokenOut).(uniswapv4.PoolMetaInfo).BlockNumber)
			clone := p.CloneState().(*uniswapv4.PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: param.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, SwapInfo: first.SwapInfo})
			changed, err := clone.CalcAmountOut(param)
			require.NoError(t, err)
			require.Less(t, changed.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), 0)
			original, err := p.CalcAmountOut(param)
			require.NoError(t, err)
			require.Equal(t, first, original)
		}
	}
}

func newPool(t *testing.T, extra Extra, native bool) *uniswapv4.PoolSimulator {
	t.Helper()
	hx, err := json.Marshal(extra)
	require.NoError(t, err)
	dynamic := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":` + string(hx) + `}`
	static, err := json.Marshal(map[string]any{"0x0": [2]bool{native, false}, "fee": 2000, "tS": 60, "hooks": HookAddress})
	require.NoError(t, err)
	token0 := "0x1000000000000000000000000000000000000000"
	if native {
		token0 = valueobject.LowerWrapped(valueobject.ChainIDMonad)
	}
	result, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address: "0x1111111111111111111111111111111111111111111111111111111111111111",
		Type:    uniswapv4.DexType, Exchange: DexType, SwapFee: 2000,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens:   []*entity.PoolToken{{Address: token0, Decimals: 18, Swappable: true}, {Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true}},
		Extra:    dynamic, StaticExtra: string(static), BlockNumber: 112361144,
	}, valueobject.ChainIDMonad)
	require.NoError(t, err)
	return result
}
