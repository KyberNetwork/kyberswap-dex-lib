package efficientprop

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	testFront   = common.HexToAddress("0x0000000000403aeb66691063cc2fd42e71d6acd6")
	testFactory = common.HexToAddress("0x39ed9993449c96503f6a629ccd2bb5f132d7cc93")
	// testFeeFactory is the Factory of the test pools: it holds their fee rates (feeFactoryABI)
	testFeeFactory = common.HexToAddress("0x5e11e5000000000000000000000000000000fee1")
	testWETH       = common.HexToAddress("0x4200000000000000000000000000000000000006")
	testUSDC       = common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	testPoolA      = common.HexToAddress("0x7000000000000000000000000000000000000a11")
	testPoolB      = common.HexToAddress("0x1111111111111111111111111111111111111111")
	// testSource is a price source the test config approves, registered as source 1, as source 65,538 and, guarded,
	// as source 4; testOther one it does not, registered guarded as source 2; testThird one it does not, registered as
	// source 3 and as source 65,537, whose low 16 bits name source 1
	testSource = common.HexToAddress("0x5000000000000000000000000000000000000001")
	testOther  = common.HexToAddress("0x5000000000000000000000000000000000000002")
	testThird  = common.HexToAddress("0x5000000000000000000000000000000000000003")
	// testImpl is the front implementation the test config lists, and the fake front's
	testImpl = common.HexToAddress("0x6000000000000000000000000000000000000001")
	// testBaseTime is the chain's millisecond-time contract the test config names, and the fake chain's: Base's
	// predeploy
	testBaseTime = common.HexToAddress("0x4200000000000000000000000000000000000030")
)

// testEntry is a storage build's source entry for a stored price lane at src: its base and quote slots 2 and 3, a
// 4 s window, the lane flag (bit 254) and, where guarded, the guard (bit 247).
func testEntry(src common.Address, guarded bool) *big.Int {
	e := new(big.Int).SetBytes(src[:])
	e.Or(e, new(big.Int).Lsh(big.NewInt(2|3<<16), 160))
	e.Or(e, new(big.Int).Lsh(big.NewInt(8), 240))
	e.SetBit(e, 254, 1)
	if guarded {
		e.SetBit(e, 247, 1)
	}
	return e
}

// testView is a live WORD listing of WETH/USDC (base WETH, token 0) on its pool's hook pair (base 0, quote 1), whose
// desk posted its word now, for 30 s, and whose depth stands for a day.
func testView(pool common.Address) DoorView {
	v := zeroView()
	v.Scopes = 2 // the fence kind WORD (scopes bits 1-2)
	v.Token0, v.Token1, v.Pool = testWETH, testUSDC, pool
	v.BaseIndex, v.QuoteIndex = 0, 1
	v.PairWord = big.NewInt(1 << 8)
	v.Flags = 1
	v.Desk = testDesk
	v.TipCap = 0x8001 // 0.1 gwei
	v.Word = testWord(time.Now().UnixMilli(), 30)
	expires := new(big.Int).Lsh(big.NewInt(time.Now().Add(24*time.Hour).Unix()), 60)
	v.Depth = [4]*big.Int{new(big.Int).Set(expires), new(big.Int), new(big.Int).Set(expires), new(big.Int)}
	return v
}

// testWord is a signed standing word at 3,000 USDC per WETH, its maker time makerMs and its life lifeS.
func testWord(makerMs int64, lifeS uint64) *big.Int {
	w := new(big.Int).Lsh(big.NewInt(int64(lifeS)), 120)
	w.Or(w, new(big.Int).Lsh(big.NewInt(makerMs), 72))
	w.Or(w, new(big.Int).Lsh(big.NewInt(3000e12), 8))
	return w.Or(w, big.NewInt(1))
}

func testChain() *fakeChain {
	return &fakeChain{
		front:   testFront,
		factory: testFactory,
		listings: map[uint32]*fakeListing{
			// bound to source 1, which a WORD listing never reads
			1: {view: testView(testPoolA), open: true, sourceIds: big.NewInt(1)},
			2: {view: testView(testPoolA)}, // reserved, its door never opened
			3: {view: testView(testPoolB), open: true, wrongFront: testPoolB},
			4: {view: testView(testPoolB), open: true},
			// 5 was never reserved: the scan ends there, and 6 is never read
			6: {view: testView(testPoolB), open: true},
		},
		balances: map[[2]common.Address]*big.Int{},
		sources: map[uint32]*big.Int{1: testEntry(testSource, false), 2: testEntry(testOther, true),
			3: testEntry(testThird, false), 4: testEntry(testSource, true), 65_537: testEntry(testThird, false),
			65_538: testEntry(testSource, false)},
		// Base's clock, mode 2, its head 400 ms into its second
		clockMode: 2,
		subMs:     400,
	}
}

// testDesk is the desk the test listings are bound to, unless a test binds another, and testDesks the desk contracts
// the test configs approve: the test front's desks 1, 7, 8 and 9.
const testDesk = 1

