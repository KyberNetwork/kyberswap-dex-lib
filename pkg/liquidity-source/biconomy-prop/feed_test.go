package biconomyprop

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// testdata/boards-base.json is a feed message built by the orchestrator's own code from Base at
// block 52847598: both makers' committed boards, plus one pending WETH -> USDC board for
// 0x72a2...890c re-signed one nonce ahead at prices 0.1% lower. The quotes below are
// PropAMMVenue.quote at that block, read from Base mainnet.
const (
	fixtureBlock     = 52847598
	fixtureTimestamp = 1792484543
	liveMaker        = "0x72a2df03aeef2a8343f9fb86bd65193a3008890c"
	darkMaker        = "0x960f1e3181e928210c02a34ee29bf3d354dd2ff5"
	testUSDC         = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/boards-base.json")
	require.NoError(t, err)
	return data
}

// withoutPending drops the pending entries, leaving the message as the chain stood at the block.
func withoutPending(t *testing.T, data []byte) []byte {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal(data, &msg))
	var kept []any
	for _, b := range msg["boards"].([]any) {
		if b.(map[string]any)["committed"].(bool) {
			kept = append(kept, b)
		}
	}
	msg["boards"] = kept
	out, err := json.Marshal(msg)
	require.NoError(t, err)
	return out
}

// fresh stamps the message as built just now, as a live feed would.
func fresh(t *testing.T, data []byte) []byte {
	t.Helper()
	var msg map[string]any
	require.NoError(t, json.Unmarshal(data, &msg))
	msg["builtAtMs"] = time.Now().UnixMilli()
	out, err := json.Marshal(msg)
	require.NoError(t, err)
	return out
}

func testFeed() *feedClient {
	return &feedClient{chainID: 8453, venue: strings.ToLower(testVenue)}
}

func TestFeedHandle_PrefersPendingBoard(t *testing.T) {
	c := testFeed()
	require.True(t, c.handle(fixture(t), time.Now()))
	snap := c.latest()
	require.NotNil(t, snap)

	assert.Equal(t, uint64(fixtureBlock), snap.blockNumber)
	assert.Equal(t, []string{darkMaker, liveMaker}, snap.makers)

	sell := snap.board(liveMaker, testToken0, testUSDC)
	require.Len(t, sell.Prices, 4)
	assert.Equal(t, "2383572429", sell.Prices[0].Dec(), "pending board, not the committed 2385958388")
	buy := snap.board(liveMaker, testUSDC, testToken0)
	assert.Equal(t, "20416093", buy.Remaining.Dec())

	dark := snap.board(darkMaker, testToken0, testUSDC)
	assert.Empty(t, dark.Sizes)
	assert.True(t, dark.Remaining.IsZero())
	missing := snap.board(liveMaker, testToken0, "0x0000000000000000000000000000000000000001")
	assert.True(t, missing.Remaining.IsZero())
}

