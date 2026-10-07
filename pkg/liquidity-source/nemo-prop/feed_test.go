package nemoprop

import (
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// rate0 is the snapshot's base -> market rate as the tracker reads it, or
// -1 without a snapshot covering the market.
func rate0(c *feedClient) float64 {
	m, ok := c.latest().market(hexAddr(testUSDC), hexAddr(testWETH))
	if !ok {
		return -1
	}
	return m.ladders[0][0].AmountOut() / m.ladders[0][0].AmountIn()
}

func eventuallyRate(t *testing.T, c *feedClient, want float64) {
	t.Helper()
	require.Eventually(t, func() bool { return rate0(c) == want }, 5*time.Second, 2*time.Millisecond,
		"rate never became %v (last %v)", want, rate0(c))
}

func TestFeed_RejectsWrongToken(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "right")
	cfg := f.config()
	cfg.Feed.AuthToken = "wrong"
	c := acquireFeed(cfg)
	require.Eventually(t, func() bool { return f.rejected.Load() >= 1 }, 5*time.Second, 5*time.Millisecond)
	assert.Zero(t, f.accepted.Load())
	assert.Nil(t, c.latest())
	f.mu.Lock()
	assert.Equal(t, "Bearer wrong", f.authSeen[0], "token only in the handshake header")
	f.mu.Unlock()
}

func TestFeed_SequenceOrdering(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	c := acquireFeed(f.config())
	f.waitConns(1)
	f.push(snapshotFrame(5, 2, 0.5))
	eventuallyRate(t, c, 2)

	// Older and repeated frames are ignored; frames apply in order, so once
	// seq 6 shows, seq 3 and the duplicate 5 are done.
	f.push(snapshotFrame(3, 3, 0.5))
	f.push(snapshotFrame(5, 3, 0.5))
	f.push(snapshotFrame(6, 2, 0.25))
	require.Eventually(t, func() bool { return c.latest().seq == 6 }, 5*time.Second, 2*time.Millisecond)
	assert.Equal(t, float64(2), rate0(c))

	// Sequence numbers are per connection.
	f.dropAll()
	f.waitConns(2)
	f.push(snapshotFrame(1, 4, 0.5))
	eventuallyRate(t, c, 4)
}

// Frames of unknown types are skipped, even when their fields clash with a
// snapshot's, so the server can add message types without withdrawing
// quotes from pool-services on older versions of this client.
func TestFeed_IgnoresUnknownMessageTypes(t *testing.T) {
	t.Parallel()
	c := &feedClient{chainID: 8453, proxy: hexAddr(testProxy)}
	frame, err := json.Marshal(snapshotFrame(1, 2, 0.5))
	require.NoError(t, err)
	require.True(t, c.handle(frame))
	snap := c.latest()
	require.NotNil(t, snap)

	for _, frame := range []string{
		`{"v":1,"type":"heartbeat"}`,
		`{"v":1,"type":"stats","seq":"many","tokens":[1,2,3]}`,
	} {
		assert.False(t, c.handle([]byte(frame)), frame)
		assert.Same(t, snap, c.latest(), frame)
	}

	// Malformed frames and other protocol versions still withdraw it.
	assert.False(t, c.handle([]byte(`{"v":2,"type":"heartbeat"}`)))
	assert.Nil(t, c.latest())
}

// Tracker refreshes race with pushes (run with -race).
func TestFeed_ConcurrentRefreshesDuringPushes(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	tracker := NewPoolTracker(f.config(), nil)
	f.waitConns(1)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				p, _ := refresh(t, tracker, testPool(t))
				if sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p}); err == nil {
					_ = quote1000(sim)
				}
			}
		})
	}
	for seq := uint64(1); seq <= 50; seq++ {
		f.push(snapshotFrame(seq, float64(1+seq%3), 0.5))
		time.Sleep(time.Millisecond)
	}
	eventuallyRate(t, tracker.feed, 3) // seq 50: rate 1+50%3 = 3
	close(stop)
	wg.Wait()
}

// One market with a bad ladder or inventory is dropped alone: it must not
// withdraw quotes for the other markets of the same snapshot.
func TestFeed_BadMarketDropsOnlyThatMarket(t *testing.T) {
	t.Parallel()
	for name, spoil := range map[string]func(m *feedMessage){
		"bad ladder": func(m *feedMessage) {
			m.Tokens[hexAddr(testCBBTC)] = feedToken{Balance: "1", Allowance: "1"}
			m.Markets[hexAddr(testCBBTC)] = feedMarket{Ladders: [2][]ladder.Point{{{2000, 2}, {1000, 3}}, nil}}
		},
		"bad inventory": func(m *feedMessage) {
			m.Tokens[hexAddr(testCBBTC)] = feedToken{Balance: "x", Allowance: "1"}
			m.Markets[hexAddr(testCBBTC)] = feedMarket{Ladders: linearLadders(2, 0.5)}
		},
	} {
		c := &feedClient{chainID: 8453, proxy: hexAddr(testProxy)}
		msg := snapshotFrame(1, 2, 0.5)
		spoil(&msg)
		frame, err := json.Marshal(msg)
		require.NoError(t, err)
		require.True(t, c.handle(frame), name)
		assert.Equal(t, float64(2), rate0(c), name)
		_, ok := c.latest().market(hexAddr(testUSDC), hexAddr(testCBBTC))
		assert.False(t, ok, name)
	}
}
