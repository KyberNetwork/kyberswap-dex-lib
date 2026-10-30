package bin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// binNodeAt serves one bin pool through multicalls at any block, its slot0 and getBin from state, and
// no lens.
func binNodeAt(t *testing.T, active uint32, bins map[uint32][2]int64) *testutil.NodeStub {
	return testutil.NewNodeStub(t, 56, 110, func(c testutil.NodeCall) ([]byte, error) {
		if c.To == nil {
			return nil, errors.New("not served")
		}
		ret, _, err := testutil.Multicall(*c.To, c.Data, 110, 0, func(_ common.Address, callData []byte) (bool, []byte) {
			m, err := shared.BinPoolManagerABI.MethodById(callData[:4])
			require.NoError(t, err)
			switch m.Name {
			case "getSlot0":
				return true, testutil.MustPack(t, m.Outputs, active, uint32(0), uint32(0))
			case "getBin":
				in, err := m.Inputs.Unpack(callData[4:])
				require.NoError(t, err)
				b := bins[uint32(in[1].(*big.Int).Uint64())]
				return true, testutil.MustPack(t, m.Outputs, big.NewInt(b[0]), big.NewInt(b[1]), new(big.Int), new(big.Int))
			}
			t.Errorf("unexpected %s", m.Name)
			return false, nil
		})
		return ret, err
	})
}

// TestRemovedSwapRereadsTheBinsItCrossed: a swap at block 110 moved the active bin from 100 to 105,
// and the pool was stored as it left bins 100 to 105. A reorg drops that swap, and the chain is back
// at active bin 100 with bins 101 to 104 holding X. Its removed event names only bin 105, so the
// bins it crossed are read again as those between the stored active bin and the one read now.
func TestRemovedSwapRereadsTheBinsItCrossed(t *testing.T) {
	chain := map[uint32][2]int64{100: {10, 10}, 101: {10, 0}, 102: {10, 0}, 103: {10, 0}, 104: {10, 0}, 105: {10, 0}}
	manager := common.HexToAddress("0xC697d2898e0D09264376196696c51D7aBbbAA4a9")
	tr := NewPoolTracker(&Config{ChainID: 56,
		LensConfig: ephemeral.LensConfig{UseLens: true}, BinPoolManagerAddress: manager.Hex()},
		binNodeAt(t, 100, chain).Client(), nil)

	poolID := common.HexToHash("0x" + strings.Repeat("ab", 32))
	swap := shared.BinPoolManagerABI.Events["Swap"]
	logData, err := swap.Inputs.NonIndexed().Pack(big.NewInt(-1), big.NewInt(1), big.NewInt(105), big.NewInt(3000),
		uint16(0))
	require.NoError(t, err)
	removed := ethtypes.Log{Address: manager, BlockNumber: 110, Removed: true,
		Topics: []common.Hash{swap.ID, poolID, {}}, Data: logData}

	p := entity.Pool{
		Address:     strings.ToLower(poolID.Hex()),
		BlockNumber: 110,
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"fee":3000}`,
		Extra: `{"protocolFee":0,"activeBinId":105,"bins":[{"id":100,"reserveX":"0","reserveY":"20"},` +
			`{"id":101,"reserveX":"0","reserveY":"10"},{"id":102,"reserveX":"0","reserveY":"10"},` +
			`{"id":103,"reserveX":"0","reserveY":"10"},{"id":104,"reserveX":"0","reserveY":"10"},` +
			`{"id":105,"reserveX":"5","reserveY":"5"}]}`,
	}
	got, err := tr.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{Logs: []ethtypes.Log{removed}})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	assert.EqualValues(t, 100, extra.ActiveBinID)
	for _, b := range extra.Bins {
		want := chain[b.ID]
		assert.Equal(t, fmt.Sprint(want), fmt.Sprintf("[%s %s]", b.ReserveX, b.ReserveY), "bin %d", b.ID)
	}
}
