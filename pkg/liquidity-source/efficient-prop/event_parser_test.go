package efficientprop

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var (
	// the desk contracts of desks 7 and 9
	testCellA = cellOf(7)
	testCellB = cellOf(9)
)

func door(id uint32) string {
	d := fakeDoorOf(testFactory, id)
	return hexutil.Encode(d[:])
}

func id(n uint32) common.Hash { return common.BigToHash(big.NewInt(int64(n))) }

// The tracker asks to be refreshed every second after the state's block time.
func TestTrackerSchedulesARefreshEverySecond(t *testing.T) {
	var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)
	tr, err := NewPoolTracker(testConfig(), nil)
	require.NoError(t, err)
	at := func(ts int64, validUntil uint64) int64 {
		b, err := json.Marshal(Extra{ValidUntil: validUntil})
		require.NoError(t, err)
		return tr.NextUpdateAt(entity.Pool{Timestamp: ts, Extra: string(b)})
	}
	assert.Equal(t, int64(1001), at(1000, 1030))
	assert.Equal(t, int64(1001), at(1000, 0), "no word: still every second")
	assert.Equal(t, int64(1001), at(1000, 1001))
}

// The parser maps each front log naming a listing to that listing's door, computed from the configured factory and
// the door codehash it answers, as the factory computes it; other logs of the front, and logs of another front, map to
// nothing.
func TestEventParserMapsTheFrontsLogsToTheDoor(t *testing.T) {
	var _ poolfactory.IPoolFactoryDecoder = (*EventParser)(nil)
	p := NewPoolFactory(testConfig(), testChain().node(t, 900).client())
	log := func(event string, topics ...common.Hash) types.Log {
		return types.Log{Address: testFront, Topics: append([]common.Hash{frontABI.Events[event].ID}, topics...)}
	}
	pair := common.BigToHash(big.NewInt(1))
	logs := []types.Log{
		log("PoolFill", pair, id(1)),
		log("DepthPosted", id(1)),
		log("ControlsPosted", id(4)),
		log("SettingsSet", id(4)),
		log("PoolStatus", id(1)),
		log("PoolRetired", id(4)),
		log("ListingDelisted", id(4)),
		log("ListingRelisted", id(4)),
		log("ListingBound", id(1), id(1)),
		log("SourceDarkened", id(1)),
		log("LeaseSet", id(1)),
		log("PoolRegistered", pair, id(7)),
		log("PoolReserved", pair, id(7)),
		log("PairAdded"),
		{Address: testPoolB, Topics: []common.Hash{frontABI.Events["PoolFill"].ID, pair, id(1)}},
	}
	got, err := p.Decode(context.Background(), logs)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Len(t, got[door(1)], 6)
	assert.Len(t, got[door(4)], 5)
	assert.Len(t, got[door(7)], 2)
}

// On a storage build the door called logs the fill's Swapped, which maps to the door itself. (On a builder build the
// listing's quote contract logs it: TestEventParserMapsADesksSwappedOnABuilderChain.)
func TestEventParserMapsADoorsSwapped(t *testing.T) {
	p := NewPoolFactory(testConfig(), nil)
	d := fakeDoorOf(testFactory, 1)
	got, err := p.Decode(context.Background(), []types.Log{{Address: d,
		Topics: []common.Hash{doorABI.Events["Swapped"].ID, {}, {}, {}}}})
	require.NoError(t, err)
	assert.Len(t, got[door(1)], 1)
}

