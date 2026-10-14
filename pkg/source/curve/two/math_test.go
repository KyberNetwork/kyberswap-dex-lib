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
