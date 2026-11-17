package uniswapv4

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// newStateBlockTracker is a tracker whose RPC answers at the block it reads from block.
func newStateBlockTracker(t *testing.T, block *atomic.Uint64) *PoolTracker {
	var calls atomic.Int32
	stub := aggregateStubAt(t, &calls, block)
	t.Cleanup(stub.Close)
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ticks":[]}}`))
	}))
	t.Cleanup(gql.Close)

	orig := NowFn
	t.Cleanup(func() { NowFn = orig; testHX = `{"a":1}`; testReserves = nil })
	return &PoolTracker{config: &Config{}, ethrpcClient: ethrpc.New(stub.URL), graphqlClient: graphqlpkg.NewClient(gql.URL)}
}

var stateBlockKey = regexp.MustCompile(`,"sB":\d+`)

// stripStateBlock is the Extra without its sB key, the other bytes untouched.
func stripStateBlock(extra string) string {
	return stateBlockKey.ReplaceAllString(extra, "")
}

func stateBlockOf(t *testing.T, extra string) uint64 {
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(extra), &e))
	return e.StateBlock
}

// A refresh and a bootstrap store the block of the read, for a plain hook and an opted-in one alike.
func TestStateBlock_SetFromReadBlock(t *testing.T) {
	for name, hook := range map[string]common.Address{"plain hook": plainHookAddr, "opted hook": optedHookAddr} {
		t.Run(name, func(t *testing.T) {
			block := new(atomic.Uint64)
			block.Store(777)
			tr := newStateBlockTracker(t, block)

			got := tr.refresh(t, schedulePool(t, hook, 0), 1, `{"a":1}`, nil, nil, nil)
			assert.Equal(t, uint64(777), stateBlockOf(t, got.Extra))

			boot := entity.Pool{Address: "0xpool", StaticExtra: `{"hooks":"` + hook.Hex() + `"}`}
			NowFn = func() int64 { return 5_000 }
			got, err := tr.BootstrapPoolState(context.Background(), boot, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			assert.Equal(t, uint64(777), stateBlockOf(t, got.Extra))
		})
	}
}

// A refresh stores the block of its read even when the state did not change: sB tells when the state
// was last read. Only the sB key differs from the Extra before.
func TestStateBlock_FollowsEveryRead(t *testing.T) {
	block := new(atomic.Uint64)
	block.Store(100)
	tr := newStateBlockTracker(t, block)

	p1 := tr.refresh(t, schedulePool(t, plainHookAddr, 0), 1, `{"a":1}`, nil, nil, nil)
	require.Equal(t, uint64(100), stateBlockOf(t, p1.Extra))

	block.Store(105)
	p2 := tr.refresh(t, p1, 2, `{"a":1}`, nil, nil, nil)
	assert.Equal(t, uint64(105), stateBlockOf(t, p2.Extra))
	assert.Equal(t, stripStateBlock(p1.Extra), stripStateBlock(p2.Extra), "an unchanged state: only sB differs")

	// the hook state changes: the state is read again at 110.
	block.Store(110)
	p3 := tr.refresh(t, p2, 3, `{"a":2}`, nil, nil, nil)
	assert.NotEqual(t, stripStateBlock(p2.Extra), stripStateBlock(p3.Extra))
	assert.Equal(t, uint64(110), stateBlockOf(t, p3.Extra))

	// a swap-fee-only change does not alter the Extra bytes but is a state change too.
	block.Store(115)
	p3.SwapFee++
	p4 := tr.refresh(t, p3, 4, `{"a":2}`, nil, nil, nil)
	assert.Equal(t, uint64(115), stateBlockOf(t, p4.Extra))
}

// buildExtra reports "changed" from the Extra built with the previous StateBlock, so the read
// block alone is never a change (markActivity and the liveness restamp stay as before sB), while the
// Extra it returns holds the block of this read.
func TestBuildExtra_ReadBlockAloneIsNotAChange(t *testing.T) {
	res := func(block uint64) *FetchRPCResult {
		return &FetchRPCResult{
			Liquidity:            big.NewInt(1e18),
			TickSpacing:          60,
			Slot0:                Slot0Data{SqrtPriceX96: big.NewInt(1e9), Tick: big.NewInt(0)},
			TicksBasedStateBlock: tickspkg.TicksBasedStateBlock{StateBlock: block},
		}
	}
	first, changed, err := buildExtra(&BaseHook{}, res(100), nil, poolState{}, 1)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, uint64(100), stateBlockOf(t, string(first)))

	prev := snapshotState(&entity.Pool{Extra: string(first)})
	require.Equal(t, uint64(100), tickspkg.TicksBasedStateBlockOf(prev.extra))

	again, changed, err := buildExtra(&BaseHook{}, res(250), nil, prev, 2)
	require.NoError(t, err)
	assert.False(t, changed, "a newer read block with the same state is no change")
	assert.Equal(t, uint64(250), stateBlockOf(t, string(again)))
	assert.Equal(t, stripStateBlock(string(first)), stripStateBlock(string(again)))

	other := res(250)
	other.HookExtra = json.RawMessage(`{"a":2}`)
	moved, changed, err := buildExtra(&BaseHook{}, other, nil, prev, 3)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, uint64(250), stateBlockOf(t, string(moved)))
}

