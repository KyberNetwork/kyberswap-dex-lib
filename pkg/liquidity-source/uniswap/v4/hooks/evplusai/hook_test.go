package evplusai

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func makeHook(t *testing.T, extra Extra) *Hook {
	t.Helper()
	data, err := json.Marshal(extra)
	require.NoError(t, err)
	hook, ok := uniswapv4.GetHook(HookAddress, &uniswapv4.HookParam{
		Cfg: &uniswapv4.Config{ChainID: 4663}, HookExtra: data,
	})
	require.True(t, ok)
	return hook.(*Hook)
}

func TestFees(t *testing.T) {
	t.Parallel()
	// Golden values from Solidity currentSwapFees, including a non-multiple of
	// ten budget and the maximum where exact-output LP rounding differs.
	for _, tc := range []struct {
		fee       uint32
		exactIn   bool
		lp, denom uint64
	}{
		{250, true, 225, 9997750}, {500, true, 450, 9995500},
		{7919, true, 7127, 9928730}, {9000, true, 8100, 9919000},
		{250, false, 225, 9999750}, {500, false, 450, 9999500},
		{7919, false, 7132, 9992081}, {9000, false, 8107, 9991000},
	} {
		lp, denominator := split(uint64(tc.fee), tc.exactIn)
		require.Equal(t, tc.lp, lp)
		require.Equal(t, tc.denom, denominator)
		for _, direction := range []bool{false, true} {
			h := makeHook(t, Extra{Fee0For1: tc.fee, Fee1For0: tc.fee, Timestamp: 100, ExpiresAt: 101})
			params := &uniswapv4.BeforeSwapParams{CalcOut: tc.exactIn, ZeroForOne: direction}
			before, err := h.BeforeSwap(params)
			require.NoError(t, err)
			require.EqualValues(t, lp, before.SwapFee)
			require.Zero(t, before.DeltaSpecified.Sign())
			require.Zero(t, before.DeltaUnspecified.Sign())
			// denominator units -> exactly numerator units; one less exercises floor.
			for _, amount := range []uint64{0, 1, denominator - 1, denominator} {
				input, output := big.NewInt(42), big.NewInt(43)
				if tc.exactIn {
					output.SetUint64(amount)
				} else {
					input.SetUint64(amount)
				}
				result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: params, AmountIn: input, AmountOut: output})
				require.NoError(t, err)
				require.EqualValues(t, amount*uint64(tc.fee)/denominator, result.HookFee.Uint64())
			}
		}
	}
}

func TestObservedSwapAndNoInputMutation(t *testing.T) {
	t.Parallel()
	// September 20, block 68251120: actual core USDG output 31.312411,
	// platform accrual 0.024974, final leg output 31.287437.
	h := makeHook(t, Extra{Fee0For1: 7919, Fee1For0: 500, Timestamp: 100, ExpiresAt: 101})
	amount := big.NewInt(31312411)
	p := &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true},
		AmountIn:         big.NewInt(12000000000000000), AmountOut: amount,
	}
	for range 2 {
		res, err := h.AfterSwap(p)
		require.NoError(t, err)
		require.Equal(t, int64(24974), res.HookFee.Int64())
		require.Equal(t, int64(31312411), amount.Int64())
	}
}

func TestDirectionalProtocolFeeAndExpiry(t *testing.T) {
	t.Parallel()
	h := makeHook(t, Extra{Fee0For1: 250, Fee1For0: 9000, Timestamp: 99, ExpiresAt: 100, ProtocolFee: 1000 | (500 << 12)})
	for _, tc := range []struct {
		sell bool
		want int64
	}{{true, 1225}, {false, 8596}} {
		p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: tc.sell}
		res, err := h.BeforeSwap(p)
		require.NoError(t, err)
		require.EqualValues(t, tc.want, res.SwapFee)
	}
	h.Timestamp = 100 // exact expiry boundary, chain timestamp from a new Track
	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true})
	require.NoError(t, err)
	require.EqualValues(t, 1450, res.SwapFee)
	h.ProtocolFee = 0
	for _, expiry := range []uint64{0, 99, 100} {
		h.ExpiresAt = expiry
		for _, side := range []bool{false, true} {
			fee, err := h.budget(side)
			require.NoError(t, err)
			require.EqualValues(t, 500, fee)
		}
	}
}

func TestInvalidStateAndAmounts(t *testing.T) {
	t.Parallel()
	for _, data := range []string{"", `{`, `{}`, `{"ts":1,"expiry":2,"f01":0,"f10":9001}`} {
		h, _ := uniswapv4.GetHook(HookAddress, &uniswapv4.HookParam{HookExtra: []byte(data)})
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true})
		require.ErrorIs(t, err, errInvalidState)
	}
	h := makeHook(t, Extra{Timestamp: 100})
	p := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}}
	for _, amount := range []*big.Int{nil, big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 127)} {
		p.AmountOut = amount
		_, err := h.AfterSwap(p)
		require.ErrorIs(t, err, errInvalidAmount)
	}
	p.CalcOut = false
	p.AmountIn = new(big.Int).Lsh(big.NewInt(1), 127)
	_, err := h.AfterSwap(p)
	require.NoError(t, err) // magnitude of int128 minimum is valid for an input
	p.AmountIn.Add(p.AmountIn, big.NewInt(1))
	_, err = h.AfterSwap(p)
	require.ErrorIs(t, err, errInvalidAmount)
	h.ProtocolFee = 1001
	_, err = h.budget(true)
	require.ErrorIs(t, err, errInvalidState)
}

func TestRegistrationAndClone(t *testing.T) {
	t.Parallel()
	h := makeHook(t, Extra{Timestamp: 100, Fee0For1: 250, Fee1For0: 9000, ExpiresAt: 101})
	require.Equal(t, valueobject.ExchangeUniswapV4EVPLUSAI, h.GetExchange())
	require.True(t, h.CanBeforeSwap(HookAddress))
	require.True(t, h.CanAfterSwap(HookAddress))
	require.Empty(t, h.GetHookData())
	copy := h.CloneState().(*Hook)
	copy.Fee0For1 = 9000
	copy.Timestamp = 101
	fee, err := h.budget(true)
	require.NoError(t, err)
	require.EqualValues(t, 250, fee)
	fee, err = copy.budget(true)
	require.NoError(t, err)
	require.EqualValues(t, 500, fee)
}
