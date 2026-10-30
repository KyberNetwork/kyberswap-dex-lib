package bin

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

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
	tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true,
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
