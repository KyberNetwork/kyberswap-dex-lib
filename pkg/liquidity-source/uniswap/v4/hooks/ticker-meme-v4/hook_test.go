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
	for fee, gas := range map[int64]int64{0: 0, 233: afterSwapGas} {
		for _, calcOut := range []bool{true, false} {
			extra := terms(fee)
			param := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: calcOut}, AmountIn: big.NewInt(2000000), AmountOut: big.NewInt(1000000)}
			want, err := extra.Extra.AfterSwap(param)
			require.NoError(t, err)
			got, err := (&Hook{Extra: extra}).AfterSwap(param)
			require.NoError(t, err)
			require.Equal(t, want.HookFee, got.HookFee)
			require.EqualValues(t, gas, got.Gas)
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
	p := newPool(t, terms(233))
	param := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Info.Tokens[0], Amount: big.NewInt(1000000)}, TokenOut: p.Info.Tokens[1]}
	result, err := p.CalcAmountOut(param)
	require.NoError(t, err)
	// This captured swap delivered 974747 on-chain; the shared Pons rounding
	// conservatively quotes one raw unit less. This PR preserves that model.
	require.Equal(t, "974746", result.TokenAmountOut.Amount.String())
	require.Equal(t, DexType, p.GetExchange())
	require.EqualValues(t, snapshotBlock, p.GetMetaInfo(param.TokenAmountIn.Token, param.TokenOut).(uniswapv4.PoolMetaInfo).BlockNumber)
}

func newPool(t *testing.T, extra Extra) *uniswapv4.PoolSimulator {
	t.Helper()
	hx, err := json.Marshal(extra)
	require.NoError(t, err)
	dynamic := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":` + string(hx) + `}`
	static, err := json.Marshal(map[string]any{"0x0": [2]bool{true, false}, "fee": 2000, "tS": 60, "hooks": HookAddress})
	require.NoError(t, err)
	token0 := valueobject.LowerWrapped(valueobject.ChainIDMonad)
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