func TestFeedHandle_PendingWinsInEitherOrder(t *testing.T) {
	var msg feedMessage
	require.NoError(t, json.Unmarshal(fixture(t), &msg))
	for i, j := 0, len(msg.Boards)-1; i < j; i, j = i+1, j-1 {
		msg.Boards[i], msg.Boards[j] = msg.Boards[j], msg.Boards[i]
	}
	snap, err := testFeed().buildSnapshot(&msg, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2383572429", snap.board(liveMaker, testToken0, testUSDC).Prices[0].Dec())
}

func TestFeedHandle_RefusesOtherChainOrVenue(t *testing.T) {
	other := testFeed()
	other.venue = "0x0000000000000000000000000000000000000001"
	assert.False(t, other.handle(fixture(t), time.Now()))
	assert.Nil(t, other.latest())

	otherChain := testFeed()
	otherChain.chainID = 84532
	assert.False(t, otherChain.handle(fixture(t), time.Now()))
}

func TestFeedHandle_IgnoresOtherFramesAndOlderSnapshots(t *testing.T) {
	c := testFeed()
	for _, frame := range []string{`{"type":"subscribed","chainId":8453}`, `{"type":"pong"}`,
		`{"type":"error","code":"RATE_LIMITED","message":"x"}`, `not json`} {
		assert.False(t, c.handle([]byte(frame), time.Now()), frame)
	}
	assert.Nil(t, c.latest())

	require.True(t, c.handle(fixture(t), time.Now()))
	first := c.latest()
	// The same message again (same build time) is ignored.
	assert.False(t, c.handle(fixture(t), time.Now()))
	assert.Same(t, first, c.latest())
	// A later build replaces it, after which the earlier one is ignored.
	later := strings.Replace(string(fixture(t)), `"builtAtMs":`, `"builtAtMs":1`, 1) // a leading 1: far later
	require.True(t, c.handle([]byte(later), time.Now()))
	assert.False(t, c.handle(fixture(t), time.Now()))
	assert.NotSame(t, first, c.latest())
}

func TestFeedHandle_DropsAMalformedBoardAlone(t *testing.T) {
	var msg feedMessage
	require.NoError(t, json.Unmarshal(fixture(t), &msg))
	for i := range msg.Boards {
		if strings.EqualFold(msg.Boards[i].TokenIn, testUSDC) && strings.EqualFold(msg.Boards[i].Mm, liveMaker) {
			msg.Boards[i].Sizes = msg.Boards[i].Sizes[:1] // sizes and prices no longer line up
		}
	}
	snap, err := testFeed().buildSnapshot(&msg, time.Now())
	require.NoError(t, err)
	assert.True(t, snap.board(liveMaker, testUSDC, testToken0).Remaining.IsZero())
	assert.Len(t, snap.board(liveMaker, testToken0, testUSDC).Sizes, 4)
}

func trackerPool(t *testing.T) entity.Pool {
	t.Helper()
	staticExtra, err := json.Marshal(StaticExtra{Venue: testVenue})
	require.NoError(t, err)
	return entity.Pool{
		Address:     testVenue + "_" + testToken0 + "_" + testToken1,
		Exchange:    "biconomy-prop",
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		StaticExtra: string(staticExtra),
	}
}

// The simulator built from the feed reproduces PropAMMVenue.quote at the fixture block exactly.
func TestTracker_FeedMatchesVenueQuote(t *testing.T) {
	feed := testFeed()
	require.True(t, feed.handle(fresh(t, withoutPending(t, fixture(t))), time.Now()))
	tracker := &PoolTracker{config: &Config{ChainID: 8453, Venue: testVenue}, feed: feed}

	p, err := tracker.GetNewPoolState(context.Background(), trackerPool(t), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(fixtureBlock), p.BlockNumber)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	sim.now = fixtureTimestamp

	for _, tc := range []struct {
		in, out, amount, want string
	}{
		{testToken0, testToken1, "300000000000000", "715787"},
		{testToken0, testToken1, "645822708000000", "1540905"},
		{testToken1, testToken0, "10000000", "4189620641290210"},
		{testToken1, testToken0, "20416093", "8553568464730056"},
	} {
		amount, _ := new(big.Int).SetString(tc.amount, 10)
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: amount},
			TokenOut:      tc.out,
		})
		require.NoError(t, err, tc.amount)
		assert.Equal(t, tc.want, res.TokenAmountOut.Amount.String(), tc.amount)
	}

	// One wei past what the board can take is refused, as the venue refuses it.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: big.NewInt(645822708000001)},
		TokenOut:      testToken1,
	})
	assert.Error(t, err)
}

func TestTracker_PendingBoardPricesBelowCommitted(t *testing.T) {
	feed := testFeed()
	require.True(t, feed.handle(fresh(t, fixture(t)), time.Now()))
	tracker := &PoolTracker{config: &Config{ChainID: 8453, Venue: testVenue}, feed: feed}
	p, err := tracker.GetNewPoolState(context.Background(), trackerPool(t), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	sim.now = fixtureTimestamp

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: big.NewInt(300000000000000)},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	assert.Less(t, res.TokenAmountOut.Amount.Int64(), int64(715787))
}

// A stale snapshot sends the tracker back on-chain; here that read fails, proving it was tried.
func TestTracker_StaleSnapshotFallsBackOnChain(t *testing.T) {
	feed := testFeed()
	require.True(t, feed.handle(fixture(t), time.Now().Add(-time.Minute)))
	tracker := &PoolTracker{
		config:       &Config{ChainID: 8453, Venue: testVenue},
		ethrpcClient: ethrpc.New("http://127.0.0.1:1"),
		feed:         feed,
	}
	_, err := tracker.GetNewPoolState(context.Background(), trackerPool(t), pool.GetNewPoolStateParams{})
	assert.Error(t, err)
}

func TestFeedClient_SubscribesWithAPIKeyAndReceives(t *testing.T) {
	data := fixture(t)
	gotKey := make(chan string, 1)
	gotSub := make(chan feedSubscribe, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey <- r.Header.Get(feedAPIKeyHeader)
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		var sub feedSubscribe
		if conn.ReadJSON(&sub) != nil {
			return
		}
		gotSub <- sub
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"subscribed","chainId":8453}`))
		_ = conn.WriteMessage(websocket.TextMessage, data)
		_, _, _ = conn.ReadMessage() // hold the connection until the client leaves
	}))
	defer srv.Close()

	cfg := &Config{
		ChainID: 8453,
		Venue:   testVenue,
		Feed:    FeedConfig{URL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/venue-stream", APIKey: "k-" + strconv.Itoa(time.Now().Nanosecond())},
	}
	c := acquireFeed(cfg)
	defer c.close()

	assert.Equal(t, cfg.Feed.APIKey, <-gotKey)
	assert.Equal(t, feedSubscribe{Type: feedMsgSubscribe, ChainID: 8453}, <-gotSub)
	require.Eventually(t, func() bool { return c.latest() != nil }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, uint64(fixtureBlock), c.latest().blockNumber)
}
