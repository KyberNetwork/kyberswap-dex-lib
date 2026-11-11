package biconomyprop

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	weth = common.HexToAddress("0x4200000000000000000000000000000000000006")
	usdc = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
)

const feedVenue = "0x0000009e5f98cdb4dd1e1d2e15b220217c88b3b8"

func loadFeed(t *testing.T) *feedResponse {
	t.Helper()
	var r feedResponse
	// Recorded from GET /venue/boards?chainId=8453 (Base, 2026-10-27). Maker 0x960f quotes both
	// directions, maker 0x72A2 is dark.
	fixture, err := os.ReadFile("testdata/feed_base.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(fixture, &r))
	return &r
}

func rawOf(t *testing.T, fb *feedBoard) rawBoard {
	t.Helper()
	rb := rawBoard{Filled: bi(fb.Filled), Remaining: bi(fb.Remaining), ExpiresAt: bi(fb.ExpiresAt),
		Sizes: []*big.Int{}, Prices: []*big.Int{}}
	for i := range fb.Sizes {
		rb.Sizes = append(rb.Sizes, bi(fb.Sizes[i]))
		rb.Prices = append(rb.Prices, bi(fb.Prices[i]))
	}
	return rb
}

// The simulator trusts Extra, so the feed path must produce exactly what the on-chain path
// produces for the same boards.
func TestFeedExtraEqualsOnchainExtra(t *testing.T) {
	r := loadFeed(t)
	nowMs := r.BuiltAtMs + 500
	members, fee, block, err := buildFeedMembers(r, weth, usdc, feedVenue, 3000, nowMs)
	require.NoError(t, err)
	assert.Equal(t, uint64(53154517), block, "pool block is stateBlock")

	// token0 = weth, token1 = usdc; board order per maker: weth->usdc, usdc->weth
	var want []MemberExtra
	for _, m := range r.Makers {
		me := MemberExtra{Maker: strings.ToLower(m)}
		for i := range r.Boards {
			b := &r.Boards[i]
			if strings.ToLower(b.MM) != me.Maker {
				continue
			}
			if strings.EqualFold(b.TokenIn, weth.Hex()) {
				me.Dir0 = toBoard(rawOf(t, b))
			} else {
				me.Dir1 = toBoard(rawOf(t, b))
			}
		}
		want = append(want, me)
	}
	got, _ := json.Marshal(Extra{Members: members, FeeBps: fee})
	exp, _ := json.Marshal(Extra{Members: want, FeeBps: 0})
	assert.JSONEq(t, string(exp), string(got))
	assert.Len(t, members[0].Dir0.Sizes, 8)
	assert.Empty(t, members[1].Dir0.Sizes)
}

// A direction the feed omits must be dark, not absent: the simulator indexes both directions.
func TestFeedMissingDirectionIsDark(t *testing.T) {
	r := loadFeed(t)
	kept := r.Boards[:0:0]
	for _, b := range r.Boards {
		if b.MM != r.Makers[0] || b.TokenIn != usdc.Hex() {
			kept = append(kept, b)
		}
	}
	r.Boards = kept
	members, _, _, err := buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs)
	require.NoError(t, err)
	d := members[0].Dir1
	assert.True(t, d.Remaining.IsZero())
	assert.Empty(t, d.Sizes)
	assert.False(t, boardLive(&d, uint64(time.Now().Unix())))
	assert.True(t, boardLive(&members[0].Dir0, uint64(r.BuiltAtMs/1000)), "present direction stays live")
}

// The venue fills committed state only; a pending board must never replace it.
func TestFeedIgnoresUncommitted(t *testing.T) {
	r := loadFeed(t)
	pending := r.Boards[0]
	pending.Committed = false
	pending.Prices = []string{"9999999999", "9999999999"}
	r.Boards = append([]feedBoard{pending}, r.Boards...)
	members, _, _, err := buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs)
	require.NoError(t, err)
	assert.Equal(t, "2314114756", members[0].Dir0.Prices[0].Dec())

	// only a pending board for the direction: dark, not the pending price
	r = loadFeed(t)
	r.Boards[0].Committed = false
	members, _, _, err = buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs)
	require.NoError(t, err)
	assert.Empty(t, members[0].Dir0.Sizes)
}

func TestFeedStaleAndInvalid(t *testing.T) {
	r := loadFeed(t)
	_, _, _, err := buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs+3001)
	assert.ErrorIs(t, err, ErrFeedStale)
	_, _, _, err = buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs-3001)
	assert.ErrorIs(t, err, ErrFeedStale)
	_, _, _, err = buildFeedMembers(r, weth, usdc, "0x0000000000000000000000000000000000000001", 3000, r.BuiltAtMs)
	assert.ErrorIs(t, err, ErrFeedInvalid)
	r.StateBlock = ""
	_, _, _, err = buildFeedMembers(r, weth, usdc, feedVenue, 3000, r.BuiltAtMs)
	assert.ErrorIs(t, err, ErrFeedInvalid)
}

func trackerPool(t *testing.T) entity.Pool {
	se, _ := json.Marshal(StaticExtra{Venue: feedVenue})
	return entity.Pool{Address: feedVenue + "_x", StaticExtra: string(se),
		Tokens: []*entity.PoolToken{{Address: weth.Hex()}, {Address: usdc.Hex()}}}
}

// A fresh feed sets Extra and BlockNumber with no RPC call; a stale one falls back to RPC.
func TestTrackerFeedThenFallback(t *testing.T) {
	var age atomic.Int64
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-Api-Key"))
		assert.Equal(t, "8453", r.URL.Query().Get("chainId"))
		resp := loadFeed(t)
		resp.BuiltAtMs = time.Now().UnixMilli() - age.Load()
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer feedSrv.Close()
	var rpcHits atomic.Int32
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rpcHits.Add(1)
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer rpcSrv.Close()

	cfg := &Config{DexID: "biconomy-prop", ChainID: 8453, Venue: feedVenue,
		HTTP: &HTTPConfig{BaseURL: feedSrv.URL, Headers: map[string]string{"X-Api-Key": "k"}}}
	tr, err := NewPoolTracker(cfg, ethrpc.New(rpcSrv.URL))
	require.NoError(t, err)

	p, err := tr.GetNewPoolState(t.Context(), trackerPool(t), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(53154517), p.BlockNumber)
	assert.Zero(t, rpcHits.Load(), "fresh feed must not touch RPC")
	assert.Contains(t, p.Extra, "2314114756")

	assert.Equal(t, int32(0), rpcHits.Load())
	age.Store(10_000)
	_, err = tr.GetNewPoolState(t.Context(), trackerPool(t), pool.GetNewPoolStateParams{})
	assert.Error(t, err, "RPC is down, so the fallback error surfaces")
	assert.NotErrorIs(t, err, ErrFeedStale)
	assert.NotZero(t, rpcHits.Load(), "stale feed must fall back to RPC: "+strconv.Itoa(int(rpcHits.Load())))
}
