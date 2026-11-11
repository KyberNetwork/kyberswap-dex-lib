package velodromev1

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

// CalcAmountIn has to return an amount that CalcAmountOut turns into at least the amount asked for. For a stable pair
// the inverse works on 1e18-scaled amounts, so it has to round up when it scales back to the decimals of the tokens
// (it used to round down, and the swap then failed K by one unit of the token with fewer decimals).
func TestPoolSimulator_CalcAmountInCoversAmountOut(t *testing.T) {
	t.Parallel()

	pools := []balanceTestPool{
		{"stable 6/18", true, [2]uint8{6, 18}, [2]string{"867079721327", "1218088927819202267755233"}, 5, 10000},
		{"stable 18/6", true, [2]uint8{18, 6}, [2]string{"374293020495613823825888", "133641919003"}, 5, 10000},
		{"stable 18/18", true, [2]uint8{18, 18}, [2]string{"82011881376752787015", "196568227299229868416"}, 1, 2000},
		{"stable 6/6", true, [2]uint8{6, 6}, [2]string{"937952881", "1092755397"}, 1, 2000},
		{"volatile 18/6", false, [2]uint8{18, 6}, [2]string{"4404229270686050804175", "387813975"}, 1, 2000},
	}
	rnd := rand.New(rand.NewSource(1))
	for _, tp := range pools {
		for _, exchange := range []string{"aero-v2-stable", "velodrome", "dystopia"} {
			reserves := [2]*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			for _, balances := range [][]*big.Int{nil, {reserves[0], reserves[1]}} {
				sim := newBalanceTestSimulatorOn(t, exchange, tp, reserves[:], balances)
				for in := 0; in < 2; in++ {
					for range 200 {
						// from 1e-9 to ~0.9 of the reserve out, plus the unit of the token
						frac := math.Pow(10, -9*rnd.Float64())
						want := new(big.Int).Div(new(big.Int).Mul(reserves[1-in], big.NewInt(int64(frac*9e8))), big.NewInt(1e9))
						if want.Sign() == 0 {
							want.SetInt64(1)
						}
						inRes, err := sim.CalcAmountIn(poolpkg.CalcAmountInParams{
							TokenAmountOut: poolpkg.TokenAmount{Token: tokenOf(1 - in), Amount: want}, TokenIn: tokenOf(in)})
						if err != nil {
							continue // no amount in to give (too small a want, a reserve left of 0, ...)
						}
						out, err := quote(sim, in, inRes.TokenAmountIn.Amount)
						if errors.Is(err, ErrK) {
							// the pair's own getAmountOut can quote an amount its K check refuses (a pair far from
							// balanced, see the 13175349 case above): the swap reverts on-chain too
							continue
						}
						if assert.NoError(t, err, "%s %s in=%d balances=%v want=%s amountIn=%s", exchange, tp.name, in, balances != nil, want, inRes.TokenAmountIn.Amount) {
							assert.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(want), 0, "%s %s in=%d want=%s amountIn=%s out=%s",
								exchange, tp.name, in, want, inRes.TokenAmountIn.Amount, out.TokenAmountOut.Amount)
						}
					}
				}
			}
		}
	}
}

// Found on a base fork (aero-v2 stable pair 0xA5e1, balance of token0 one wei over its reserve): 0.5045 of the 18
// decimals token out needs 500001 of the 6 decimals token in; the inverse alone gave 500000, which fails K.
func TestPoolSimulator_CalcAmountInRoundsUpToTheTokenUnit(t *testing.T) {
	t.Parallel()

	tp := balanceTestPool{"stable 6/18", true, [2]uint8{6, 18}, [2]string{"867079721327", "1218088927819202267755233"}, 5, 10000}
	r0, r1 := bi(tp.reserves[0]), bi(tp.reserves[1])
	for _, exchange := range []string{"aero-v2-stable", "velodrome"} {
		sim := newBalanceTestSimulatorOn(t, exchange, tp, []*big.Int{r0, r1}, []*big.Int{new(big.Int).Add(r0, big.NewInt(1)), r1})
		want := bi("504541079585834501")
		res, err := sim.CalcAmountIn(poolpkg.CalcAmountInParams{TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: want}, TokenIn: tokenA})
		require.NoError(t, err, exchange)
		// the pair already credits one wei: the router sends what is left of the 500001
		assert.Equal(t, "500000", res.TokenAmountIn.Amount.String(), exchange)
		out, err := quote(sim, 0, res.TokenAmountIn.Amount)
		require.NoError(t, err, exchange)
		assert.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(want), 0, exchange)
	}
}
