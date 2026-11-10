package velodromev1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The stable quotes below are pair.getAmountOut(amountIn, tokenIn) read on-chain (anvil forks of optimism, polygon
// and base). The velodrome v1 and dystopia pairs iterate on the Solidly f(x0, y), which rounds differently from the
// (x*y)*(x*x+y*y) form of _k (and of the aero-v2 pair): a few wei apart when the reserves are far from balanced.
func TestPoolSimulator_StableQuoteMatchesPairView(t *testing.T) {
	t.Parallel()

	velodrome := balanceTestPool{"sAMM 6/18", true, [2]uint8{6, 18}, [2]string{"13308434", "1695718482117307092129253"}, 5, 10000}
	dystopia618 := balanceTestPool{"sAMM 6/18", true, [2]uint8{6, 18}, [2]string{"68061", "125499547057812022415470"}, 1, 2000}
	dystopia1818 := balanceTestPool{"sAMM 18/18", true, [2]uint8{18, 18}, [2]string{"1061691822", "1235029058321091782190"}, 1, 2000}
	aero := balanceTestPool{"sAMM 18/6", true, [2]uint8{18, 6}, [2]string{"132542362954804", "69"}, 5, 10000}

	for _, tc := range []struct {
		exchange string
		pool     balanceTestPool
		in       int
		amountIn string
		want     string
	}{
		{"velodrome", velodrome, 0, "4451", "188917058326151579110"},
		{"velodrome", velodrome, 0, "1492", "63363901422910439961"},
		{"velodrome", velodrome, 0, "61011", "2582118187720147659783"},
		{"velodrome", velodrome, 1, "6178185571255590563366811", "13175349"},
		{"dystopia", dystopia618, 0, "1", "614636467222946124"},
		{"dystopia", dystopia618, 0, "9976", "5591361204027563138933"},
		{"dystopia", dystopia1818, 0, "4", "1551291474536"},
		{"dystopia", dystopia1818, 0, "155621912", "55020710572468203889"},
		// aero-v2 keeps the form of _k
		{"aero-v2-stable", aero, 1, "1", "992021665753"},
	} {
		t.Run(tc.exchange+" "+tc.pool.name+" "+tc.amountIn, func(t *testing.T) {
			sim := newBalanceTestSimulatorOn(t, tc.exchange, tc.pool, []*big.Int{bi(tc.pool.reserves[0]), bi(tc.pool.reserves[1])}, nil)
			res, err := quote(sim, tc.in, bi(tc.amountIn))
			require.NoError(t, err)
			assert.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
		})
	}
}