// A desk cell's DeskPush and LaneFill map to the doors of every listing bound to its desk, learned from the front;
// the parser learns again after a binding, and when a cell it does not know logs.
func TestEventParserMapsADesksCellToItsListings(t *testing.T) {
	chain := testChain()
	chain.listings[1].view.Desk = 7
	chain.listings[4].view.Desk = 7
	n := chain.node(t, 900)
	p := NewPoolFactory(testConfig(), n.client())
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	fill := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	got, err := p.Decode(context.Background(), []types.Log{push, fill})
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Len(t, got[door(1)], 2)
	assert.Len(t, got[door(4)], 2)

	// listing 4 rebound to desk 2: the binding's log makes the parser learn again at the next cell log
	chain.listings[4].view.Desk = 9
	bound := types.Log{Address: testFront,
		Topics: []common.Hash{frontABI.Events["ListingBound"].ID, id(4), id(2)}}
	pushB := types.Log{Address: testCellB, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	got, err = p.Decode(context.Background(), []types.Log{bound, push, pushB})
	require.NoError(t, err)
	assert.Equal(t, []types.Log{push}, got[door(1)])
	assert.Equal(t, []types.Log{bound, pushB}, got[door(4)])

	// a cell no desk of the front has maps to nothing, and is not learned for again at once
	before := len(n.calls())
	other := types.Log{Address: common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd"),
		Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	got, err = p.Decode(context.Background(), []types.Log{other})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(n.calls()), "learned again within relearnAfter")
	p.learnedAt = time.Now().Add(-relearnAfter)
	_, err = p.Decode(context.Background(), []types.Log{other})
	require.NoError(t, err)
	assert.Greater(t, len(n.calls()), before)
}

// A listing not live when the parser learns (delisted, or reserved and not yet live) is learned too: once it is
// relisted, its desk's posts reach its door.
func TestEventParserMapsADesksCellToAListingRelistedLater(t *testing.T) {
	chain := testChain()
	chain.listings[1].view.Desk = 7
	chain.listings[4].view.Desk = 7
	chain.listings[4].view.Flags = 0
	n := chain.node(t, 900)
	p := NewPoolFactory(testConfig(), n.client())
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	_, err := p.Decode(context.Background(), []types.Log{push})
	require.NoError(t, err)

	chain.listings[4].view.Flags = 1
	relisted := types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["ListingRelisted"].ID, id(4)}}
	got, err := p.Decode(context.Background(), []types.Log{relisted, push})
	require.NoError(t, err)
	assert.Equal(t, []types.Log{relisted, push}, got[door(4)])
	assert.Equal(t, []types.Log{push}, got[door(1)])
}

// A listing reserved after the parser learned makes it learn again: the reservation's log marks what it learned stale,
// so the next post of a desk it already knows reaches the new listing's door too.
func TestEventParserLearnsAgainAfterAReservation(t *testing.T) {
	chain := testChain()
	chain.listings[1].view.Desk = 7
	n := chain.node(t, 900)
	p := NewPoolFactory(testConfig(), n.client())
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	got, err := p.Decode(context.Background(), []types.Log{push})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {push}}, got)

	// listing 5 reserved on desk 1
	v := testView(testPoolA)
	v.Desk = 7
	chain.listings[5] = &fakeListing{view: v}
	reserved := types.Log{Address: testFront,
		Topics: []common.Hash{frontABI.Events["PoolReserved"].ID, id(1), id(5)}}
	got, err = p.Decode(context.Background(), []types.Log{reserved, push})
	require.NoError(t, err)
	assert.Equal(t, []types.Log{reserved, push}, got[door(5)], "the new listing's door")
	assert.Equal(t, []types.Log{push}, got[door(1)])
}

// posted is a stored price lane's Posted log, which names no listing: the lane is its address, the feeds it stored
// its data.
func posted(lane common.Address, stored int64) types.Log {
	return types.Log{Address: lane, Topics: []common.Hash{crypto.Keccak256Hash([]byte("Posted(uint256)"))},
		Data: common.BigToHash(big.NewInt(stored)).Bytes()}
}

