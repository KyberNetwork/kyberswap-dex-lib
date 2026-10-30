package liquiditybookv21

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// zeroNode answers every multicall with zeros but the pair's reserves, 5 and 7, which no bin of zeros
// adds up to, and records the block each one asked for. It fails every lens call, recording its
// block apart.
func zeroNode(t *testing.T) (*testutil.NodeStub, func() []string, func() []string) {
	var mu sync.Mutex
	var blocks, lensBlocks []string
	reserves := pairABI.Methods[pairMethodGetReserves]
	node := testutil.NewNodeStub(t, 43114, 0, func(c testutil.NodeCall) ([]byte, error) {
		if c.To == nil {
			mu.Lock()
			lensBlocks = append(lensBlocks, c.Tag)
			mu.Unlock()
			return nil, errors.New("header not found")
		}
		ret, targets, err := testutil.Multicall(*c.To, c.Data, 0, 0, func(_ common.Address, callData []byte) (bool, []byte) {
			if !bytes.HasPrefix(callData, reserves.ID) {
				return true, make([]byte, 512)
			}
			values := []any{big.NewInt(5), big.NewInt(7), big.NewInt(8388608)}[:len(reserves.Outputs)]
			return true, testutil.MustPack(t, reserves.Outputs, values...)
		})
		if targets != nil { // not a call of the multicall's own, such as its block timestamp
			mu.Lock()
			blocks = append(blocks, c.Tag)
			mu.Unlock()
		}
		return ret, err
	})
	return node, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), blocks...)
		}, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), lensBlocks...)
		}
}

// swapLog is a Swap of pair in bin 8388608 at block.
func swapLog(t *testing.T, pair common.Address, block uint64, removed bool) ethtypes.Log {
	return testutil.EventLog(t, pairABI.Events["Swap"], pair, block, removed, map[string]any{"id": big.NewInt(8388608)})
}

// TestReorgReadsBinsWithThePair: a reorg under a pair at block 120 reverting block 110 reads the
// pair's reserves and active bin at 110, its logs' block, and its bins there too, so they add up to
// the reserves read with them.
func TestReorgReadsBinsWithThePair(t *testing.T) {
	node, blocks, _ := zeroNode(t)
	tracker := NewPoolTracker(&Config{}, node.Client(), nil)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), BlockNumber: 120, Extra: `{}`}
	_, _ = tracker.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{
		Logs: []ethtypes.Log{swapLog(t, pair, 110, true), swapLog(t, pair, 110, false)},
	})

	read := blocks()
	require.GreaterOrEqual(t, len(read), 2, "the pair's state and its bins are read")
	for _, block := range read {
		assert.Equal(t, hexutil.EncodeUint64(110), block)
	}
}

// TestReorgReadsLensBinsWithThePair: bins that no longer add up to the reserves are read again
// through the lens at the block the reserves were read at, the reorg's, not at the pair's newer one.
func TestReorgReadsLensBinsWithThePair(t *testing.T) {
	node, blocks, lensBlocks := zeroNode(t)
	tracker := NewPoolTracker(&Config{
		LensConfig: ephemeral.LensConfig{UseLens: true}}, node.Client(), nil)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), BlockNumber: 120,
		Extra: `{"bins":[{"id":8388608,"reserveX":1,"reserveY":1}]}`}
	_, _ = tracker.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{
		Logs: []ethtypes.Log{swapLog(t, pair, 110, true), swapLog(t, pair, 110, false)},
	})

	require.NotEmpty(t, lensBlocks(), "the bins were read again through the lens")
	for _, block := range append(blocks(), lensBlocks()...) {
		assert.Equal(t, hexutil.EncodeUint64(110), block)
	}
}
