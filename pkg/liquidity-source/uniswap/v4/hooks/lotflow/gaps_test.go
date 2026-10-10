package lotflow

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// DELTA3's surviving mutants, each pinned by a test.

// SqrtPriceLimit is called by the outer v4 simulator outside quote's own recover: a state the
// structural validation would refuse (a constituent without NAV inputs) refuses the limit, it does
// not panic.
func TestSqrtPriceLimitRecoversNumericPanic(t *testing.T) {
	h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	require.NotNil(t, h.SqrtPriceLimit(true), "control")
	h.Market.Constituents = []MarketConstituent{{}}
	for _, zfo := range []bool{true, false} {
		var lim *uint256.Int
		require.NotPanics(t, func() { lim = h.SqrtPriceLimit(zfo) })
		require.Nil(t, lim)
		_, err := h.priceLimit(zfo)
		require.ErrorIs(t, err, ErrNumeric)
	}
}

// validatePool at a tick boundary: slot0's tick may be one below the boundary its sqrtPrice sits on
// (a downward stop), never one above; a tick at the current index belongs below it (its liquidityNet
// is already in the pool's liquidity); gross liquidity is a uint128.
func TestValidatePoolBoundaries(t *testing.T) {
	at := func(i int) *big.Int { s, err := SqrtPriceAtTick(i); require.NoError(t, err); return s }
	maxU128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	g := uint256.MustFromBig(maxU128)
	pool := func(tick int) *LegPool {
		return &LegPool{Kind: KindV3, SqrtPriceX96: at(120), Tick: tick, Liquidity: new(big.Int).Set(maxU128),
			TickSpacing: 60,
			Ticks: []uniswapv3.TickU256{
				{Index: 120, LiquidityGross: g, LiquidityNet: int256.MustFromBig(maxU128)},
				{Index: 240, LiquidityGross: g, LiquidityNet: int256.MustFromBig(new(big.Int).Neg(maxU128))},
			}}
	}
	// on the boundary at tick 120 after an upward cross: range [120, 240) holds all the liquidity
	require.NoError(t, validatePool(pool(120)), "tick at the current index is below the price")
	require.ErrorIs(t, validatePool(pool(121)), errInvalid, "one above the boundary")
	p := pool(120)
	p.Ticks[1].LiquidityGross = uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 128))
	require.ErrorIs(t, validatePool(p), errInvalid, "gross liquidity above uint128")
}

// The per-call path (pool-service tooling): a context that ends during a 429 back-off is a transport
// error, not a revert.
func TestPerCallCtxDuring429IsTransport(t *testing.T) {
	var n int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&n, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Rate Limit"))
	}))
	defer srv.Close()
	l := &RPCLoader{c: ethrpc.New(srv.URL).GetETHClient().Client(), Block: "0x1"}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_, err := l.call(ctx, "0x0000000000000000000000000000000000000001", sel("foo()"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.True(t, isTransport(err), "%v (%T) after %d requests", err, err, atomic.LoadInt64(&n))
}

// ctxTwapFault ends the caller's context once the market is loaded, and every TWAP slot read then
// fails with a plain (unclassified) error, as a loader whose context ended would.
type ctxTwapFault struct {
	*RPCLoader
	cancel context.CancelFunc
}

func (f ctxTwapFault) PrefetchTwap(ctx context.Context, m *Market) error {
	if err := f.RPCLoader.PrefetchTwap(ctx, m); err != nil {
		return err
	}
	m.Mq.ConvTwap.Obs = map[int]Observation{}
	m.Mq.ConvTwap.Fetch = func(int) (Observation, error) { return Observation{}, errors.New("read aborted") }
	f.cancel()
	return nil
}

// A checked NAV cut short by the caller's context publishes nothing, whatever error it returned.
func TestCheckedNavCancelledPublishesNothing(t *testing.T) {
	param, load := mqTrackParts(t)
	tr, l, lot := load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &Hook{}
	raw, err := h.trackMarket(ctx, param, tr, ctxTwapFault{l, cancel}, lot, time.Now())
	require.True(t, isTransport(err), "%v", err)
	require.Nil(t, raw)
	require.False(t, h.Tracked)
}

// A "latest" node that lags the pinned one (head older than the pin) is not a stale pin: Track tracks.
func TestTrackPinAheadOfLatest(t *testing.T) {
	for _, v := range liveVenues {
		h, raw, err, _ := tapeTrack(t, v, replayFaults{headBehind: 5})
		require.NoError(t, err, v.name)
		require.NotNil(t, raw, v.name)
		require.True(t, h.Tracked && h.Market != nil, v.name)
	}
}

// The recheck request itself failing (HTTP error on the plain-eth_call batch) is a transport error:
// the failed sub-call is never cached as an empty success.
func TestRecheckRequestErrorIsTransport(t *testing.T) {
	to := "0x00000000000000000000000000000000000000AA"
	data := sel("decimals()")
	tape := &rpcTape{Block: "0x1", Calls: map[string]tapeEntry{mcKey(to, data): {OK: true, Ret: encWord(big.NewInt(6))}}}
	srv := replayServerWith(t, tape, &rpcCounter{}, replayFaults{
		batchOOG: func(string, []byte) bool { return true },
		httpFail: func(n int) bool { return n >= 2 }, // request 1 = tryAggregate, 2.. = the recheck
	})
	defer srv.Close()
	m := newMulticaller(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), big.NewInt(1))
	out, err := m.get(context.Background(), to, data)
	require.True(t, isTransport(err), "%v", err)
	require.Nil(t, out)
}

// One unpredicted read is already worth the warning.
func TestStatsWarningOneMiss(t *testing.T) {
	require.Contains(t, statsWarning(MulticallStats{Misses: 1}), "1 reads not predicted")
}

// Track reads the pinned block's header once: the loader reuses the timestamp Track already has.
func TestTrackReadsHeaderOnce(t *testing.T) {
	for _, v := range liveVenues {
		_, _, err, ctr := tapeTrack(t, v, replayFaults{})
		require.NoError(t, err)
		require.Equal(t, 2, ctr.Methods["eth_getBlockByNumber"], "%s: the pinned header + the head (pin age)", v.name)
	}
}
