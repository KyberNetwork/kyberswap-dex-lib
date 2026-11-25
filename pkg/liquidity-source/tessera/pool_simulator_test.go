package tessera

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testBase  = "0x1111111111111111111111111111111111111111"
	testQuote = "0x2222222222222222222222222222222222222222"
)

func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()
	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  "0xpool",
			Exchange: "tessera",
			Type:     DexType,
			Tokens:   []string{testBase, testQuote},
			Reserves: []*big.Int{big.NewInt(1_000_000_000), big.NewInt(1_000_000_000)},
		},
		extra: Extra{
			BaseToQuotePrefetches: []PrefetchRate{
				{AmountIn: uint256.NewInt(1000), Rate: uint256.NewInt(990)},
				{AmountIn: uint256.NewInt(10000), Rate: uint256.NewInt(9800)},
				{AmountIn: uint256.NewInt(100000), Rate: uint256.NewInt(95000)},
			},
			QuoteToBasePrefetches: []PrefetchRate{
				{AmountIn: uint256.NewInt(1000), Rate: uint256.NewInt(1010)},
				{AmountIn: uint256.NewInt(10000), Rate: uint256.NewInt(10100)},
				{AmountIn: uint256.NewInt(100000), Rate: uint256.NewInt(102000)},
			},
			TradingEnabled: true,
			IsInitialised:  true,
		},
	}
}

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one over several inputs and limits.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	amounts := []int64{1, 999, 1000, 5000, 10000, 50000, 100000, 200000}
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1_000_000_000),
		"binds-out": uint256.NewInt(1),
	}

	for name, lim := range limits {
		for _, amt := range amounts {
			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{testQuote: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testBase, Amount: big.NewInt(amt)},
				TokenOut:      testQuote,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *uint256.NewInt(uint64(amt))},
				TokenOut:      testQuote,
				Limit:         swapLimit,
			})
			require.Equal(t, legacyErr, nativeErr, "%s/%d error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, legacy.TokenAmountOut.Token, native.TokenAmountOut.Token, "%s/%d token out", name, amt)
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%d amount out", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%d gas", name, amt)
		}
	}
}

// TestCalcAmtOutGolden pins an exact prefetch point to its rate, so a core math change fails.
func TestCalcAmtOutGolden(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *uint256.NewInt(10000)},
		TokenOut:      testQuote,
	})
	require.NoError(t, err)
	require.Equal(t, "9800", res.TokenAmountOut.Amount.Dec())
}

func TestCalcAmtOutLimitBinds(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{testQuote: uint256.NewInt(1)})
	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *uint256.NewInt(10000)},
		TokenOut:      testQuote,
		Limit:         limit,
	})
	require.ErrorIs(t, err, ErrSwapReverted)
}
