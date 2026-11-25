package big256

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// SubSigned must agree with big.Int subtraction of the signed delta, including a result that crosses zero.
func TestSubSigned(t *testing.T) {
	t.Parallel()
	vals := []int64{0, 1, 5, 100, -1, -5, -100}
	for _, a := range vals {
		for _, d := range vals {
			if a < 0 { // a sign-magnitude start value: only the non-negative magnitude with neg flag
				continue
			}
			for _, startNeg := range []bool{false, true} {
				mag := uint256.NewInt(uint64(a))
				dU := FromBig(big.NewInt(d))
				if d < 0 {
					dU = new(uint256.Int).Neg(uint256.NewInt(uint64(-d)))
				}
				want := big.NewInt(a)
				if startNeg {
					want.Neg(want)
				}
				want.Sub(want, big.NewInt(d))

				neg, overflow := SubSigned(mag, startNeg, dU)
				got := mag.ToBig()
				if neg {
					got.Neg(got)
				}
				require.False(t, overflow)
				require.Equal(t, want.String(), got.String(), "%d (neg=%v) - %d", a, startNeg, d)
			}
		}
	}
	mag := new(uint256.Int).Set(UMax)
	_, overflow := SubSigned(mag, false, new(uint256.Int).Neg(uint256.NewInt(1)))
	require.True(t, overflow)
}

// AddSigned must agree with big.Int addition of the signed delta, including a result that crosses zero.
func TestAddSigned(t *testing.T) {
	t.Parallel()
	vals := []int64{0, 1, 5, 100, -1, -5, -100}
	for _, a := range []int64{0, 1, 5, 100} {
		for _, d := range vals {
			for _, startNeg := range []bool{false, true} {
				mag := uint256.NewInt(uint64(a))
				dU := new(uint256.Int).Neg(uint256.NewInt(uint64(-d)))
				if d >= 0 {
					dU = uint256.NewInt(uint64(d))
				}
				want := big.NewInt(a)
				if startNeg {
					want.Neg(want)
				}
				want.Add(want, big.NewInt(d))

				neg, overflow := AddSigned(mag, startNeg, dU)
				got := mag.ToBig()
				if neg {
					got.Neg(got)
				}
				require.False(t, overflow)
				require.Equal(t, want.String(), got.String(), "%d (neg=%v) + %d", a, startNeg, d)
			}
		}
	}
	mag := new(uint256.Int).Set(UMax)
	_, overflow := AddSigned(mag, false, uint256.NewInt(1))
	require.True(t, overflow)
}
