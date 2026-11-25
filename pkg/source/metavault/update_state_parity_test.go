package metavault

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	testutil.TestUpdateStateParity(t, func() any { return newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy) },
		[]string{"1000000", "1000000000", "1000000000000000000"})
}
