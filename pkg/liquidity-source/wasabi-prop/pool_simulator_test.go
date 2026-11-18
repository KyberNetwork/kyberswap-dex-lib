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

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one over several limits.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	amounts := []string{"1", "999", "1000", "5000", "10000", "50000", "100000"}
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1_000_000_000),
		"binds-out": uint256.NewInt(1),
	}

	for name, lim := range limits {
		for _, amt := range amounts {
			in, ok := new(big.Int).SetString(amt, 10)
			require.True(t, ok)

			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{testQuote: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testBase, Amount: in},
				TokenOut:      testQuote,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: testBase, Amount: *uint256.MustFromBig(in)},
				TokenOut:      testQuote,
				Limit:         swapLimit,
			})
			require.Equal(t, legacyErr, nativeErr, "%s/%s error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%s amount out", name, amt)
			require.Equal(t, 0, legacy.Fee.Amount.Sign(), "%s/%s fee", name, amt)
			require.True(t, native.Fee.Amount.IsZero(), "%s/%s fee", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%s gas", name, amt)
		}
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
