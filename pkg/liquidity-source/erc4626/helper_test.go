package erc4626

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Both swap directions share one curve through the prefetched samples, so CalcAmountIn (isExactOut) must be the
// min inverse of CalcAmountOut; picking a rate per direction made the two disagree.
func TestGetClosestRate(t *testing.T) {
	t.Parallel()
	rates := []*uint256.Int{
		uint256.NewInt(945_745),
		uint256.NewInt(945_745_537_757),
		uint256.MustFromDecimal("945745537757005980"),
		nil, // failed prefetch is skipped
		uint256.MustFromDecimal("945745537757005980110202556790"),
	}

	t.Run("samples are exact", func(t *testing.T) {
		for i, rate := range rates {
			if rate != nil {
				got, err := GetClosestRate(rates, PrefetchAmounts[i], false)
				require.NoError(t, err)
				assert.Equal(t, rate, got)
			}
		}
	})

	t.Run("linear between samples, last rate beyond", func(t *testing.T) {
		// 945745 + floor((1e9 - 1e6) * (945745537757 - 945745) / (1e12 - 1e6))
		got, err := GetClosestRate(rates, u256.TenPow(9), false)
		require.NoError(t, err)
		assert.Equal(t, "945745537", got.Dec())

		// 1e31 * 945745537757005980110202556790 / 1e30
		got, err = GetClosestRate(rates, u256.TenPow(31), false)
		require.NoError(t, err)
		assert.Equal(t, "9457455377570059801102025567900", got.Dec())
	})

	rng := rand.New(rand.NewPCG(1, 2))
	for name, rates := range map[string][]*uint256.Int{
		"linear rates":     rates,
		"bad first sample": append([]*uint256.Int{uint256.NewInt(5)}, rates[1:]...),
		"decreasing rates": {uint256.NewInt(2_000_000), uint256.NewInt(1_000_000), u256.TenPow(18)},
	} {
		t.Run(name+": exact out is the min inverse", func(t *testing.T) {
			for range 2000 {
				var amountOut uint256.Int
				amountOut.SetUint64(1 + rng.Uint64N(1e18))
				amountOut.Mul(&amountOut, u256.TenPow(rng.IntN(14)))

				amountIn, err := GetClosestRate(rates, &amountOut, true)
				require.NoError(t, err)
				got, err := GetClosestRate(rates, amountIn, false)
				require.NoError(t, err)
				require.False(t, got.Lt(&amountOut), "amountIn %s gives %s < %s", amountIn, got, &amountOut)

				got, err = GetClosestRate(rates, new(uint256.Int).SubUint64(amountIn, 1), false)
				require.NoError(t, err)
				require.True(t, got.Lt(&amountOut), "amountIn %s is not minimal for %s", amountIn, &amountOut)
			}
		})
	}

	_, err := GetClosestRate([]*uint256.Int{nil, nil}, u256.TenPow(18), false)
	assert.ErrorIs(t, err, ErrInvalidRate)
}
