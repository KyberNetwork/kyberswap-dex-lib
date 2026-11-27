package sellonly

import (
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestSellOnlyAcceptsOnlyItsDirection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		sellsZero bool
		hook      func() (uniswapv4.Hook, bool)
	}{
		{"SellOnly", true, func() (uniswapv4.Hook, bool) { return uniswapv4.GetHook(SellOnly, nil) }},
		{"SellOnlyOne", false, func() (uniswapv4.Hook, bool) { return uniswapv4.GetHook(SellOnlyOne, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			hook, ok := tc.hook()
			require.True(t, ok)
			require.IsType(t, &Hook{}, hook)
			require.Equal(t, valueobject.ExchangeUniswapV4SellOnly, hook.GetExchange())

			allowed, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: tc.sellsZero})
			require.NoError(t, err)
			require.Zero(t, allowed.DeltaSpecified.Sign())
			require.Zero(t, allowed.DeltaUnspecified.Sign())
			require.Zero(t, allowed.SwapFee)

			_, err = hook.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: !tc.sellsZero})
			require.ErrorIs(t, err, ErrWrongDirection)
		})
	}
}
