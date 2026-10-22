package liquiditybookv20

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// countingTransport counts the HTTP requests a client makes that the node answered, and sends a
// request the node rate-limited again, after a pause, so a shared endpoint's limit does not fail
// the test.
type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		req.Body = io.NopCloser(bytes.NewReader(body))
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 20 {
			c.n.Add(1)
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
	}
}

// TestLiveLens: on Avalanche, Arbitrum and Base, for two v2.0 pairs each, a bootstrap through the
// lens holds every bin getBin reports for those ids at the same block and adds up to the reserves,
// a gas cap that forces paging reads the same set, a stored set missing a bin gets it back where
// reading the stored ids again cannot, and a pair without state gets its bins whole from its first
// update. Each chain runs with the multicall its production config names. Run with any of:
//
//	LIQUIDITYBOOK_V20_AVALANCHE_RPC_URL=https://... LIQUIDITYBOOK_V20_ARBITRUM_RPC_URL=https://... \
//	LIQUIDITYBOOK_V20_BASE_RPC_URL=https://... go test ./pkg/source/liquiditybookv20/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	const multicall3, arbMulticall2 = "0xcA11bde05977b3631167028862bE2a173976CA11",
		"0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF"
	pairs := []struct {
		env, multicall, name, address string
	}{
		{"LIQUIDITYBOOK_V20_AVALANCHE_RPC_URL", multicall3, "avalanche traderjoe-v20 a",
			"0xc01961ede437bf0cc41d064b1a3f6f0ea6aa2a40"},
		{"LIQUIDITYBOOK_V20_AVALANCHE_RPC_URL", multicall3, "avalanche traderjoe-v20 b",
			"0xb5352a39c11a81fe6748993d586ec448a01f08b5"},
		{"LIQUIDITYBOOK_V20_ARBITRUM_RPC_URL", arbMulticall2, "arbitrum traderjoe-v20 a",
			"0x7ec3717f70894f6d9ba0be00774610394ce006ee"},
		{"LIQUIDITYBOOK_V20_ARBITRUM_RPC_URL", arbMulticall2, "arbitrum traderjoe-v20 b",
			"0xa51ee8b744e6cc1f2ac12b9eeaae8deb27619c6b"},
		{"LIQUIDITYBOOK_V20_BASE_RPC_URL", multicall3, "base sectorone-v20 a", "0x253f975e354a9910ef5514f8e73f2e94fa6603cc"},
		{"LIQUIDITYBOOK_V20_BASE_RPC_URL", multicall3, "base sectorone-v20 b", "0xbed7d791b4dfd69f2ed2b0124319967ee24b9329"},
	}
	ran := false
	for _, pc := range pairs {
		rpcURL := os.Getenv(pc.env)
		if rpcURL == "" {
			continue
		}
		ran = true
		t.Run(pc.name, func(t *testing.T) { checkLivePair(t, rpcURL, pc.multicall, pc.address) })
	}
	if !ran {
		t.Skip("set LIQUIDITYBOOK_V20_{AVALANCHE,ARBITRUM,BASE}_RPC_URL to run this")
	}
}

