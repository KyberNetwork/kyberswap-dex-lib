package feltir

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testToken0 = "0x1111111111111111111111111111111111111111"
	testToken1 = "0x2222222222222222222222222222222222222222"
)

func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{Samples: [2][][2]*big.Int{
		{{big.NewInt(1000), big.NewInt(990)}, {big.NewInt(10000), big.NewInt(9800)}, {big.NewInt(100000), big.NewInt(95000)}},
		{{big.NewInt(1000), big.NewInt(1010)}, {big.NewInt(10000), big.NewInt(10100)}, {big.NewInt(100000), big.NewInt(102000)}},
	}})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     "0xpool",
		Exchange:    "feltir",
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		Reserves:    []string{"1000000000", "1000000000"},
		Extra:       string(extra),
		StaticExtra: `{"feltirAddress":"0x3333333333333333333333333333333333333333"}`,
	})
	require.NoError(t, err)
	return sim
}

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one over several inputs and limits.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	amounts := []int64{1, 999, 1000, 5000, 10000, 50000, 100000}
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1_000_000_000),
		"binds-out": uint256.NewInt(1),
	}

	for name, lim := range limits {
		for _, amt := range amounts {
			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{testToken1: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: big.NewInt(amt)},
				TokenOut:      testToken1,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: testToken0, Amount: *uint256.NewInt(uint64(amt))},
				TokenOut:      testToken1,
				Limit:         swapLimit,
			})
			require.Equal(t, legacyErr, nativeErr, "%s/%d error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%d amount out", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%d gas", name, amt)
		}
	}
}

// TestCalcAmtOutGolden pins outputs to values worked out from the fixture, so a core math change fails.
func TestCalcAmtOutGolden(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testToken0, Amount: *uint256.NewInt(5000)},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	require.Equal(t, "4950", res.TokenAmountOut.Amount.Dec())
}

func TestCalcAmtOutLimitBinds(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{testToken1: uint256.NewInt(1)})
	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testToken0, Amount: *uint256.NewInt(5000)},
		TokenOut:      testToken1,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}
