package alphix

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getLiquidityForAmounts against v3-periphery's LiquidityAmounts test vectors (test/LiquidityAmounts.spec.ts): the
// range [encodePriceSqrt(100, 110), encodePriceSqrt(110, 100)] with 100 token0 and 200 token1 holds 2148 liquidity at
// price 1, 1048 below the range (token0 only) and 2097 above it (token1 only).
func TestGetLiquidityForAmounts_PeripheryVectors(t *testing.T) {
	t.Parallel()
	u := uint256.MustFromDecimal
	lower, upper := u("75541088972021052632782079082"), u("83095197869223157896060286990")
	for _, tc := range []struct {
		name, price, want string
	}{
		{"inside", "79228162514264337593543950336", "2148"},
		{"below", "75162434512514379355924140470", "1048"},
		{"above", "83472048772503575395058907992", "2097"},
		{"at the lower edge", "75541088972021052632782079082", "1048"},
		{"at the upper edge", "83095197869223157896060286990", "2097"},
	} {
		got := getLiquidityForAmounts(u(tc.price), lower, upper, uint256.NewInt(100), uint256.NewInt(200))
		assert.Equal(t, tc.want, got.Dec(), tc.name)
	}
	assert.True(t, getLiquidityForAmounts(q96(), upper, lower, uint256.NewInt(100), uint256.NewInt(200)).IsZero(),
		"an inverted range holds none")
}

func jitPrices() map[string]*uint256.Int {
	return map[string]*uint256.Int{
		"tick0":   q96(),
		"low":     new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(9997)), uint256.NewInt(10000)),
		"high":    new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(10003)), uint256.NewInt(10000)),
		"outside": new(uint256.Int).Mul(q96(), uint256.NewInt(2)),
		"below":   new(uint256.Int).Div(q96(), uint256.NewInt(2)),
	}
}

// TestInvariant_JitCloneMonotonicBounded: UpdateBalance on a clone moves the clone's price and not the
// original's; exact-in output is non-decreasing in the input and never above the yield source's amount of the
// output token.
func TestInvariant_JitCloneMonotonicBounded(t *testing.T) {
	t.Parallel()
	for name, price := range jitPrices() {
		h := jitHook(price)
		before := *h
		for _, zfo := range []bool{true, false} {
			res, err := beforeSwap(h, zfo, true, big.NewInt(1_000_000))
			require.NoError(t, err, name)
			clone := h.CloneState()
			clone.UpdateBalance(res.SwapInfo)
			assert.Equal(t, before, *h, "%s: UpdateBalance on the clone moved the original", name)

			prev := new(big.Int)
			for _, a := range []int64{1, 10, 1_000, 99_999, 1_000_000, 100_000_000, 999_999_999} {
				r, err := beforeSwap(h, zfo, true, big.NewInt(a))
				require.NoError(t, err)
				out := new(big.Int).Neg(r.DeltaUnspecified)
				assert.GreaterOrEqual(t, out.Cmp(prev), 0, "%s zfo=%v: not monotonic at %d", name, zfo, a)
				avail := h.Amount1Available
				if !zfo {
					avail = h.Amount0Available
				}
				assert.LessOrEqual(t, out.Cmp(avail.ToBig()), 0, "%s: out above the yield source", name)
				prev = out
			}
		}
	}
}

