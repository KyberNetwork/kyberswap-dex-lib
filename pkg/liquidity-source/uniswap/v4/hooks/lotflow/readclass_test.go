package lotflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// ADV2-3: the read-classification paths, each pinned by a test (the mutants that survived the suite).

// rpcCodeErr is a JSON-RPC error as go-ethereum's client surfaces it (rpc.Error).
type rpcCodeErr struct {
	code int
	msg  string
}

func (e rpcCodeErr) Error() string  { return e.msg }
func (e rpcCodeErr) ErrorCode() int { return e.code }

// classifyRPC: a node reports a revert as code 3 (geth, revert data present) or as -32000 "execution
// reverted" (geth, no revert data); everything else is a transport failure.
func TestClassifyRPC(t *testing.T) {
	for _, tc := range []struct {
		err    error
		revert bool
	}{
		{rpcCodeErr{3, "execution reverted: NotProfitable"}, true},
		{rpcCodeErr{3, "VM execution error"}, true}, // the code alone decides
		{rpcCodeErr{-32000, "execution reverted"}, true},
		{errors.New("execution reverted"), true},
		{rpcCodeErr{-32000, "upstream request timeout"}, false},
		{rpcCodeErr{-32005, "429 Too Many Requests"}, false},
		{errors.New("dial tcp 127.0.0.1:1: connect: connection refused"), false},
	} {
		got := classifyRPC(tc.err)
		require.Error(t, got, tc.err.Error())
		require.Equal(t, tc.revert, errors.Is(got, ErrReverted), "%q", tc.err)
		require.Equal(t, !tc.revert, isTransport(got), "%q", tc.err)
	}
	require.NoError(t, classifyRPC(nil))
}

// errorString is Solidity's Error(string) revert payload: 100 bytes, enough to decode as words.
func errorString(msg string) []byte {
	out := append([]byte{}, sel("Error(string)")...)
	out = append(out, encWord(big.NewInt(32))...)
	out = append(out, encWord(big.NewInt(int64(len(msg))))...)
	pad := make([]byte, (len(msg)+31)/32*32)
	copy(pad, msg)
	return append(out, pad...)
}

// A reverted sub-call is never decoded as data, even when its revert payload is long enough to read
// as words: both the multicall path and a word read refuse it as ErrReverted.
func TestRevertedSubCallIsNeverData(t *testing.T) {
	to := "0x00000000000000000000000000000000000000AA"
	bad, good := sel("latestRoundData()"), sel("decimals()")
	tape := &rpcTape{Block: "0x1", Calls: map[string]tapeEntry{
		mcKey(to, bad):  {OK: false, Ret: errorString("feed is down, not a price")},
		mcKey(to, good): {OK: true, Ret: encWord(big.NewInt(8))},
	}}
	srv := replayServer(t, tape, &rpcCounter{})
	defer srv.Close()
	rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
	ctx := context.Background()
	m := newMulticaller(rc, big.NewInt(1))
	require.NoError(t, m.fetch(ctx, []*callReq{{to: to, data: bad}, {to: to, data: good}}))
	out, err := m.get(ctx, to, bad)
	require.ErrorIs(t, err, ErrReverted)
	require.Nil(t, out)
	out, err = m.get(ctx, to, good) // control: the successful sub-call of the same request is data
	require.NoError(t, err)
	require.Equal(t, encWord(big.NewInt(8)), out)

	l := &RPCLoader{c: rc.GetETHClient().Client(), Block: "0x1", mc: m}
	_, err = l.word(ctx, to, "latestRoundData()")
	require.ErrorIs(t, err, ErrReverted)
	require.False(t, isTransport(err))
}

// A rate-limited multicall request (aggregate) and a rate-limited recheck are retried, not failed.
func TestRateLimitedRequestsAreRetried(t *testing.T) {
	to := "0x00000000000000000000000000000000000000AA"
	data := sel("decimals()")
	tape := &rpcTape{Block: "0x1", Calls: map[string]tapeEntry{mcKey(to, data): {OK: true, Ret: encWord(big.NewInt(6))}}}
	for _, tc := range []struct {
		name     string
		limited  int  // the HTTP request answered 429
		batchOOG bool // the sub-call fails inside tryAggregate, so request 2 is the recheck
	}{{"aggregate", 1, false}, {"recheck", 2, true}} {
		var limited atomic.Int64
		f := replayFaults{rateLimit: func(n int) bool {
			if n == tc.limited {
				limited.Add(1)
				return true
			}
			return false
		}}
		if tc.batchOOG {
			f.batchOOG = func(string, []byte) bool { return true }
		}
		ctr := &rpcCounter{}
		srv := replayServerWith(t, tape, ctr, f)
		m := newMulticaller(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), big.NewInt(1))
		out, err := m.get(context.Background(), to, data)
		srv.Close()
		require.Equal(t, int64(1), limited.Load(), "%s: the 429 was served", tc.name)
		require.NoError(t, err, tc.name)
		require.Equal(t, encWord(big.NewInt(6)), out, tc.name)
	}
}

