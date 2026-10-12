package lotflow

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// liveMulticall is the ArbMulticall2-style multicall on Robinhood Chain (the address #1740's live tests
// configure); its getBlockNumber() is the L2 block number, unlike canonical Multicall3's (README).
const liveMulticall = "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1"

const tapeDir = "testdata/rpc"

// maxTrackRequests / maxTrackSeconds: the per-venue Track budget (HTTP requests incl. eth_blockNumber).
const (
	maxTrackRequests = 25
	maxTrackSeconds  = 10
)

func tapePath(poolID string) string { return filepath.Join(tapeDir, poolID[:18]+".json.gz") }

func venueParam(rc *ethrpc.Client, v struct{ name, hook, lot, poolID string }, block *big.Int) *uniswapv4.HookParam {
	usdgIs0 := addrLess(parityUsdg, v.lot)
	t0, t1 := lotUsdg(v.lot, usdgIs0)
	pool := &entity.Pool{Address: v.poolID, Tokens: []*entity.PoolToken{{Address: strings.ToLower(t0)},
		{Address: strings.ToLower(t1)}}}
	return &uniswapv4.HookParam{RpcClient: rc, Pool: pool, HookAddress: common.HexToAddress(v.hook), BlockNumber: block,
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDRobinhood}}
}

// trackDirect is Track as of cc383eb5: one eth_call per read (JSON-RPC batches of 25 for ticks).
// It is the reference the multicall path must reproduce.
func trackDirect(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	p := param.Pool
	ec := param.RpcClient.GetETHClient()
	block := hexutil.EncodeBig(param.BlockNumber)
	hook := hexutil.Encode(param.HookAddress[:])
	pmW, err := (&RPCLoader{c: ec.Client(), Block: block}).word(ctx, hook, "poolManager()")
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig("", "", FinalisedRules)
	l, err := NewRPCLoaderFromClient(ctx, ec.Client(), block, hook, wordAddr(pmW), cfg)
	if err != nil {
		return nil, err
	}
	var lot string
	for _, t := range p.Tokens {
		if !strings.EqualFold(t.Address, l.Usdg) {
			lot = t.Address
		}
	}
	id, err := l.word(ctx, hook, "venueOf(address)", encAddr(lot))
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(common.BigToHash(id).Hex(), p.Address) {
		return nil, ErrNotAVenue
	}
	tr := &Tracker{Cfg: cfg, Hook: hook, PoolManager: l.PM, NavGuard: l.NavGuard, Registry: l.Registry,
		VenueBook: l.VenueBook}
	return (&Hook{}).trackMarket(ctx, param, tr, l, lot, time.Now())
}

// comparable strips the per-run timestamp: everything else in the extra must be byte-identical.
func comparableExtra(t testing.TB, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &m))
	delete(m, "trackedAt")
	return m
}

func requireSameExtra(t testing.TB, name string, want, got json.RawMessage) {
	t.Helper()
	w, g := comparableExtra(t, want), comparableExtra(t, got)
	require.NotEqual(t, "null", string(w["market"]), name+": reference market is empty")
	require.Equal(t, len(w), len(g), name)
	for k := range w {
		require.Equal(t, string(w[k]), string(g[k]), "%s: extra field %q differs", name, k)
	}
}

// Offline (recorded live state): the multicall Track builds the byte-identical extra the per-call
// reference builds, inside the request budget.
func TestLoaderEquivalenceTape(t *testing.T) {
	for _, v := range liveVenues {
		t.Run(v.name, func(t *testing.T) {
			tape := loadTape(t, tapePath(v.poolID))
			var ctr rpcCounter
			srv := replayServer(t, tape, &ctr)
			defer srv.Close()
			rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
			block, err := hexutil.DecodeBig(tape.Block)
			require.NoError(t, err)
			ctx := context.Background()

			ctr.reset()
			want, err := trackDirect(ctx, venueParam(rc, v, block))
			require.NoError(t, err)
			t.Logf("per-call reference: %s", &ctr)

			ctr.reset()
			got, err := (&Hook{}).Track(ctx, venueParam(rc, v, block))
			require.NoError(t, err)
			t.Logf("multicall Track:    %s", &ctr)
			requireSameExtra(t, v.name, want, got)
		})
	}
}

