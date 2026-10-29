package liquiditybookv21

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// swapInFixture is one LBRouter exact-out swap, recorded from the chain: the pair's state at the
// block before, the swap's time, what the swap asked for, took in and gave out, and the pair's bins
// after it. The swap is the pair's only transaction in its block.
type swapInFixture struct {
	Chain  string      `json:"chain"`
	Block  uint64      `json:"block"`
	Tx     string      `json:"tx"`
	Router string      `json:"router"`
	Method string      `json:"method"`
	Pool   entity.Pool `json:"pool"`
	Swap   struct {
		SwapForY      bool   `json:"swapForY"`
		AmountOut     string `json:"amountOut"`     // what the swap asked for
		AmountIn      string `json:"amountIn"`      // what the pair took in, its protocol fee included: getSwapIn of AmountOut
		AmountOutPaid string `json:"amountOutPaid"` // what the pair gave out for AmountIn
	} `json:"swap"`
	After struct {
		ActiveBinID uint32 `json:"activeBinId"`
		Bins        []Bin  `json:"bins"`
	} `json:"after"`
}

// binString is a bin as the test compares it: its ID and reserves.
func binString(id uint32, x, y fmt.Stringer) string {
	return fmt.Sprintf("%d %s %s", id, x, y)
}

// TestSwapInUpdateMatchesChain: each fixture is a swap through LBRouter's swapTokensForExactTokens.
// The router quotes the amount in with the pair's getSwapIn, and the pair swaps that amount in, which
// can give out a few wei more than asked, in the bin the swap ends in. The simulator, quoting the
// amount out and applying the swap, must take the same amount in and leave every bin as the pair
// did, the last one holding that difference more. The swaps cross 2 and 31 bins. A fixture keeps
// the bins the swap crossed and the five non-empty bins on each side of them, before and after.
func TestSwapInUpdateMatchesChain(t *testing.T) {
	for name, fixture := range loadFixtures[swapInFixture](t, "testdata/swapin/*.json") {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)

			in, out := fixture.Pool.Tokens[0].Address, fixture.Pool.Tokens[1].Address
			if !fixture.Swap.SwapForY {
				in, out = out, in
			}
			amountOut, _ := new(big.Int).SetString(fixture.Swap.AmountOut, 10)
			res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: amountOut},
				TokenIn:        in,
			})
			require.NoError(t, err)
			assert.Equal(t, fixture.Swap.AmountIn, res.TokenAmountIn.Amount.String(), "the amount in")

			sim.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  *res.TokenAmountIn,
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: amountOut},
				SwapInfo:       res.SwapInfo,
			})
			assert.Equal(t, fixture.After.ActiveBinID, sim.activeBinID, "the active bin")

			paid, _ := new(big.Int).SetString(fixture.Swap.AmountOutPaid, 10)
			more := new(big.Int).Sub(paid, amountOut)
			want := make([]string, len(fixture.After.Bins))
			for i, b := range fixture.After.Bins {
				x, y := new(big.Int).Set(b.ReserveX), new(big.Int).Set(b.ReserveY)
				if b.ID == fixture.After.ActiveBinID {
					if fixture.Swap.SwapForY {
						y.Add(y, more)
					} else {
						x.Add(x, more)
					}
				}
				want[i] = binString(b.ID, x, y)
			}
			got := make([]string, len(sim.bins))
			for i, b := range sim.bins {
				got[i] = binString(b.ID, b.ReserveX, b.ReserveY)
			}
			assert.Equal(t, want, got, "the bins after the swap")
		})
	}
}
