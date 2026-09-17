package lunya

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func annFor(a uint64) *uint256.Int {
	return uint256.NewInt(a * aPrecision * 4)
}

func TestSqrtPriceAtPar(t *testing.T) {
	t.Parallel()

	for _, x := range []string{"1", "1000000000000000000", "123456789000000000000000000"} {
		price, err := sqrtPriceFromReserves(uint256.MustFromDecimal(x), uint256.MustFromDecimal(x))
		require.NoError(t, err)
		assert.Equal(t, q96.Dec(), price.Dec(), x)
	}
}

func TestGetDOfABalancedPoolIsTheSum(t *testing.T) {
	t.Parallel()

	x := uint256.MustFromDecimal("1000000000000000000000")
	for _, a := range []uint64{1, 100, 10_000} {
		d, err := getD(x, x, annFor(a))
		require.NoError(t, err)
		assert.Equal(t, new(uint256.Int).Lsh(x, 1).Dec(), d.Dec(), "A=%d", a)
	}
}

func TestGetOtherReserveInvertsGetD(t *testing.T) {
	t.Parallel()

	x := uint256.MustFromDecimal("700000000000000000000")
	y := uint256.MustFromDecimal("1300000000000000000000")
	for _, a := range []uint64{1, 100, 10_000} {
		ann := annFor(a)
		d, err := getD(x, y, ann)
		require.NoError(t, err)
		solved, err := getOtherReserve(x, &d, ann, &d)
		require.NoError(t, err)

		var diff uint256.Int
		absDiff(&diff, &solved, y)
		assert.True(t, diff.CmpUint64(1_000_000) <= 0, "A=%d: solved %s for %s", a, solved.Dec(), y.Dec())
	}
}

// newTestPool builds a pool of 6-decimal token0 and 18-decimal token1 on the same terms a STABLE pool
// holds them in: curve reserves, rates, and the slot0 price the reserves imply.
func newTestPool(t *testing.T, reserve0, reserve1 string, fee uint32, feeToken uint8) *PoolSimulator {
	t.Helper()

	rate0, rate1 := uint256.NewInt(1_000_000_000_000), uint256.NewInt(1)
	var priceScale uint256.Int
	require.NoError(t, mulDiv(&priceScale, rate1, q192, rate0))
	priceScale.Sqrt(&priceScale)

	r0, r1 := uint256.MustFromDecimal(reserve0), uint256.MustFromDecimal(reserve1)
	var curve0, curve1 uint256.Int
	curve0.Mul(r0, rate0)
	curve1.Mul(r1, rate1)
	scaled, err := sqrtPriceFromReserves(&curve0, &curve1)
	require.NoError(t, err)
	var sqrtPrice uint256.Int
	require.NoError(t, mulDiv(&sqrtPrice, &scaled, q96, &priceScale))

	extra, err := json.Marshal(Extra{
		SqrtPriceX96:      &sqrtPrice,
		Liquidity:         uint256.NewInt(1),
		CurveReserve0:     r0,
		CurveReserve1:     r1,
		Rate0:             rate0,
		Rate1:             rate1,
		PriceScaleSqrtQ96: &priceScale,
		AmplificationX100: 100 * aPrecision,
		Fee:               fee,
		FeeToken:          feeToken,
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     "0xpool",
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{reserve0, reserve1},
		Tokens:      []*entity.PoolToken{{Address: "token0"}, {Address: "token1"}},
		Extra:       string(extra),
		StaticExtra: `{"poolType":2}`,
	})
	require.NoError(t, err)
	return sim
}

func TestRoundTripCreatesNoValue(t *testing.T) {
	t.Parallel()

	for _, feeToken := range []uint8{feeTokenPaid, feeTokenToken0, feeTokenToken1} {
		for _, fee := range []uint32{0, 400} {
			sim := newTestPool(t, "1000000000", "1300000000000000000000", fee, feeToken)

			for _, amount := range []string{"1", "1000", "1000000", "250000000"} {
				start := uint256.MustFromDecimal(amount).ToBig()
				out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: "token0", Amount: start},
					TokenOut:      "token1",
				})
				if err != nil {
					assert.ErrorIs(t, err, ErrZeroAmount, "fee %d feeToken %d amount %s", fee, feeToken, amount)
					continue
				}

				clone := sim.CloneState()
				clone.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: "token0", Amount: start},
					TokenAmountOut: *out.TokenAmountOut,
					SwapInfo:       out.SwapInfo,
				})
				back, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: *out.TokenAmountOut,
					TokenOut:      "token0",
				})
				if err != nil {
					assert.ErrorIs(t, err, ErrZeroAmount)
					continue
				}
				assert.LessOrEqual(t, back.TokenAmountOut.Amount.Cmp(start), 0,
					"fee %d feeToken %d: %s came back as %s", fee, feeToken, amount, back.TokenAmountOut.Amount)
			}
		}
	}
}

func TestExactOutputDeliversWhatItQuotes(t *testing.T) {
	t.Parallel()

	for _, feeToken := range []uint8{feeTokenPaid, feeTokenToken0, feeTokenToken1} {
		sim := newTestPool(t, "1000000000", "900000000000000000000", 400, feeToken)

		for _, amountOut := range []string{"1000000000000", "5000000000000000000", "300000000000000000000"} {
			in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: "token1", Amount: uint256.MustFromDecimal(amountOut).ToBig()},
				TokenIn:        "token0",
			})
			require.NoError(t, err, "feeToken %d amountOut %s", feeToken, amountOut)

			out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: *in.TokenAmountIn,
				TokenOut:      "token1",
			})
			require.NoError(t, err)
			// paying the quoted input as an exact input reaches at least the requested output
			assert.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(uint256.MustFromDecimal(amountOut).ToBig()), 0,
				"feeToken %d: %s for %s", feeToken, out.TokenAmountOut.Amount, amountOut)
		}
	}
}

func TestAmplificationRamp(t *testing.T) {
	t.Parallel()

	sim := &PoolSimulator{
		amplificationX100: 10_000,
		ramp:              &AmplificationRamp{StartAmplification: 10_000, TargetAmplification: 20_000, StartTime: 1_000, EndTime: 3_000},
	}
	assert.EqualValues(t, 10_000, sim.amplificationAt(1_000))
	assert.EqualValues(t, 15_000, sim.amplificationAt(2_000))
	assert.EqualValues(t, 20_000, sim.amplificationAt(3_000))
	assert.EqualValues(t, 20_000, sim.amplificationAt(9_000))

	sim.ramp = &AmplificationRamp{StartAmplification: 20_000, TargetAmplification: 10_000, StartTime: 1_000, EndTime: 3_000}
	assert.EqualValues(t, 15_000, sim.amplificationAt(2_000))

	sim.ramp = nil
	assert.EqualValues(t, 10_000, sim.amplificationAt(1<<40))
}
