package cl

import (
	"math/big"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// countingTransport counts the HTTP requests a client makes.
type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(req)
}

type liveChain struct {
	env     string
	chainID valueobject.ChainID
	manager string
	vault   string
	pools   []string
}

var liveChains = []liveChain{{
	env:     "PANCAKE_INFINITY_CL_BSC_RPC_URL",
	chainID: valueobject.ChainIDBSC,
	manager: "0xa0FfB9c1CE1Fe56963B0321B32E7A0302114058b",
	vault:   "0x238a358808379702088667322f80aC48bAd5e6c4",
	pools: []string{
		"0xd0949ad5874cee7991ccd4cc2c5f7f80a7be4a220689cf5947ccb12a946fa50d", // spacing 10
		"0xe2d5738b246526194b19adf0dcdf5fc511126d276798e8fe5102ed464f7720e2", // spacing 10
		"0xba2eab70102e8e1ca982dce825328cadf1cd33ca4e0d1bb4e7408fe9bcdecec4", // spacing 1
	},
}, {
	env:     "PANCAKE_INFINITY_CL_BASE_RPC_URL",
	chainID: valueobject.ChainIDBase,
	manager: "0xa0FfB9c1CE1Fe56963B0321B32E7A0302114058b",
	vault:   "0x238a358808379702088667322f80aC48bAd5e6c4",
	pools:   []string{"0x25424bf60fbb8b43bdf2efee7bb9b89b1ff991f0256f8e1e5206fd23cd6fe2db"},
}}

// TestLiveLens: per pool, a bootstrap through the lens holds every tick CLPoolManager reports for
// those indices at the same block, with the price and liquidity it reports there; the lens reads at
// the block it is asked; a gas cap that forces paging reads the same set; a stored set missing a
// tick gets it back; and a pool as the factory builds it gets its state whole from its first log.
// Run with an endpoint per chain:
//
//	PANCAKE_INFINITY_CL_BSC_RPC_URL=https://... PANCAKE_INFINITY_CL_BASE_RPC_URL=https://... \
//	  go test ./pkg/liquidity-source/pancake/infinity/cl/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	ran := false
	for _, chain := range liveChains {
		rpcURL := os.Getenv(chain.env)
		if rpcURL == "" {
			continue
		}
		ran = true
		for _, id := range chain.pools {
			t.Run(chain.chainID.String()+"/"+id[:10], func(t *testing.T) { checkLivePool(t, chain, rpcURL, id) })
		}
	}
	if !ran {
		t.Skip("set PANCAKE_INFINITY_CL_BSC_RPC_URL or PANCAKE_INFINITY_CL_BASE_RPC_URL to run this")
	}
}