// Track's own block-number read failing is a transport error: nothing is published.
func TestTrackBlockNumberTransportError(t *testing.T) {
	v := liveVenues[0]
	tape := loadTape(t, tapePath(v.poolID))
	srv := replayServerWith(t, tape, &rpcCounter{}, replayFaults{httpFail: func(n int) bool { return n == 1 }})
	defer srv.Close()
	h := &Hook{}
	raw, err := h.Track(context.Background(),
		venueParam(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), v, nil))
	require.Error(t, err)
	require.True(t, isTransport(err), "%v (%T)", err, err)
	require.Nil(t, raw)
	require.False(t, h.Tracked)
}

// directLoader is the per-call RPCLoader of a recorded venue (the path the admission reads share with
// pool-service tooling), under faults.
func directLoader(t *testing.T, v struct{ name, hook, lot, poolID string }, f replayFaults) *RPCLoader {
	tape := loadTape(t, tapePath(v.poolID))
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
	return l
}

// F9 for the two optional reads: oraclePaused() and an MQ token pool's factory() revert as "read
// failed" (READ_GAS semantics), but a transport error on them is returned, never dropped.
func TestOptionalReadsReturnTransportErrors(t *testing.T) {
	ctx := context.Background()
	paused := func(_ string, d []byte) bool { return bytes.Equal(d, sel("oraclePaused()")) }
	v := liveVenues[0]
	stocks, _, err := directLoader(t, v, replayFaults{}).Constituents(ctx, v.lot)
	require.NoError(t, err)
	_, err = directLoader(t, v, replayFaults{revert: paused}).Stock(ctx, stocks[0])
	require.NoError(t, err, "control: a reverting oraclePaused() is a failed read")
	_, err = directLoader(t, v, replayFaults{nodeError: paused}).Stock(ctx, stocks[0])
	require.True(t, isTransport(err), "oraclePaused transport: %v", err)

	mq := liveVenues[1]
	tokenPool := func(f replayFaults) (*RPCLoader, string, *MqMarket) {
		l := directLoader(t, mq, f)
		q, err := l.MqMarket(ctx)
		require.NoError(t, err)
		stocks, _, err := l.Constituents(ctx, mq.lot)
		require.NoError(t, err)
		for _, s := range stocks {
			snap, err := l.Stock(ctx, s)
			require.NoError(t, err)
			if !sameAddr(snap.Registry.Stable, l.Usdg) {
				return l, s, q
			}
		}
		t.Fatal("the MQ venue has a WETH-quoted constituent")
		return nil, "", nil
	}
	l, s, q := tokenPool(replayFaults{})
	tok, _, err := l.MqToken(ctx, s, q)
	require.NoError(t, err)
	require.NotEqual(t, "", tok.Factory, "control: the recorded factory() read succeeds")
	pool := strings.ToLower(tok.Pool)
	factoryOf := func(to string, d []byte) bool { return strings.ToLower(to) == pool && bytes.Equal(d, sel("factory()")) }
	l, s, q = tokenPool(replayFaults{revert: factoryOf})
	_, _, err = l.MqToken(ctx, s, q)
	require.NoError(t, err, "a reverting factory() is a failed read")
	l, s, q = tokenPool(replayFaults{nodeError: factoryOf})
	_, _, err = l.MqToken(ctx, s, q)
	require.True(t, isTransport(err), "MqToken factory transport: %v", err)
}

// twapFault wraps the MQ venue's RPCLoader and, after the TWAP prefetch, empties the conversion ring
// and makes its loader fail with err: a slot the checked NAV needs then fails to load.
type twapFault struct {
	*RPCLoader
	err error
}

func (f twapFault) PrefetchTwap(ctx context.Context, m *Market) error {
	if err := f.RPCLoader.PrefetchTwap(ctx, m); err != nil {
		return err
	}
	m.Mq.ConvTwap.Obs = map[int]Observation{}
	m.Mq.ConvTwap.Fetch = func(int) (Observation, error) { return Observation{}, f.err }
	return nil
}

