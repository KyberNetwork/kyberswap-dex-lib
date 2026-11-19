package integral

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

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one on both directions, with a loose
// limit and a limit that binds on the output token.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	limits := map[string]*uint256.Int{
		"loose":     uint256.MustFromDecimal("1000000000000000000000000"),
		"binds-out": uint256.NewInt(1),
	}
	cases := []struct {
		tokenIn, tokenOut string
		amount            *big.Int
	}{
		{_token0, _token1, _amount0In},
		{_token1, _token0, _amount1In},
		{_token0, _token1, big.NewInt(1000)},
		{_token1, _token0, big.NewInt(1000000000000)},
	}

	for name, lim := range limits {
		for _, c := range cases {
			swapLimit := swaplimit.NewInventory(DexTypeIntegral, map[string]*uint256.Int{_token0: lim, _token1: lim})
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: c.tokenIn, Amount: c.amount},
				TokenOut:      c.tokenOut,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: c.tokenIn, Amount: *uint256.MustFromBig(c.amount)},
				TokenOut:      c.tokenOut,
				Limit:         swapLimit,
			})
			require.ErrorIs(t, nativeErr, legacyErr, "%s/%s error", name, c.amount)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%s amount out", name, c.amount)
			require.Equal(t, 0, legacy.Fee.Amount.Cmp(native.Fee.Amount.ToBig()), "%s/%s fee", name, c.amount)
			require.Equal(t, sim.gas.Swap, native.Gas, "%s/%s gas", name, c.amount)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo, "%s/%s swap info", name, c.amount)
		}
	}
}

func TestCalcAmtOutLimitBindsAndNilLimit(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	bind := swaplimit.NewInventory(DexTypeIntegral, map[string]*uint256.Int{_token1: uint256.NewInt(1)})
	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: _token0, Amount: *uint256.MustFromBig(_amount0In)},
		TokenOut:      _token1,
		Limit:         bind,
	})
	require.ErrorIs(t, err, ErrTR3A)

	_, err = sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: _token0, Amount: *uint256.MustFromBig(_amount0In)},
		TokenOut:      _token1,
	})
	require.ErrorIs(t, err, ErrNoSwapLimit)
}

func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(Extra{
		IsEnabled:                true,
		SwapFee:                  _swapFee,
		Price:                    _price,
		InvertedPrice:            _invertedPrice,
		Token0LimitMin:           _token0LimitMin,
		Token1LimitMin:           _token1LimitMin,
		Token0LimitMaxMultiplier: _token0LimitMaxMultiplier,
		Token1LimitMaxMultiplier: _token1LimitMaxMultiplier,
	})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{_reserve0.String(), _reserve1.String()},
		Tokens: []*entity.PoolToken{
			{Address: _token0, Decimals: 18, Swappable: true},
			{Address: _token1, Decimals: 6, Swappable: true},
		},
		Extra: string(extraBytes),
	})
	require.NoError(t, err)
	return sim
}
