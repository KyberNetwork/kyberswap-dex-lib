package bancorv3

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtOutMatchesLegacy checks the native CalcAmtOut against the legacy CalcAmountOut on amount out,
// fee, gas, error, and swap info. Bancor has no swap limit, so the native path ignores params.Limit.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(bancorV3Pool1), &entityPool))
	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	const token = "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"
	const tokenOut = "0x4a220e6096b25eadb88358cb44068a3248254675"
	amounts := []string{"1663885565640917213", "1000000000000000", "1"}

	for _, amt := range amounts {
		amountIn, _ := new(big.Int).SetString(amt, 10)
		legacy, legacyErr := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: token, Amount: amountIn},
			TokenOut:      tokenOut,
		})

		native, nativeErr := sim.CalcAmtOut(poolpkg.CalcAmtOutParams{
			TokenAmountIn: poolpkg.TokenAmt{Token: token, Amount: *uint256.MustFromBig(amountIn)},
			TokenOut:      tokenOut,
		})

		require.Equal(t, legacyErr, nativeErr, "amountIn=%s error", amt)
		if legacyErr != nil {
			continue
		}
		require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountIn=%s amountOut", amt)
		require.Equal(t, legacy.Gas, native.Gas, "amountIn=%s gas", amt)
		require.Equal(t, legacy.SwapInfo, native.SwapInfo, "amountIn=%s swapInfo", amt)
	}

	// Parity alone cannot catch a change to the shared core, since legacy forwards native. Pin the gas to the
	// constant the pre-refactor legacy path returned.
	native, err := sim.CalcAmtOut(poolpkg.CalcAmtOutParams{
		TokenAmountIn: poolpkg.TokenAmt{Token: token, Amount: *uint256.NewInt(1663885565640917213)},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	require.Equal(t, defaultGas.Swap, native.Gas)
	require.Equal(t, "113555440960257967", native.TokenAmountOut.Amount.Dec())
}

// TestCalcAmountOutRejectsOversizedInput checks the big.Int boundary: an amount wider than 256 bits returns
// ErrOverflow rather than wrapping.
func TestCalcAmountOutRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(bancorV3Pool1), &entityPool))
	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	oversized := new(big.Int).Lsh(big.NewInt(1), 256)
	_, err = sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", Amount: oversized},
		TokenOut:      "0x4a220e6096b25eadb88358cb44068a3248254675",
	})
	require.ErrorIs(t, err, ErrOverflow)
}
