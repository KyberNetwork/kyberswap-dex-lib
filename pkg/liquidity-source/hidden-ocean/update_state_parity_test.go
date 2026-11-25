package hiddenocean

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	testutil.TestUpdateStateParity(t, func() any {
		return newTestPool(t, "79228162514264337593543950336", "1000000000000000000",
			"74505858973476688699204988068", "84377791587915437410404619782", 3000)
	}, []string{"1000000", "100000000000000"})
}
