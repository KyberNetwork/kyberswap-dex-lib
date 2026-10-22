package integral

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
)

// positionLog builds a log of event ev whose bottomTick and topTick are bottom and top, every other
// argument zero.
func positionLog(t *testing.T, ev abi.Event, block uint64, removed bool, bottom, top int64) ethtypes.Log {
	topics := []common.Hash{ev.ID}
	var data []any
	for _, in := range ev.Inputs {
		var v any
		switch {
		case in.Name == "bottomTick":
			v = big.NewInt(bottom)
		case in.Name == "topTick":
			v = big.NewInt(top)
		case in.Type.T == abi.AddressTy:
			v = common.Address{}
		default:
			v = new(big.Int)
		}
		if !in.Indexed {
			data = append(data, v)
			continue
		}
		if n, ok := v.(*big.Int); ok {
			topics = append(topics, common.BigToHash(math.U256(new(big.Int).Set(n))))
		} else {
			topics = append(topics, common.Hash{})
		}
	}
	packed, err := ev.Inputs.NonIndexed().Pack(data...)
	require.NoError(t, err)
	return ethtypes.Log{Address: common.HexToAddress("0x1"), Topics: topics, Data: packed, BlockNumber: block,
		Removed: removed}
}

// TestLateLogsReadTicksAtThePoolsBlock: the ticks a batch's logs touch are read at the pool's block
// when the logs are older, so a retried or replayed batch does not put them back to older values;
// a reorg still reads at its own block.
func TestLateLogsReadTicksAtThePoolsBlock(t *testing.T) {
	tr := NewPoolTracker(&Config{}, nil, nil)
	var blocks []uint64
	tr.readTicks = func(_ context.Context, _ string, indexes []int, block uint64,
		_ map[common.Address]gethclient.OverrideAccount) ([]tickspkg.Tick, error) {
		blocks = append(blocks, block)
		out := make([]tickspkg.Tick, 0, len(indexes))
		for _, i := range indexes {
			net := int64(5)
			if i > 0 {
				net = -5
			}
			out = append(out, tickspkg.Tick{TickIdx: i, LiquidityGross: big.NewInt(5), LiquidityNet: big.NewInt(net)})
		}
		return out, nil
	}
	p := entity.Pool{Address: "0x1", BlockNumber: 200}
	mint := poolV12ABI.Events["Mint"]

	pool, err := tr.newTicksBasedPool(t.Context(), p, []ethtypes.Log{positionLog(t, mint, 150, false, -60, 60)}, nil)
	require.NoError(t, err)
	assert.Equal(t, []uint64{200}, blocks, "a late log")
	assert.EqualValues(t, 200, pool.BlockNumber)
	assert.Len(t, pool.Ticks, 2)

	blocks = nil
	_, err = tr.newTicksBasedPool(t.Context(), p, []ethtypes.Log{positionLog(t, mint, 250, false, -60, 60)}, nil)
	require.NoError(t, err)
	assert.Equal(t, []uint64{250}, blocks, "a new log")

	blocks = nil
	_, err = tr.newTicksBasedPool(t.Context(), p, []ethtypes.Log{positionLog(t, mint, 150, true, -60, 60),
		positionLog(t, mint, 150, false, -60, 60)}, nil)
	require.NoError(t, err)
	assert.Equal(t, []uint64{150}, blocks, "a reorg")
}