func testDesks() []string {
	var out []string
	for _, d := range []uint32{testDesk, 7, 8, 9} {
		c := cellOf(d)
		out = append(out, hexutil.Encode(c[:]))
	}
	return out
}

func testConfig() *Config {
	return &Config{ChainID: valueobject.ChainIDBase, ApprovedDesks: testDesks(), Fronts: []Front{{
		Address:     hexutil.Encode(testFront[:]),
		DoorFactory: hexutil.Encode(testFactory[:]),
		FillMode:    FillModeFront,
		// the source the print-reading tests bind
		ApprovedSources: []string{hexutil.Encode(testSource[:])},
		Implementations: []string{hexutil.Encode(testImpl[:])},
		BaseTime:        hexutil.Encode(testBaseTime[:]),
	}}}
}

// One pool per opened door: the door is the pool, its tokens the listing's pair, and its static data the front,
// the listing id and the pool holding its custody. A door never opened, or one whose immutables name another front,
// is not listed, and the scan stops at the first listing the front never reserved.
func TestListerListsOpenedDoors(t *testing.T) {
	chain := testChain()
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)

	pools, meta, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, meta)
	require.Len(t, pools, 2)

	for k, id := range []uint32{1, 4} {
		p := pools[k]
		door := fakeDoorOf(testFactory, id)
		assert.Equal(t, hexutil.Encode(door[:]), p.Address)
		assert.Equal(t, DexType, p.Exchange)
		assert.Equal(t, DexType, p.Type)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves, "discovery never sets reserves")
		require.Len(t, p.Tokens, 2)
		assert.Equal(t, hexutil.Encode(testWETH[:]), p.Tokens[0].Address)
		assert.Equal(t, hexutil.Encode(testUSDC[:]), p.Tokens[1].Address)
		assert.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)

		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		assert.Equal(t, hexutil.Encode(testFront[:]), se.Front)
		assert.Equal(t, id, se.Listing)
		assert.Equal(t, p.Address, se.Door, "the door, as the adapter's data names it")
		assert.Equal(t, modeFront, se.Mode, "the front's configured fill mode")
		want := chain.listings[id].view.Pool
		assert.Equal(t, hexutil.Encode(want[:]), se.Custody)
	}

	// every read after the first (the factory's front and door codehash, the front's desk codehash) is pinned to the
	// block the first answered at
	pinned := hexutil.EncodeUint64(900)
	calls := n.calls()
	require.Greater(t, len(calls), 3)
	for _, c := range calls[:3] {
		assert.Equal(t, "latest", c.tag)
	}
	for _, c := range calls[3:] {
		assert.Equal(t, pinned, c.tag, "a read at another block: %x", c.data)
	}
	// listing 6 lies past the first listing never reserved
	for _, c := range calls {
		if c.to == testFront && len(c.data) == 36 {
			assert.LessOrEqual(t, new(big.Int).SetBytes(c.data[4:]).Int64(), int64(scanBatch))
		}
	}
}

// A front configured to fill through its doors lists its pools with the door's mode.
func TestListerCarriesTheFrontsFillMode(t *testing.T) {
	for mode, want := range map[string]uint8{FillModeDoor: modeDoor, FillModeFront: modeFront} {
		cfg := testConfig()
		cfg.Fronts[0].FillMode = mode
		u, err := NewPoolsListUpdater(cfg, testChain().node(t, 900).client())
		require.NoError(t, err)
		pools, _, err := u.GetNewPools(context.Background(), nil)
		require.NoError(t, err)
		require.NotEmpty(t, pools)
		for _, p := range pools {
			var se StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
			assert.Equal(t, want, se.Mode, mode)
			assert.Equal(t, p.Address, se.Door, mode)
		}
	}
}

// A factory that serves another front lists nothing: its doors forward to that front.
func TestListerRefusesAFactoryOfAnotherFront(t *testing.T) {
	chain := testChain()
	chain.factoryFront = testPoolB
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	assert.ErrorIs(t, err, ErrFrontMismatch)
	assert.Empty(t, pools)
}

// A factory that answers no door code hash lists nothing. The config names none, and a factory of another door build
// is listed by its own doorOf (TestDoorsComeFromTheFactorysOwnCodeHash).
func TestListerRefusesAFactoryThatAnswersNoDoorCodeHash(t *testing.T) {
	chain := testChain()
	chain.noDoorHash = true
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	assert.ErrorIs(t, err, ErrDoorCodehash)
	assert.Empty(t, pools)
}

// A factory not yet deployed answers no front, which is the same refusal.
func TestListerRefusesAnUndeployedFactory(t *testing.T) {
	chain := testChain()
	chain.factory = common.HexToAddress("0x2222222222222222222222222222222222222222")
	n := chain.node(t, 900)
	cfg := testConfig()
	cfg.Fronts[0].DoorFactory = "0x3333333333333333333333333333333333333333"
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)

	_, _, err = u.GetNewPools(context.Background(), nil)
	assert.ErrorIs(t, err, ErrFrontMismatch)
}

