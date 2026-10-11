package tickermemev4

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func terms(fee, tax uint16) Extra {
	return Extra{ImplementationHash: ImplementationCodeHash, Registered: true, HookFeeBps: fee, CreatorTaxBps: tax}
}

func TestAfterSwapSeparateFloors(t *testing.T) {
	for _, tc := range []struct{ gross, fee int64 }{{1, 0}, {99, 0}, {100, 2}, {199, 2}, {200, 4}, {10001, 200}} {
		for _, direction := range []bool{false, true} {
			h := &Hook{Extra: terms(100, 100)}
			gross := big.NewInt(tc.gross)
			result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
				BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: direction},
				AmountIn:         big.NewInt(999999), AmountOut: gross,
			})
			require.NoError(t, err)
			require.Equal(t, tc.fee, result.HookFee.Int64())
			require.Equal(t, tc.gross, gross.Int64(), "quote changed its input")
			if tc.fee > 0 {
				require.EqualValues(t, afterSwapGas, result.Gas)
			} else {
				require.Zero(t, result.Gas)
			}
		}
	}
}

func TestAfterSwapRejectsInvalidState(t *testing.T) {
	param := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountOut: big.NewInt(10000)}
	for _, extra := range []Extra{{}, terms(1001, 0), terms(1000, 1001)} {
		_, err := (&Hook{Extra: extra}).AfterSwap(param)
		require.Error(t, err)
	}
	h := &Hook{Extra: terms(1000, 1000)}
	_, err := h.AfterSwap(param)
	require.NoError(t, err, "20 percent is permitted")
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 127)} {
		param.AmountOut = amount
		_, err = h.AfterSwap(param)
		require.Error(t, err)
	}
	param.AmountOut = big.NewInt(10000)
	param.CalcOut = false
	_, err = h.AfterSwap(param)
	require.Error(t, err, "reverse quotes must use the exact-input path")
}

func TestUnregisteredAndClone(t *testing.T) {
	h := &Hook{Extra: terms(100, 133)}
	h.Registered = false
	fee, err := h.fee(big.NewInt(10000))
	require.NoError(t, err)
	require.Zero(t, fee.Sign())
	clone := h.CloneState().(*Hook)
	clone.HookFeeBps = 1000
	require.EqualValues(t, 100, h.HookFeeBps)
}

func TestFactoryRejectsWrongChainAndMalformedExtra(t *testing.T) {
	for _, param := range []*uniswapv4.HookParam{
		{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDBase}, HookAddress: HookAddress},
		{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: common.Address{}},
		{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, HookExtra: uniswapv4.HookExtra(`{"r":`)},
	} {
		require.Error(t, NewHook(param).(*Hook).validate())
	}
}

func TestMalformedExtraRemainsRejectedAfterMsgpack(t *testing.T) {
	for _, suffix := range []string{`"f":65536}`, `"f":"bad"}`, `"f":100,"t":65536}`} {
		raw := `{"impl":"` + ImplementationCodeHash.Hex() + `","r":true,` + suffix
		h := NewHook(&uniswapv4.HookParam{
			Cfg:         &uniswapv4.Config{ChainID: valueobject.ChainIDMonad},
			HookAddress: HookAddress,
			HookExtra:   uniswapv4.HookExtra(raw),
		}).(*Hook)
		require.Error(t, h.validate())
		encoded, err := msgpack.Marshal(h)
		require.NoError(t, err)
		var restored Hook
		require.NoError(t, msgpack.Unmarshal(encoded, &restored))
		_, err = restored.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
			AmountOut:        big.NewInt(10000),
		})
		require.Error(t, err, "serialization must not make partially decoded fees quotable")
	}
}

func TestV4QuotesAndUpdates(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, tokenInIndex := range []int{0, 1} {
			hooked := newPool(t, terms(100, 133), native)
			plain := newPool(t, terms(0, 0), native)
			tokens := hooked.Info.Tokens
			meta := hooked.GetMetaInfo(tokens[0], tokens[1]).(uniswapv4.PoolMetaInfo)
			require.EqualValues(t, snapshotBlock, meta.BlockNumber)
			require.Equal(t, HookAddress, meta.HookAddress)
			require.Equal(t, native, hooked.SwapReceiveNativeIn(tokens[0], tokens[1], valueobject.ChainIDMonad))
			require.Equal(t, native, hooked.SwapReturnNativeOut(tokens[1], tokens[0], valueobject.ChainIDMonad))
			quote := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[tokenInIndex], Amount: big.NewInt(1000000)}, TokenOut: tokens[1-tokenInIndex]}
			gross, err := plain.CalcAmountOut(quote)
			require.NoError(t, err)
			result, err := hooked.CalcAmountOut(quote)
			require.NoError(t, err)
			// Independent arithmetic oracle; do not use the hook's fee helper.
			expected := new(big.Int).Set(gross.TokenAmountOut.Amount)
			for _, rate := range []int64{100, 133} {
				part := new(big.Int).Mul(gross.TokenAmountOut.Amount, big.NewInt(rate))
				expected.Sub(expected, part.Quo(part, big.NewInt(10000)))
			}
			require.Equal(t, expected, result.TokenAmountOut.Amount)
			require.Equal(t, DexType, hooked.GetExchange())
			second, err := hooked.CalcAmountOut(quote)
			require.NoError(t, err)
			require.Equal(t, result.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
			clone := hooked.CloneState().(*uniswapv4.PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: quote.TokenAmountIn, TokenAmountOut: *result.TokenAmountOut, SwapInfo: result.SwapInfo})
			changed, err := clone.CalcAmountOut(quote)
			require.NoError(t, err)
			require.Less(t, changed.TokenAmountOut.Amount.Cmp(result.TokenAmountOut.Amount), 0)
			unchanged, err := hooked.CalcAmountOut(quote)
			require.NoError(t, err)
			require.Equal(t, result.TokenAmountOut.Amount, unchanged.TokenAmountOut.Amount)
			// Only the core swap changes tick/liquidity state; pending hook fees
			// are separate balances and conversion happens in a later sweep.
			plain.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: quote.TokenAmountIn, TokenAmountOut: *gross.TokenAmountOut, SwapInfo: gross.SwapInfo})
			require.Equal(t, plain.V3Pool.SqrtRatioX96, clone.V3Pool.SqrtRatioX96)
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
