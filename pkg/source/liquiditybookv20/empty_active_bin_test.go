package liquiditybookv20

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestWithActiveBin(t *testing.T) {
	bin := func(id uint32, x, y int64) Bin { return Bin{ID: id, ReserveX: big.NewInt(x), ReserveY: big.NewInt(y)} }
	bins := []Bin{bin(9, 0, 5), bin(12, 5, 0)}

	got := withActiveBin(bins, 10)
	assert.Equal(t, []Bin{bin(9, 0, 5), bin(10, 0, 0), bin(12, 5, 0)}, got)
	assert.Len(t, bins, 2, "the input set is left as it was")

	assert.Equal(t, bins, withActiveBin(bins, 12), "a set with the active bin is kept")
	assert.Equal(t, []Bin{bin(7, 0, 0)}, withActiveBin(nil, 7))
}

// TestEmptyActiveBinQuotes: a pair whose active bin is empty swaps from its next non-empty bin either
// way. The simulator finds that bin by the active one's place in the set, so it quotes the pair when
// the set keeps the empty active bin, and cannot quote it at all when the bin is dropped.
func TestEmptyActiveBinQuotes(t *testing.T) {
	const active = 8388608
	bins := []Bin{
		{ID: active - 1, ReserveX: new(big.Int), ReserveY: bignumber.TenPowInt(18)},
		{ID: active + 1, ReserveX: bignumber.TenPowInt(18), ReserveY: new(big.Int)},
	}
	swap := func(sim *PoolSimulator, in string) (*pool.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(1000)},
			TokenOut:      otherToken[in],
		})
	}

	kept, dropped := newTestSim(t, withActiveBin(bins, active), active), newTestSim(t, bins, active)
	for _, in := range []string{"0xa", "0xb"} {
		_, err := swap(kept, in)
		assert.NoError(t, err, "%s in, with the empty active bin kept", in)
		_, err = swap(dropped, in)
		assert.Error(t, err, "%s in, with the empty active bin dropped, as the trackers stored it before", in)
	}

	// A swap moves the active bin to one it filled with the token in, which the simulator keeps
	// when it drops the bins the swap left empty: the pair quotes again both ways.
	res, err := swap(kept, "0xa")
	require.NoError(t, err)
	kept.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xa", Amount: big.NewInt(1000)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
	for _, in := range []string{"0xa", "0xb"} {
		_, err := swap(kept, in)
		assert.NoError(t, err, "%s in, after a swap", in)
	}
}
