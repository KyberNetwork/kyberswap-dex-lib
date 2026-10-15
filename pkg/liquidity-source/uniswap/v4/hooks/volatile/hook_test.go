package volatile

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func TestBeforeSwap_RequiresTracked(t *testing.T) {
	h := &Hook{}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: true, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.ErrorIs(t, err, ErrPoolIsNotTracked)
}

func TestBeforeSwap_UsesDirectionalFee(t *testing.T) {
	h := &Hook{Extra: Extra{Tracked: true, FeeZeroForOne: 900, FeeOneForZero: 1200}}

	res0, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: true, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.NoError(t, err)
	require.Equal(t, uniswapv4.FeeAmount(900), res0.SwapFee)
	require.Equal(t, HookGas, res0.Gas)
	require.Zero(t, res0.DeltaSpecified.Sign())
	require.Zero(t, res0.DeltaUnspecified.Sign())

	res1, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: false, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.NoError(t, err)
	require.Equal(t, uniswapv4.FeeAmount(1200), res1.SwapFee)
	require.Zero(t, res1.DeltaSpecified.Sign())
	require.Zero(t, res1.DeltaUnspecified.Sign())
}
