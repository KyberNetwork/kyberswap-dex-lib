package two

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestNewtonY checks the solver directly against the previous implementation on real pool balances, for both
// output coins and trade sizes from 1e-9 of the reserve to the whole reserve.
func TestNewtonY(t *testing.T) {
	t.Parallel()
	var compared int
	for name, s := range allStates {
		p, err := NewPoolSimulator(s.entityPool())
		require.NoError(t, err)
		for i := range nCoins {
			for _, frac := range []int64{1e9, 1e6, 1e3, 10, 1} {
				reserve, _ := new(big.Int).SetString(s.balances[i], 10)
				dx := new(big.Int).Div(reserve, big.NewInt(frac))

				// xp exactly as get_dy builds it, in both representations
				legacyXp := []*big.Int{p.Info.Reserves[0], p.Info.Reserves[1]}
				legacyXp[i] = new(big.Int).Add(legacyXp[i], dx)
				legacyXp[0] = new(big.Int).Mul(legacyXp[0], p.Precisions[0].ToBig())
				priceScale := new(big.Int).Mul(p.PriceScale.ToBig(), p.Precisions[1].ToBig())
				legacyXp[1] = new(big.Int).Div(new(big.Int).Mul(legacyXp[1], priceScale), Precision.ToBig())
				var xp [nCoins]uint256.Int
				for k := range nCoins {
					xp[k] = *uint256.MustFromBig(legacyXp[k])
				}

				j := 1 - i
				want, wantErr := legacyNewtonY(p.A.ToBig(), p.Gamma.ToBig(), legacyXp, p.D.ToBig(), j)
				got, err := newtonY(p.A, p.Gamma, xp, p.D, j)
				require.Equal(t, wantErr == nil, err == nil, "%s i=%d dx=%s: legacy=%v new=%v", name, i, dx, wantErr, err)
				if wantErr == nil {
					require.Equal(t, want.String(), got.Dec(), "%s i=%d dx=%s", name, i, dx)
					compared++
				}
			}
		}
	}
	require.Greater(t, compared, 20)
}

// newton_D asserts MIN_A <= ANN <= MAX_A ("unsafe values A"); the previous implementation did not check A here.
func TestNewtonDRejectsUnsafeA(t *testing.T) {
	t.Parallel()
	gamma := *uint256.NewInt(145000000000000) // CVX/ETH
	x1 := *uint256.MustFromDecimal("1000000000000000000000000")
	x := [nCoins]uint256.Int{x1, x1}

	D, err := newtonD(*uint256.NewInt(400000), gamma, x)
	require.NoError(t, err)
	require.Equal(t, "2000000000000000000000000", D.Dec())

	for _, ann := range []*uint256.Int{new(uint256.Int).SubUint64(MinA, 1), new(uint256.Int).AddUint64(MaxA, 1)} {
		_, err = newtonD(*ann, gamma, x)
		require.ErrorIs(t, err, errUnsafeA, ann.Dec())
		_, err = legacyNewtonD(ann.ToBig(), gamma.ToBig(), []*big.Int{x1.ToBig(), x1.ToBig()})
		require.NoError(t, err, "the previous implementation accepted A=%s", ann.Dec())
	}
}

// newton_y asserts 10**16 * N_COINS <= K0_i <= 10**20 * N_COINS, where K0_i = 10**18 * N_COINS * x_j / D. Just below
// the lower bound the previous implementation still returned a y, while the contract reverts ("unsafe values x[i]").
func TestNewtonYRejectsUnsafeK0(t *testing.T) {
	t.Parallel()
	ann, gamma := *uint256.NewInt(400000), *uint256.NewInt(145000000000000) // CVX/ETH
	D := *uint256.MustFromDecimal("1000000000000000000000000")              // 1e24
	for _, tc := range []struct {
		xj      string
		wantErr error
	}{
		{"10000000000000000000000", nil},       // x_j = D / 100: K0_i = 2e16, the smallest allowed
		{"9999999999999999999999", errUnsafeX}, // K0_i = 2e16 - 1
		{"5000000000000000000000", errUnsafeX},
	} {
		x := [nCoins]uint256.Int{*uint256.MustFromDecimal(tc.xj), {}}
		_, err := newtonY(ann, gamma, x, D, 1)
		if tc.wantErr == nil {
			require.NoError(t, err, tc.xj)
		} else {
			require.ErrorIs(t, err, tc.wantErr, tc.xj)
		}
		_, err = legacyNewtonY(ann.ToBig(), gamma.ToBig(), []*big.Int{x[0].ToBig(), big.NewInt(0)}, D.ToBig(), 1)
		require.NoError(t, err, "the previous implementation accepted x_j=%s", tc.xj)
	}
}
