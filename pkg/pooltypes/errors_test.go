package pooltypes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	altfun "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/alt-fun"
	bouncetech "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bounce-tech"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	fluiddext1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/integral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Callers (e.g. router-service trade data generator) retry a larger swap on pool.ErrSwapAmountBelowMin instead of
// marking the pool invalid, so every pool-minimum error must wrap it.
func TestErrSwapAmountBelowMin(t *testing.T) {
	t.Parallel()
	for name, err := range map[string]error{
		"alt-fun":     altfun.ErrBelowMinAmount,
		"bounce-tech": bouncetech.ErrBelowMinAmount,
		"clipper":     clipper.ErrMinAmountInNotEnough,
		"erc4626":     erc4626.ErrERC4626DepositLessThanMin,
		"fluid-dex":   fluiddext1.ErrAmountInTooSmall,
		"integral":    integral.ErrTR03,
		"lo1inch":     lo1inch.ErrAmountInBelowOrderMin,
		"order-book":  orderbook.ErrInvalidAmountIn,
	} {
		assert.ErrorIs(t, err, pool.ErrSwapAmountBelowMin, name)
	}
	// amountIn <= 0 is invalid input, not a pool minimum: retrying a larger amount is meaningless.
	assert.NotErrorIs(t, fluiddext1.ErrInvalidAmountIn, pool.ErrSwapAmountBelowMin)
	// unfilled because orders ran out: a larger swap fails too.
	assert.NotErrorIs(t, lo1inch.ErrCannotFulfillAmountIn, pool.ErrSwapAmountBelowMin)
}
