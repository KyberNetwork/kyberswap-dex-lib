package liquiditybookv21

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TestLiveLens: on Avalanche and Arbitrum, for a v2.1 and a v2.2 pair each, a bootstrap through the
// lens holds every bin getBin reports for those ids at the same block and adds up to the reserves,
// a gas cap that forces paging reads the same set, a stored set missing a bin gets it back where
// reading the stored ids again cannot, and a pair without state gets its bins whole from its first
// update, a real recent log of its own. Run with either or both of:
//
//	LIQUIDITYBOOK_AVALANCHE_RPC_URL=https://... LIQUIDITYBOOK_ARBITRUM_RPC_URL=https://... \
//	  go test ./pkg/source/liquiditybookv21/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	pairs := []struct {
		env, name, address string
	}{
		{"LIQUIDITYBOOK_AVALANCHE_RPC_URL", "avalanche v2.1 WAVAX/USDC", "0xd446eb1660f766d533beceef890df7a69d26f7d1"},
		{"LIQUIDITYBOOK_AVALANCHE_RPC_URL", "avalanche v2.2 WAVAX/USDC", "0x864d4e5ee7318e97483db7eb0912e09f161516ea"},
		{"LIQUIDITYBOOK_ARBITRUM_RPC_URL", "arbitrum v2.1 WETH/USDC", "0x69f1216cb2905bf0852f74624d5fa7b5fc4da710"},
		{"LIQUIDITYBOOK_ARBITRUM_RPC_URL", "arbitrum v2.2 ARB/WETH", "0xc09f4ad33a164e29df3c94719ffd5f7b5b057781"},
	}
	ran := false
	for _, pc := range pairs {
		rpcURL := os.Getenv(pc.env)
		if rpcURL == "" {
			continue
		}
		ran = true
		t.Run(pc.name, func(t *testing.T) { checkLivePair(t, rpcURL, pc.address) })
	}
	if !ran {
		t.Skip("set LIQUIDITYBOOK_AVALANCHE_RPC_URL or LIQUIDITYBOOK_ARBITRUM_RPC_URL to run this")
	}
}