// A swap fee or a reserves change alone is a state change.
func TestBuildExtra_FeeAndReservesChanges(t *testing.T) {
	res := func(swapFee uint32, reserves entity.PoolReserves) *FetchRPCResult {
		return &FetchRPCResult{
			Liquidity:            big.NewInt(1e18),
			TickSpacing:          60,
			Slot0:                Slot0Data{SqrtPriceX96: big.NewInt(1e9), Tick: big.NewInt(0)},
			SwapFee:              swapFee,
			Reserves:             reserves,
			TicksBasedStateBlock: tickspkg.TicksBasedStateBlock{StateBlock: 130},
		}
	}
	reserves := entity.PoolReserves{"1", "2"}
	stored, changed, err := buildExtra(&BaseHook{}, res(3000, reserves), nil, poolState{}, 1)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, uint64(130), stateBlockOf(t, string(stored)))
	prev := poolState{extra: string(stored), swapFee: 3000, reserves: reserves, stateBlock: 130}

	_, changed, err = buildExtra(&BaseHook{}, res(3000, reserves), nil, prev, 2)
	require.NoError(t, err)
	assert.False(t, changed, "the same fee and reserves")
	_, changed, err = buildExtra(&BaseHook{}, res(500, reserves), nil, prev, 2)
	require.NoError(t, err)
	assert.True(t, changed, "a swap fee change alone")
	_, changed, err = buildExtra(&BaseHook{}, res(3000, entity.PoolReserves{"1", "3"}), nil, prev, 2)
	require.NoError(t, err)
	assert.True(t, changed, "a reserves change alone")
}

// A liveness restamp (lv) of an unchanged state stores the block of its read too.
func TestStateBlock_LivenessRestampStoresReadBlock(t *testing.T) {
	block := new(atomic.Uint64)
	block.Store(100)
	tr := newStateBlockTracker(t, block)

	p0 := tr.refresh(t, schedulePool(t, liveActiveHookAddr, 0), 1_000, `{"a":1}`, nil, nil, nil)
	require.Equal(t, uint64(100), stateBlockOf(t, p0.Extra))

	block.Store(120)
	p1 := tr.refresh(t, p0, 1_000+int64(testInterval.Seconds()), `{"a":1}`, nil, nil, nil)
	m0, lv0 := extraSansLv(t, stripStateBlock(p0.Extra))
	m1, lv1 := extraSansLv(t, stripStateBlock(p1.Extra))
	require.Greater(t, lv1, lv0, "lv was restamped")
	assert.Equal(t, m0, m1, "only lv and sB differ")
	assert.Equal(t, uint64(120), stateBlockOf(t, p1.Extra))
}

// An Extra from before the field decodes with StateBlock 0, and the next refresh of its unchanged
// state only adds the sB key.
func TestStateBlock_OldExtraDecodesAndGetsStateBlock(t *testing.T) {
	block := new(atomic.Uint64)
	block.Store(100)
	tr := newStateBlockTracker(t, block)

	// settled state without sB: what the tracker wrote before the field existed.
	settled := tr.refresh(t, schedulePool(t, plainHookAddr, 0), 1, `{"a":1}`, nil, nil, nil)
	require.Contains(t, settled.Extra, `,"sB":100`)
	old := stripStateBlock(settled.Extra)
	settled.Extra = old

	var e Extra
	var e256 ExtraU256
	require.NoError(t, json.Unmarshal([]byte(old), &e))
	require.NoError(t, json.Unmarshal([]byte(old), &e256))
	assert.Zero(t, e.StateBlock)
	assert.Zero(t, e256.StateBlock)

	block.Store(130)
	same := tr.refresh(t, settled, 2, `{"a":1}`, nil, nil, nil)
	assert.Equal(t, uint64(130), stateBlockOf(t, same.Extra))
	assert.Equal(t, old, stripStateBlock(same.Extra), "only the sB key is new")
}

