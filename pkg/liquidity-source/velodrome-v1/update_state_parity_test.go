package velodromev1

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			testutil.TestUpdateStateParity(t, func() any { return newBalanceTestSimulator(t, tp, reserves, nil) },
				[]string{"1000000", "1000000000"})
		})
	}
}
