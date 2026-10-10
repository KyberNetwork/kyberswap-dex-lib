package lotflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// F2: a read that REVERTS on chain (feed, float, MQ feeds, NAV inputs) publishes the fail-closed state
// (Tracked, no Market: reserves 0, quotes refused), as #1740 publishes empty ladders (16ac2b74); a
// TRANSPORT failure returns the error and publishes nothing. AUDIT's repro, adopted.

type failingLoader struct {
	fixtureLoader
	feed, float error
}

func (l *failingLoader) Feed(ctx context.Context, stock string) (string, *FeedState, error) {
	if l.feed != nil {
		return "", nil, l.feed
	}
	return l.fixtureLoader.Feed(ctx, stock)
}

func (l *failingLoader) PoolManagerLotBalance(ctx context.Context, lot string) (*big.Int, error) {
	if l.float != nil {
		return nil, l.float
	}
	return l.fixtureLoader.PoolManagerLotBalance(ctx, lot)
}

var (
	errRevertRead = fmt.Errorf("%w: feed 0x117cc2133c37B721F49dE2A7a74833232B3B4C0C", ErrReverted)
	errTransport  = &TransportError{Err: errors.New("dial tcp: i/o timeout")}
)

func requireFailClosed(t *testing.T, h *Hook, raw json.RawMessage, err error, name string) {
	t.Helper()
	require.NoError(t, err, name)
	var published Extra
	require.NoError(t, json.Unmarshal(raw, &published), name)
	require.True(t, published.Tracked, name)
	require.Nil(t, published.Market, name)
	_, qerr := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true,
		AmountSpecified: big.NewInt(1_000_000_000)})
	require.ErrorIs(t, qerr, ErrNotTracked, name+": the previous state must not stay quotable")
	reserves, err := h.GetReserves(context.Background(), nil)
	require.NoError(t, err, name)
	require.Equal(t, entity.PoolReserves{"0", "0"}, reserves, name)
}

func TestTrackReadRevertPublishesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name        string
		feed, float error
	}{{"control", nil, nil}, {"feed revert", errRevertRead, nil}, {"float revert", nil, errRevertRead}} {
		h, c := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
		raw, err := h.trackMarket(context.Background(), &uniswapv4.HookParam{BlockNumber: big.NewInt(123)},
			trackerOf(c), &failingLoader{fixtureLoader{c}, tc.feed, tc.float}, c.Pre.Lot.Address, time.Now())
		if tc.feed == nil && tc.float == nil {
			require.NoError(t, err)
			require.NotNil(t, h.Market, "control publishes a market")
			continue
		}
		requireFailClosed(t, h, raw, err, tc.name)
	}
}

func TestTrackTransportErrorPublishesNothing(t *testing.T) {
	for _, tc := range []struct {
		name        string
		feed, float error
	}{{"feed transport", errTransport, nil}, {"float transport", nil, errTransport}} {
		h, c := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
		before := h.Market
		raw, err := h.trackMarket(context.Background(), &uniswapv4.HookParam{BlockNumber: big.NewInt(123)},
			trackerOf(c), &failingLoader{fixtureLoader{c}, tc.feed, tc.float}, c.Pre.Lot.Address, time.Now())
		require.ErrorIs(t, err, errTransport, tc.name)
		require.Nil(t, raw, tc.name)
		require.Same(t, before, h.Market, tc.name+": nothing installed")
	}
}

var (
	selLatestRound = sel("latestRoundData()")
	selBalanceOf   = sel("balanceOf(address)")
)

//nolint:staticcheck // ST1008: Track's hook, extra and error come first; test instrumentation follows.
func tapeTrack(t *testing.T, v struct{ name, hook, lot, poolID string }, f replayFaults) (*Hook, json.RawMessage,
	error, *rpcCounter) {
	t.Helper()
	h, raw, err, ctr, _ := tapeTrackTimed(t, v, f)
	return h, raw, err, ctr
}

