package bin

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func bin(id uint32, x, y uint64) Bin {
	return Bin{ID: id, ReserveX: uint256.NewInt(x), ReserveY: uint256.NewInt(y)}
}

// TestWithActiveBin: the active bin is in the set, in its place, empty when it holds nothing, and a
// set that has it is left as it is.
func TestWithActiveBin(t *testing.T) {
	bins := []Bin{bin(1, 1, 0), bin(9, 0, 1)}
	got := withActiveBin(bins, 5)
	assert.Equal(t, []uint32{1, 5, 9}, lo.Map(got, func(b Bin, _ int) uint32 { return b.ID }))
	assert.True(t, got[1].ReserveX.IsZero() && got[1].ReserveY.IsZero())
	assert.Len(t, bins, 2, "the caller's slice is not written to")
	assert.Equal(t, bins, withActiveBin(bins, 9))
	assert.Equal(t, []uint32{3}, lo.Map(withActiveBin(nil, 3), func(b Bin, _ int) uint32 { return b.ID }))
}

// TestEmptyActiveBinStillQuotes: a swap starts in the active bin; dropped from the set when empty, the
// pool cannot quote at all, while kept empty it quotes from the next bin, as the manager swaps.
func TestEmptyActiveBinStillQuotes(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &p))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))

	quote := func(bins []Bin) error {
		e := extra
		e.Bins = bins
		b, err := json.Marshal(e)
		require.NoError(t, err)
		q := p
		q.Extra = string(b)
		sim, err := NewPoolSimulator(q, valueobject.ChainID(chainID))
		require.NoError(t, err)
		out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: bignumber.NewBig10("1000000000000000")},
			TokenOut:      p.Tokens[1].Address,
		})
		if err == nil && out.TokenAmountOut.Amount.Sign() <= 0 {
			t.Fatal("quoted nothing")
		}
		return err
	}

	withoutActive := lo.Filter(extra.Bins, func(b Bin, _ int) bool { return b.ID != extra.ActiveBinID })
	require.Less(t, len(withoutActive), len(extra.Bins), "the sample's active bin holds reserves")
	assert.ErrorIs(t, quote(withoutActive), ErrBinIDNotFound)
	assert.NoError(t, quote(withActiveBin(withoutActive, extra.ActiveBinID)))
}

// TestHasState: a pool factory writes the bins as null; a tracker as a list, empty or not.
func TestHasState(t *testing.T) {
	assert.False(t, hasState(entity.Pool{}))
	assert.False(t, hasState(entity.Pool{Extra: `{"protocolFee":0,"activeBinId":8388608,"bins":null}`}))
	assert.False(t, hasState(entity.Pool{Extra: `{"activeBinId":8388608,"lensDue":5}`}))
	assert.True(t, hasState(entity.Pool{Extra: `{"activeBinId":8388608,"bins":[]}`}))
	assert.True(t, hasState(entity.Pool{Extra: `{"bins":[{"id":1,"reserveX":"1","reserveY":"0"}]}`}))
}

// TestLacksBin: the stored set lacked a bin when the active bin, read now, holds reserves it had no
// bin for and no Mint of the batch put there.
func TestLacksBin(t *testing.T) {
	stored := []Bin{bin(4, 1, 0), bin(6, 0, 1)}
	none := map[uint32]struct{}{}
	assert.True(t, lacksBin(stored, []Bin{bin(5, 1, 1)}, none, 5))
	assert.False(t, lacksBin(stored, []Bin{bin(5, 1, 1)}, map[uint32]struct{}{5: {}}, 5), "minted in the batch")
	assert.False(t, lacksBin(stored, []Bin{bin(5, 0, 0)}, none, 5), "the active bin is empty")
	assert.False(t, lacksBin(stored, []Bin{bin(4, 2, 0)}, none, 4), "the set had it")
	assert.True(t, lacksBin(append(stored, bin(5, 0, 0)), []Bin{bin(5, 1, 1)}, none, 5),
		"an empty active bin kept in the set is no bin")
}

// TestResetPoolState: a reset drops the bins and raises lensDue, so the next update reads the pool
// whole; without the lens the pool is kept.
func TestResetPoolState(t *testing.T) {
	p := entity.Pool{Extra: `{"protocolFee":7,"activeBinId":5,"bins":[{"id":5,"reserveX":"1","reserveY":"1"}],"lensDue":3}`}

	kept, ok := (&PoolTracker{config: &Config{}}).ResetPoolState(p, 10)
	assert.False(t, ok)
	assert.Equal(t, p, kept)

	reset, ok := (&PoolTracker{config: &Config{UseLens: true}}).ResetPoolState(p, 10)
	require.True(t, ok)
	assert.False(t, hasState(reset))
	assert.JSONEq(t, `{"protocolFee":7,"activeBinId":5,"lensDue":10}`, reset.Extra)
}
