package uniswapv4

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// stubHook returns fixed BeforeSwap and AfterSwap results, so each test isolates one swapping-hook term.
type stubHook struct {
	*BaseHook
	canBefore, canAfter bool
	before              *BeforeSwapResult
	beforeErr           error
	after               *AfterSwapResult
}

func (h *stubHook) CanBeforeSwap(common.Address) bool { return h.canBefore }
func (h *stubHook) CanAfterSwap(common.Address) bool  { return h.canAfter }
func (h *stubHook) BeforeSwap(*BeforeSwapParams) (*BeforeSwapResult, error) {
	return h.before, h.beforeErr
}
func (h *stubHook) AfterSwap(*AfterSwapParams) (*AfterSwapResult, error) { return h.after, nil }

// TestCalcAmtOutSwappingHookParity: for each swapping-hook term, native CalcAmtOut must match the legacy
// CalcAmountOut on amount, gas, fee and error. The golden amounts are the legacy outputs captured before
// the native path existed, so a drift in the shared body fails here too.
func TestCalcAmtOutSwappingHookParity(t *testing.T) {
	t.Parallel()
	var poolEnt entity.Pool
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))
	base, err := NewPoolSimulator(poolEnt, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	dynamic := *base
	dynamic.staticExtra.Fee = 0x800000

	z := func(v int64) *big.Int { return big.NewInt(v) }
	u := func(v int64) *uint256.Int { return uint256.NewInt(uint64(v)) }
	beforeOnly := func(s, d int64, swapFee FeeAmount, gas int64) *BeforeSwapResult {
		return &BeforeSwapResult{DeltaSpecified: u(s), DeltaUnspecified: u(d), SwapFee: swapFee, Gas: gas}
	}
	cases := []struct {
		name    string
		sim     PoolSimulator
		hook    *stubHook
		wantOut string
		wantGas int64
		wantErr error
	}{
		{"delta-specified", *base,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, before: beforeOnly(100, 0, 0, 0)},
			"1070524822913424539490499", 129869, nil},
		{"delta-unspecified", *base,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, before: beforeOnly(0, 5000, 0, 0)},
			"1070524829112273927796315", 129869, nil},
		{"swap-fee", dynamic,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, before: beforeOnly(0, 0, 5000, 0)},
			"1075374024931016850814429", 129869, nil},
		{"before-gas", *base,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, before: beforeOnly(0, 0, 0, 1234)},
			"1070524829112273927801315", 131103, nil},
		{"after-hook-fee", *base,
			&stubHook{BaseHook: &BaseHook{}, canAfter: true, after: &AfterSwapResult{HookFee: u(77), Gas: 999}},
			"1070524829112273927801238", 130868, nil},
		{"all-terms", dynamic,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, canAfter: true,
				before: beforeOnly(100, 5000, 5000, 1234), after: &AfterSwapResult{HookFee: u(77), Gas: 999}},
			"1075374018738589681063711", 132102, nil},
		{"before-error", *base,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, beforeErr: fmt.Errorf("boom")},
			"", 0, errors.New("[BeforeSwap] boom")},
		{"fee-at-max", dynamic,
			&stubHook{BaseHook: &BaseHook{}, canBefore: true, before: beforeOnly(0, 0, FeeMax, 0)},
			"", 0, ErrInvalidFee},
	}
	const usdc, rez = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0x3b50805453023a91a8bf641e279401a0b23fa6f9"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sim := tc.sim
			sim.hook = tc.hook

			want, wantErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: z(15497045801)},
				TokenOut:      rez,
			})
			got, gotErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: usdc, Amount: *uint256.NewInt(15497045801)},
				TokenOut:      rez,
			})
			if tc.wantErr != nil {
				require.Error(t, gotErr)
				require.Error(t, wantErr)
				assert.Equal(t, wantErr.Error(), gotErr.Error(), "error")
				return
			}
			require.NoError(t, wantErr)
			require.NoError(t, gotErr)
			assert.Equal(t, tc.wantOut, got.TokenAmountOut.Amount.Dec(), "golden amountOut")
			assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.Dec(), "amountOut")
			assert.Equal(t, tc.wantGas, got.Gas, "golden gas")
			assert.Equal(t, want.Gas, got.Gas, "gas")
			assert.Equal(t, want.Fee.Token, got.Fee.Token, "fee token")
		})
	}
}
