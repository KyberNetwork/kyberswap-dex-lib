package uniswaplo

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtOutMatchesLegacy checks native CalcAmtOut against legacy CalcAmountOut on amount out, fee, gas,
// swap info, and the remaining amount. The 1500-in case fills one 1000-taking order and leaves 500 unfilled.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	const (
		tokenIn  = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		tokenOut = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	)
	sim := newParitySim(t)

	for _, amt := range []string{"1500", "1000", "2000", "1"} {
		amountIn, ok := new(big.Int).SetString(amt, 10)
		require.True(t, ok)

		legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
			TokenOut:      tokenOut,
		})
		native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *uint256.MustFromBig(amountIn)},
			TokenOut:      tokenOut,
		})

		require.Equal(t, legacyErr, nativeErr, "amountIn=%s error", amt)
		if legacyErr != nil {
			continue
		}
		require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountIn=%s amountOut", amt)
		require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec(), "amountIn=%s fee", amt)
		require.Equal(t, legacy.Gas, native.Gas, "amountIn=%s gas", amt)
		require.Equal(t, legacy.SwapInfo, native.SwapInfo, "amountIn=%s swapInfo", amt)
		require.Equal(t, legacy.RemainingTokenAmountIn.Amount.String(), native.RemainingTokenAmountIn.Amount.Dec(),
			"amountIn=%s remaining", amt)
	}
}

// TestCalcAmtOutPinnedPartialFill anchors native CalcAmtOut to the values the pre-refactor legacy path returned.
// Parity alone cannot catch a change to the shared core, since legacy forwards native.
func TestCalcAmtOutPinnedPartialFill(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	native, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: *uint256.NewInt(1500)},
		TokenOut:      "0xdac17f958d2ee523a2206206994597c13d831ec7",
	})
	require.NoError(t, err)
	require.Equal(t, "100", native.TokenAmountOut.Amount.Dec())
	require.Equal(t, "500", native.RemainingTokenAmountIn.Amount.Dec())
}

// newParitySim builds a pool with one open order that takes 1000 and gives 100.
func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()

	extra := Extra{
		TakeToken0Orders: []*DutchOrder{{
			OrderHash:   "3001",
			Type:        "Dutch",
			OrderStatus: OpenOrderStatus,
			Swapper:     common.HexToAddress("0x0"),
			Input: Input{
				Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				StartAmount: uint256.NewInt(100),
				EndAmount:   uint256.NewInt(100),
			},
			Outputs: []Output{{
				Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
				StartAmount: uint256.NewInt(1000),
				EndAmount:   uint256.NewInt(1000),
				Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
			}},
		}},
	}
	sExtra, err := json.Marshal(extra)
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Tokens:      []*entity.PoolToken{{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, {Address: "0xdac17f958d2ee523a2206206994597c13d831ec7"}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
		Extra:       string(sExtra),
	})
	require.NoError(t, err)
	return sim
}
