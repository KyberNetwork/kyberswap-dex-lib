package wasabiprop

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testBase  = "0x1111111111111111111111111111111111111111"
	testQuote = "0x2222222222222222222222222222222222222222"
)

func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0xpool",
		Exchange: "wasabi-prop",
		Type:     DexType,
		Tokens:   []*entity.PoolToken{{Address: testBase}, {Address: testQuote}},
		Reserves: []string{"1000000000", "1000000000"},
		Extra: `{"samples":[` +
			`[[1000,990],[10000,9800],[100000,95000]],` +
			`[[1000,1010],[10000,10100],[100000,102000]]]}`,
		StaticExtra: `{"routerAddress":"0x3333333333333333333333333333333333333333"}`,
	})
	require.NoError(t, err)
	return sim
}

// TestCalcAmtOutLimits checks the inventory limit against the output: a quote whose output exceeds the limit fails.
func TestCalcAmtOutLimits(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	cases := []struct {
		name    string
		amtIn   string
		limit   *uint256.Int
		wantErr error
	}{
		{"no-limit", "10000", nil, nil},
		{"loose", "10000", uint256.NewInt(1_000_000_000), nil},
		{"binds-out", "10000", uint256.NewInt(1), pool.ErrNotEnoughInventory},
		{"exact-limit", "10000", uint256.NewInt(9800), nil},
		{"one-below", "10000", uint256.NewInt(9799), pool.ErrNotEnoughInventory},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := uint256.MustFromDecimal(tc.amtIn)
			var swapLimit pool.SwapLimit
			if tc.limit != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{testQuote: tc.limit})
			}
			_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *in},
				TokenOut:      testQuote,
				Limit:         swapLimit,
			})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestCalcAmtOutGolden pins outputs to values worked out by hand from the fixture, so a core math change fails.
func TestCalcAmtOutGolden(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	cases := map[string]string{
		"1000":  "990",
		"5000":  "4950",
		"10000": "9800",
	}
	for amt, want := range cases {
		in, ok := new(big.Int).SetString(amt, 10)
		require.True(t, ok)
		res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *uint256.MustFromBig(in)},
			TokenOut:      testQuote,
		})
		require.NoError(t, err)
		require.Equal(t, want, res.TokenAmountOut.Amount.Dec(), "amountIn %s", amt)
	}
}

func TestCalcAmtOutLimitBinds(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	in := uint256.NewInt(1000)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{testQuote: uint256.NewInt(1)})

	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *in},
		TokenOut:      testQuote,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}

// TestCloneStateIsIPoolSim checks the clone satisfies IPoolSim, the uint256 interface the factory returns.
func TestCloneStateIsIPoolSim(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	var clone pool.IPoolSim = sim.CloneState()
	require.NotNil(t, clone)
	require.Equal(t, DexType, clone.GetType())
}

// TestCloneStateDeepCopies checks that a swap on the clone leaves the original's reserves and samples unchanged.
func TestCloneStateDeepCopies(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	origReserves := sim.GetReserves()
	origSample := new(big.Int).Set(sim.Samples[0][0][0])

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  pool.TokenAmt{Token: testBase, Amount: *uint256.NewInt(1000)},
		TokenAmountOut: pool.TokenAmt{Token: testQuote, Amount: *uint256.NewInt(990)},
	})
	clone.Samples[0][0][0].SetInt64(7)

	require.Equal(t, origReserves, sim.GetReserves(), "UpdateState on the clone changed the original reserves")
	require.Equal(t, 0, origSample.Cmp(sim.Samples[0][0][0]), "clone changed the original samples")
}
