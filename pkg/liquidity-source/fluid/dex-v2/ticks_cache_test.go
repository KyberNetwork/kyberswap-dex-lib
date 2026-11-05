package dexv2

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestCanReuseTicks(t *testing.T) {
	ticks := []Tick{{Index: 1, LiquidityGross: big.NewInt(1), LiquidityNet: big.NewInt(1)}}
	require.False(t, canReuseTicks(nil, 100))
	require.False(t, canReuseTicks(&Extra{TicksBlock: 90}, 100), "no cached ticks")
	require.False(t, canReuseTicks(&Extra{Ticks: ticks}, 100), "legacy state without scan block")
	require.False(t, canReuseTicks(&Extra{Ticks: ticks, TicksBlock: 101}, 100), "snapshot behind scan")
	require.True(t, canReuseTicks(&Extra{Ticks: ticks, TicksBlock: 100}, 100))
	require.True(t, canReuseTicks(&Extra{Ticks: ticks, TicksBlock: 100}, 100+ticksFullRescanBlocks))
	require.False(t, canReuseTicks(&Extra{Ticks: ticks, TicksBlock: 100}, 101+ticksFullRescanBlocks), "forced rescan")
}

func TestTicksLogQuery(t *testing.T) {
	dex := common.HexToAddress("0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261")
	id := common.HexToHash("0xe018f09af38956affdfeab72c2cefbcd4e6fee44d09df7525ec9dba3e51356a5")
	q := ticksLogQuery(dex, 3, id, 1000, 1050)
	require.Equal(t, uint64(1000-ticksReorgOverlap), q.FromBlock.Uint64())
	require.Equal(t, uint64(1050), q.ToBlock.Uint64())
	require.Equal(t, []common.Address{dex}, q.Addresses)
	require.Len(t, q.Topics, 3)
	require.Len(t, q.Topics[0], 4, "deposit, withdraw, borrow, payback")
	require.Equal(t, common.BigToHash(big.NewInt(3)), q.Topics[1][0])
	require.Equal(t, id, q.Topics[2][0])
	require.Equal(t, uint64(0), ticksLogQuery(dex, 3, id, 5, 10).FromBlock.Uint64())
}

func TestLiquidityChanged(t *testing.T) {
	data := make([]byte, 256)
	require.False(t, liquidityChanged(ethtypes.Log{Data: data}), "fee-only call")
	data[255] = 1
	require.True(t, liquidityChanged(ethtypes.Log{Data: data}))
	require.True(t, liquidityChanged(ethtypes.Log{Data: data[:224]}), "unknown layout")
}