// A post of a stored price lane the front's config approves (Posted, the lane its address) maps to the doors of every
// listing that reads a print bound to a source on that lane, its primary or its backup, learned from the front's
// storage; a listing reading no print, a tied one, one bound to another source, a lane not approved and a post that
// stored nothing map to nothing; a listing whose primary and backup are both on the lane gets its post once. The
// binding is read at the block the parser learns at. A post of an approved lane bound to no listing does not make the
// parser learn again, a source change does.
func TestEventParserMapsALanesPostToItsListings(t *testing.T) {
	chain := testChain()
	ls := chain.listings
	ls[1].view.Scopes, ls[1].sourceIds = 0, new(big.Int).SetUint64(1|65_538<<32) // PRINT, both entries on the lane
	ls[2].view.Scopes, ls[2].sourceIds = 6, new(big.Int).SetUint64(3|4<<32)      // SESSION, its backup guarded on it
	ls[3].view.Scopes, ls[3].sourceIds = 0, big.NewInt(3)                        // PRINT on a lane not approved
	ls[4].view.Scopes, ls[4].sourceIds = 2, big.NewInt(1)                        // WORD: reads no print
	ls[6].view.Scopes, ls[6].sourceIds = 0, big.NewInt(1)                        // never reached by the scan
	n := chain.node(t, 900)
	p := NewPoolFactory(testConfig(), n.client())

	post := posted(testSource, 3)
	got, err := p.Decode(context.Background(), []types.Log{post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {post}, door(2): {post}}, got)
	learnedAt := hexutil.EncodeUint64(900)
	for _, slot := range []common.Hash{recordSlot(1, 10), sourceEntrySlot(1), sourceEntrySlot(65_538)} {
		assert.Equal(t, []string{learnedAt}, storageReads(chain, n, slot), "the binding read at the learned block")
	}

	// a lane not approved, and a post that stored nothing: nothing, and no learning again
	before := len(n.calls())
	got, err = p.Decode(context.Background(), []types.Log{posted(testThird, 2), posted(testSource, 0)})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(n.calls()), "no learning for a lane not approved or an empty post")

	// tied on the lane: not quoted, so not mapped
	ls[1].view.Scopes = 16
	bound := types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["ListingBound"].ID, id(1), id(1)}}
	got, err = p.Decode(context.Background(), []types.Log{bound, post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {bound}, door(2): {post}}, got)

	// listing 2's source moved off the lane, listing 3's onto it: the front's setSource logs a source change
	// (SettingsSet of kind 34) for each, which makes the parser learn again
	ls[2].sourceIds, ls[3].sourceIds = big.NewInt(3), big.NewInt(65_538)
	moved2, moved3 := sourceSet(2, 3), sourceSet(3, 65_538)
	got, err = p.Decode(context.Background(), []types.Log{moved2, moved3, post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(2): {moved2}, door(3): {moved3, post}}, got)

	// an approved lane no listing is bound to maps to nothing, and does not make the parser learn again
	ls[3].sourceIds = big.NewInt(3)
	p.stale = true
	_, err = p.Decode(context.Background(), []types.Log{post})
	require.NoError(t, err)
	before = len(n.calls())
	p.learnedAt = time.Now().Add(-relearnAfter)
	got, err = p.Decode(context.Background(), []types.Log{post})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(n.calls()), "an approved lane is known though no listing is bound to it")
	// a lane not approved is not looked up, however long ago the parser learned
	got, err = p.Decode(context.Background(), []types.Log{posted(testThird, 2)})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(n.calls()), "a lane not approved makes the parser learn again")
}

// sourceSet is the front's log of a listing's source change (setSource): SettingsSet(listing, kind 34, the source ids
// as record word 10 holds them).
func sourceSet(listing uint32, ids uint64) types.Log {
	data := append(common.BigToHash(big.NewInt(34)).Bytes(), common.BigToHash(new(big.Int).SetUint64(ids)).Bytes()...)
	return types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["SettingsSet"].ID, id(listing)},
		Data: data}
}

// A listing's source change (the front's SettingsSet of kind 34, which logs no ListingBound) makes the parser learn
// again, so the new lane's posts reach its door at once, and the old lane's no longer do; a SettingsSet of another
// kind maps to the listing's door and does not.
func TestEventParserLearnsAgainAfterASourceChange(t *testing.T) {
	chain := testChain()
	ls := chain.listings
	ls[1].view.Scopes, ls[1].sourceIds = 0, big.NewInt(3) // PRINT on testThird, approved below
	n := chain.node(t, 900)
	p := NewPoolFactory(approving(testSource, testThird), n.client())

	onThird, onSource := posted(testThird, 1), posted(testSource, 1)
	got, err := p.Decode(context.Background(), []types.Log{onThird, onSource})
	require.NoError(t, err)
	require.Equal(t, map[string][]types.Log{door(1): {onThird}}, got, "before the change")

	// a change of the listing's settings (kind 0): its door, and no learning again
	before := len(n.calls())
	settings := types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["SettingsSet"].ID, id(1)},
		Data: append(common.BigToHash(big.NewInt(0)).Bytes(), common.BigToHash(big.NewInt(7)).Bytes()...)}
	got, err = p.Decode(context.Background(), []types.Log{settings, onThird})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {settings, onThird}}, got)
	assert.Equal(t, before, len(n.calls()), "a settings change makes the parser learn again")

	// the operator's setSource(listing 1, source 1, no backup) applies: word 10 names source 1, on testSource
	ls[1].sourceIds = big.NewInt(1)
	set := sourceSet(1, 1)
	got, err = p.Decode(context.Background(), []types.Log{set, onSource, onThird})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {set, onSource}}, got, "the new lane's post right after the change")

	// and however long after
	p.mu.Lock()
	p.learnedAt = time.Now().Add(-10 * relearnAfter)
	p.mu.Unlock()
	got, err = p.Decode(context.Background(), []types.Log{onSource, onThird})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {onSource}}, got, "the new lane's post long after the change")
}