// tapeTrackTimed is tapeTrack, also returning the instant just before Track was called.
//
//nolint:staticcheck // ST1008: same result order as tapeTrack.
func tapeTrackTimed(t *testing.T, v struct{ name, hook, lot, poolID string }, f replayFaults) (*Hook,
	json.RawMessage, error, *rpcCounter, time.Time) {
	t.Helper()
	tape := loadTape(t, tapePath(v.poolID))
	ctr := &rpcCounter{}
	srv := replayServerWith(t, tape, ctr, f)
	t.Cleanup(srv.Close)
	block, err := hexutil.DecodeBig(tape.Block)
	require.NoError(t, err)
	h := &Hook{}
	param := venueParam(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), v, block)
	start := time.Now()
	raw, err := h.Track(context.Background(), param)
	return h, raw, err, ctr, start
}

// Recorded live state, real Track: every feed (incl. the MQ ETH/USD and USDG/USD feeds) or the LOT
// float reverting publishes the fail-closed state.
func TestTrackTapeRevertFailClosed(t *testing.T) {
	for _, v := range liveVenues {
		for _, tc := range []struct {
			name   string
			revert func(to string, data []byte) bool
		}{
			{"feed", func(_ string, d []byte) bool { return bytes.HasPrefix(d, selLatestRound) }},
			{"float", func(_ string, d []byte) bool {
				return bytes.HasPrefix(d, selBalanceOf) && bytes.Contains(d, common.HexToAddress(livePM).Bytes())
			}},
		} {
			var hits atomic.Int64
			f := replayFaults{revert: func(to string, d []byte) bool {
				if tc.revert(to, d) {
					hits.Add(1)
					return true
				}
				return false
			}}
			h, raw, err, _ := tapeTrack(t, v, f)
			require.Positive(t, hits.Load(), "%s %s: the fault must hit a recorded read", v.name, tc.name)
			requireFailClosed(t, h, raw, err, v.name+" "+tc.name)
		}
	}
}

// A transport failure mid-Track returns the error and publishes nothing.
func TestTrackTapeTransportError(t *testing.T) {
	for _, v := range liveVenues {
		for _, at := range []int{1, 3, 5} {
			h, raw, err, _ := tapeTrack(t, v, replayFaults{httpFail: func(n int) bool { return n >= at }})
			require.Error(t, err, "%s: request %d fails", v.name, at)
			require.True(t, isTransport(err), "%s: %v", v.name, err)
			require.Nil(t, raw)
			require.False(t, h.Tracked)
		}
	}
}

// F9: admission reads (oraclePaused, venuesOf, factory/slot0/liquidity/tickSpacing) treat a revert as
// "read failed" (the READ_GAS semantics) but return a transport error instead of dropping the read.
func TestStockAdmissionReadClassification(t *testing.T) {
	v := liveVenues[0]
	tape := loadTape(t, tapePath(v.poolID))
	direct := func(f replayFaults) (*RPCLoader, []string) {
		srv := replayServerWith(t, tape, &rpcCounter{}, f)
		t.Cleanup(srv.Close)
		rc := ethrpc.New(srv.URL)
		ctx := context.Background()
		hook := strings.ToLower(v.hook)
		pm, err := (&RPCLoader{c: rc.GetETHClient().Client(), Block: tape.Block}).word(ctx, hook, "poolManager()")
		require.NoError(t, err)
		l, err := NewRPCLoaderFromClient(ctx, rc.GetETHClient().Client(), tape.Block, hook, wordAddr(pm),
			DefaultConfig("", "", FinalisedRules))
		require.NoError(t, err)
		stocks, _, err := l.Constituents(ctx, v.lot)
		require.NoError(t, err)
		return l, stocks
	}
	selFactory := sel("factory()")
	isFactory := func(_ string, d []byte) bool { return bytes.Equal(d, selFactory) }

	l, stocks := direct(replayFaults{})
	booked := 0
	for _, s := range stocks {
		snap, err := l.Stock(context.Background(), s)
		require.NoError(t, err)
		for _, bv := range snap.Venues {
			require.True(t, bv.FactoryOK, "control: recorded factory() reads succeed")
			booked++
		}
	}
	require.Positive(t, booked, "the venue's stocks have VenueBook sources")

	l, stocks = direct(replayFaults{revert: isFactory})
	for _, s := range stocks {
		snap, err := l.Stock(context.Background(), s)
		require.NoError(t, err, "a reverting admission read is a failed read, not an error")
		for _, bv := range snap.Venues {
			require.False(t, bv.FactoryOK)
		}
	}

	l, stocks = direct(replayFaults{nodeError: isFactory})
	failed := 0
	for _, s := range stocks {
		if _, err := l.Stock(context.Background(), s); err != nil {
			require.True(t, isTransport(err), "%v", err)
			failed++
		}
	}
	require.Positive(t, failed, "a transport error on an admission read must not silently drop the source")
}

