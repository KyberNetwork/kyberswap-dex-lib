package hyeth

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// poolFrom builds getPoolJSON with the extra's fields replaced (key -> raw JSON value).
func poolFrom(t *testing.T, repl map[string]string) *PoolSimulator {
	t.Helper()
	var poolE entity.Pool
	require.NoError(t, json.Unmarshal([]byte(getPoolJSON), &poolE))
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(poolE.Extra), &extra))
	for k, v := range repl {
		var val any
		require.NoError(t, json.Unmarshal([]byte(v), &val))
		extra[k] = val
	}
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	poolE.Extra = string(b)
	p, err := NewPoolSimulator(poolE)
	require.NoError(t, err)
	return p
}

// 1e18 in, each way, at block 22317092's state with external units, with issue and redeem fees, and under vault
// caps: the outputs are the DebtIssuanceModule and vault math (by a math/big model of the contracts, retired once
// it pinned these), the issue's post-trade supply and shares, and 2e18 WETH in is above a 1e18 maxDeposit.
func TestCalcAmountOut_FeesAndUnits(t *testing.T) {
	t.Parallel()
	epru := `["1000000000000000","0","2500000000000000"]`
	for _, tc := range []struct {
		name          string
		repl          map[string]string
		redeem, issue string
	}{
		{"no external units", nil, "1270365070930608945", "787175295419172397"},
		{"external units", map[string]string{"epru": epru}, "1273906807273470106", "784986777910614914"},
		{"fees", map[string]string{"epru": `["1000000000000000"]`, "feeI": `"1000000000000000"`,
			"feeR": `"3000000000000000"`}, "1267562864613197854", "785762997488831301"},
	} {
		p := poolFrom(t, tc.repl)
		r, err := quote(p, testHyeth, testWeth, big.NewInt(1e18))
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.redeem, r.TokenAmountOut.Amount.String(), "%s redeem", tc.name)
		r, err = quote(p, testWeth, testHyeth, big.NewInt(1e18))
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.issue, r.TokenAmountOut.Amount.String(), "%s issue", tc.name)
	}
	// The issue mints the quantity and its fee, and takes components for both, rounded up.
	r, err := quote(poolFrom(t, map[string]string{"epru": `["1000000000000000"]`, "feeI": `"1000000000000000"`}),
		testWeth, testHyeth, big.NewInt(1e18))
	require.NoError(t, err)
	si := r.SwapInfo.(SwapInfo)
	assert.Equal(t, "810020099575571514674", si.HyethSupply.Dec(), "hyETH supply after the issue")
	assert.Equal(t, "1016895889890752776020", si.HyethBalance.Dec(), "hyETH's shares after the issue")

	capped := poolFrom(t, map[string]string{"maxDeposit": `"1000000000000000000"`})
	_, err = quote(capped, testWeth, testHyeth, big.NewInt(2e18))
	assert.Error(t, err, "above maxDeposit")
}

// A clone's swap leaves the original's quote, and concurrent quotes agree, in both directions, with external
// units.
func TestCloneStateAndConcurrentQuotes(t *testing.T) {
	t.Parallel()
	p := newPoolWithEpru(t, "1000000000000000")
	for _, d := range [][2]string{{testHyeth, testWeth}, {testWeth, testHyeth}} {
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: d[0], Amount: big.NewInt(1e18)},
			TokenOut: d[1]}
		testutil.TestCloneState(t, p, params, nil)
		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) { return p.CalcAmountOut(params) })
		require.NoError(t, err)
	}
}

// TestCalcAmountOut_RedeemBounds pins each redeem bound on its own. With the external units of epru-mix a hyETH
// takes 1.2589e18 shares, so 808e18 hyETH (below the 809.23e18 supply) needs 1017.2e18 shares, above hyETH's
// 1015.9e18; with no external units 809e18 fits both and 810e18 is above the supply.
func TestCalcAmountOut_RedeemBounds(t *testing.T) {
	t.Parallel()
	e18 := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }
	for _, tc := range []struct {
		epru []string
		amt  *big.Int
		err  error
	}{
		{nil, e18(809), nil},
		{nil, e18(810), ErrRedeemMoreThanSupply},
		{[]string{"1000000000000000", "0", "2500000000000000"}, e18(806), nil},
		{[]string{"1000000000000000", "0", "2500000000000000"}, e18(808), ErrRedeemMoreThanSupply},
	} {
		_, err := quote(newPoolWithEpru(t, tc.epru...), testHyeth, testWeth, tc.amt)
		if tc.err == nil {
			assert.NoError(t, err, "%v %s", tc.epru, tc.amt)
		} else {
			assert.ErrorIs(t, err, tc.err, "%v %s", tc.epru, tc.amt)
		}
	}
}