// GetMetaInfo exposes sB = max(StateBlock, BlockNumber) next to the untouched
// blockNumber, and omits it when the pool has no StateBlock.
func TestPoolMetaInfo_StateBlock(t *testing.T) {
	var stored entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &stored))
	cursor := stored.BlockNumber
	require.NotZero(t, cursor)

	meta := func(stateBlock uint64) (PoolMetaInfo, string) {
		p := stored
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &m))
		if stateBlock > 0 {
			m["sB"] = json.RawMessage(strconv.FormatUint(stateBlock, 10))
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		p.Extra = string(b)
		sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.NoError(t, err)
		info := sim.GetMetaInfo(p.Tokens[0].Address, p.Tokens[1].Address)
		out, err := json.Marshal(info)
		require.NoError(t, err)
		return info.(PoolMetaInfo), string(out)
	}

	m, out := meta(cursor + 7)
	assert.Equal(t, cursor, m.BlockNumber, "blockNumber is still the tick cursor")
	assert.Equal(t, cursor+7, m.StateBlock)
	assert.Contains(t, out, `"sB":`)

	m, _ = meta(cursor - 7)
	assert.Equal(t, cursor, m.BlockNumber)
	assert.Equal(t, cursor, m.StateBlock, "never below the tick cursor")

	m, out = meta(0)
	assert.Equal(t, cursor, m.BlockNumber)
	assert.Zero(t, m.StateBlock)
	assert.NotContains(t, out, "sB", "omitted when unknown")
}

// A state read under eth_call overrides (a pending tx's prestate) exists at no block: it stores no
// StateBlock, not even the one the pool had.
func TestStateBlock_NotStoredUnderOverrides(t *testing.T) {
	block := new(atomic.Uint64)
	block.Store(100)
	tr := newStateBlockTracker(t, block)
	NowFn = func() int64 { return 1 }
	testHX = `{"a":1}`

	p := schedulePool(t, plainHookAddr, 0)
	got, err := tr.GetNewPoolStateWithOverrides(context.Background(), p, poolpkg.GetNewPoolStateWithOverridesParams{
		Overrides: map[common.Address]gethclient.OverrideAccount{{1}: {}},
	})
	require.NoError(t, err)
	assert.NotEqual(t, p.Extra, got.Extra, "the state did change")
	assert.Zero(t, stateBlockOf(t, got.Extra))

	// without overrides the same refresh stores it.
	got, err = tr.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(100), stateBlockOf(t, got.Extra))

	// a pool that has a StateBlock loses it under overrides.
	got, err = tr.GetNewPoolStateWithOverrides(context.Background(), got, poolpkg.GetNewPoolStateWithOverridesParams{
		Overrides: map[common.Address]gethclient.OverrideAccount{{1}: {}},
	})
	require.NoError(t, err)
	assert.Zero(t, stateBlockOf(t, got.Extra))
}

// oldPlainExtra and oldLiveExtra are the Extra the tracker of the release before the StateBlock wrote
// for schedulePool and the chain of aggregateStub, taken from that release's code, for a plain hook and
// for a hook with liveness (lv). The pool had swap fee 0 and reserves 0, 0.
const (
	oldPlainExtra = `{"liquidity":79228162514264337593543950336,"sqrtPriceX96":1000000000000000000,"tickSpacing":0,"tick":0,` +
		`"ticks":[],"hX":{"a":1}}`
	oldLiveExtra = `{"liquidity":79228162514264337593543950336,"sqrtPriceX96":1000000000000000000,"tickSpacing":0,"tick":0,` +
		`"ticks":[],"hX":{"a":1},"lv":1000}`
)

// The Extra of an unchanged pool is the one the release before wrote, byte for byte, except the sB
// key, and so are its swap fee and reserves.
func TestStateBlock_UnchangedStateIdenticalToOldExtraExceptStateBlock(t *testing.T) {
	for name, tc := range map[string]struct {
		hook  common.Address
		now   int64
		extra string
	}{
		"plain hook":    {plainHookAddr, 1, oldPlainExtra},
		"liveness hook": {liveActiveHookAddr, 1_000, oldLiveExtra},
	} {
		t.Run(name, func(t *testing.T) {
			block := new(atomic.Uint64)
			block.Store(100)
			tr := newStateBlockTracker(t, block)

			// the state of an old pool, on a chain that has not changed
			fresh := tr.refresh(t, schedulePool(t, tc.hook, 0), tc.now, `{"a":1}`, nil, nil, nil)
			assert.Equal(t, tc.extra, stripStateBlock(fresh.Extra), "the old Extra and the sB key, no other change")

			old := fresh
			old.Extra = tc.extra
			got := tr.refresh(t, old, tc.now, `{"a":1}`, nil, nil, nil)
			assert.Equal(t, old.Extra, stripStateBlock(got.Extra))
			assert.Equal(t, uint64(100), stateBlockOf(t, got.Extra))
			assert.Equal(t, old.SwapFee, got.SwapFee)
			assert.Equal(t, old.Reserves, got.Reserves)
		})
	}
}
