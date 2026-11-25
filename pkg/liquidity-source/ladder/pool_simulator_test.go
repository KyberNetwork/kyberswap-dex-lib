package ladder

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testToken0 = "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec"
	testToken1 = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
)

func newTestPool(t *testing.T, extraJSON string) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{
		"address":"0xpool",
		"exchange":"ladder-test",
		"type":"ladder-test",
		"reserves":["10000000000000000000000","10000000000"],
		"tokens":[
			{"address":"`+testToken0+`","decimals":18,"swappable":true},
			{"address":"`+testToken1+`","decimals":6,"swappable":true}
		],
		"extra":`+extraJSON+`,
		"blockNumber":1000
	}`), &ep))
	sim := lo.Must(NewPoolSimulator(ep))
	sim.Gas = 100_000
	return sim
}

func TestNewPoolSimulatorWith_Staleness(t *testing.T) {
	t.Parallel()
	ep := entity.Pool{
		Address:   "0xpool",
		Tokens:    []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		Reserves:  entity.PoolReserves{"1", "1"},
		Extra:     "{}",
		Timestamp: time.Now().Add(-time.Minute).Unix(),
	}

	_, err := NewPoolSimulatorWith(ep, time.Second)
	assert.ErrorIs(t, err, ErrStale)

	sim, err := NewPoolSimulatorWith(ep, time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, sim)
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, `"{\"l\":[[[1000000000000,1000000],[2000000000000,1900000]],[[1000000,1000000000000]]]}"`)

	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1000000000000")},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	assert.Equal(t, "1000000", out.TokenAmountOut.Amount.String())
	assert.Equal(t, int64(100_000), out.Gas)
}

func TestPoolSimulator_EmptyExtraMeansInactive(t *testing.T) {
	t.Parallel()
	// No "l" key at all -- a paused pool (or one with no tracker data yet) is
	// represented by empty ladders in both directions, not a separate flag.
	sim := newTestPool(t, `"{}"`)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1")},
		TokenOut:      testToken1,
	})
	assert.ErrorIs(t, err, ErrNoQuote)
}

func TestPoolSimulator_EmptyLadderOneDirectionOnly(t *testing.T) {
	t.Parallel()
	// token0 -> token1 has data; token1 -> token0 doesn't (empty ladder) and
	// must be independently rejected without affecting the other direction.
	sim := newTestPool(t, `"{\"l\":[[[100,200]],[]]}"`)

	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("100")},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	assert.Equal(t, "200", out.TokenAmountOut.Amount.String())

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken1, Amount: bignumber.NewBig10("1")},
		TokenOut:      testToken0,
	})
	assert.ErrorIs(t, err, ErrNoQuote)
}

func TestPoolSimulator_UpdateBalanceAndCloneState(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, `"{\"l\":[[[1000000000000,1000000],[2000000000000,1900000]],[]]}"`)

	clone := sim.CloneState().(*PoolSimulator)

	out, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1000000000000")},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1000000000000")},
		TokenAmountOut: *out.TokenAmountOut,
	})

	// Original must be untouched by the clone's UpdateBalance.
	origOut, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1000000000000")},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	assert.Equal(t, "1000000", origOut.TokenAmountOut.Amount.String())

	// Clone's second quote reflects the curve position already consumed.
	cloneOut2, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: bignumber.NewBig10("1000000000000")},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	assert.Equal(t, "900000", cloneOut2.TokenAmountOut.Amount.String())
}

// TestPoolSimulator_CalcAmtOutParity pins the native uint256 quote to the legacy big.Int quote. Both run the same
// quote helper, so a divergence in amount out, fee, gas, or error means one path changed on its own.
func TestPoolSimulator_CalcAmtOutParity(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, `"{\"l\":[[[1000000000000,1000000],[2000000000000,1900000]],[[1000000,1000000000000]]]}"`)

	cases := []struct {
		name     string
		tokenIn  string
		amountIn string
		limit    pool.SwapLimitBig
	}{
		{"direct", testToken0, "1000000000000", nil},
		{"small", testToken0, "100", nil},
		{"reverse", testToken1, "1000000", nil},
		{"over ladder", testToken0, "3000000000000", nil},
		{"zero in", testToken0, "0", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: bignumber.NewBig10(tc.amountIn)},
				TokenOut:      otherToken(tc.tokenIn),
				Limit:         tc.limit,
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tc.tokenIn, Amount: *uint256.MustFromDecimal(tc.amountIn)},
				TokenOut:      otherToken(tc.tokenIn),
			})
			require.ErrorIs(t, nativeErr, legacyErr)
			if legacyErr != nil {
				return
			}
			assert.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.ToBig().String(), "amount out")
			assert.Equal(t, legacy.Gas, native.Gas, "gas")
		})
	}
}

// TestU256ToFloat_MatchesBigFloat pins the native uint256 -> float64 step to big.Int.Float64, which the legacy
// path uses. Amounts above 2^64 must round to nearest, not truncate, or native and legacy quotes diverge.
func TestU256ToFloat_MatchesBigFloat(t *testing.T) {
	t.Parallel()
	check := func(v *big.Int) {
		t.Helper()
		var u uint256.Int
		require.False(t, u.SetFromBig(v))
		want, _ := v.Float64()
		assert.Equal(t, want, u256ToFloat(&u), v.String())
	}
	check(bignumber.NewBig10("1000000000000000000000000000000"))
	check(new(big.Int).Lsh(big.NewInt(1), 255))
	rng := rand.New(rand.NewSource(1))
	for range 10000 {
		var u uint256.Int
		u[0], u[1], u[2], u[3] = rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()
		check(u.ToBig())
	}
}

func otherToken(token string) string {
	if token == testToken0 {
		return testToken1
	}
	return testToken0
}
