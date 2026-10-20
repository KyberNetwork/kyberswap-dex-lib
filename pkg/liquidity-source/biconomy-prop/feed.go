package biconomyprop

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
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/holiman/uint256"
)

// Venue board feed (Biconomy's /venue-stream, JSON text frames):
//
//	client -> server, after an "x-api-key: <key>" handshake:
//	  {"type":"subscribe","chainId":8453}
//
//	server -> client, a complete snapshot (never a delta) on every new head and on every
//	maker update:
//	  {"type":"boards","chainId":8453,"venue":"0x...","blockNumber":"...","timestamp":"...",
//	   "feeBps":0,"makers":["0x...",...],"builtAtMs":...,
//	   "boards":[{"mm":"0x...","tokenIn":"0x...","tokenOut":"0x...","committed":true,
//	              "sizes":["..."],"prices":["..."],"filled":"...","remaining":"...",
//	              "expiresAt":"...", ...}, ...]}
//
//	  Each board is PropAMMExecutor.board() for (mm, tokenIn, tokenOut) at blockNumber. A board
//	  a maker has signed and the keeper has not landed yet appears a second time with
//	  "committed": false, priced as board() will price it once it lands.
//
// Other frame types ("subscribed", "pong", "error") are ignored, so the server can add types
// without breaking pool-services on an older dex-lib. A disconnect keeps the last snapshot until
// it is older than the maximum age; then the tracker reads the venue on-chain again.
const (
	feedMsgSubscribe = "subscribe"
	feedMsgBoards    = "boards"
	feedMsgError     = "error"

	feedAPIKeyHeader = "x-api-key"

	feedDialTimeout    = 10 * time.Second
	feedWriteTimeout   = 5 * time.Second
	feedPingInterval   = 5 * time.Second
	feedReadTimeout    = 20 * time.Second
	feedMaxMessageSize = 8 << 20
	feedBackoffMin     = 250 * time.Millisecond
	feedBackoffMax     = 30 * time.Second
)

var errFeedReconnect = errors.New("feed api key rotated")

type (
	feedSubscribe struct {
		Type    string `json:"type"`
		ChainID int64  `json:"chainId"`
	}

	feedHeader struct {
		Type string `json:"type"`
	}

	feedMessage struct {
		ChainID     int64       `json:"chainId"`
		Venue       string      `json:"venue"`
		BlockNumber string      `json:"blockNumber"`
		FeeBps      uint64      `json:"feeBps"`
		Makers      []string    `json:"makers"`
		Boards      []feedBoard `json:"boards"`
		BuiltAtMs   int64       `json:"builtAtMs"`
		Code        string      `json:"code"`
		Message     string      `json:"message"`
	}

	feedBoard struct {
		Mm        string   `json:"mm"`
		TokenIn   string   `json:"tokenIn"`
		TokenOut  string   `json:"tokenOut"`
		Committed bool     `json:"committed"`
		Sizes     []string `json:"sizes"`
		Prices    []string `json:"prices"`
		Filled    string   `json:"filled"`
		Remaining string   `json:"remaining"`
		ExpiresAt string   `json:"expiresAt"`
	}
)

// feedSnapshot is one validated feed message. Immutable once published.
type feedSnapshot struct {
	venue       string
	blockNumber uint64
	feeBps      uint64
	makers      []string
	// boards holds one board per "mm|tokenIn|tokenOut", lowercase: the pending board when the
	// message has one, the committed board otherwise.
	boards     map[string]Board
	builtAt    time.Time
	receivedAt time.Time
}

func boardKey(mm, tokenIn, tokenOut string) string {
	return strings.ToLower(mm) + "|" + strings.ToLower(tokenIn) + "|" + strings.ToLower(tokenOut)
}

// board returns a maker's board for a direction, or an empty (dark) board when the snapshot has
// none: the venue skips a maker with nothing to fill.
func (s *feedSnapshot) board(mm, tokenIn, tokenOut string) Board {
	if b, ok := s.boards[boardKey(mm, tokenIn, tokenOut)]; ok {
		return b
	}
	return Board{Filled: new(uint256.Int), Remaining: new(uint256.Int)}
}

