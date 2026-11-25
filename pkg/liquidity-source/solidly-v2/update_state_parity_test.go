package solidlyv2

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	amounts := []string{"1000", "1000000", "1000000000", "1000000000000000"}
	for name, sim := range map[string]*PoolSimulator{"volatile": poolSim, "stable": stablePoolSim} {
		t.Run(name, func(t *testing.T) {
			testutil.TestUpdateStateParity(t, func() any { return sim.CloneState() }, amounts)
		})
	}
}