// limitedNode is a node behind a provider that refuses (HTTP 413) every JSON-RPC batch refuse says no to, and the
// largest batch it was sent.
func limitedNode(t *testing.T, n *node, refuse func([]rpcMessage) bool) (*ethrpc.Client, *atomic.Int64) {
	largest := new(atomic.Int64)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var msgs []rpcMessage
		if json.Unmarshal(body, &msgs) == nil {
			if int64(len(msgs)) > largest.Load() {
				largest.Store(int64(len(msgs)))
			}
			if refuse(msgs) {
				http.Error(w, "batch refused", http.StatusRequestEntityTooLarge)
				return
			}
		}
		resp, err := http.Post(n.URL, "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(proxy.Close)
	return ethrpc.New(proxy.URL).SetMulticallContract(multicall3), largest
}

// manyPrintListings is a chain of count PRINT listings of desk 1 (cell testCellA), each bound to source 1 on the
// approved lane.
func manyPrintListings(count uint32) *fakeChain {
	chain := testChain()
	chain.listings = map[uint32]*fakeListing{}
	for id := uint32(1); id <= count; id++ {
		v := testView(testPoolA)
		v.Scopes, v.Desk = 0, 7
		chain.listings[id] = &fakeListing{view: v, open: true, sourceIds: big.NewInt(1)}
	}
	return chain
}

// The lanes' binding is read in batches of scanBatch storage reads, so a provider that refuses a batch of more than
// 100 requests serves it, and every batch is read: with 120 print listings both a desk's post and the lane's reach
// every door.
func TestEventParserReadsTheLanesInBatches(t *testing.T) {
	chain := manyPrintListings(120)
	client, largest := limitedNode(t, chain.node(t, 900), func(msgs []rpcMessage) bool { return len(msgs) > 100 })
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	post := posted(testSource, 1)
	got, err := NewPoolFactory(testConfig(), client).Decode(context.Background(), []types.Log{push, post})
	require.NoError(t, err)
	assert.Len(t, got, 120)
	for id := uint32(1); id <= 120; id++ {
		assert.Equal(t, []types.Log{push, post}, got[door(id)], "listing %d", id)
	}
	assert.LessOrEqual(t, largest.Load(), int64(scanBatch), "the largest batch sent")
}

// A refused read of the lanes' binding leaves the lanes unmapped and keeps what the same learning read of the desks:
// a desk's post still reaches its doors, a lane's post reaches none, and the parser learns again at any log, a desk's
// post included, no sooner than relearnAfter.
func TestEventParserKeepsTheCellsWhereTheLanesAreRefused(t *testing.T) {
	chain := manyPrintListings(3)
	n := chain.node(t, 900)
	var refusing atomic.Bool
	refusing.Store(true)
	client, _ := limitedNode(t, n, func(msgs []rpcMessage) bool {
		return refusing.Load() && slices.ContainsFunc(msgs, func(m rpcMessage) bool { return m.Method == "eth_getStorageAt" })
	})
	p := NewPoolFactory(testConfig(), client)
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	post := posted(testSource, 1)
	got, err := p.Decode(context.Background(), []types.Log{push, post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {push}, door(2): {push}, door(3): {push}}, got)

	// the lane's next post within relearnAfter: no learning again
	before := len(n.calls())
	got, err = p.Decode(context.Background(), []types.Log{post})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(n.calls()), "learned again within relearnAfter")

	// the provider serves the reads again, and relearnAfter has passed: the desk's next post makes the parser learn
	// again, and the lane's post reaches its doors
	refusing.Store(false)
	p.mu.Lock()
	p.learnedAt = time.Now().Add(-relearnAfter)
	p.mu.Unlock()
	got, err = p.Decode(context.Background(), []types.Log{push})
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Greater(t, len(n.calls()), before, "learned again at the desk's post")
	got, err = p.Decode(context.Background(), []types.Log{post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {post}, door(2): {post}, door(3): {post}}, got)
}

