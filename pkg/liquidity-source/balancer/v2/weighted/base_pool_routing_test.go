package weighted

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/composable-stable"
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
		_, err := s.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tc.tokenIn, Amount: *uint256.NewInt(1e6)},
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

// TestComposableStableBaseIsRouted checks a real composable-stable base pool passes the shared.IBasePool
// assertion in NewPoolSimulator and is found by getBasePool. A base that fails the assertion is dropped.
func TestComposableStableBaseIsRouted(t *testing.T) {
	t.Parallel()

	base, err := composablestable.NewPoolSimulator(entity.Pool{
		Address:  "0xbase",
		Exchange: composablestable.DexType,
		Type:     composablestable.DexType,
		Reserves: []string{"1000000000000000000", "1000000000000000000", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xdai"}, {Address: "0xusdc"}, {Address: "0xusdt"},
		},
		Extra:       `{"amp":"0x1388","swapFeePercentage":"0x2D79883D2000","scalingFactors":["1000000000000000000","1000000000000000000","1000000000000000000"]}`,
		StaticExtra: `{"poolId":"0xbase","poolType":"ComposableStable","poolTypeVersion":3,"bptIndex":-1,"vault":"0xvault"}`,
	})
	require.NoError(t, err)

	main, err := NewPoolSimulator(entity.Pool{
		Address:  "0xmain",
		Exchange: DexType,
		Type:     DexType,
		Reserves: []string{"1000", "1000"},
		Tokens:   []*entity.PoolToken{{Address: "0xbase"}, {Address: "0xweth"}},
		Extra:    `{"swapFeePercentage":"0x2386F26FC10000","totalSupply":"1000000000000000000000"}`,
		StaticExtra: `{"poolId":"0xmain","poolType":"Weighted","poolTypeVer":3,` +
			`"scalingFactors":["1000000000000000000","1000000000000000000"],` +
			`"normalizedWeights":["500000000000000000","500000000000000000"],` +
			`"vault":"0xvault","basePools":{"0xbase":["0xdai","0xusdc","0xusdt"]}}`,
	}, map[string]pool.IPoolSim{"0xbase": base})
	require.NoError(t, err)

	got, err := main.getBasePool("0xusdc", "")
	require.NoError(t, err)
	assert.Equal(t, "0xbase", got.GetAddress())
}