// F3: TrackedAt is the time Track STARTED (before its reads), so the StaleCheck age of a state includes
// the time its reads took. Track on recorded state over a slow RPC (250 ms per request).
func TestTrackedAtIsTrackStart(t *testing.T) {
	v := liveVenues[1]
	h, _, err, ctr, start := tapeTrackTimed(t, v, replayFaults{delay: 250 * time.Millisecond})
	require.NoError(t, err)
	took := time.Since(start)
	require.Greater(t, took, 2*time.Second, "instrument: the slow RPC makes Track take seconds (%d requests)",
		ctr.Requests)
	require.LessOrEqual(t, h.TrackedAt, start.Unix(), "TrackedAt must not postdate the start of the reads")
	age := time.Now().Unix() - h.TrackedAt
	require.GreaterOrEqual(t, age, int64(took/time.Second), "StaleCheck age includes the Track duration")
	t.Logf("Track took %s over %d requests; TrackedAt = start (%d), age at return %d s", took, ctr.Requests,
		h.TrackedAt, age)
}

// Chain gate (#1740 navjit hook.go QuoterByChain; evplusai hook.go): LOTFLOW tracks and quotes only on
// Robinhood Chain (4663). Off-chain Track refuses before any read; an extra handed to the factory on
// another chain is untracked.
func TestChainGate(t *testing.T) {
	v := liveVenues[0]
	tape := loadTape(t, tapePath(v.poolID))
	var ctr rpcCounter
	srv := replayServer(t, tape, &ctr)
	defer srv.Close()
	block, err := hexutil.DecodeBig(tape.Block)
	require.NoError(t, err)
	rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
	param := venueParam(rc, v, block)
	raw, err := (&Hook{}).Track(context.Background(), param)
	require.NoError(t, err, "control: chain 4663")

	for _, cfg := range []*uniswapv4.Config{nil, {ChainID: valueobject.ChainIDEthereum}, {ChainID: valueobject.ChainIDBase}} {
		ctr.reset()
		p := venueParam(rc, v, block)
		p.Cfg = cfg
		_, err := (&Hook{}).Track(context.Background(), p)
		require.ErrorIs(t, err, ErrUnsupportedChain, "%+v", cfg)
		require.Zero(t, ctr.Requests, "refused before any read")
	}
	for _, chain := range []valueobject.ChainID{valueobject.ChainIDEthereum, valueobject.ChainIDBase} {
		h := uniswapv4.HookFactories[HookAddresses[0]](&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw),
			Cfg: &uniswapv4.Config{ChainID: chain}}).(*Hook)
		require.False(t, h.Tracked, "chain %d", chain)
	}
	h := uniswapv4.HookFactories[HookAddresses[0]](&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw),
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDRobinhood}}).(*Hook)
	require.True(t, h.Tracked, "chain 4663")
}

// F-N3: a caller deadline that lands during the 429 back-off is a transport failure (Track returns it
// and publishes nothing), never an unclassified error that Track would publish as an empty market.
func TestCtxDeadlineDuring429BackoffIsTransport(t *testing.T) {
	var n int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&n, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Rate Limit"))
	}))
	defer srv.Close()
	m := newMulticaller(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), big.NewInt(1))
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_, err := m.get(ctx, "0x0000000000000000000000000000000000000001", sel("foo()"))
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.True(t, isTransport(err), "%v (%T) after %d requests", err, err, atomic.LoadInt64(&n))
	require.NotErrorIs(t, err, ErrReverted)
}