// A desk's stop, revive and key change map to the doors of every listing bound to that desk, a listing's maker key and
// asset module to its door, and the front's pause and unpause, a pair's retirement, a pair's or the protocol's asset
// module and a lease set for every listing to every door of the front.
func TestEventParserMapsDeskAndFrontWideLogs(t *testing.T) {
	chain := testChain()
	chain.listings[1].view.Desk = 7
	chain.listings[4].view.Desk = 9
	n := chain.node(t, 900)
	p := NewPoolFactory(testConfig(), n.client())
	log := func(event string, topics ...common.Hash) types.Log {
		ev, ok := frontABI.Events[event]
		require.True(t, ok, "%s is not in the front's ABI", event)
		return types.Log{Address: testFront, Topics: append([]common.Hash{ev.ID}, topics...)}
	}
	for _, event := range []string{"DeskStopped", "DeskRevived", "DeskKeySet"} {
		l := log(event, id(9))
		got, err := p.Decode(context.Background(), []types.Log{l})
		require.NoError(t, err)
		assert.Equal(t, map[string][]types.Log{door(4): {l}}, got, event)
	}
	key := log("MakerKeySet", id(1))
	got, err := p.Decode(context.Background(), []types.Log{key})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {key}}, got)

	// a listing's own asset module: its door
	module := log("AssetModuleSet", id(1), id(4))
	got, err = p.Decode(context.Background(), []types.Log{module})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(4): {module}}, got)

	for name, l := range map[string]types.Log{
		"AllPaused":   log("AllPaused"),
		"AllUnpaused": log("AllUnpaused"),
		// a pair retired: its listings fill no more
		"PairRetired": log("PairRetired", id(1)),
		// a pair's module bound or removed for good, or the protocol's (pair 0): every listing of the pair, or later
		"AssetModuleSet of a pair": log("AssetModuleSet", id(1), id(0)),
		"AssetModuleSet of pair 0": log("AssetModuleSet", id(0), id(0)),
		// every lease cut, or that cut lifted
		"LeaseSet of every listing": log("LeaseSet", id(0)),
	} {
		got, err := p.Decode(context.Background(), []types.Log{l})
		require.NoError(t, err)
		// every listing the front reserved, live or not, its door opened or not: 1 to 4
		assert.Len(t, got, 4, name)
		for i := uint32(1); i <= 4; i++ {
			assert.Equal(t, []types.Log{l}, got[door(i)], "%s, listing %d", name, i)
		}
	}
}

// The three events' topics are the front's: their signatures as IPropAMMFront declares them.
func TestPairAndModuleEventTopics(t *testing.T) {
	for event, sig := range map[string]string{
		"PairRetired":    "PairRetired(uint32)",
		"AssetModuleSet": "AssetModuleSet(uint32,uint24,address)",
		"LeaseSet":       "LeaseSet(uint24,uint32,address)",
	} {
		ev, ok := frontABI.Events[event]
		require.True(t, ok, "%s is not in the front's ABI", event)
		assert.Equal(t, crypto.Keccak256Hash([]byte(sig)), ev.ID, event)
	}
	assert.Equal(t, 2, listingTopic[frontABI.Events["AssetModuleSet"].ID], "AssetModuleSet names its listing at topic 2")
}

// The offline door address is CREATE2's: EIP-1014's example 5 (deployer 0xdeadbeef, salt 0xcafebabe, creation code
// 0xdeadbeef), the listing id as the salt. The fork test checks it against a deployed factory's doorOf.
func TestDoorOfIsCreate2OfTheListingId(t *testing.T) {
	f := Front{DoorFactory: "0x00000000000000000000000000000000deadbeef"}
	assert.Equal(t, "0x60f3f640a8508fc6a86d45df051962668e1e8ac7",
		doorOf(f, crypto.Keccak256Hash(common.FromHex("0xdeadbeef")), 0xcafebabe))
}

// The topics are the contracts' own: DeskPush as the cell logs it (log1 in its assembly), PoolFill as the front logs
// it on the fork.
func TestEventTopics(t *testing.T) {
	assert.Equal(t, common.HexToHash("0x5f51a2679791c14504a0315e6dd07f842f3b15cc7cea98909f754edea4056eec"), topicDeskPush)
	assert.Equal(t, common.HexToHash("0x9375a6f96e6b8f213540246148c2554d750c4e06a2a0d1a93584ed6f11c66366"),
		frontABI.Events["PoolFill"].ID)
	assert.Equal(t, common.HexToHash("0x1eeaa4acf3c225a4033105c2647625dbb298dec93b14e16253c4231e26c02b1d"), topicSwapped)
	assert.Equal(t, crypto.Keccak256Hash([]byte("Posted(uint256)")), topicPosted, "a stored price lane's post")
}
