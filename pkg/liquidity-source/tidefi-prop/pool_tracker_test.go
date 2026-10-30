package tidefiprop

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// TideFi's quote caps amountIn at what the engine will actually fill and
// returns that cap, so every probe beyond it comes back as an identical
// (amountIn, amountOut) pair. Keeping those repeats makes consecutive ladder
// points share the same x, which turns the spline's secant into 0/0 and
// panics the simulator, so only the first one may survive.
func TestCollectLadder_DropsCappedRepeats(t *testing.T) {
	t.Parallel()

	results := []quoteResult{
		{AmountIn: big.NewInt(100), AmountOut: big.NewInt(200)},
		{AmountIn: big.NewInt(400), AmountOut: big.NewInt(700)},
		{AmountIn: big.NewInt(546), AmountOut: big.NewInt(900)},
		{AmountIn: big.NewInt(546), AmountOut: big.NewInt(900)},
		{AmountIn: big.NewInt(546), AmountOut: big.NewInt(900)},
		// Reverted (nil) and zero-output probes are still dropped.
		{AmountIn: nil, AmountOut: big.NewInt(1)},
		{AmountIn: big.NewInt(700), AmountOut: big.NewInt(0)},
	}

	assert.Equal(t, []ladder.Point{{100, 200}, {400, 700}, {546, 900}}, collectLadder(results))
}