// F1: one Track stays within the request budget (offline: recorded state, resolving the block).
func TestTrackRPCBudget(t *testing.T) {
	defer logTrackStats(t)()
	for _, v := range liveVenues {
		t.Run(v.name, func(t *testing.T) {
			tape := loadTape(t, tapePath(v.poolID))
			var ctr rpcCounter
			srv := replayServer(t, tape, &ctr)
			defer srv.Close()
			rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
			ctr.reset()
			param := venueParam(rc, v, nil) // Track resolves the block (eth_blockNumber) itself
			_, err := (&Hook{}).Track(context.Background(), param)
			require.NoError(t, err)
			t.Logf("%s: %s", v.name, &ctr)
			require.LessOrEqual(t, ctr.Requests, maxTrackRequests, "HTTP requests per Track")
		})
	}
}

func logTrackStats(t testing.TB) func() {
	trackStats = func(s MulticallStats) { t.Logf("  multicall: %+v", s) }
	return func() { trackStats = nil }
}

// Live, read-only (eth_call at one pinned block): the per-call reference and the multicall Track build
// byte-identical extras; requests, JSON-RPC calls and wall time of each are logged. With
// LOTFLOW_RECORD=1 the reads of both are saved as the offline tapes.
func TestLoaderEquivalenceLive(t *testing.T) {
	url := os.Getenv("LOTFLOW_RPC")
	if url == "" {
		t.Skip("LOTFLOW_RPC not set (live chain-4663 loader equivalence)")
	}
	record := os.Getenv("LOTFLOW_RECORD") == "1"
	ctx := context.Background()
	rc0, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	var bn hexutil.Big
	require.NoError(t, rc0.CallContext(ctx, &bn, "eth_blockNumber"))
	block := new(big.Int).Sub(bn.ToInt(), big.NewInt(2))
	if b := os.Getenv("LOTFLOW_BLOCK"); b != "" {
		block, _ = new(big.Int).SetString(b, 0)
	}
	t.Logf("pinned block %s", block)
	defer logTrackStats(t)()
	for _, v := range liveVenues {
		tape := &rpcTape{Block: hexutil.EncodeBig(block), Calls: map[string]tapeEntry{}}
		var ctr rpcCounter
		srv := recordingProxy(t, url, tape, &ctr)
		rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))

		ctr.reset()
		start := time.Now()
		want, err := trackDirect(ctx, venueParam(rc, v, block))
		require.NoError(t, err, v.name)
		t.Logf("%s per-call reference: %s, %.1fs", v.name, &ctr, time.Since(start).Seconds())

		ctr.reset()
		start = time.Now()
		h := &Hook{}
		got, err := h.Track(ctx, venueParam(rc, v, block))
		el := time.Since(start)
		require.NoError(t, err, v.name)
		t.Logf("%s multicall Track:    %s, %.1fs", v.name, &ctr, el.Seconds())
		requireSameExtra(t, v.name, want, got)
		t.Logf("%s: extras byte-identical (%d bytes, trackedAt excluded)", v.name, len(got))
		srv.Close()
		require.LessOrEqual(t, ctr.Requests, maxTrackRequests, v.name)
		require.LessOrEqual(t, el.Seconds(), float64(maxTrackSeconds), v.name)
		if record {
			tape.save(t, tapePath(v.poolID))
			t.Logf("%s: recorded %d reads to %s", v.name, len(tape.Calls), tapePath(v.poolID))
		}
	}
}

// F-N6: a Track whose collect passes stopped predicting its reads (misses) or that re-read failed
// sub-calls (debug) is reported in production, not only through the trackStats test hook.
func TestMulticallStatsWarning(t *testing.T) {
	require.Empty(t, statsWarning(MulticallStats{Rounds: 7, Requests: 9, Calls: 983}))
	require.Contains(t, statsWarning(MulticallStats{Misses: 3}), "3 reads not predicted")
	require.Empty(t, statsWarning(MulticallStats{Rechecks: 2})) // every live Track re-reads its real reverts
	// the recorded live Tracks: rechecks happen (real reverting admission reads), misses do not
	for _, v := range liveVenues {
		var got MulticallStats
		trackStats = func(s MulticallStats) { got = s }
		_, _, err, _ := tapeTrack(t, v, replayFaults{})
		trackStats = nil
		require.NoError(t, err)
		require.Zero(t, got.Misses, v.name)
		require.Positive(t, got.Rechecks, v.name)
	}
}
