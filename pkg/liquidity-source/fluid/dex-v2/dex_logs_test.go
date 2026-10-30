package dexv2

import (
	"cmp"
	"os"
	"slices"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
)

type dexLogsFixture struct {
	Pool  string         `json:"pool"`
	Logs  []ethtypes.Log `json:"logs"`
	Ticks []Tick         `json:"ticks"`
}

func loadDexLogs(t *testing.T) dexLogsFixture {
	t.Helper()
	b, err := os.ReadFile("testdata/dex_logs_polygon.json")
	require.NoError(t, err)
	var fx dexLogsFixture
	require.NoError(t, json.Unmarshal(b, &fx))
	return fx
}

// TestDecodePoolAddressesFromDexLogs: every log the dex emitted for one pool on Polygon, from its
// LogInitialize through its deposits, withdrawals and swaps, decodes to that pool. The dex indexes
// dexType, dexId and user, and a swap is a LogSwapIn.
func TestDecodePoolAddressesFromDexLogs(t *testing.T) {
	fx := loadDexLogs(t)
	f := NewPoolFactory(&Config{DexID: "fluid-dex-v2", ChainID: 137})
	for i, log := range fx.Logs {
		pools, err := f.DecodePoolAddressesFromFactoryLog(t.Context(), log)
		require.NoError(t, err, "log %d", i)
		assert.Equal(t, []string{fx.Pool}, pools, "log %d", i)
	}
}

// TestTicksFromDexLogs: the pool's liquidity logs, applied as the tracker applies them, give the
// ticks the pool holds on chain after them.
//
// The fixture's logs are those of the dex 0x7822b394 on Polygon for the pool, from its
// LogInitialize at block 79889908 (tx 0xaafc296f) to its last LogSwapIn at block 93210176
// (tx 0xbcccb2bd); its last liquidity log is a LogWithdraw at block 84438661 (tx 0x529f8eba).
// Each log equals the log of its transaction's receipt. The expected ticks are every tick set in
// the pool's tick bitmap (slot 2) with its liquidityGross and liquidityNet (slots 3 and 4), read
// at blocks 84438661, 93210176 and 96265000, where they are the same.
func TestTicksFromDexLogs(t *testing.T) {
	fx := loadDexLogs(t)
	tr := &PoolTracker{}
	ticks := map[int]tickspkg.Tick{}
	for i, log := range fx.Logs {
		lower, upper, delta, err := tr.extractEventData(log)
		require.NoError(t, err, "log %d", i)
		if delta.Sign() == 0 {
			continue
		}
		require.True(t, tr.applyLiquidityChange(ticks, lower, delta, true), "log %d", i)
		require.True(t, tr.applyLiquidityChange(ticks, upper, delta, false), "log %d", i)
	}

	var got []Tick
	for _, tick := range ticks {
		if tick.LiquidityGross.Sign() != 0 {
			got = append(got, Tick{Index: tick.TickIdx, LiquidityGross: tick.LiquidityGross, LiquidityNet: tick.LiquidityNet})
		}
	}
	slices.SortFunc(got, func(a, b Tick) int { return cmp.Compare(a.Index, b.Index) })
	require.Len(t, got, len(fx.Ticks))
	for i := range got {
		assert.Equal(t, fx.Ticks[i].Index, got[i].Index)
		assert.Zero(t, fx.Ticks[i].LiquidityGross.Cmp(got[i].LiquidityGross), "tick %d gross", got[i].Index)
		assert.Zero(t, fx.Ticks[i].LiquidityNet.Cmp(got[i].LiquidityNet), "tick %d net", got[i].Index)
	}
}
