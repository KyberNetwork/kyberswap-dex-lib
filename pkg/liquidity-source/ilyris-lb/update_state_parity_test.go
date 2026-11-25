package ilyrislb

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	testutil.TestUpdateStateParity(t, func() any { return newTestSim(t) },
		[]string{"1000000000000000", "100000000"})
}
