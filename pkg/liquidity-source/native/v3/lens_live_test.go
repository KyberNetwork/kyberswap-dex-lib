package v3

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// livePool is a Native V3 pool as its factory log creates it: two LP tokens and its spacing.
type livePool struct {
	address, token0, token1 string
	decimals0, decimals1    uint8
	spacing                 uint64
	ticks                   bool // whether it holds positions, so the lens must find ticks
}

func (lp livePool) pool() entity.Pool {
	return entity.Pool{
		Address:  lp.address,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: lp.token0, Decimals: lp.decimals0, Swappable: true},
			{Address: lp.token1, Decimals: lp.decimals1, Swappable: true},
		},
		StaticExtra: fmt.Sprintf(`{"tickSpacing":%d}`, lp.spacing),
	}
}

var livePools = map[string][]livePool{
	"NATIVE_V3_ETHEREUM_RPC_URL": {
		{"0x933347830121ed36d1a98a8f3ddf75b6393b4ed3", "0x5994258ec80cc6853e2b6f047ec6d213fe89b24b",
			"0x91f70f89915f8e5fc9fdd8078685067a49cc6c28", 18, 6, 10, true},
		{"0xe79d39ef1f0bce80e065ec4670bf15d2cf62d230", "0x5994258ec80cc6853e2b6f047ec6d213fe89b24b",
			"0xb3c455dd8e45524264cad476195aa7c4bd27d2a8", 18, 18, 60, true},
		{"0x1659cc9f13ab09ff35b39a2aee9a0b1111d300ab", "0x5994258ec80cc6853e2b6f047ec6d213fe89b24b",
			"0x66550d6453dbb70103b96fac81052f106d4fffeb", 18, 18, 60, true},
	},
	"NATIVE_V3_ARBITRUM_RPC_URL": {
		{"0x050de6e6bb2c2e20cfdabd1f2a01176641fddc84", "0x21d5d0ebc83d877a06f7ae92462075f3d324010d",
			"0xce55b8c3bb83458cd14f42e803ff49076296535e", 6, 18, 10, true},
		{"0xee51e5de5d4138332edd12b55051025aa39c5e2a", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831", 18, 6, 10, false},
	},
}

// TestLiveLens: on each chain with an endpoint, a bootstrap through the lens holds every tick the
// pool reports for those indices at the same block, reports the block it was pinned to, a gas cap
// that forces paging reads the same set, a stored set missing a tick gets it back, and a pool
// without state gets it whole, its underlying tokens included. Run with:
//
//	NATIVE_V3_ETHEREUM_RPC_URL=https://... NATIVE_V3_ARBITRUM_RPC_URL=https://... \
//	    go test ./pkg/liquidity-source/native/v3/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	for env, pools := range livePools {
		t.Run(env, func(t *testing.T) {
			rpcURL := os.Getenv(env)
			if rpcURL == "" {
				t.Skipf("set %s to run this", env)
			}
			for _, lp := range pools {
				t.Run(lp.address, func(t *testing.T) { checkLiveLens(t, rpcURL, lp) })
			}
		})
	}
}

func checkLiveLens(t *testing.T, rpcURL string, lp livePool) {
	ctx := t.Context()
	live := testutil.DialLive(t, rpcURL)
	client := live.Ethrpc
	cfg := &Config{DexID: DexType, UseLens: true}
	tracker := NewPoolTracker(cfg, client, nil)

	p, err := tracker.BootstrapPoolState(ctx, lp.pool(), sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotZero(t, p.BlockNumber)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	if lp.ticks {
		require.NotEmpty(t, extra.Ticks)
	}

	// Each tick against the pool's own answer for it, at the lens's block.
	viaRPC, err := tracker.queryRPCTicksByIndexes(ctx, p.Address,
		lo.Map(extra.Ticks, func(tk Tick, _ int) int { return tk.Index }), p.BlockNumber)
	require.NoError(t, err)
	byIndex := lo.SliceToMap(viaRPC, func(tk tickspkg.Tick) (int, tickspkg.Tick) { return tk.TickIdx, tk })
	for _, tk := range extra.Ticks {
		got := byIndex[tk.Index]
		assert.Zero(t, tk.LiquidityGross.Cmp(got.LiquidityGross), "gross of tick %d", tk.Index)
		assert.Zero(t, tk.LiquidityNet.Cmp(got.LiquidityNet), "net of tick %d", tk.Index)
	}
	rpcData, err := tracker.FetchRPCData(ctx, &p, p.BlockNumber)
	require.NoError(t, err)
	assert.Zero(t, extra.Liquidity.Cmp(rpcData.Liquidity), "liquidity")
	assert.Zero(t, extra.SqrtPriceX96.Cmp(rpcData.Slot0.SqrtPriceX96), "price")

	block := new(big.Int).SetUint64(p.BlockNumber)
	before := live.Requests()
	whole, err := tracker.snapshot(ctx, p.Address, block)
	require.NoError(t, err)
	assert.Equal(t, p.BlockNumber, whole.block, "the lens reported another block than the one pinned")
	wholeCalls := live.Requests() - before
	cfg.LensGas = 250_000
	before = live.Requests()
	paged, err := tracker.snapshot(ctx, p.Address, block)
	require.NoError(t, err)
	assert.Greater(t, live.Requests()-before, wholeCalls, "the cap did not force paging")
	assert.Equal(t, whole.ticks, paged.ticks, "paging changed the set")
	cfg.LensGas = 0

	if len(extra.Ticks) > 0 {
		// A stored set missing a tick, as logs can leave it, is replaced by the full one.
		tbp, err := tickspkg.NewTicksBasedPool(p)
		require.NoError(t, err)
		dropped := extra.Ticks[len(extra.Ticks)/2].Index
		delete(tbp.Ticks, dropped)
		require.True(t, tracker.ticksFromLens(ctx, &tbp, p.BlockNumber))
		assert.Contains(t, tbp.Ticks, dropped, "the missing tick was not restored")
		assert.Len(t, tbp.Ticks, len(extra.Ticks))
	}

	// A pool created from its factory log has no state: its first update reads it whole at the
	// batch's newest block, whatever logs the batch carries, and gets its underlying tokens. A new
	// tracker, as the guard keeps the pool just read above in its cooldown.
	head := p.BlockNumber
	fresh, err := NewPoolTracker(&Config{DexID: DexType,
		UseLens: true},
		client, nil).GetNewPoolState(ctx, lp.pool(), sourcePool.GetNewPoolStateParams{
		Logs:         []ethtypes.Log{{BlockNumber: head - 1}},
		BlockHeaders: map[uint64]entity.BlockHeader{head - 1: {}, head: {}},
	})
	require.NoError(t, err)
	assert.True(t, tickspkg.HasState(fresh))
	assert.Equal(t, head, fresh.BlockNumber)
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
	assert.Len(t, freshExtra.Ticks, len(extra.Ticks))
	if rpcData.UnderlyingTokens[0] != (common.Address{}) {
		assert.Len(t, fresh.Tokens, 4, "the underlying tokens were not added")
	}

	t.Logf("%d ticks at block %d, liquidity %s, tick %s, %d tokens after the first update",
		len(whole.ticks), whole.block, extra.Liquidity, extra.Tick, len(fresh.Tokens))
}
