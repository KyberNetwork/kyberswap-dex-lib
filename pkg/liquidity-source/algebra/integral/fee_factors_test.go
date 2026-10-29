package integral

import (
	"math/big"
	"testing"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculateFeeFactorsFollowsThePlugin checks the fee factors after a price move against the
// plugin's int256 arithmetic, for a fall as well as a rise in price.
func TestCalculateFeeFactorsFollowsThePlugin(t *testing.T) {
	const priceChangeFactor = 1000
	one := new(big.Int).Lsh(big.NewInt(1), 96)
	two := new(big.Int).Lsh(one, 1)
	zto := new(big.Int).Div(new(big.Int).Mul(one, big.NewInt(3)), big.NewInt(10)) // 0.3
	otz := new(big.Int).Sub(two, zto)                                             // 1.7

	for _, delta := range []int32{-887272, -5000, -8, -3, -1, 1, 3, 15, 5000, 887272} {
		// the plugin's _calculateFeeFactors, in signed big integers
		var sqrt v3Utils.Uint160
		require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(int(delta), &sqrt))
		s := sqrt.ToBig()
		ratio := new(big.Int).Sub(new(big.Int).Div(new(big.Int).Mul(s, s), one), one)
		impact := new(big.Int).Quo(new(big.Int).Mul(ratio, big.NewInt(priceChangeFactor)), big.NewInt(1000))
		wantZto, wantOtz := new(big.Int).Sub(zto, impact), new(big.Int).Add(otz, impact)
		switch {
		case wantZto.Sign() <= 0:
			wantZto, wantOtz = big.NewInt(0), two
		case wantZto.Cmp(two) >= 0:
			wantZto, wantOtz = two, big.NewInt(0)
		}

		factors := &SlidingFeeConfig{
			ZeroToOneFeeFactor: uint256.MustFromBig(zto),
			OneToZeroFeeFactor: uint256.MustFromBig(otz),
			PriceChangeFactor:  priceChangeFactor,
		}
		got, err := calculateFeeFactors(factors, delta, 0)
		require.NoError(t, err)
		assert.Equal(t, zto.String(), factors.ZeroToOneFeeFactor.Dec(), "delta %d: factors changed", delta)
		assert.Equal(t, otz.String(), factors.OneToZeroFeeFactor.Dec(), "delta %d: factors changed", delta)
		assert.Equal(t, wantZto.String(), got.ZeroToOneFeeFactor.Dec(), "delta %d", delta)
		assert.Equal(t, wantOtz.String(), got.OneToZeroFeeFactor.Dec(), "delta %d", delta)
	}
}