// age is how old the snapshot is, from when it was built, capped at receipt so a server clock
// running ahead cannot extend its life.
func (s *feedSnapshot) age(now time.Time) time.Duration {
	origin := s.receivedAt
	if s.builtAt.Before(origin) {
		origin = s.builtAt
	}
	return now.Sub(origin)
}

// feedClient is one persistent, authenticated connection, shared by every tracker of a chain in
// this process. Only trackers use it, so it runs in pool-service and never in router processes.
type feedClient struct {
	url     string
	chainID int64
	venue   string

	apiKey   atomic.Pointer[string]
	snapshot atomic.Pointer[feedSnapshot]

	reconnect chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

var feeds sync.Map

// acquireFeed returns the process-wide client for cfg's feed, starting it on first use, or nil
// when no feed is configured. A tracker built with a rotated API key reconnects the shared
// client with it. It never blocks on the network.
func acquireFeed(cfg *Config) *feedClient {
	if cfg.Feed.URL == "" || cfg.Venue == "" {
		return nil
	}
	venue := strings.ToLower(cfg.Venue)
	key := cfg.Feed.URL + "#" + strconv.FormatInt(int64(cfg.ChainID), 10) + "#" + venue
	apiKey := cfg.Feed.APIKey

	c := &feedClient{
		url:       cfg.Feed.URL,
		chainID:   int64(cfg.ChainID),
		venue:     venue,
		reconnect: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	c.apiKey.Store(&apiKey)
	if v, loaded := feeds.LoadOrStore(key, c); loaded {
		c = v.(*feedClient)
		if old := c.apiKey.Swap(&apiKey); *old != apiKey {
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

// latest returns the newest valid snapshot, or nil. Safe on a nil client.
func (c *feedClient) latest() *feedSnapshot {
	if c == nil {
		return nil
	}
	return c.snapshot.Load()
}

func (c *feedClient) close() {
	c.closeOnce.Do(func() { close(c.done) })
}

func (c *feedClient) log() logger.Logger {
	return logger.WithFields(logger.Fields{"dexType": DexType, "feed": c.url, "venue": c.venue})
}

// run reconnects forever with jittered exponential backoff. A session that delivered a valid
// snapshot resets the backoff; an API key rotation reconnects immediately.
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
			c.log().Warnf("biconomy-prop feed disconnected: %v", err)
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

// session runs one connection until it fails. healthy reports whether it delivered at least one
// valid snapshot.
func (c *feedClient) session() (healthy bool, err error) {
	select {
	case <-c.done:
		return false, nil
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), feedDialTimeout)
	header := http.Header{feedAPIKeyHeader: {*c.apiKey.Load()}}
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

	sub, _ := json.Marshal(feedSubscribe{Type: feedMsgSubscribe, ChainID: c.chainID})
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

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if rotated.Load() {
				return healthy, errFeedReconnect
			}
			return healthy, err
		}
		_ = conn.SetReadDeadline(time.Now().Add(feedReadTimeout))
		if c.handle(data, time.Now()) {
			healthy = true
		}
	}
}

// handle applies one frame and reports whether it published a snapshot.
func (c *feedClient) handle(data []byte, receivedAt time.Time) bool {
	var head feedHeader
	if err := json.Unmarshal(data, &head); err != nil {
		c.log().Warnf("biconomy-prop feed: unreadable frame: %v", err)
		return false
	}
	var msg feedMessage
	switch head.Type {
	case feedMsgBoards:
		if err := json.Unmarshal(data, &msg); err != nil {
			c.log().Warnf("biconomy-prop feed: unreadable snapshot: %v", err)
			return false
		}
	case feedMsgError:
		_ = json.Unmarshal(data, &msg)
		c.log().Warnf("biconomy-prop feed error %s: %s", msg.Code, msg.Message)
		return false
	default:
		return false
	}

	snap, err := c.buildSnapshot(&msg, receivedAt)
	if err != nil {
		c.log().Warnf("biconomy-prop feed: dropping snapshot: %v", err)
		return false
	}
	// Snapshots can arrive out of order across a reconnect: never go back in block or build time.
	if prev := c.snapshot.Load(); prev != nil &&
		(snap.blockNumber < prev.blockNumber || !snap.builtAt.After(prev.builtAt)) {
		return false
	}
	c.snapshot.Store(snap)
	return true
}

// buildSnapshot validates a boards frame. A frame for another chain or venue is refused whole; a
// board that does not parse is dropped alone, which leaves that maker dark for the direction.
func (c *feedClient) buildSnapshot(msg *feedMessage, receivedAt time.Time) (*feedSnapshot, error) {
	if msg.ChainID != c.chainID || !strings.EqualFold(msg.Venue, c.venue) {
		return nil, errors.New("snapshot for another chain or venue")
	}
	blockNumber, err := strconv.ParseUint(msg.BlockNumber, 10, 64)
	if err != nil || msg.BuiltAtMs <= 0 {
		return nil, errors.New("snapshot without block number or build time")
	}
	if msg.FeeBps >= bpsDenominator {
		return nil, errors.New("fee out of range")
	}

	boards := make(map[string]Board, len(msg.Boards))
	pending := make(map[string]bool, len(msg.Boards))
	for i := range msg.Boards {
		fb := &msg.Boards[i]
		key := boardKey(fb.Mm, fb.TokenIn, fb.TokenOut)
		// A pending board is what the chain holds once the keeper's commit lands, which on Base
		// is within a block or two of signing: well before a routed swap executes.
		if pending[key] && fb.Committed {
			continue
		}
		b, err := parseFeedBoard(fb)
		if err != nil {
			c.log().Warnf("biconomy-prop feed: dropping board %s: %v", key, err)
			continue
		}
		boards[key] = b
		pending[key] = !fb.Committed
	}
	makers := make([]string, len(msg.Makers))
	for i, m := range msg.Makers {
		makers[i] = strings.ToLower(m)
	}
	return &feedSnapshot{
		venue:       c.venue,
		blockNumber: blockNumber,
		feeBps:      msg.FeeBps,
		makers:      makers,
		boards:      boards,
		builtAt:     time.UnixMilli(msg.BuiltAtMs),
		receivedAt:  receivedAt,
	}, nil
}

var errFeedBoard = errors.New("malformed board")

func parseFeedBoard(fb *feedBoard) (Board, error) {
	if len(fb.Sizes) != len(fb.Prices) {
		return Board{}, errFeedBoard
	}
	b := Board{
		Sizes:  make([]*uint256.Int, len(fb.Sizes)),
		Prices: make([]*uint256.Int, len(fb.Prices)),
	}
	var err error
	for i := range fb.Sizes {
		if b.Sizes[i], err = uint256.FromDecimal(fb.Sizes[i]); err != nil {
			return Board{}, errFeedBoard
		}
		if b.Prices[i], err = uint256.FromDecimal(fb.Prices[i]); err != nil {
			return Board{}, errFeedBoard
		}
		// Cumulative sizes only grow; a ladder that does not is not one the executor holds.
		if i > 0 && !b.Sizes[i].Gt(b.Sizes[i-1]) {
			return Board{}, errFeedBoard
		}
	}
	if b.Filled, err = uint256.FromDecimal(fb.Filled); err != nil {
		return Board{}, errFeedBoard
	}
	if b.Remaining, err = uint256.FromDecimal(fb.Remaining); err != nil {
		return Board{}, errFeedBoard
	}
	if b.ExpiresAt, err = strconv.ParseUint(fb.ExpiresAt, 10, 64); err != nil {
		return Board{}, errFeedBoard
	}
	return b, nil
}
