package liquiditybookv20

import (
	"encoding/json"
	"errors"
	"maps"
	"math/big"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// emptyActiveBinChain serves a pair at block 101 whose active bin is empty, with liquidity on both
// sides of it. The lens returns only the bins that hold liquidity, as on chain.
func emptyActiveBinChain(t *testing.T) (*PoolTracker, func() []string, func()) {
	const active = 8388608
	chain, lensBlocks := lensChain(t, 101, map[uint64]lensState{
		101: {active: active, rx: 1e18, ry: 1e18, bins: map[uint32][2]int64{
			active - 1: {0, 1e18}, active: {0, 0}, active + 1: {1e18, 0}}},
	})
	return NewPoolTracker(&Config{UseLens: true}, chain.Client(), nil), lensBlocks, chain.Close
}

// emptyActiveBinWant is the set the RPC path stores for that pair since it keeps the active bin.
func emptyActiveBinWant() []Bin {
	const active = 8388608
	e18 := big.NewInt(1e18)
	return []Bin{
		{ID: active - 1, ReserveX: new(big.Int), ReserveY: e18},
		{ID: active, ReserveX: new(big.Int), ReserveY: new(big.Int)},
		{ID: active + 1, ReserveX: e18, ReserveY: new(big.Int)},
	}
}

func storedBins(t *testing.T, p entity.Pool) []Bin {
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	return extra.Bins
}

// assertQuotesBothWays: the pair swaps from its next non-empty bin either way, and so must the
// simulator built from what the tracker stored.
func assertQuotesBothWays(t *testing.T, p entity.Pool, label string) {
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	for _, in := range []string{"0xa", "0xb"} {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(1000)},
			TokenOut:      otherToken[in],
		})
		assert.NoError(t, err, "%s: swap %s -> %s", label, in, otherToken[in])
	}
}

// TestLensBootstrapKeepsEmptyActiveBin: a pair the lens bootstraps is stored with its empty active
// bin, as the RPC bootstrap stores it.
func TestLensBootstrapKeepsEmptyActiveBin(t *testing.T) {
	tracker, lensBlocks, closeChain := emptyActiveBinChain(t)
	defer closeChain()

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), Exchange: "traderjoe-v20", Type: DexTypeLiquidityBookV20,
		Tokens: []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}}}
	got, err := tracker.BootstrapPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotEmpty(t, lensBlocks(), "the pair was read through the lens")

	assertSameBins(t, emptyActiveBinWant(), storedBins(t, got), "bins the lens bootstrap stored")
	assertQuotesBothWays(t, got, "as the lens bootstrap stored it")
}

// TestLensRepairKeepsEmptyActiveBin: a withdrawal empties the active bin and the bins stored no
// longer add up to the reserves. The lens reads the set again, and the update stores the same bins,
// the empty active one among them, as the RPC path that reads the stored bins again.
func TestLensRepairKeepsEmptyActiveBin(t *testing.T) {
	const active = 8388608
	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	stale, err := json.Marshal(Extra{ActiveBinID: active, Bins: []Bin{
		{ID: active - 1, ReserveX: new(big.Int), ReserveY: big.NewInt(1e18)},
		{ID: active, ReserveX: big.NewInt(5), ReserveY: big.NewInt(5)},
		{ID: active + 1, ReserveX: big.NewInt(2e18), ReserveY: new(big.Int)},
	}})
	require.NoError(t, err)
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), Exchange: "traderjoe-v20", Type: DexTypeLiquidityBookV20,
		Tokens: []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}}, BlockNumber: 100, Extra: string(stale)}
	logs := []ethtypes.Log{swapLog(t, pair, 101, false)}

	for _, useLens := range []bool{true, false} {
		tracker, lensBlocks, closeChain := emptyActiveBinChain(t)
		tracker.cfg.UseLens = useLens
		got, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{Logs: logs})
		closeChain()
		require.NoError(t, err)
		if useLens {
			require.NotEmpty(t, lensBlocks(), "the bins were read again through the lens")
		} else {
			require.Empty(t, lensBlocks())
		}

		label := "RPC repair"
		if useLens {
			label = "lens repair"
		}
		assertSameBins(t, emptyActiveBinWant(), storedBins(t, got), "bins the "+label+" stored")
		assertQuotesBothWays(t, got, "after the "+label)
	}
}

// lensState is a pair at one block.
type lensState struct {
	active uint32
	rx, ry int64
	bins   map[uint32][2]int64
}

// lensChain serves a pair whose state is states[block]: multicalls of its views, and the lens.
// It records the block each lens call was pinned to.
func lensChain(t *testing.T, head uint64, states map[uint64]lensState) (*testutil.NodeStub, func() []string) {
	var mu sync.Mutex
	var lensBlocks []string
	chain := testutil.NewNodeStub(t, 43114, head, func(c testutil.NodeCall) ([]byte, error) {
		st, ok := states[c.Block]
		if c.To == nil {
			mu.Lock()
			lensBlocks = append(lensBlocks, c.Tag)
			mu.Unlock()
		}
		if !ok {
			return nil, errors.New("header not found")
		}
		if c.To == nil {
			var bins []struct{ Id, ReserveX, ReserveY *big.Int }
			for _, id := range slices.Sorted(maps.Keys(st.bins)) {
				if b := st.bins[id]; b[0] != 0 || b[1] != 0 {
					bins = append(bins, struct{ Id, ReserveX, ReserveY *big.Int }{
						big.NewInt(int64(id)), big.NewInt(b[0]), big.NewInt(b[1])})
				}
			}
			out, err := lens.PackResult(big.NewInt(int64(st.active)), big.NewInt(st.rx), big.NewInt(st.ry),
				new(big.Int).SetUint64(c.Block), big.NewInt(0), true, bins)
			require.NoError(t, err)
			return nil, testutil.Revert(out)
		}
		ret, _, err := testutil.Multicall(*c.To, c.Data, c.Block, 0, func(_ common.Address, callData []byte) (bool, []byte) {
			m, err := pairABI.MethodById(callData[:4])
			if err != nil { // a call the pair does not have, such as one of the multicall's own
				return true, make([]byte, 32)
			}
			var out []byte
			switch m.Name {
			case pairMethodGetReservesAndID:
				out, err = m.Outputs.Pack(big.NewInt(st.rx), big.NewInt(st.ry), big.NewInt(int64(st.active)))
			case pairMethodGetBin:
				in, err2 := m.Inputs.Unpack(callData[4:])
				require.NoError(t, err2)
				b := st.bins[uint32(in[0].(*big.Int).Uint64())]
				out, err = m.Outputs.Pack(big.NewInt(b[0]), big.NewInt(b[1]))
			default:
				out = make([]byte, 32*20)
			}
			require.NoError(t, err)
			return true, out
		})
		return ret, err
	})
	return chain, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), lensBlocks...)
	}
}
