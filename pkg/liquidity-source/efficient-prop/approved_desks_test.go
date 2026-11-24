package efficientprop

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// deskListServer is an https server answering a desk list: body where set, else the list of chain, updatedAt and
// desks, with status (200 where 0).
type deskListServer struct {
	*httptest.Server
	mu        sync.Mutex
	chain     uint64
	updatedAt uint64
	desks     []common.Address
	status    int
	body      string
	hits      int
}

func newDeskListServer(t *testing.T, chain uint64, desks ...common.Address) *deskListServer {
	s := &deskListServer{chain: chain, updatedAt: 1, desks: desks}
	s.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.hits++
		if s.status != 0 {
			w.WriteHeader(s.status)
			return
		}
		if s.body != "" {
			_, _ = w.Write([]byte(s.body))
			return
		}
		quoted := make([]string, len(s.desks))
		for i, d := range s.desks {
			quoted[i] = `"` + hexutil.Encode(d[:]) + `"`
		}
		_, _ = fmt.Fprintf(w, `{"chainId":%d,"updatedAt":%d,"desks":[%s]}`, s.chain, s.updatedAt,
			strings.Join(quoted, ","))
	}))
	t.Cleanup(s.Close)
	return s
}

// set changes what the server answers, under its lock.
func (s *deskListServer) set(f func(s *deskListServer)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

// gateOf is the maker gate of a config naming static and, where srv is set, its url, on a clock the test moves.
func gateOf(srv *deskListServer, clock *time.Time, static ...common.Address) *approvedDesks {
	cfg := &Config{ChainID: valueobject.ChainIDEthereum}
	for _, a := range static {
		cfg.ApprovedDesks = append(cfg.ApprovedDesks, hexutil.Encode(a[:]))
	}
	if srv != nil {
		cfg.ApprovedDesksURL = srv.URL
	}
	d := newApprovedDesks(cfg)
	d.now = func() time.Time { return *clock }
	d.warn = func(error) {}
	if srv != nil {
		d.client = srv.Client()
	}
	return d
}

var (
	deskA = common.HexToAddress("0x00000000000000000000000000000000000000a1")
	deskB = common.HexToAddress("0x00000000000000000000000000000000000000b2")
	deskC = common.HexToAddress("0x00000000000000000000000000000000000000c3")
)

// With no url the static set rules, and an empty one approves nothing.
func TestApprovedDesksStaticSet(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ctx := context.Background()
	none := gateOf(nil, &now)
	assert.False(t, none.approved(ctx, deskA), "nothing is approved by default")
	d := gateOf(nil, &now, deskA)
	assert.True(t, d.approved(ctx, deskA))
	assert.False(t, d.approved(ctx, deskB))
	assert.False(t, d.approved(ctx, common.Address{}))
}

// A fetched list rules while it is fresh: a desk it names is approved though the static set does not name it, and a
// desk of the static set it leaves out is not. Once the list is stale and the url fails, the static set rules, less
// the desks the list had left out: the failure brings no desk in.
func TestApprovedDesksStaleListFallsBackToTheStaticSet(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ctx := context.Background()
	srv := newDeskListServer(t, 1, deskA, deskB)
	d := gateOf(srv, &now, deskA, deskC)
	assert.True(t, d.approved(ctx, deskA))
	assert.True(t, d.approved(ctx, deskB), "the list names it")
	assert.False(t, d.approved(ctx, deskC), "the fresh list leaves it out")

	// the url fails from here on: the held list rules to its max age
	srv.set(func(s *deskListServer) { s.status = http.StatusInternalServerError })
	now = now.Add(defaultDeskListMaxAge)
	assert.True(t, d.approved(ctx, deskB), "held to its max age")
	now = now.Add(time.Second)
	assert.True(t, d.approved(ctx, deskA), "the static set's, and the list named it")
	assert.False(t, d.approved(ctx, deskB), "the list's alone: gone once the list is stale")
	assert.False(t, d.approved(ctx, deskC), "the list had dropped it: a failing url does not bring it back")

	// a url that never answered: the static set
	down := newDeskListServer(t, 1)
	down.set(func(s *deskListServer) { s.status = http.StatusBadGateway })
	d = gateOf(down, &now, deskA)
	assert.True(t, d.approved(ctx, deskA))
	assert.False(t, d.approved(ctx, deskB))
}

// The list is fetched every deskListEvery at most, and a desk it drops is approved no more from that fetch on.
func TestApprovedDesksRevokedDeskDropsWithinOneFetch(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ctx := context.Background()
	srv := newDeskListServer(t, 1, deskA, deskB)
	d := gateOf(srv, &now, deskA, deskB)
	require.True(t, d.approved(ctx, deskB))
	srv.set(func(s *deskListServer) { s.desks, s.updatedAt = []common.Address{deskA}, 2 })
	now = now.Add(deskListEvery - time.Second)
	assert.True(t, d.approved(ctx, deskB), "no fetch is due yet")
	assert.Equal(t, 1, srv.hits)
	now = now.Add(time.Second)
	assert.False(t, d.approved(ctx, deskB), "dropped by the next fetch")
	assert.True(t, d.approved(ctx, deskA))
	assert.Equal(t, 2, srv.hits)
	// an empty list approves nothing
	srv.set(func(s *deskListServer) { s.desks, s.updatedAt = nil, 3 })
	now = now.Add(deskListEvery)
	assert.False(t, d.approved(ctx, deskA))
}

// A list is refused whole, and the one held stands, where it names another chain, is too long, names a bad address or
// no updatedAt, or is older than the one held.
func TestApprovedDesksRefusesABadList(t *testing.T) {
	ctx := context.Background()
	for name, bad := range map[string]func(s *deskListServer){
		"another chain": func(s *deskListServer) { s.chain, s.updatedAt = 56, 5 },
		"too long": func(s *deskListServer) {
			s.body = `{"chainId":1,"updatedAt":5,"desks":[],"pad":"` + strings.Repeat("x", maxDeskListBytes) + `"}`
		},
		"a bad address": func(s *deskListServer) {
			s.body = `{"chainId":1,"updatedAt":5,"desks":["0x00000000000000000000000000000000000000b2","0x12"]}`
		},
		"no updatedAt":  func(s *deskListServer) { s.body = `{"chainId":1,"desks":["` + hexutil.Encode(deskB[:]) + `"]}` },
		"an older list": func(s *deskListServer) { s.desks, s.updatedAt = []common.Address{deskB}, 1 },
		"not json":      func(s *deskListServer) { s.body = "<html>" },
	} {
		now := time.Unix(1_800_000_000, 0)
		srv := newDeskListServer(t, 1, deskA)
		srv.set(func(s *deskListServer) { s.updatedAt = 2 })
		d := gateOf(srv, &now)
		var warned int
		d.warn = func(error) { warned++ }
		require.True(t, d.approved(ctx, deskA), name)
		srv.set(bad)
		now = now.Add(deskListEvery)
		assert.False(t, d.approved(ctx, deskB), "%s: the bad list's desk", name)
		assert.True(t, d.approved(ctx, deskA), "%s: the held list stands", name)
		assert.Equal(t, 1, warned, name)
	}
}

// The url is https or the config is refused, and a list never comes over plain http.
func TestApprovedDesksURLIsHTTPS(t *testing.T) {
	cfg := testConfig()
	cfg.ApprovedDesksURL = "http://desks.example/list"
	assert.ErrorIs(t, cfg.validate(), ErrInvalidConfig)
	cfg.ApprovedDesksURL = "https://desks.example/list"
	assert.NoError(t, cfg.validate())
	cfg.ApprovedDesks = []string{"0x12"}
	assert.ErrorIs(t, cfg.validate(), ErrInvalidConfig)

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"chainId":1,"updatedAt":1,"desks":["%s"]}`, hexutil.Encode(deskB[:]))
	}))
	defer plain.Close()
	now := time.Unix(1_800_000_000, 0)
	d := gateOf(nil, &now)
	d.url = plain.URL
	assert.False(t, d.approved(context.Background(), deskB))
}

// A listing is listed and quoted only where its desk's contract is approved: not under a config that approves none,
// not where it approves another desk, and never a listing bound to no desk. On a chain read at the head and on a
// builder chain.
func TestOnlyAnApprovedDesksListingIsListedAndQuoted(t *testing.T) {
	ctx := context.Background()
	other := cellOf(9)
	quoted := func(p entity.Pool) bool {
		e := extraOf(t, p)
		return len(e.Ladders[0])+len(e.Ladders[1]) > 0
	}

	// read at the head
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	require.True(t, quoted(track(t, n, p)), "its desk is approved")
	for name, desks := range map[string][]string{"none approved": nil, "another desk": {hexutil.Encode(other[:])}} {
		cfg := testConfig()
		cfg.ApprovedDesks = desks
		u, err := NewPoolsListUpdater(cfg, n.client())
		require.NoError(t, err)
		pools, _, err := u.GetNewPools(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, pools, name)
		tr, err := NewPoolTracker(cfg, n.client())
		require.NoError(t, err)
		got, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		assert.False(t, quoted(got), name)
	}
	chain.listings[1].view.Desk = 0
	assert.False(t, quoted(track(t, chain.node(t, 1234), p)), "a listing bound to no desk")

	// a builder chain
	eth := ethChain()
	en := builderNode(t, eth, 1234)
	srv := newStateServer(t)
	srv.set(slotAfter(en.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(srv.URL)
	ep := ethListed(t, en, cfg)
	require.True(t, quoted(ethTrack(t, en, cfg, ep)), "its desk is approved")
	for name, desks := range map[string][]string{"none approved": nil, "another desk": {hexutil.Encode(other[:])}} {
		cfg := ethConfig(srv.URL)
		cfg.ApprovedDesks = desks
		u, err := NewPoolsListUpdater(cfg, en.client())
		require.NoError(t, err)
		pools, _, err := u.GetNewPools(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, pools, name)
		assert.False(t, quoted(ethTrack(t, en, cfg, ep)), name)
	}
}

// A tracker under a desk list url quotes a listing while the list names its desk, and no more from the refresh after
// the fetch that drops it.
func TestTrackerDropsARevokedDeskWithinOneRefresh(t *testing.T) {
	ctx := context.Background()
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	srv := newDeskListServer(t, uint64(valueobject.ChainIDBase), cellOf(testDesk))
	cfg := testConfig()
	cfg.ApprovedDesks, cfg.ApprovedDesksURL = nil, srv.URL
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	now := time.Now()
	tr.desks.client, tr.desks.now = srv.Client(), func() time.Time { return now }
	ladders := func() int {
		got, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		e := extraOf(t, got)
		return len(e.Ladders[0]) + len(e.Ladders[1])
	}
	require.NotZero(t, ladders(), "the list names its desk")
	srv.set(func(s *deskListServer) { s.desks, s.updatedAt = nil, 2 })
	now = now.Add(deskListEvery)
	assert.Zero(t, ladders(), "revoked: the next refresh quotes nothing")
}