func checkLivePool(t *testing.T, chain liveChain, rpcURL, id string) {
	ctx := t.Context()
	counter := &countingTransport{}
	rc, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	eth := ethclient.NewClient(rc)
	client := ethrpc.NewWithClient(eth).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{
		ChainID:              int(chain.chainID),
		DexID:                DexType,
		CLPoolManagerAddress: chain.manager,
		VaultAddress:         chain.vault,
		UseLens:              true,
	}
	tracker := NewPoolTracker(cfg, client, nil)

	head, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	pinned := head - 5
	factoryPool := newLivePool(t, client, cfg, id, pinned)
	require.False(t, tickspkg.HasState(factoryPool))

	// The lens reads at the block it is asked, which is the chain's own block on every chain.
	snap, err := tracker.snapshot(ctx, id, new(big.Int).SetUint64(pinned))
	require.NoError(t, err)
	assert.Equal(t, pinned, snap.block, "the lens read another block than the one pinned")

	p, err := tracker.BootstrapPoolState(ctx, factoryPool, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotEmpty(t, extra.Ticks)
	require.NotZero(t, extra.LensBlock)
	assert.Equal(t, extra.LensBlock, p.BlockNumber)
	assert.NotEmpty(t, p.Exchange, "the hook did not resolve the exchange")
	assert.Len(t, p.Reserves, 2)

	checked := lo.Map(extra.Ticks, func(tk Tick, _ int) ephemeral.Tick {
		return ephemeral.Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	})
	require.NoError(t, ephemeral.CheckTicks(checked, int(extra.Tick.Int64()), extra.Liquidity))

	// The price and liquidity against CLPoolManager's own answer, at the lens's block.
	rpcData, _, _, err := tracker.fetchOnchainState(ctx, &factoryPool, p.BlockNumber)
	require.NoError(t, err)
	assert.Zero(t, rpcData.Slot0.SqrtPriceX96.Cmp(extra.SqrtPriceX96), "price")
	assert.Zero(t, rpcData.Liquidity.Cmp(extra.Liquidity), "liquidity")
	assert.Equal(t, rpcData.TickSpacing, extra.TickSpacing, "tick spacing")

	// Each tick against CLPoolManager's own answer for it, at the lens's block.
	viaRPC, err := tracker.queryRPCTicksByIndexes(ctx, p.Address,
		lo.Map(extra.Ticks, func(tk Tick, _ int) int { return tk.Index }), p.BlockNumber)
	require.NoError(t, err)
	byIndex := lo.SliceToMap(viaRPC, func(tk tickspkg.Tick) (int, tickspkg.Tick) { return tk.TickIdx, tk })
	for _, tk := range extra.Ticks {
		got, ok := byIndex[tk.Index]
		require.True(t, ok, "tick %d", tk.Index)
		assert.Zero(t, tk.LiquidityGross.Cmp(got.LiquidityGross), "gross of tick %d", tk.Index)
		assert.Zero(t, tk.LiquidityNet.Cmp(got.LiquidityNet), "net of tick %d", tk.Index)
	}

	block := new(big.Int).SetUint64(p.BlockNumber)
	before := counter.n.Load()
	whole, err := tracker.snapshot(ctx, id, block)
	require.NoError(t, err)
	wholeCalls := counter.n.Load() - before
	cfg.LensGas = 1_000_000
	before = counter.n.Load()
	paged, err := tracker.snapshot(ctx, id, block)
	require.NoError(t, err)
	pagedCalls := counter.n.Load() - before
	cfg.LensGas = 0
	assert.Greater(t, pagedCalls, wholeCalls, "the cap did not force paging")
	assert.Equal(t, whole.ticks, paged.ticks, "paging changed the set")

	// A stored set missing a tick, as logs can leave it, is replaced by the full one.
	tbp, err := tickspkg.NewTicksBasedPool(p)
	require.NoError(t, err)
	tbp.BlockNumber = p.BlockNumber
	dropped := extra.Ticks[len(extra.Ticks)/2].Index
	delete(tbp.Ticks, dropped)
	require.NoError(t, tracker.replaceTicksFromLens(ctx, &tbp))
	assert.Contains(t, tbp.Ticks, dropped, "the missing tick was not restored")
	assert.Len(t, tbp.Ticks, len(extra.Ticks))

	// A pool as the factory builds it has no state: its first update reads it whole at the batch's
	// newest block, which holds the batch's logs.
	logs := recentLogs(t, eth, chain.manager, id, head)
	if len(logs) == 0 {
		t.Logf("no log of %s in the last blocks; the stateless update is not checked", id)
	} else {
		last := lo.MaxBy(logs, func(a, b ethtypes.Log) bool { return a.BlockNumber > b.BlockNumber })
		// The factory dates a pool at its Initialize event, older than any of its logs.
		stateless := factoryPool
		stateless.BlockNumber = lo.MinBy(logs, func(a, b ethtypes.Log) bool { return a.BlockNumber < b.BlockNumber }).
			BlockNumber - 1
		fresh, err := tracker.GetNewPoolState(ctx, stateless, poolpkg.GetNewPoolStateParams{
			Logs:         logs,
			BlockHeaders: map[uint64]entity.BlockHeader{last.BlockNumber: {}},
		})
		require.NoError(t, err)
		assert.True(t, tickspkg.HasState(fresh))
		assert.Equal(t, last.BlockNumber, fresh.BlockNumber)
		var freshExtra Extra
		require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
		assert.Equal(t, last.BlockNumber, freshExtra.LensBlock, "logs the snapshot holds would be read again")

		// The same logs again, as a retry delivers them, change nothing the snapshot holds.
		again, err := tracker.GetNewPoolState(ctx, fresh, poolpkg.GetNewPoolStateParams{Logs: logs})
		require.NoError(t, err)
		var againExtra Extra
		require.NoError(t, json.Unmarshal([]byte(again.Extra), &againExtra))
		assert.Equal(t, freshExtra.Ticks, againExtra.Ticks, "a replayed log changed the ticks")
		assert.Equal(t, freshExtra.LensBlock, againExtra.LensBlock)
	}

	t.Logf("%s: spacing %d, %d ticks at block %d; %d call(s) whole, %d under a 1M cap",
		id, whole.tickSpacing, len(whole.ticks), whole.block, wholeCalls, pagedCalls)
}

// newLivePool builds the pool the factory would from its Initialize event, out of the key and
// price CLPoolManager holds.
func newLivePool(t *testing.T, client *ethrpc.Client, cfg *Config, id string, block uint64) entity.Pool {
	var key shared.PoolKey
	var slot0 Slot0Data
	_, err := client.NewRequest().SetBlockNumber(new(big.Int).SetUint64(block)).AddCall(&ethrpc.Call{
		ABI: shared.CLPoolManagerABI, Target: cfg.CLPoolManagerAddress, Method: "poolIdToPoolKey",
		Params: []any{common.HexToHash(id)},
	}, []any{&key}).AddCall(&ethrpc.Call{
		ABI: shared.CLPoolManagerABI, Target: cfg.CLPoolManagerAddress, Method: shared.CLPoolManagerMethodGetSlot0,
		Params: []any{common.HexToHash(id)},
	}, []any{&slot0}).Aggregate()
	require.NoError(t, err)
	p, err := NewPoolFactory(cfg, client).newPool(&abi.PancakeInfinityPoolManagerInitialize{
		Id:           common.HexToHash(id),
		Currency0:    key.Currency0,
		Currency1:    key.Currency1,
		Hooks:        key.Hooks,
		Fee:          key.Fee,
		Parameters:   key.Parameters,
		SqrtPriceX96: slot0.SqrtPriceX96,
		Tick:         slot0.Tick,
	}, block)
	require.NoError(t, err)
	return *p
}

// recentLogs returns the pool's logs of a few recent blocks, the newest window holding any, read in
// windows small enough for any provider.
func recentLogs(t *testing.T, eth *ethclient.Client, manager, id string, head uint64) []ethtypes.Log {
	for to := head; to > head-2000; to -= 10 {
		logs, err := eth.FilterLogs(t.Context(), ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(to - 9),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: []common.Address{common.HexToAddress(manager)},
			Topics:    [][]common.Hash{nil, {common.HexToHash(id)}},
		})
		require.NoError(t, err)
		if len(logs) > 0 {
			return logs
		}
	}
	return nil
}
