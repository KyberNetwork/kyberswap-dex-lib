package nemoprop

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// Live feed protocol, version 1 (JSON text frames):
//
//	client -> server, once per connection, after an
//	"Authorization: Bearer <token>" handshake:
//	  {"v":1,"type":"subscribe","chainId":8453,"proxy":"0x..."}
//
//	server -> client, a complete snapshot (never a delta), resent faster
//	than the client's fresh window (1s by default) even when nothing
//	changed:
//	  {"v":1,"type":"snapshot","chainId":8453,"proxy":"0x...","seq":42,
//	   "timestampMs":...,"blockNumber":...,"base":"0x...",
//	   "tokens":{"0x...":{"balance":"...","allowance":"..."}},
//	   "markets":{"0x<asset>":{"l":[[[in,out],...],[[in,out],...]]}}}
//
//	  Ladders are (amountIn, amountOut) samples in pool order (direction 0
//	  is base -> market), unbuffered and untruncated: the client truncates
//	  them at deliverable inventory and applies the buffer. "tokens" holds
//	  the vault's balance and its allowance to the proxy for the base and
//	  every market asset.
//	  "seq" starts at 1 on every connection and must increase strictly;
//	  older or repeated frames are ignored.
//
//	server -> client, withdrawing the current state:
//	  {"v":1,"type":"unavailable","reason":"..."}
//
// Frames of any other type are ignored, so the server can add message types
// without breaking routers that run an older version of this client.
// Invalid frames and withdrawals clear the cached snapshot until the next
// valid one. A disconnect keeps it: past the fresh window its quotes decay
// (or stop, with decay disabled), and past the maximum age they stop. Both
// are timed from receipt on the client's own clock.
const (
	feedProtocolVersion = 1

	feedMsgSubscribe   = "subscribe"
	feedMsgSnapshot    = "snapshot"
	feedMsgUnavailable = "unavailable"

	feedDialTimeout    = 10 * time.Second
	feedWriteTimeout   = 5 * time.Second
	feedPingInterval   = 3 * time.Second
	feedReadTimeout    = 10 * time.Second
	feedMaxMessageSize = 8 << 20
	feedBackoffMin     = 250 * time.Millisecond
	feedBackoffMax     = 30 * time.Second
)

var (
	errFeedInvalid   = errors.New("invalid feed frame")
	errFeedReconnect = errors.New("feed settings rotated")
)

type (
	feedSubscribe struct {
		V       int    `json:"v"`
		Type    string `json:"type"`
		ChainID int64  `json:"chainId"`
		Proxy   string `json:"proxy"`
	}

	// feedHeader is read before a frame's type is known, so a frame of an
	// unknown type is skipped whatever its other fields hold.
	feedHeader struct {
		V    int    `json:"v"`
		Type string `json:"type"`
	}

	feedMessage struct {
		V           int                   `json:"v"`
		Type        string                `json:"type"`
		ChainID     int64                 `json:"chainId"`
		Proxy       string                `json:"proxy"`
		Seq         uint64                `json:"seq"`
		TimestampMs int64                 `json:"timestampMs"`
		BlockNumber uint64                `json:"blockNumber"`
		Base        string                `json:"base"`
		Tokens      map[string]feedToken  `json:"tokens"`
		Markets     map[string]feedMarket `json:"markets"`
		Reason      string                `json:"reason"`
	}

	feedToken struct {
		Balance   string `json:"balance"`
		Allowance string `json:"allowance"`
	}

	feedMarket struct {
		Ladders [2][]ladder.Point `json:"l"`
	}
)

// feedSnapshot is one complete, coherent feed state with every market's
// quoting state prebuilt, so quotes only do a map lookup. Immutable once
// published.
type feedSnapshot struct {
	seq         uint64
	receivedAt  time.Time
	blockNumber uint64
	base        string
	markets     map[string]*ladder.PoolSimulator

	raw *feedMessage // kept to reshape on a buffer change
}

// market returns the quoting state for (base, market), or nil if there is
// no snapshot or it doesn't cover the market.
func (s *feedSnapshot) market(base, market string) *ladder.PoolSimulator {
	if s == nil || s.base != base {
		return nil
	}
	return s.markets[market]
}

type versionedSettings struct {
	FeedSettings
	version int64
}

func (s *versionedSettings) freshWindow() time.Duration {
	if s.FreshMs > 0 {
		return time.Duration(s.FreshMs) * time.Millisecond
	}
	return defaultFeedFresh
}

// maxAge bounds how long after receipt a snapshot quotes at all, decayed or
// not. Unset, it is defaultFeedMaxAge, or the fresh window if that's longer.
func (s *versionedSettings) maxAge() time.Duration {
	if s.MaxAgeMs > 0 {
		return time.Duration(s.MaxAgeMs) * time.Millisecond
	}
	return max(defaultFeedMaxAge, s.freshWindow())
}