// The scan reads in batches and stops at the configured cap.
func TestListerScansInBatchesUpToTheCap(t *testing.T) {
	chain := testChain()
	chain.listings = map[uint32]*fakeListing{}
	for id := uint32(1); id <= 100; id++ {
		chain.listings[id] = &fakeListing{view: testView(testPoolA), open: id%10 == 0}
	}
	n := chain.node(t, 900)
	cfg := testConfig()
	cfg.MaxListings = 70
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, pools, 7, "doors 10, 20, ..., 70")
}

func retired(v DoorView) DoorView {
	v.Flags &^= flagLive
	return v
}

// A retired or delisted listing keeps its pool in doorView, so the scan goes past it to the listings after it, and
// lists none of them that is not live.
func TestListerSkipsARetiredListingAndScansOn(t *testing.T) {
	chain := testChain()
	chain.listings = map[uint32]*fakeListing{
		1: {view: testView(testPoolA), open: true},
		2: {view: retired(testView(testPoolA)), open: true},
		3: {view: testView(testPoolB), open: true},
	}
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	for k, id := range []uint32{1, 3} {
		door := fakeDoorOf(testFactory, id)
		assert.Equal(t, hexutil.Encode(door[:]), pools[k].Address)
	}
}

// A listing whose taker tip cap is under the configured floor is not listed: a route that tips more reverts there.
// A listing whose cap is 0 passes any floor.
func TestListerSkipsAListingUnderTheTipCapFloor(t *testing.T) {
	chain := testChain()
	chain.listings[4].view.TipCap = 0
	n := chain.node(t, 900)
	cfg := testConfig()
	cfg.MinTipCapWei = 2e8 // listing 1's cap is 1e8
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	door := fakeDoorOf(testFactory, 4)
	assert.Equal(t, hexutil.Encode(door[:]), pools[0].Address)

	cfg.MinTipCapWei = 1e8
	u, err = NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, pools, 2, "a cap at the floor passes")
}

// A door whose LISTING immutable names another listing is not listed, nor one whose pair is out of order (token0
// above token1), though its immutables match the view.
func TestListerChecksTheDoorsListingAndPairOrder(t *testing.T) {
	chain := testChain()
	chain.listings[1].wrongListing = 4
	v := chain.listings[4].view
	v.Token0, v.Token1 = testUSDC, testWETH
	chain.listings[4].view = v
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pools)
}

func TestNewPoolsListUpdaterValidates(t *testing.T) {
	_, err := NewPoolsListUpdater(&Config{}, nil)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

// On a builder chain a front that answers another CELL_CODEHASH() than the config names lists nothing, so nothing is
// listed that the tracker could not quote. On a chain read at the head the config names none: the front's own answer
// names its desk contracts, so the desks approved are the CREATE2 children under that hash and no other.
func TestListerRefusesAFrontWhoseCellCodehashIsNotTheConfigs(t *testing.T) {
	chain := ethChain()
	chain.cellHash = common.HexToHash("0xeb1c76ea")
	n := builderNode(t, chain, 900)
	u, err := NewPoolsListUpdater(ethConfig("http://127.0.0.1:1"), n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	assert.ErrorIs(t, err, ErrCellCodehash)
	assert.Empty(t, pools)

	base := testChain()
	base.cellHash = common.HexToHash("0xeb1c76ea")
	nb := base.node(t, 900)
	u, err = NewPoolsListUpdater(testConfig(), nb.client())
	require.NoError(t, err)
	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pools, "the approved desks are the children under another hash")
	cfg := testConfig()
	child := deskContractOf(testFront, base.cellHash, testDesk)
	cfg.ApprovedDesks = []string{hexutil.Encode(child[:])}
	u, err = NewPoolsListUpdater(cfg, nb.client())
	require.NoError(t, err)
	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, pools)
}

// The factory's doorOf and the CREATE2 door of the listing id under DOOR_CODEHASH, which the parser names a door by,
// are held together: a listing whose doorOf says another address is left out of the list, and the others are listed.
func TestListerLeavesOutAListingWhoseDoorIsNotTheCreate2Door(t *testing.T) {
	chain := testChain()
	chain.doorOfs = map[uint32]common.Address{4: common.HexToAddress("0x4444444444444444444444444444444444444444")}
	chain.listings[4].open = true
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, p := range pools {
		got[p.Address] = true
	}
	assert.True(t, got[door(1)], "listing 1's door is the CREATE2 door")
	assert.False(t, got[door(4)] || got["0x4444444444444444444444444444444444444444"], "listing 4's is not")
}

// The doors' immutables are read scanBatch doors a multicall, as the scan reads its ids: no multicall grows with the
// front's listings, and every door past the first batch is still listed.
func TestListerReadsTheDoorsInBatches(t *testing.T) {
	chain := manyPrintListings(2*scanBatch + 22)
	n := chain.node(t, 900)
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, pools, 2*scanBatch+22)
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, k := range n.multis {
		assert.LessOrEqual(t, k, 5*scanBatch) // four immutables and the pool's factory a door
	}
}
