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
//	   "timestampMs":<pricing time>,"blockNumber":...,"base":"0x...",
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
// without breaking pool-services that run an older version of this client.
// Invalid frames and withdrawals clear the cached snapshot until the next
// valid one. A disconnect keeps it: past the fresh window its quotes decay
// (or stop, with decay disabled), and past the maximum age they stop. Both
// are timed from receipt on the tracker's own clock.
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
	errFeedReconnect = errors.New("feed auth token rotated")
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

// feedSnapshot is one complete, coherent feed state: every market's raw,
// validated ladders and the vault's deliverable inventory per token.
// Immutable once published.
type feedSnapshot struct {
	seq         uint64
	pricedAt    time.Time
	blockNumber uint64
	base        string
	reserves    map[string]*uint256.Int
	markets     map[string][2][]ladder.Point
}

// snapshotMarket is one market's view of a snapshot: raw ladders in pool order
// and the deliverable inventory of [base, asset].
type snapshotMarket struct {
	ladders  [2][]ladder.Point
	reserves [2]*uint256.Int
}

// market returns (base, asset)'s ladders and inventory, and false if there
// is no snapshot or it doesn't cover the market.
func (s *feedSnapshot) market(base, asset string) (snapshotMarket, bool) {
	if s == nil {
		return snapshotMarket{}, false
	}
	base, asset = strings.ToLower(base), strings.ToLower(asset)
	ladders, ok := s.markets[asset]
	if !ok || s.base != base {
		return snapshotMarket{}, false
	}
	return snapshotMarket{ladders: ladders, reserves: [2]*uint256.Int{s.reserves[base], s.reserves[asset]}}, true
}

// feedClient is one persistent, authenticated connection, shared by every
// tracker of a proxy in this process. Only trackers use it, so it runs in
// pool-service and never in router processes. Like angstrom's
// AttestationController it lives for the whole process.
type feedClient struct {
	url     string
	chainID int64
	proxy   string

	authToken atomic.Pointer[string]
	snapshot  atomic.Pointer[feedSnapshot]

	reconnect chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	lastSeq   uint64 // owned by the connection goroutine
}

// feeds holds one *feedClient per feedKey for the process lifetime.
var feeds sync.Map

func feedKey(url string, chainID int64, proxy string) string {
	return url + "#" + strconv.FormatInt(chainID, 10) + "#" + proxy
}

// acquireFeed returns the process-wide client for cfg's feed and proxy,
// starting its connection on first use, or nil if no feed is configured.
// A tracker built with a rotated auth token reconnects the shared client
// with it. It never blocks on the network.
func acquireFeed(cfg *Config) *feedClient {
	if cfg.Feed.URL == "" || cfg.Address == "" {
		return nil
	}
	proxy := strings.ToLower(cfg.Address)
	key := feedKey(cfg.Feed.URL, int64(cfg.ChainID), proxy)
	token := cfg.Feed.AuthToken

	c := &feedClient{
		url:       cfg.Feed.URL,
		chainID:   int64(cfg.ChainID),
		proxy:     proxy,
		reconnect: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	c.authToken.Store(&token)
	if v, loaded := feeds.LoadOrStore(key, c); loaded {
		c = v.(*feedClient)
		if old := c.authToken.Swap(&token); *old != token {
			select {
			case c.reconnect <- struct{}{}:
			default:
			}
		}
		return c
	}
	go c.run()
	return c
}

// latest returns the newest valid snapshot, or nil. Lock-free; safe on a
// nil client (no feed configured).
func (c *feedClient) latest() *feedSnapshot {
	if c == nil {
		return nil
	}
	return c.snapshot.Load()
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
// delivered a valid snapshot resets the backoff; an auth token rotation
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

	ctx, cancel := context.WithTimeout(context.Background(), feedDialTimeout)
	header := http.Header{"Authorization": {"Bearer " + *c.authToken.Load()}}
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
		// Ignored, not invalid: pool-service runs whichever dex-lib version Kyber
		// last deployed, so new message types must not withdraw Nemo's quotes.
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

	snap, err := c.buildFeedSnapshot(&msg, time.Now())
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

// buildFeedSnapshot validates a snapshot frame. A bad base or base inventory
// invalidates the frame; a bad market (ladder or inventory) is dropped alone,
// so it can't withdraw the other markets' quotes.
func (c *feedClient) buildFeedSnapshot(msg *feedMessage, receivedAt time.Time) (*feedSnapshot, error) {
	if !common.IsHexAddress(msg.Base) {
		return nil, errFeedInvalid
	}
	if msg.TimestampMs <= 0 {
		return nil, errFeedInvalid
	}
	base := strings.ToLower(msg.Base)

	reserves := make(map[string]*uint256.Int, len(msg.Tokens))
	for token, t := range msg.Tokens {
		balance, err := uint256.FromDecimal(t.Balance)
		if err != nil {
			continue
		}
		allowance, err := uint256.FromDecimal(t.Allowance)
		if err != nil {
			continue
		}
		reserves[strings.ToLower(token)] = deliverable(balance, allowance)
	}
	if _, ok := reserves[base]; !ok {
		return nil, errFeedInvalid
	}

	markets := make(map[string][2][]ladder.Point, len(msg.Markets))
	for asset, m := range msg.Markets {
		asset = strings.ToLower(asset)
		if _, ok := reserves[asset]; !ok || !common.IsHexAddress(asset) || asset == base ||
			validateLadder(m.Ladders[0]) != nil || validateLadder(m.Ladders[1]) != nil {
			c.log().Warnf("nemo-prop feed: snapshot %d: dropping invalid market %s", msg.Seq, asset)
			continue
		}
		markets[asset] = m.Ladders
	}

	return &feedSnapshot{
		seq:         msg.Seq,
		pricedAt:    pricedAt(msg.TimestampMs, receivedAt),
		blockNumber: msg.BlockNumber,
		base:        base,
		reserves:    reserves,
		markets:     markets,
	}, nil
}

// pricedAt is when Nemo priced the snapshot (timestampMs), capped at receipt
// so a server clock running ahead can't extend a snapshot's life.
func pricedAt(timestampMs int64, receivedAt time.Time) time.Time {
	if t := time.UnixMilli(timestampMs); t.Before(receivedAt) {
		return t
	}
	return receivedAt
}
