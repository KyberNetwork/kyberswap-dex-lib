package upshiftusdf

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const startOfYear = 1640995200 // 2022-01-01

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// staleExtra is the live vault at mainnet block 26204144: 84 days behind, so compute() caps at 30.
func staleExtra() Extra {
	return Extra{StartOfYear: startOfYear, CurrentPeriod: 1665, Apr: u("1100"), TokenPrice: u("1414381"),
		MinDeposit: u("100000"), MaxSupply: u("10000000000000"), TotalSupply: u("2829657973978")}
}

// freshExtra is the same vault after compute() caught up (period 1749, price 1439665).
func freshExtra() Extra {
	e := staleExtra()
	e.CurrentPeriod, e.TokenPrice = 1749, u("1439665")
	return e
}

func dayStart(period uint64) int64 { return int64(startOfYear + period*secondsPerDay) }

func newSim(t testing.TB, e Extra, at int64) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(e)
	require.NoError(t, err)
	r := Reserves(&e)
	sim, err := NewPoolSimulator(entity.Pool{
		Address: Vault, Exchange: DexType, Type: DexType,
		Reserves:    []string{r[0].String(), r[1].String()},
		Tokens:      []*entity.PoolToken{{Address: USDC, Decimals: 6, Swappable: true}, {Address: USDF, Decimals: 6, Swappable: true}},
		Extra:       string(extraBytes),
		BlockNumber: 26204144,
	})
	require.NoError(t, err)
	sim.now = func() time.Time { return time.Unix(at, 0) }
	return sim
}

func quote(sim *PoolSimulator, amount string) (*pool.CalcAmountOutResult, error) {
	a, _ := new(big.Int).SetString(amount, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDC, Amount: a}, TokenOut: USDF})
}

// Expected values are USDF minted by real Vault.deposit on an anvil fork of block 26204144
// at the given block timestamp (see NOTES.md).
func TestCalcAmountOutMatchesChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		extra  Extra
		at     int64
		amount string
		out    string
		gas    int64
	}{
		{"stale min", staleExtra(), 1792142314, "100000", "70253", 2820604},
		{"stale 1e9", staleExtra(), 1792142262, "1000000000", "702537777", 2820604},
		{"stale large", staleExtra(), 1792142262, "123456789012", "86733058134", 2820604},
		{"fresh 0d", freshExtra(), 1792142366, "1000000000", "694606036", 76404},
		{"fresh 1d", freshExtra(), 1792238401, "1000000000", "694460841", 170294},
		{"fresh 3d", freshExtra(), 1792411201, "1000000000", "694170632", 353074},
		{"fresh 10d", freshExtra(), 1793016003, "1000000000", "693156809", 992804},
		{"fresh 30d", freshExtra(), 1794744038, "1000000000", "690276421", 2820604},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sim := newSim(t, tc.extra, tc.at)
			res, err := quote(sim, tc.amount)
			require.NoError(t, err)
			assert.Equal(t, tc.out, res.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.gas, res.Gas)
		})
	}
}

func TestMinDeposit(t *testing.T) {
	sim := newSim(t, staleExtra(), 1792142262)
	_, err := quote(sim, "99999")
	assert.ErrorIs(t, err, ErrBelowMinDeposit)
}

func TestSupplyCap(t *testing.T) {
	e := staleExtra()
	// headroom 702537777 USDF
	e.TotalSupply = new(uint256.Int).Sub(e.MaxSupply, u("702537777"))
	sim := newSim(t, e, 1792142262)
	res, err := quote(sim, "1000000000")
	require.NoError(t, err)
	assert.Equal(t, "702537777", res.TokenAmountOut.Amount.String())
	_, err = quote(sim, "1000001000")
	assert.ErrorIs(t, err, ErrSupplyExceeded)

	e.MaxSupply = new(uint256.Int) // unlimited
	_, err = quote(newSim(t, e, 1792142262), "123456789012")
	assert.NoError(t, err)
}

func TestZeroShares(t *testing.T) {
	e := staleExtra()
	e.MinDeposit = u("1")
	_, err := quote(newSim(t, e, 1792142262), "1")
	assert.ErrorIs(t, err, ErrZeroShares)
}

// Near a day boundary a lagging vault would roll a different number of days, so refuse.
func TestPeriodBoundary(t *testing.T) {
	next := dayStart(1750)
	for _, tc := range []struct {
		at  int64
		err error
	}{
		{next - boundaryAfterSec - 1, nil},
		{next - boundaryAfterSec, ErrPeriodBoundary},
		{next - 1, ErrPeriodBoundary},
		{next + boundaryBeforeSec - 1, ErrPeriodBoundary},
		{next + boundaryBeforeSec, nil},
	} {
		_, err := quote(newSim(t, freshExtra(), tc.at), "1000000000")
		assert.Equal(t, tc.err, err, "at %d", tc.at-next)
	}
	// 30-day cap makes the boundary irrelevant while the vault is far behind.
	_, err := quote(newSim(t, staleExtra(), dayStart(1749)-1), "1000000000")
	assert.NoError(t, err)
}

// A second deposit in the same route rolls another 30 days, as on-chain.
func TestUpdateBalanceRollsAgain(t *testing.T) {
	sim := newSim(t, staleExtra(), 1792142262)
	res, err := quote(sim, "1000000000")
	require.NoError(t, err)
	clone := sim.CloneState().(*PoolSimulator)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	assert.EqualValues(t, 1695, sim.CurrentPeriod)
	assert.Equal(t, "1423411", sim.TokenPrice.Dec())
	assert.Equal(t, "2830360511755", sim.TotalSupply.Dec()) // 2829657973978 + 702537777
	assert.EqualValues(t, 1665, clone.CurrentPeriod, "clone is isolated")
	assert.Equal(t, "1414381", clone.TokenPrice.Dec())

	res2, err := quote(sim, "1000000000")
	require.NoError(t, err)
	assert.Equal(t, int64(2820604), res2.Gas)
	assert.Equal(t, "1432441", res2.SwapInfo.(SwapInfo).NewPrice.Dec())
	assert.Equal(t, "698109032", res2.TokenAmountOut.Amount.String()) // chain: second sequential deposit
}

func TestInvalid(t *testing.T) {
	sim := newSim(t, staleExtra(), 1792142262)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDF, Amount: big.NewInt(1e9)}, TokenOut: USDC})
	assert.ErrorIs(t, err, ErrInvalidSwapPath)
	_, err = quote(sim, "0")
	assert.ErrorIs(t, err, ErrZeroAmount)
}

func TestReserves(t *testing.T) {
	e := staleExtra()
	r := Reserves(&e)
	assert.Equal(t, "7170342026022", r[1].String())  // max - total
	assert.Equal(t, "10141595525107", r[0].String()) // *1414381/1e6
	e.MaxSupply = new(uint256.Int)
	assert.Equal(t, 1, Reserves(&e)[1].Sign())
}
