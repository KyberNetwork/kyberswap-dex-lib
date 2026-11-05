package liquiditybookv21

import (
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// unpinnedChain serves a pair at block 101 from a node that ignores a pinned block, as HyperEVM's
// does: the lens's second page, pinned to the first's block, runs at the node's next block instead.
// It counts the lens calls.
func unpinnedChain(t *testing.T) (*testutil.NodeStub, func() int) {
	const active = 8388608
	bins := map[uint32][2]int64{active - 1: {0, 1e18}, active + 1: {1e18, 0}}
	var mu sync.Mutex
	lensCalls := 0
	chain := testutil.NewNodeStub(t, 43114, 101, func(c testutil.NodeCall) ([]byte, error) {
		if c.To == nil {
			mu.Lock()
			lensCalls++
			first := lensCalls == 1
			mu.Unlock()
			block, next, done, id := int64(101), big.NewInt(active+1), false, uint32(active-1)
			if !first {
				block, next, done, id = 102, big.NewInt(0), true, active+1
			}
			out, err := lens.PackResult(big.NewInt(active), big.NewInt(1e18), big.NewInt(1e18), big.NewInt(block),
				next, done, []struct{ Id, ReserveX, ReserveY *big.Int }{
					{big.NewInt(int64(id)), big.NewInt(bins[id][0]), big.NewInt(bins[id][1])}})
			require.NoError(t, err)
			return nil, testutil.Revert(out)
		}
		if c.Block != 101 {
			return nil, errors.New("header not found")
		}
		ret, _, err := testutil.Multicall(*c.To, c.Data, c.Block, 0, func(_ common.Address, callData []byte) (bool, []byte) {
			m, err := pairABI.MethodById(callData[:4])
			if err != nil { // a call the pair does not have, such as one of the multicall's own
				return true, make([]byte, 32)
			}
			var out []byte
			switch m.Name {
			case pairMethodGetReserves:
				out, err = m.Outputs.Pack(big.NewInt(1e18), big.NewInt(1e18))
			case pairMethodGetActiveID:
				out, err = m.Outputs.Pack(big.NewInt(active))
			case pairMethodGetBinStep:
				out, err = m.Outputs.Pack(uint16(25))
			case pairMethodGetBin:
				in, err2 := m.Inputs.Unpack(callData[4:])
				require.NoError(t, err2)
				b := bins[uint32(in[0].(*big.Int).Uint64())]
				out, err = m.Outputs.Pack(big.NewInt(b[0]), big.NewInt(b[1]))
			default:
				out = make([]byte, 32*len(m.Outputs))
			}
			require.NoError(t, err)
			return true, out
		})
		return ret, err
	})
	return chain, func() int {
		mu.Lock()
		defer mu.Unlock()
		return lensCalls
	}
}

// staleSubgraph serves the pair's bins as a subgraph behind the chain does: their reserves are
// those of an older block.
func staleSubgraph(t *testing.T) *httptest.Server {
	var mu sync.Mutex
	served := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if served {
			_, _ = io.WriteString(w, `{"data":{"lbpair":{"tokenX":{"decimals":"18"},"tokenY":{"decimals":"18"},"bins":[]}}}`)
			return
		}
		served = true
		_, _ = io.WriteString(w, `{"data":{"lbpair":{"tokenX":{"decimals":"18"},"tokenY":{"decimals":"18"},"bins":[`+
			`{"binId":"8388607","reserveX":"0","reserveY":"0.5","totalSupply":"1"},`+
			`{"binId":"8388609","reserveX":"2","reserveY":"0","totalSupply":"1"}]}}}`)
	}))
	t.Cleanup(s.Close)
	return s
}

// TestBootstrapWithoutTheLensReadsTheBinsAgain: on a node that cannot serve the lens, a bootstrap
// falls back to the subgraph and RPC. FetchPoolTicks leaves the bins to the lens when it is on, so
// the fallback reads the subgraph's bins again at the block of the state itself: the pair is stored
// with the chain's bins, not the subgraph's older ones.
func TestBootstrapWithoutTheLensReadsTheBinsAgain(t *testing.T) {
	const active = 8388608
	chain, lensCalls := unpinnedChain(t)
	tracker := NewPoolTracker(&Config{UseLens: true}, chain.Client(),
		graphqlpkg.NewClient(staleSubgraph(t).URL))

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), Exchange: "traderjoe-v21", Type: DexTypeLiquidityBookV21,
		Tokens: []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}}}
	got, err := tracker.BootstrapPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, lensCalls(), 2, "the pair was read through the lens across pages first")

	got, err = tracker.FetchPoolTicks(t.Context(), got)
	require.NoError(t, err)
	e18 := big.NewInt(1e18)
	assertSameBins(t, []Bin{
		{ID: active - 1, ReserveX: new(big.Int), ReserveY: e18},
		{ID: active, ReserveX: new(big.Int), ReserveY: new(big.Int)},
		{ID: active + 1, ReserveX: e18, ReserveY: new(big.Int)},
	}, storedBins(t, got), "bins read again at the state's block")
}
