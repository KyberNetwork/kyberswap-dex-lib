package integral

import (
	"sync"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type feeCase struct {
	volatility uint64
	want       uint16
}

// Expected values are computed with a line by line port of AdaptiveFee.getFee of the Algebra Integral base plugin
// (verified source of thena-fusion-v3 plugin 0x3E6e46Ea6E7f5D659dD1b3aB889cb8Bc199A072F on BSC).
var feeTests = []struct {
	name  string
	cfg   DynamicFeeConfig
	cases []feeCase
}{
	{name: "thena", cfg: DynamicFeeConfig{Alpha1: 500, Alpha2: 200, Beta1: 360, Beta2: 60000, Gamma1: 59, Gamma2: 8500, BaseFee: 25},
		cases: []feeCase{
			{0, 25}, {5000, 218}, {5400, 275}, {6000, 357}, {15000, 525}, {240000, 526}, {315000, 527}, {450000, 530}, {900000, 625}, {1125000, 695}, {1350000, 719}, {1650000, 724}, {2100000, 725}, {1000000000, 725}}},
	{name: "default", cfg: DynamicFeeConfig{Alpha1: 2990, Alpha2: 12000, Beta1: 360, Beta2: 60000, Gamma1: 59, Gamma2: 8500, BaseFee: 100},
		cases: []feeCase{
			{0, 100}, {5000, 1258}, {5400, 1595}, {6000, 2088}, {15000, 3090}, {240000, 3157}, {315000, 3210}, {450000, 3431}, {900000, 9090}, {1125000, 13335}, {1350000, 14748}, {1650000, 15056}, {2100000, 15090}, {1000000000, 15090}}},
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	for _, tc := range feeTests {
		for _, c := range tc.cases {
			cfg := tc.cfg
			require.Equal(t, c.want, getFee(uint256.NewInt(c.volatility), &cfg), "%s volatility=%d", tc.name, c.volatility)
		}
	}
}

// getFee must not modify its input (it is reused by both sigmoids) nor the shared CLOSEST_VALUE_* constants.
func TestGetFee_NoSharedStateMutation(t *testing.T) {
	t.Parallel()
	consts := []*uint256.Int{CLOSEST_VALUE_0, CLOSEST_VALUE_1, CLOSEST_VALUE_2, CLOSEST_VALUE_3, CLOSEST_VALUE_4,
		CLOSEST_VALUE_DEFAULT, E_HALF_MULTIPLIER, E_MULTIPLIER_BIG}
	before := make([]uint256.Int, len(consts))
	for i, c := range consts {
		before[i] = *c
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, tc := range feeTests {
				for round := range 3 { // repeated calls must give the same result
					for _, c := range tc.cases {
						cfg, vol := tc.cfg, uint256.NewInt(c.volatility)
						require.Equal(t, c.want, getFee(vol, &cfg), "round %d %s volatility=%d", round, tc.name, c.volatility)
						require.Equal(t, c.volatility, vol.Uint64(), "input modified")
					}
				}
			}
		}()
	}
	wg.Wait()
	for i, c := range consts {
		require.Equal(t, before[i], *c, "constant %d modified", i)
	}
}
