package bin

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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
		out, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: p.Tokens[0].Address, Amount: bignumber.NewBig10("1000000000000000")},
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

	reset, ok := (&PoolTracker{config: &Config{
		LensConfig: ephemeral.LensConfig{UseLens: true}}}).ResetPoolState(p, 10)
	require.True(t, ok)
	assert.False(t, hasState(reset))
	assert.JSONEq(t, `{"protocolFee":7,"activeBinId":5,"lensDue":10}`, reset.Extra)
}

// binChain is a node at block 100 holding one bin pool whose active bin is activeID and holds
// reserves: the lens reads that bin, and a multicall reads the pool's slot0.
func binChain(t *testing.T, activeID uint32) *testutil.NodeStub {
	id := new(big.Int).SetUint64(uint64(activeID))
	return testutil.NewNodeStub(t, 56, 100, func(c testutil.NodeCall) ([]byte, error) {
		if c.To == nil {
			bins := []struct{ Id, ReserveX, ReserveY *big.Int }{{id, big.NewInt(5), big.NewInt(7)}}
			out, err := lens.PackResult(id, big.NewInt(100), big.NewInt(0), true, bins)
			require.NoError(t, err)
			return nil, testutil.Revert(out)
		}
		ret, _, err := testutil.Multicall(*c.To, c.Data, 100, 0, func(common.Address, []byte) (bool, []byte) {
			return true, append(common.LeftPadBytes(id.Bytes(), 32), make([]byte, 64)...)
		})
		return ret, err
	})
}

// TestLensBootstrapIsTimestamped: a pool given its state from the lens at the batch's head is
// stamped with the time it was read, as every other read of it is.
func TestLensBootstrapIsTimestamped(t *testing.T) {
	const activeID = 8388608
	tr := NewPoolTracker(&Config{ChainID: 56,
		LensConfig:            ephemeral.LensConfig{UseLens: true},
		BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, binChain(t, activeID).Client(), nil)

	p := entity.Pool{
		Address:     "0x" + strings.Repeat("ab", 32),
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"fee":3000}`,
	}
	got, err := tr.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{
		BlockHeaders: map[uint64]entity.BlockHeader{100: {}},
	})
	require.NoError(t, err)
	require.True(t, hasState(got), "the lens did not give the pool its state")
	assert.EqualValues(t, 100, got.BlockNumber)
	assert.NotZero(t, got.Timestamp)
}

// fallbackChain is a node whose lens read, when noProgress, comes back with no bin read and its
// cursor where it started, as one under a gas cap too low for the pool does, and otherwise fails as
// a node behind does; every other call fails too. subgraphAsked tells whether the bootstrap without
// the lens ran.
func fallbackChain(t *testing.T, noProgress bool) (node *testutil.NodeStub, subgraph *httptest.Server,
	subgraphAsked *atomic.Bool) {
	node = testutil.NewNodeStub(t, 56, 100, func(c testutil.NodeCall) ([]byte, error) {
		if c.To != nil || !noProgress {
			return nil, errors.New("header not found")
		}
		out, err := lens.PackResult(big.NewInt(8388608), big.NewInt(100), big.NewInt(0), false,
			[]struct{ Id, ReserveX, ReserveY *big.Int }{})
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
	subgraphAsked = new(atomic.Bool)
	subgraph = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		subgraphAsked.Store(true)
		_, _ = w.Write([]byte(`{"data":{"lbpair":null}}`))
	}))
	return node, subgraph, subgraphAsked
}

// TestLensBootstrapFallsBackWhereTheLensCannotRead: a node that cannot serve the lens read at all,
// here as one that caps a call under the pool's densest step, has the pool bootstrapped without the
// lens rather than left unsaved; a failure of the node's own is the bootstrap's, to be tried again.
func TestLensBootstrapFallsBackWhereTheLensCannotRead(t *testing.T) {
	for _, noProgress := range []bool{true, false} {
		node, subgraph, subgraphAsked := fallbackChain(t, noProgress)
		tr := NewPoolTracker(&Config{ChainID: 56,
			LensConfig:            ephemeral.LensConfig{UseLens: true},
			BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, node.Client(), graphqlpkg.NewClient(subgraph.URL))
		_, _ = tr.BootstrapPoolState(t.Context(), entity.Pool{
			Address:     "0x" + strings.Repeat("ab", 32),
			Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
			StaticExtra: `{"fee":3000}`,
		}, poolpkg.GetNewPoolStateParams{})
		assert.Equal(t, noProgress, subgraphAsked.Load(), "the bootstrap without the lens ran")
		node.Close()
		subgraph.Close()
	}
}
