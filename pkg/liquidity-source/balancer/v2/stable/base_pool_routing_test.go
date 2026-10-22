package stable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Mirrors the Ethereum weighted pool 0x08775ccb (tokens [BPT 0x79c58f70, WETH]) nested on the
// USD composable stable pool 0x79c58f70 (tokens [DAI, BPT, USDC, USDT]). Routing USDC -> BPT
// through the main pool used to emit a BPT -> BPT hop that reverts with BAL#509.
func newNestedPools(withSecondBasePool bool) *PoolSimulator {
	newPool := func(address string, tokens ...string) *PoolSimulator {
		return &PoolSimulator{Info: pool.PoolInfo{Address: address, Tokens: tokens}}
	}

	basePools := map[string]shared.IBasePool{
		"bpt": newPool("bpt", "dai", "bpt", "usdc", "usdt"),
	}
	mainTokens := []string{"bpt", "weth"}
	if withSecondBasePool {
		basePools["bpt2"] = newPool("bpt2", "bpt2", "usdc", "frax")
		mainTokens = append(mainTokens, "bpt2")
	}

	main := newPool("main", mainTokens...)
	main.basePools = basePools
	main.batchSwapEnabled = true

	return main
}

func TestBasePoolBptSwapRejected(t *testing.T) {
	t.Parallel()

	s := newNestedPools(false)

	for _, tc := range []struct{ tokenIn, tokenOut string }{
		{"usdc", "bpt"}, // swapFromBase2Main
		{"bpt", "usdc"}, // swapFromMain2Base
	} {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: big.NewInt(1e6)},
			TokenOut:      tc.tokenOut,
		})
		assert.ErrorIs(t, err, ErrBasePoolBptSwapNotAllowed, "%s -> %s", tc.tokenIn, tc.tokenOut)
	}

	assert.NotContains(t, s.CanSwapTo("usdc"), "bpt")
	assert.Contains(t, s.CanSwapTo("usdc"), "weth")
	// Every underlying token lives only in the BPT's own base pool.
	assert.ElementsMatch(t, []string{"weth"}, s.CanSwapTo("bpt"))
}

func TestBasePoolBptSwapUsesOtherBasePool(t *testing.T) {
	t.Parallel()

	s := newNestedPools(true)

	for range 20 { // basePools is a map; exclusion must not depend on iteration order
		basePool, err := s.getBasePool("usdc", "bpt")
		require.NoError(t, err)
		assert.Equal(t, "bpt2", basePool.GetAddress())
	}

	assert.Contains(t, s.CanSwapTo("usdc"), "bpt")
	assert.Contains(t, s.CanSwapTo("bpt"), "usdc")
	assert.NotContains(t, s.CanSwapTo("bpt"), "dai")
}