// A transport error inside the checked NAV (a TWAP slot read) is returned and publishes nothing; a
// revert there publishes the fail-closed market.
// mqTrackParts is the recorded MQ venue's Track setup (wiring round, loader, tracker, prefetch) on a
// fresh multicaller per call, for driving trackMarket directly.
func mqTrackParts(t *testing.T) (*uniswapv4.HookParam, func() (*Tracker, *RPCLoader, string)) {
	v := liveVenues[1]
	tape := loadTape(t, tapePath(v.poolID))
	srv := replayServer(t, tape, &rpcCounter{})
	t.Cleanup(srv.Close)
	ctx := context.Background()
	rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
	block, err := hexutil.DecodeBig(tape.Block)
	require.NoError(t, err)
	param := venueParam(rc, v, block)
	hook := strings.ToLower(v.hook)
	load := func() (*Tracker, *RPCLoader, string) { // Track's setup, on its own multicaller
		mc := newMulticaller(rc, block)
		var wiring []*callReq // Track's round 0
		for _, sig := range []string{"poolManager()", "registry()", "venueBook()", "navGuard()", "usdg()",
			"usdgDecimals()", "v3Factory()"} {
			wiring = append(wiring, &callReq{to: hook, data: sel(sig)})
		}
		for _, tok := range param.Pool.Tokens {
			wiring = append(wiring, &callReq{to: hook, data: append(sel("venueOf(address)"), encAddr(tok.Address)...)})
		}
		require.NoError(t, mc.fetch(ctx, wiring))
		pm, err := (&RPCLoader{c: rc.GetETHClient().Client(), Block: tape.Block, mc: mc}).word(ctx, hook, "poolManager()")
		require.NoError(t, err)
		l, err := newRPCLoader(ctx, rc.GetETHClient().Client(), mc, tape.Block, hook, wordAddr(pm),
			DefaultConfig("", "", FinalisedRules))
		require.NoError(t, err)
		tr := &Tracker{Cfg: l.Cfg, Hook: hook, PoolManager: l.PM, NavGuard: l.NavGuard, Registry: l.Registry,
			VenueBook: l.VenueBook}
		require.NoError(t, l.prefetch(ctx, tr, v.lot))
		return tr, l, strings.ToLower(v.lot)
	}
	return param, load
}

func TestCheckedNavTransportPublishesNothing(t *testing.T) {
	ctx := context.Background()
	param, load := mqTrackParts(t)
	tr, l, lot := load()
	h := &Hook{}
	raw, err := h.trackMarket(ctx, param, tr, l, lot, time.Now())
	require.NoError(t, err)
	require.NotNil(t, raw)
	require.NotNil(t, h.Market, "control: the recorded MQ venue tracks")

	tr, l, lot = load()
	h = &Hook{}
	raw, err = h.trackMarket(ctx, param, tr, twapFault{l, fmt.Errorf("%w: observations", ErrReverted)}, lot, time.Now())
	require.NoError(t, err)
	require.NotNil(t, raw)
	require.True(t, h.Tracked)
	require.Nil(t, h.Market, "a reverting TWAP read publishes the fail-closed market")

	tr, l, lot = load()
	h = &Hook{}
	raw, err = h.trackMarket(ctx, param, tr, twapFault{l, &TransportError{Err: errors.New("read timed out")}}, lot,
		time.Now())
	require.True(t, isTransport(err), "%v", err)
	require.Nil(t, raw)
	require.False(t, h.Tracked)
}

// The Track round count is a cost guard (offline, recorded state): the oracle rings are fetched whole
// in one round, not searched lazily one round per probe.
func TestTrackRounds(t *testing.T) {
	var st MulticallStats
	trackStats = func(s MulticallStats) { st = s }
	defer func() { trackStats = nil }()
	for _, tc := range []struct {
		venue     int
		maxRounds int
	}{{0, 7}, {1, 11}} {
		v := liveVenues[tc.venue]
		_, _, err, _ := tapeTrack(t, v, replayFaults{})
		require.NoError(t, err)
		t.Logf("%s: %+v", v.name, st)
		require.LessOrEqual(t, st.Rounds, tc.maxRounds, v.name)
		require.Zero(t, st.Misses, v.name)
	}
}

// BeforeSwap on a tracked instance without a venue refuses instead of dereferencing it (the factory
// never builds one; a Hook value can).
func TestBeforeSwapWithoutVenue(t *testing.T) {
	h := &Hook{BaseHook: &uniswapv4.BaseHook{Exchange: Exchange}, Extra: Extra{Tracked: true, Market: &Market{}}}
	var err error
	require.NotPanics(t, func() {
		_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1e6)})
	})
	require.ErrorIs(t, err, ErrNotTracked)
}