func checkLivePair(t *testing.T, rpcURL, address string) {
	ctx := t.Context()
	live := testutil.DialLive(t, rpcURL)
	eth := live.Eth
	client := live.Ethrpc
	// The guard's attempt timeout covers the pauses of a rate-limited shared endpoint.
	guard := ephemeral.GuardConfig{TimeoutSecs: 300}
	cfg := &Config{DexID: "liquiditybook-v21",
		LensConfig: ephemeral.LensConfig{UseLens: true, LensGuard: guard}}
	tracker := NewPoolTracker(cfg, client, nil)
	stateless := entity.Pool{Address: address, Exchange: "liquiditybook-v21", Type: DexTypeLiquidityBookV21,
		Reserves: entity.PoolReserves{"0", "0"}}

	p, err := tracker.BootstrapPoolState(ctx, stateless, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotEmpty(t, extra.Bins)
	require.NotZero(t, p.BlockNumber)
	head, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	assert.InDelta(t, float64(head), float64(p.BlockNumber), 1_000, "the lens block is not the chain's own")

	// Each bin against the pair's own answer for it, the reserves and the active bin against
	// FetchRPCData, all at the lens's block.
	viaRPC, err := tracker.queryRPCBins(ctx, address, lo.Map(extra.Bins, func(b Bin, _ int) uint32 { return b.ID }),
		p.BlockNumber)
	require.NoError(t, err)
	assertSameBins(t, extra.Bins, viaRPC, "getBin")
	rpcData, err := tracker.FetchRPCData(ctx, &p, p.BlockNumber)
	require.NoError(t, err)
	assert.Equal(t, rpcData.ActiveBinID, extra.ActiveBinID)
	assert.Equal(t, entity.PoolReserves{rpcData.Reserves.ReserveX.String(), rpcData.Reserves.ReserveY.String()},
		p.Reserves)
	require.NoError(t, tracker.validateBins(ctx, &p, extra.Bins))
	assert.True(t, lo.ContainsBy(extra.Bins, func(b Bin) bool { return b.ID == extra.ActiveBinID }),
		"the active bin holds liquidity on these pairs")

	block := new(big.Int).SetUint64(p.BlockNumber)
	before := live.Requests()
	whole, err := tracker.snapshot(ctx, address, block)
	require.NoError(t, err)
	wholeCalls := live.Requests() - before
	cfg.LensGas = 2_000_000
	before = live.Requests()
	paged, err := tracker.snapshot(ctx, address, block)
	require.NoError(t, err)
	pagedCalls := live.Requests() - before
	cfg.LensGas = 0
	assert.Greater(t, pagedCalls, wholeCalls, "the cap did not force paging")
	assertSameBins(t, whole.bins, paged.bins, "paging changed the set")

	// A stored set missing a bin, as logs can leave it: reading the stored ids again cannot find it,
	// the lens can.
	dropped := extra.Bins[len(extra.Bins)/2]
	missing := p
	missingExtra := extra
	missingExtra.Bins = lo.Reject(extra.Bins, func(b Bin, _ int) bool { return b.ID == dropped.ID })
	b, err := json.Marshal(missingExtra)
	require.NoError(t, err)
	missing.Extra = string(b)
	reread, err := tracker.fetchAllBins(ctx, &missing, nil, p.BlockNumber)
	require.NoError(t, err)
	assert.Error(t, tracker.validateBins(ctx, &missing, reread), "reading the stored ids found the bin")
	restored, err := tracker.refetchAllBins(ctx, &missing, nil, p.BlockNumber)
	require.NoError(t, err)
	assertSameBins(t, extra.Bins, restored, "the missing bin was not restored")

	// A pair without state, as a pair factory creates it, given a real log of its own: through the
	// lens it gets every bin at that log's block; without, only the bins the log names.
	swap, found := recentLog(t, ctx, eth, address, head)
	if !found {
		// No log of its own lately: the empty log that triggers an update, at the head.
		swap = ethtypes.Log{BlockNumber: head}
	}
	param := pool.GetNewPoolStateParams{
		Logs:         []ethtypes.Log{swap},
		BlockHeaders: map[uint64]entity.BlockHeader{swap.BlockNumber: {}},
	}
	// A tracker of its own: the guard keeps this pair cooling down after the read above.
	first := NewPoolTracker(&Config{DexID: "liquiditybook-v21",
		LensConfig: ephemeral.LensConfig{UseLens: true, LensGuard: guard}}, client, nil)
	fresh, err := first.GetNewPoolState(ctx, stateless, param)
	require.NoError(t, err)
	assert.True(t, hasState(fresh))
	assert.Equal(t, swap.BlockNumber, fresh.BlockNumber)
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
	require.NoError(t, tracker.validateBins(ctx, &fresh, freshExtra.Bins))

	plain := NewPoolTracker(&Config{DexID: "liquiditybook-v21"}, client, nil)
	old, err := plain.GetNewPoolState(ctx, stateless, param)
	require.NoError(t, err)
	var oldExtra Extra
	require.NoError(t, json.Unmarshal([]byte(old.Extra), &oldExtra))

	t.Logf("%d bins at block %d (active %d), %d calls whole, %d paged under 2M gas; a pair without state "+
		"given its log (found %v) at block %d: %d bins through the lens, %d without", len(whole.bins), whole.block,
		whole.activeID, wholeCalls, pagedCalls, found, swap.BlockNumber, len(freshExtra.Bins), len(oldExtra.Bins))
}

// assertSameBins compares bins by value: a zero reserve decodes with or without a backing slice.
func assertSameBins(t *testing.T, want, got []Bin, msg string) {
	t.Helper()
	require.Len(t, got, len(want), msg)
	for i := range want {
		if want[i].ID != got[i].ID || want[i].ReserveX.Cmp(got[i].ReserveX) != 0 ||
			want[i].ReserveY.Cmp(got[i].ReserveY) != 0 {
			t.Fatalf("%s: bin %d is %d (%s, %s), want %d (%s, %s)", msg, i, got[i].ID, got[i].ReserveX,
				got[i].ReserveY, want[i].ID, want[i].ReserveX, want[i].ReserveY)
		}
	}
}

// recentLog finds the newest log the pair emitted in the last 2000 blocks before head, reading ten
// blocks at a time, as some providers limit eth_getLogs to.
func recentLog(t *testing.T, ctx context.Context, eth *ethclient.Client, address string,
	head uint64) (ethtypes.Log, bool) {
	t.Helper()
	for to := head; to > head-2_000; to -= 10 {
		logs, err := eth.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(to - 9),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: []common.Address{common.HexToAddress(address)},
		})
		require.NoError(t, err)
		if len(logs) > 0 {
			return logs[len(logs)-1], true
		}
	}
	return ethtypes.Log{}, false
}
