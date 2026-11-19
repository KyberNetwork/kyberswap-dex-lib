package lo1inch

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

// Native CalcAmtOut must match legacy CalcAmountOut on amount out, fee, gas and error, for the same limit.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()
	sim := newParitySim(t, Extra{
		TakeToken0Orders: []*Order{
			{OrderHash: "1001", MakingAmount: uint256.MustFromDecimal("100"), TakingAmount: uint256.MustFromDecimal("1000"),
				RemainingMakerAmount: uint256.MustFromDecimal("100"), MakerBalance: uint256.MustFromDecimal("100000000000000000000"),
				MakerAllowance: uint256.MustFromDecimal("100000000000000000000")},
			{OrderHash: "1002", MakingAmount: uint256.MustFromDecimal("100"), TakingAmount: uint256.MustFromDecimal("2000"),
				RemainingMakerAmount: uint256.MustFromDecimal("50"), MakerBalance: uint256.MustFromDecimal("100000000000000000000"),
				MakerAllowance: uint256.MustFromDecimal("100000000000000000000")},
		},
	})
	inv := swaplimit.NewInventory("", swaplimit.ToU256Balances(sim.CalculateLimit()))
	for _, amount := range []string{"100", "1100", "2000", "2001"} {
		amountIn := uint256.MustFromDecimal(amount)
		legacy, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: amountIn.ToBig()}, TokenOut: "B", Limit: pool.AsSwapLimitBig(inv)})
		native, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: "A", Amount: *amountIn}, TokenOut: "B", Limit: inv})
		require.Equal(t, lerr, nerr, "error amountIn=%s", amount)
		if lerr != nil {
			continue
		}
		require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountOut amountIn=%s", amount)
		require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec(), "fee amountIn=%s", amount)
		require.Equal(t, legacy.Gas, native.Gas, "gas amountIn=%s", amount)
	}
}

// A shared limit that cannot cover the maker's order must reject the quote the same way on both paths.
func TestCalcAmtOutLimitBindsLikeLegacy(t *testing.T) {
	t.Parallel()
	sim := newParitySim(t, Extra{
		TakeToken0Orders: []*Order{
			{OrderHash: "1001", MakingAmount: uint256.MustFromDecimal("100"), TakingAmount: uint256.MustFromDecimal("1000"),
				RemainingMakerAmount: uint256.MustFromDecimal("100"), MakerBalance: uint256.MustFromDecimal("100000000000000000000"),
				MakerAllowance: uint256.MustFromDecimal("100000000000000000000")},
		},
	})
	empty := swaplimit.NewInventory("", swaplimit.ToU256Balances(map[string]*big.Int{}))
	amountIn := uint256.MustFromDecimal("100")
	_, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "A", Amount: amountIn.ToBig()}, TokenOut: "B", Limit: pool.AsSwapLimitBig(empty)})
	_, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: "A", Amount: *amountIn}, TokenOut: "B", Limit: empty})
	require.Error(t, lerr)
	require.Equal(t, lerr, nerr)
}

func newParitySim(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()
	sExtra, err := json.Marshal(extra)
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"token0":"A","token1":"B"}`,
		Extra:       string(sExtra),
	})
	require.NoError(t, err)
	return p
}
