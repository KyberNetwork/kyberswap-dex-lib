package liquiditybookv20

import (
	"errors"
	"maps"
	"math/big"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

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
