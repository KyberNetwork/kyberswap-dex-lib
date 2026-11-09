package dexv2

import (
	"math/big"
	"os"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
)

type replayFixture struct {
	Address  string         `json:"address"`
	EndBlock uint64         `json:"endBlock"`
	EndTicks []Tick         `json:"endTicks"`
	Logs     []ethtypes.Log `json:"logs"`
}

// Fixture: Ethereum mainnet D3 pools of the live Fluid DEX v2. EndTicks are the initialized ticks read from the
// pool's tick bitmap at EndBlock; Logs are every DEX log of the pool in the ~6 days of history the node still served
// (LogDeposit and LogSwapOut among them), all at or before EndBlock.
func loadReplay(t *testing.T) []replayFixture {
	t.Helper()
	b, err := os.ReadFile("testdata/ethereum_tick_replay.json")
	require.NoError(t, err)
	var fx []replayFixture
	require.NoError(t, json.Unmarshal(b, &fx))
	return fx
}

func applyPositionLogs(t *testing.T, ticks map[int]tickspkg.Tick, logs []ethtypes.Log) {
	t.Helper()
	tr := &PoolTracker{}
	for i, l := range logs {
		lower, upper, delta, err := tr.extractEventData(l)
		require.NoError(t, err, "log %d", i)
		if delta.Sign() == 0 {
			continue
		}
		require.True(t, tr.applyLiquidityChange(ticks, lower, delta, true))
		require.True(t, tr.applyLiquidityChange(ticks, upper, delta, false))
	}
}

// Deposits rebuilt from logs give exactly the on-chain gross/net: three LP deposits into the USDC/ETH pool
// (150159835666 + 114793393957 on one range, 942005700598 on another) are the whole liquidity of those four ticks.
func TestRealDepositLogsMatchOnchainTicks(t *testing.T) {
	var fx replayFixture
	for _, f := range loadReplay(t) {
		if f.Address == "0xd0adee4ff64cfb16d76667ef79e093420a3f01cd648ffaae9bfab849449930fd_3" {
			fx = f
		}
	}
	require.NotEmpty(t, fx.Logs)
	rebuilt := map[int]tickspkg.Tick{}
	applyPositionLogs(t, rebuilt, fx.Logs)

	onchain := map[int]Tick{}
	for _, tk := range fx.EndTicks {
		onchain[tk.Index] = tk
	}
	require.Len(t, rebuilt, 4)
	for idx, got := range rebuilt {
		want, ok := onchain[idx]
		require.True(t, ok, "tick %d initialized on chain", idx)
		require.Zero(t, want.LiquidityGross.Cmp(got.LiquidityGross), "tick %d gross", idx)
		require.Zero(t, want.LiquidityNet.Cmp(got.LiquidityNet), "tick %d net", idx)
	}
}

// A bootstrap that read ticks at EndBlock already includes these deposits. The worker may still replay their logs
// (its checkpoint comes from a different source), and applying them again would double the liquidity.
func TestReplayOfAlreadyIncludedLogsIsDropped(t *testing.T) {
	for _, fx := range loadReplay(t) {
		current := func() map[int]tickspkg.Tick {
			m := map[int]tickspkg.Tick{}
			for _, tk := range fx.EndTicks {
				m[tk.Index] = tickspkg.Tick{TickIdx: tk.Index, LiquidityGross: new(big.Int).Set(tk.LiquidityGross), LiquidityNet: new(big.Int).Set(tk.LiquidityNet)}
			}
			return m
		}

		// Without the filter the deposits are counted twice.
		doubled := current()
		applyPositionLogs(t, doubled, fx.Logs)
		changed := false
		for _, tk := range fx.EndTicks {
			if doubled[tk.Index].LiquidityGross.Cmp(tk.LiquidityGross) != 0 {
				changed = true
			}
		}
		require.True(t, changed, "%s: replay must change ticks when nothing is dropped", fx.Address)

		// With TicksBlock = EndBlock every one of those logs is already included.
		kept := dropAppliedLogs(fx.Logs, fx.EndBlock)
		require.Empty(t, kept)
		same := current()
		applyPositionLogs(t, same, kept)
		for _, tk := range fx.EndTicks {
			require.Zero(t, same[tk.Index].LiquidityGross.Cmp(tk.LiquidityGross), "%s tick %d", fx.Address, tk.Index)
			require.Zero(t, same[tk.Index].LiquidityNet.Cmp(tk.LiquidityNet), "%s tick %d", fx.Address, tk.Index)
		}

		// A log after the snapshot block is still applied.
		later := fx.Logs[0]
		later.BlockNumber = fx.EndBlock + 1
		require.Len(t, dropAppliedLogs([]ethtypes.Log{later}, fx.EndBlock), 1)
	}
}