// validate's bounds at their edges: NavJitHookV17.MAX_BAND_BPS = 1000 (setParams), and a TWAP ring
// index inside its cardinality.
func TestValidateBounds(t *testing.T) {
	stock, err := LoadCase("testdata/fixtures/navjit2-buy-1kusd.json")
	require.NoError(t, err)
	m := marketOf(t, stock)
	m.Venue.Params.BandBps = 1000
	require.NoError(t, validateMarket(m), "band at MAX_BAND_BPS")
	m.Venue.Params.BandBps = 1001
	require.ErrorIs(t, validateMarket(m), errInvalid, "band above MAX_BAND_BPS")

	mq, err := LoadCase("testdata/mq/mq-buy-1kusd.json")
	require.NoError(t, err)
	m = marketOf(t, mq)
	tw := m.Mq.ConvTwap
	require.NoError(t, validateMarket(m))
	idx := tw.Index
	tw.Index = tw.Cardinality - 1
	require.NoError(t, validateTwap(tw), "last ring slot")
	tw.Index = tw.Cardinality
	require.ErrorIs(t, validateTwap(tw), errInvalid, "index == cardinality")
	tw.Index = idx
}

// The leg window refuses a zero tick spacing (a pool read the chain could not have answered) instead
// of dividing by it.
func TestWindowRefusesZeroSpacing(t *testing.T) {
	l := &RPCLoader{Cfg: DefaultConfig("", "", FinalisedRules)}
	sqrtP, err := SqrtPriceAtTick(1000)
	require.NoError(t, err)
	_, _, err = l.window(sqrtP, 60)
	require.NoError(t, err, "control")
	require.NotPanics(t, func() { _, _, err = l.window(sqrtP, 0) })
	require.Error(t, err)
}

// priceLimit refuses a band at or under the 25 bps buffer as an invalid limit (ErrPriceLimit), before
// the spread check (which would also refuse it, by another name).
func TestPriceLimitBandAtBuffer(t *testing.T) {
	h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	_, err := h.priceLimit(true)
	require.NoError(t, err, "control")
	h.Market.Venue.Params.BandBps = priceLimitBufferBps
	h.Market.Venue.Params.BuySpreadBps, h.Market.Venue.Params.SellSpreadBps = 0, 0
	for _, zfo := range []bool{true, false} {
		_, err = h.priceLimit(zfo)
		require.ErrorIs(t, err, ErrPriceLimit)
	}
}

// An oracle ring up to maxRingPrefetch slots is fetched whole, in one multicall round, before the
// checked NAV (the live venues' rings are larger and take the lazy path; this pins the small one).
func TestPrefetchTwapSmallRingOneRound(t *testing.T) {
	pool := "0x00000000000000000000000000000000000000Bb"
	const card, newest = 8, 3
	tape := &rpcTape{Block: "0x1", Calls: map[string]tapeEntry{}}
	for i := 0; i < card; i++ {
		age := (newest - i + card) % card // 0 = newest
		obs := concat([][]byte{encWord(big.NewInt(int64(10_000 - 100*age))), encWord(big.NewInt(0)),
			encWord(big.NewInt(0)), encWord(big.NewInt(1))})
		r := observationReq(pool, i)
		tape.Calls[mcKey(r.to, r.data)] = tapeEntry{OK: true, Ret: obs}
	}
	srv := replayServer(t, tape, &rpcCounter{})
	defer srv.Close()
	rc := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall))
	m := newMulticaller(rc, big.NewInt(1))
	l := &RPCLoader{c: rc.GetETHClient().Client(), Block: "0x1", mc: m}
	tw := &TwapState{Index: newest, Cardinality: card, Obs: map[int]Observation{}}
	tw.Fetch = func(i int) (Observation, error) {
		r := observationReq(pool, i)
		o, err := l.call(context.Background(), r.to, r.data)
		if err != nil {
			return Observation{}, err
		}
		return Observation{Timestamp: wordAt(o, 0).Uint64(), TickCumulative: signedBits(wordAt(o, 1), 256),
			Initialized: wordAt(o, 3).Sign() != 0}, nil
	}
	require.NoError(t, l.PrefetchTwap(context.Background(), &Market{Timestamp: 10_000, Mq: &MqMarket{ConvPool: pool,
		ConvTwap: tw}}))
	st := m.Stats()
	require.Equal(t, 1, st.Rounds, "%+v", st)
	require.Equal(t, card, st.Calls, "%+v", st)
}
