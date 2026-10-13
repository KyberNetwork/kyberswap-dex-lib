package tickermemev4

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func terms(fee, tax uint16) Extra {
	return Extra{Registered: true, HookFeeBps: fee, CreatorTaxBps: tax, ImplementationHash: ImplementationCodeHash}
}

func TestOutputFeeCallbacks(t *testing.T) {
	h := &Hook{Extra: terms(100, 100)}
	for _, tc := range []struct{ gross, fee int64 }{{99, 0}, {199, 2}, {200, 4}} {
		result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountOut: big.NewInt(tc.gross),
		})
		require.NoError(t, err)
		require.EqualValues(t, tc.fee, result.HookFee.Int64())
		if tc.fee == 0 {
			require.Zero(t, result.Gas)
		} else {
			require.EqualValues(t, afterSwapGas, result.Gas)
		}
	}
	net := big.NewInt(197)
	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: net})
	require.NoError(t, err)
	// Combined-rate gross-up starts at 201; separately rounded fees leave 197.
	require.EqualValues(t, 4, before.DeltaSpecified.Int64())
	require.Zero(t, before.DeltaUnspecified.Sign())
	require.EqualValues(t, 197, net.Int64())
	after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false}})
	require.NoError(t, err)
	require.Zero(t, after.HookFee.Sign())
	require.Zero(t, after.Gas)
	for _, extra := range []Extra{terms(0, 0), {ImplementationHash: ImplementationCodeHash}} {
		before, err := (&Hook{Extra: extra}).BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: net})
		require.NoError(t, err)
		require.Zero(t, before.DeltaSpecified.Sign())
		require.Zero(t, before.Gas)
	}
	before, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: net})
	require.NoError(t, err)
	require.Zero(t, before.DeltaSpecified.Sign())
}

func TestFactoryStateAndSerialization(t *testing.T) {
	extra := terms(100, 133)
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, HookExtra: raw}
	hook, ok := uniswapv4.GetHook(HookAddress, param)
	require.True(t, ok)
	h := hook.(*Hook)
	require.NoError(t, h.validate())
	require.Equal(t, "uniswap-v4-ticker-meme", h.GetExchange())
	require.True(t, h.CanAfterSwap(HookAddress))
	require.True(t, h.CanBeforeSwap(HookAddress))
	encoded, err := msgpack.Marshal(h)
	require.NoError(t, err)
	var restored Hook
	require.NoError(t, msgpack.Unmarshal(encoded, &restored))
	require.NoError(t, restored.validate())
	require.Equal(t, extra, restored.Extra)
	clone := h.CloneState().(*Hook)
	clone.HookFeeBps = 1000
	require.EqualValues(t, 100, h.HookFeeBps)

	// Combined-rate snapshots must be refreshed before quoting with separate rates.
	for _, invalid := range []string{
		`{}`, `{"implementation":"` + ImplementationCodeHash.Hex() + `","r":true,"f":233}`,
		`{"impl":"` + ImplementationCodeHash.Hex() + `","r":true,"f":100,"t":133}`,
		`{"implementationV2":"` + ImplementationCodeHash.Hex() + `","r":true,"f":"bad"}`,
		`{"implementationV2":"` + ImplementationCodeHash.Hex() + `","r":true,"f":-1}`,
		`{"implementationV2":"` + ImplementationCodeHash.Hex() + `","r":true,"f":2001}`,
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
	p := newPool(t, terms(100, 133))
	param := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Info.Tokens[0], Amount: big.NewInt(1000000)}, TokenOut: p.Info.Tokens[1]}
	result, err := p.CalcAmountOut(param)
	require.NoError(t, err)
	require.Equal(t, "974747", result.TokenAmountOut.Amount.String())
	require.Equal(t, "uniswap-v4-ticker-meme", p.GetExchange())
	require.EqualValues(t, snapshotBlock, p.GetMetaInfo(param.TokenAmountIn.Token, param.TokenOut).(uniswapv4.PoolMetaInfo).BlockNumber)
}

func TestV4ReverseReplay(t *testing.T) {
	for _, rates := range [][2]uint16{{100, 133}, {1000, 1000}} {
		for _, index := range []int{0, 1} {
			p := newPool(t, terms(rates[0], rates[1]))
			for _, desired := range []int64{197, 1000000} {
				net := big.NewInt(desired)
				reverse, err := p.CalcAmountIn(pool.CalcAmountInParams{
					TokenIn:        p.Info.Tokens[index],
					TokenAmountOut: pool.TokenAmount{Token: p.Info.Tokens[1-index], Amount: net},
				})
				require.NoError(t, err)
				forward, err := p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: *reverse.TokenAmountIn, TokenOut: p.Info.Tokens[1-index],
				})
				require.NoError(t, err)
				require.GreaterOrEqual(t, forward.TokenAmountOut.Amount.Cmp(net), 0,
					"rates=%v direction=%d desired=%d", rates, index, desired)
				require.EqualValues(t, desired, net.Int64())
			}
		}
	}
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