// feedClient is one persistent, authenticated connection, shared by every
// tracker and simulator of a proxy in this process. Like angstrom's
// AttestationController it lives for the whole process. Quotes only touch
// the atomics; mu serializes writers.
type feedClient struct {
	url     string
	chainID int64
	proxy   string

	settings atomic.Pointer[versionedSettings]
	snapshot atomic.Pointer[feedSnapshot]

	mu        sync.Mutex
	reconnect chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	lastSeq   uint64 // owned by the connection goroutine
}

// feeds holds one *feedClient per feedKey for the process lifetime.
var feeds sync.Map

// feedKey identifies a shared connection. Empty means the feed is disabled.
func feedKey(s FeedSettings, proxy string) string {
	if s.URL == "" || proxy == "" {
		return ""
	}
	return s.URL + "#" + strconv.FormatInt(s.ChainID, 10) + "#" + proxy
}

// acquireFeed returns the process-wide client for key, starting its
// connection on first use, and offers s (pool state written at version) as
// possibly newer settings. It never blocks on the network.
func acquireFeed(key string, s FeedSettings, proxy string, version int64) *feedClient {
	if v, ok := feeds.Load(key); ok {
		c := v.(*feedClient)
		c.offer(s, version)
		return c
	}

	c := &feedClient{
		url:       s.URL,
		chainID:   s.ChainID,
		proxy:     proxy,
		reconnect: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	c.settings.Store(&versionedSettings{FeedSettings: s, version: version})
	if v, loaded := feeds.LoadOrStore(key, c); loaded {
		c = v.(*feedClient)
		c.offer(s, version)
		return c
	}
	go c.run()
	return c
}

// offer adopts s if it comes from fresher pool state than the current
// settings, which is how a rotated auth token reaches every process.
func (c *feedClient) offer(s FeedSettings, version int64) {
	if cur := c.settings.Load(); cur.FeedSettings == s || version <= cur.version {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	cur := c.settings.Load()
	if cur.FeedSettings == s || version <= cur.version {
		return
	}
	c.settings.Store(&versionedSettings{FeedSettings: s, version: version})

	if s.AuthToken != cur.AuthToken {
		select {
		case c.reconnect <- struct{}{}:
		default:
		}
	}
	if s.Buffer != cur.Buffer {
		if snap := c.snapshot.Load(); snap != nil {
			reshaped, _ := buildFeedSnapshot(snap.raw, s.Buffer, snap.receivedAt)
			// Swap, not store: the connection clears a withdrawn snapshot
			// without taking mu, and that must not be undone here.
			c.snapshot.CompareAndSwap(snap, reshaped)
		}
	}
}

// quotable returns the latest snapshot and its decay: DecayBps times the
// milliseconds it is past the fresh window, 0 while fresh. It returns nil
// past the maximum age, or past the fresh window with decay disabled.
// Lock-free.
func (c *feedClient) quotable(now time.Time) (*feedSnapshot, uint64) {
	snap := c.snapshot.Load()
	if snap == nil {
		return nil, 0
	}
	settings := c.settings.Load()
	age := now.Sub(snap.receivedAt)
	if age >= settings.maxAge() {
		return nil, 0
	}
	stale := age - settings.freshWindow()
	if stale <= 0 {
		return snap, 0
	}
	if settings.DecayBps <= 0 {
		return nil, 0
	}
	return snap, uint64(settings.DecayBps) * uint64(stale.Milliseconds())
}

func (c *feedClient) clear() {
	c.snapshot.Store(nil)
}

// close stops the connection for good. Production clients live for the
// whole process; this exists for tests.
func (c *feedClient) close() {
	c.closeOnce.Do(func() { close(c.done) })
}

func (c *feedClient) log() logger.Logger {
	return logger.WithFields(logger.Fields{"dexType": DexType, "feed": c.url, "proxy": c.proxy})
}

// run reconnects forever with jittered exponential backoff. A session that
// delivered a valid snapshot resets the backoff; a settings rotation
// reconnects immediately.
func (c *feedClient) run() {
	backoff := feedBackoffMin
	for {
		healthy, err := c.session()
		if healthy {
			backoff = feedBackoffMin
		}

		if errors.Is(err, errFeedReconnect) {
			continue
		}
		if err != nil {
			c.log().Warnf("nemo-prop feed disconnected: %v", err)
		}

		delay := backoff/2 + rand.N(backoff/2+1)
		backoff = min(backoff*2, feedBackoffMax)
		select {
		case <-c.done:
			return
		case <-c.reconnect:
		case <-time.After(delay):
		}
	}
}

// session runs one connection until it fails. healthy reports whether it
// delivered at least one valid snapshot.
func (c *feedClient) session() (healthy bool, err error) {
	select {
	case <-c.done:
		return false, nil
	default:
	}

	settings := c.settings.Load()
	ctx, cancel := context.WithTimeout(context.Background(), feedDialTimeout)
	header := http.Header{"Authorization": {"Bearer " + settings.AuthToken}}
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, c.url, header)
	cancel()
	if err != nil {
		if resp != nil {
			return false, errors.New("dial: " + resp.Status)
		}
		return false, err
	}
	defer func() { _ = conn.Close() }()

	conn.SetReadLimit(feedMaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(feedReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(feedReadTimeout))
	})

	sub, _ := json.Marshal(feedSubscribe{
		V: feedProtocolVersion, Type: feedMsgSubscribe, ChainID: c.chainID, Proxy: c.proxy,
	})
	_ = conn.SetWriteDeadline(time.Now().Add(feedWriteTimeout))
	if err := conn.WriteMessage(websocket.TextMessage, sub); err != nil {
		return false, err
	}

	var rotated atomic.Bool
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(feedPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(feedWriteTimeout)) != nil {
					_ = conn.Close()
					return
				}
			case <-c.reconnect:
				rotated.Store(true)
				_ = conn.Close()
				return
			case <-c.done:
				_ = conn.Close()
				return
			case <-stop:
				return
			}
		}
	}()

	c.lastSeq = 0
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if rotated.Load() {
				return healthy, errFeedReconnect
			}
			return healthy, err
		}
		_ = conn.SetReadDeadline(time.Now().Add(feedReadTimeout))
		if c.handle(data) {
			healthy = true
		}
	}
}