func checkLivePair(t *testing.T, rpcURL, multicall, address string) {
	ctx := t.Context()
	counter := &countingTransport{}
	rc, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	eth := ethclient.NewClient(rc)
	client := ethrpc.NewWithClient(eth).SetMulticallContract(common.HexToAddress(multicall))
	// The guard's attempt timeout covers the pauses of a rate-limited shared endpoint.
	guard := ephemeral.GuardConfig{TimeoutSecs: 300}
	cfg := &Config{DexID: "liquiditybook-v20", UseLens: true, LensGuard: guard}
	tracker := NewPoolTracker(cfg, client, nil)
	stateless := entity.Pool{Address: address, Exchange: "liquiditybook-v20", Type: DexTypeLiquidityBookV20,
		Reserves: entity.PoolReserves{"0", "0"}}

	head, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	p, err := tracker.BootstrapPoolState(ctx, stateless, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotEmpty(t, extra.Bins)
	assert.GreaterOrEqual(t, p.BlockNumber, head, "the lens block is not the chain's own")
	assert.Less(t, p.BlockNumber, head+1_000, "the lens block is not the chain's own")

	// Each bin against the pair's own answer for it, the reserves and the active bin against
	// FetchRPCData, all at the lens's block.
	viaRPC, err := tracker.queryRPCBins(ctx, address, lo.Map(extra.Bins, func(b Bin, _ int) uint32 { return b.ID }),
		p.BlockNumber)
	require.NoError(t, err)
	assertSameBins(t, extra.Bins, viaRPC, "getBin")
	rpcData, err := tracker.FetchRPCData(ctx, &p, p.BlockNumber)
	require.NoError(t, err)
	assert.EqualValues(t, rpcData.ReservesAndID.ActiveId.Uint64(), extra.ActiveBinID)
	assert.Equal(t, entity.PoolReserves{rpcData.ReservesAndID.ReserveX.String(), rpcData.ReservesAndID.ReserveY.String()},
		p.Reserves)
	assert.Equal(t, rpcData.FeeParameters, extra.FeeParameters)
	require.NoError(t, tracker.validateBins(&p, extra.Bins))

	block := new(big.Int).SetUint64(p.BlockNumber)
	before := counter.n.Load()
	whole, err := tracker.snapshot(ctx, address, block)
	require.NoError(t, err)
	wholeCalls := counter.n.Load() - before
	cfg.LensGas = 400_000
	before = counter.n.Load()
	paged, err := tracker.snapshot(ctx, address, block)
	require.NoError(t, err)
	pagedCalls := counter.n.Load() - before
	cfg.LensGas = 0
	assert.Equal(t, p.BlockNumber, whole.block, "a read pinned to a block reports that block")
	if len(whole.bins) > 30 {
		assert.Greater(t, pagedCalls, wholeCalls, "the cap did not force paging")
	}
	assertSameBins(t, whole.bins, paged.bins, "paging changed the set")

	// A stored set missing a bin, as logs can leave it: reading the stored ids again cannot find it,
	// the lens can.
	if len(extra.Bins) > 1 {
		dropped := extra.Bins[len(extra.Bins)/2]
		missing := p
		missingExtra := extra
		missingExtra.Bins = lo.Reject(extra.Bins, func(b Bin, _ int) bool { return b.ID == dropped.ID })
		b, err := json.Marshal(missingExtra)
		require.NoError(t, err)
		missing.Extra = string(b)
		reread, err := tracker.fetchAllBins(ctx, &missing, nil, p.BlockNumber)
		require.NoError(t, err)
		assert.Error(t, tracker.validateBins(&missing, reread), "reading the stored ids found the bin")
		restored, err := tracker.refetchAllBins(ctx, &missing, nil, p.BlockNumber)
		require.NoError(t, err)
		assertSameBins(t, extra.Bins, restored, "the missing bin was not restored")
	}

	// A pair without state, as a pair factory creates it, given a real log of its own: through the
	// lens it gets every bin at that log's block; without, only the bins the log names.
	trigger, found := recentLog(t, ctx, eth, address, head)
	if !found {
		// No log of its own lately: the empty log that triggers an update, at the head.
		trigger = ethtypes.Log{BlockNumber: head}
	}
	param := pool.GetNewPoolStateParams{
		Logs:         []ethtypes.Log{trigger},
		BlockHeaders: map[uint64]entity.BlockHeader{trigger.BlockNumber: {}},
	}
	// A tracker of its own: the guard keeps this pair cooling down after the reads above.
	first := NewPoolTracker(&Config{DexID: "liquiditybook-v20", UseLens: true, LensGuard: guard}, client, nil)
	fresh, err := first.GetNewPoolState(ctx, stateless, param)
	require.NoError(t, err)
	assert.True(t, hasState(fresh))
	assert.Equal(t, trigger.BlockNumber, fresh.BlockNumber)
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
	require.NoError(t, tracker.validateBins(&fresh, freshExtra.Bins))

	plain := NewPoolTracker(&Config{DexID: "liquiditybook-v20"}, client, nil)
	old, err := plain.GetNewPoolState(ctx, stateless, param)
	require.NoError(t, err)
	var oldExtra Extra
	require.NoError(t, json.Unmarshal([]byte(old.Extra), &oldExtra))

	t.Logf("%d bins at block %d (active %d), %d calls whole, %d paged under 400k gas; a pair without state "+
		"given its log (found %v) at block %d: %d bins through the lens, %d without", len(whole.bins), whole.block,
		whole.activeID, wholeCalls, pagedCalls, found, trigger.BlockNumber, len(freshExtra.Bins), len(oldExtra.Bins))
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