// A load cut short by the caller's context publishes nothing, whatever error the loader returned.
func TestTrackCancelledContextPublishesNothing(t *testing.T) {
	h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	before := h.Extra
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw, err := h.trackMarket(ctx, &uniswapv4.HookParam{}, &Tracker{Cfg: h.cfg()}, ctxLoader{}, h.Lot, time.Now())
	require.Nil(t, raw)
	require.True(t, isTransport(err), "%v", err)
	require.Equal(t, before, h.Extra)
	// control: the same unclassified error under a live context is the chain's answer (fail-closed)
	raw, err = h.trackMarket(context.Background(), &uniswapv4.HookParam{}, &Tracker{Cfg: h.cfg()}, ctxLoader{},
		h.Lot, time.Now())
	require.NoError(t, err)
	require.NotNil(t, raw)
	require.True(t, h.Tracked)
	require.Nil(t, h.Market)
}

// ctxLoader fails every read with a plain (unclassified) error, as a loader would when its context ends.
type ctxLoader struct{ Loader }

func (ctxLoader) Constituents(context.Context, string) ([]string, []*big.Int, error) {
	return nil, nil, errors.New("read aborted")
}

// Item 7: pool-service pins Track at the block number its multicall returns (uniswap/v4
// pool_tracker.go:117). A multicall contract reporting the EVM block.number on 4663 (the L1 number,
// canonical Multicall3) would pin an old L2 block; on an archive node Track would then publish weeks-old
// state stamped with the current TrackedAt. Track refuses a pinned block more than MaxAgeSec older than
// the chain head, and publishes nothing.
func TestTrackRefusesStalePin(t *testing.T) {
	for _, v := range liveVenues {
		h, raw, err, _ := tapeTrack(t, v, replayFaults{headAhead: MaxAgeSec + 1})
		require.ErrorIs(t, err, ErrStalePin, v.name)
		require.Nil(t, raw, v.name)
		require.False(t, h.Tracked, v.name)
		// control: a head within MaxAgeSec of the pin tracks
		h, raw, err, _ = tapeTrack(t, v, replayFaults{headAhead: MaxAgeSec})
		require.NoError(t, err, v.name)
		require.NotNil(t, raw, v.name)
		require.True(t, h.Tracked && h.Market != nil, v.name)
	}
}

// F-N5: a sub-call that fails inside tryAggregate is not necessarily an on-chain revert (a sub-call
// starved of gas in a large batch fails too). Failed sub-calls are re-read one by one; only a plain
// eth_call that reverts counts as a revert. A batch-only failure must not change the published state.
func TestBatchOnlyFailureIsRechecked(t *testing.T) {
	for _, v := range liveVenues {
		_, want, err, _ := tapeTrack(t, v, replayFaults{})
		require.NoError(t, err)
		var hits atomic.Int64
		h, got, err, _ := tapeTrack(t, v, replayFaults{batchOOG: func(_ string, d []byte) bool {
			if bytes.HasPrefix(d, selLatestRound) {
				hits.Add(1)
				return true
			}
			return false
		}})
		require.Positive(t, hits.Load(), v.name)
		require.NoError(t, err, v.name)
		require.NotNil(t, h.Market, "%s: a batch-only failure published the fail-closed state", v.name)
		require.Equal(t, stripTrackedAt(t, want), stripTrackedAt(t, got), v.name)
		// the recheck itself failing at the node is a transport error: nothing is published
		feed := func(_ string, d []byte) bool { return bytes.HasPrefix(d, selLatestRound) }
		_, raw, err, _ := tapeTrack(t, v, replayFaults{batchOOG: feed, nodeError: feed})
		require.True(t, isTransport(err), "%s: %v", v.name, err)
		require.Nil(t, raw, v.name)
	}
}

func stripTrackedAt(t *testing.T, raw json.RawMessage) string {
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	delete(m, "trackedAt")
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return string(b)
}
