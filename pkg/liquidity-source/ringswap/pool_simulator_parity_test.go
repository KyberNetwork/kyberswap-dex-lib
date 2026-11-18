package ringswap

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// newParityPool is the pool from the "[swap0to1]" case in TestPoolSimulator_CalcAmountOut. Its tokenOut is an
// unwrap, so the swap reads the wrapped-token limit.
func newParityPool() PoolSimulator {
	return PoolSimulator{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
				Tokens: []string{
					"0x25f233c3e3676f9e900a89644a3fe5404d643c84",
					"0x4300000000000000000000000000000000000004",
					"0x18755d2cec785ab87680edb8e117615e4b005430",
					"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1",
				},
				Reserves: []*big.Int{bignumber.NewBig("10089138480746"), bignumber.NewBig("10066716097576"), big.NewInt(1), big.NewInt(1)},
			},
		},
		originalReserves: uniswapv2.ReserveData{
			Reserve0: bignumber.NewBig("10089138480746"),
			Reserve1: bignumber.NewBig("10066716097576"),
		},
		fee:          number.NewUint256("3"),
		feePrecision: number.NewUint256("1000"),
	}
}

// TestCalcAmtOutMatchesLegacy checks native CalcAmtOut against legacy CalcAmountOut on amount out, fee, gas, error,
// and swap info. The 1-wei limit binds on the unwrap, so both paths must reject it.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	const tokenIn = "0x25f233c3e3676f9e900a89644a3fe5404d643c84"
	const tokenOut = "0x4300000000000000000000000000000000000004"
	sim := newParityPool()

	cases := []struct {
		name    string
		limit   map[string]*big.Int
		amtIn   string
		wantErr error
	}{
		{"limit clears output", map[string]*big.Int{"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": bignumber.NewBig("10066716097576")}, "125224746", nil}, {"limit binds on unwrap", map[string]*big.Int{"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": big.NewInt(1)}, "125224746", uniswapv2.ErrInsufficientLiquidity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			amountIn := bignumber.NewBig(tc.amtIn)
			inventory := swaplimit.NewInventory("ringswap", swaplimit.ToU256Balances(tc.limit))

			legacy, legacyErr := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
				Limit:         poolpkg.AsSwapLimitBig(inventory),
			})
			native, nativeErr := sim.CalcAmtOut(poolpkg.CalcAmtOutParams{
				TokenAmountIn: poolpkg.TokenAmt{Token: tokenIn, Amount: *uint256.MustFromBig(amountIn)},
				TokenOut:      tokenOut,
				Limit:         inventory,
			})

			require.ErrorIs(t, legacyErr, tc.wantErr)
			require.ErrorIs(t, nativeErr, tc.wantErr)
			if tc.wantErr != nil {
				return
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec())
			require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec())
			require.Equal(t, legacy.Gas, native.Gas)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo)
		})
	}
}

// TestCalcAmtOutPinnedUnwrap anchors native CalcAmtOut to the amount the existing "[swap0to1]" case expects. Parity
// alone cannot catch a change to the shared core, since legacy forwards native.
func TestCalcAmtOutPinnedUnwrap(t *testing.T) {
	t.Parallel()

	sim := newParityPool()
	inventory := swaplimit.NewInventory("ringswap", swaplimit.ToU256Balances(map[string]*big.Int{
		"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": bignumber.NewBig("10066716097576"),
	}))
	native, err := sim.CalcAmtOut(poolpkg.CalcAmtOutParams{
		TokenAmountIn: poolpkg.TokenAmt{Token: "0x25f233c3e3676f9e900a89644a3fe5404d643c84", Amount: *uint256.NewInt(125224746)},
		TokenOut:      "0x4300000000000000000000000000000000000004",
		Limit:         inventory,
	})
	require.NoError(t, err)
	require.Equal(t, "124570062", native.TokenAmountOut.Amount.Dec())
	require.Equal(t, int64(defaultGas), native.Gas)
}

// TestCalcAmountOutRejectsOversizedInput checks the big.Int boundary returns ErrInvalidAmountIn, not a panic.
func TestCalcAmountOutRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	sim := newParityPool()
	oversized := new(big.Int).Lsh(big.NewInt(1), 256)
	_, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: "0x25f233c3e3676f9e900a89644a3fe5404d643c84", Amount: oversized},
		TokenOut:      "0x4300000000000000000000000000000000000004",
		Limit:         poolpkg.AsSwapLimitBig(swaplimit.NewInventory("ringswap", swaplimit.ToU256Balances(map[string]*big.Int{}))),
	})
	require.ErrorIs(t, err, uniswapv2.ErrInvalidAmountIn)
}