// handle applies one frame and reports whether it published a snapshot.
func (c *feedClient) handle(data []byte) bool {
	var head feedHeader
	if err := json.Unmarshal(data, &head); err != nil {
		c.invalidate(err)
		return false
	}
	if head.V != feedProtocolVersion {
		c.invalidate(errors.New("unsupported protocol version " + strconv.Itoa(head.V)))
		return false
	}

	var msg feedMessage
	switch head.Type {
	case feedMsgUnavailable:
		c.clear()
		_ = json.Unmarshal(data, &msg) // only for the reason
		c.log().Infof("nemo-prop feed unavailable: %s", msg.Reason)
		return false
	case feedMsgSnapshot:
		if err := json.Unmarshal(data, &msg); err != nil {
			c.invalidate(err)
			return false
		}
	default:
		// Ignored, not invalid: routers run whichever dex-lib version Kyber
		// last deployed, so new message types must not withdraw their quotes.
		return false
	}

	if msg.Seq <= c.lastSeq {
		return false
	}
	c.lastSeq = msg.Seq
	if msg.ChainID != c.chainID || !strings.EqualFold(msg.Proxy, c.proxy) {
		c.invalidate(errors.New("snapshot for another chain or proxy"))
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	snap, err := buildFeedSnapshot(&msg, c.settings.Load().Buffer, time.Now())
	if err != nil {
		c.snapshot.Store(nil)
		c.log().Warnf("nemo-prop feed: dropping snapshot %d: %v", msg.Seq, err)
		return false
	}
	c.snapshot.Store(snap)
	return true
}

func (c *feedClient) invalidate(err error) {
	c.clear()
	c.log().Warnf("nemo-prop feed: %v: %v", errFeedInvalid, err)
}

// buildFeedSnapshot validates a snapshot frame and prebuilds every market's
// quoting state from it, shaped with buffer.
func buildFeedSnapshot(msg *feedMessage, buffer int64, receivedAt time.Time) (*feedSnapshot, error) {
	if !common.IsHexAddress(msg.Base) {
		return nil, errFeedInvalid
	}
	base := strings.ToLower(msg.Base)

	reserves := make(map[string]*uint256.Int, len(msg.Tokens))
	for token, t := range msg.Tokens {
		balance, err := uint256.FromDecimal(t.Balance)
		if err != nil {
			return nil, err
		}
		allowance, err := uint256.FromDecimal(t.Allowance)
		if err != nil {
			return nil, err
		}
		reserves[strings.ToLower(token)] = deliverable(balance, allowance)
	}
	baseReserve, ok := reserves[base]
	if !ok {
		return nil, errFeedInvalid
	}

	markets := make(map[string]*ladder.PoolSimulator, len(msg.Markets))
	for asset, m := range msg.Markets {
		asset = strings.ToLower(asset)
		assetReserve, ok := reserves[asset]
		if !ok || !common.IsHexAddress(asset) || asset == base {
			return nil, errFeedInvalid
		}
		if validateLadder(m.Ladders[0]) != nil || validateLadder(m.Ladders[1]) != nil {
			return nil, errInvalidLadder
		}
		poolReserves := [2]*uint256.Int{baseReserve, assetReserve}
		state, err := newMarketState(base, asset, poolReserves, shapeLadders(m.Ladders, poolReserves, buffer),
			msg.BlockNumber)
		if err != nil {
			return nil, err
		}
		markets[asset] = state
	}

	return &feedSnapshot{
		seq:         msg.Seq,
		receivedAt:  receivedAt,
		blockNumber: msg.BlockNumber,
		base:        base,
		markets:     markets,
		raw:         msg,
	}, nil
}
