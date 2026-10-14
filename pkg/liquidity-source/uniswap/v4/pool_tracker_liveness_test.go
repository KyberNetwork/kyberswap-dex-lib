package uniswapv4

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const testInterval = 20 * time.Second

// liveHook has liveness only; liveActiveHook also opts in to the activity bump.
type liveHook struct{ plainHook }

func (*liveHook) LivenessInterval() time.Duration { return testInterval }

type liveActiveHook struct{ liveHook }

func (*liveActiveHook) TracksActivity() {}

func (*liveActiveHook) AllowEmptyTicks() bool { return true } // lets the simulator build with no ticks

var (
	liveHookAddr       = common.HexToAddress("0x00000000000000000000000000000000000d0031")
	liveActiveHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0032")
	capturedTrackedAt  int64

	_ = RegisterHooks(&liveHook{plainHook{&BaseHook{}}}, liveHookAddr)
	_ = RegisterHooksFactory(func(p *HookParam) Hook {
		capturedTrackedAt = p.TrackedAt
		return &liveActiveHook{liveHook{plainHook{&BaseHook{}}}}
	}, liveActiveHookAddr)
)

// extraSansLv decodes Extra and drops lv, to show everything else is identical.
func extraSansLv(t *testing.T, extra string) (map[string]any, int64) {
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(extra), &m))
	lv, _ := m["lv"].(float64)
	delete(m, "lv")
	return m, int64(lv)
}

// A quiet liveness pool keeps byte-identical Extra between stamps, so pool-service skips the save;
// once per interval only lv changes, and Timestamp/BlockNumber (the archiving clock) never move for it.
func TestLiveness_QuietPoolStampsOncePerInterval(t *testing.T) {
	tr := newActivityTracker(t)
	p0 := tr.refresh(t, schedulePool(t, liveActiveHookAddr, 0), 1_000, `{"a":1}`, nil, nil, nil)
	_, lv0 := extraSansLv(t, p0.Extra)
	require.Equal(t, int64(1_000), lv0, "first track stamps lv")
	require.Equal(t, int64(1_000), p0.Timestamp)

	// 19 s later: nothing changed, lv not due.
	p1 := tr.refresh(t, p0, 1_019, `{"a":1}`, nil, nil, nil)
	assert.Equal(t, p0.Extra, p1.Extra, "quiet pool inside the interval: identical Extra bytes")
	assert.Equal(t, p0.Timestamp, p1.Timestamp)
	assert.Equal(t, p0.BlockNumber, p1.BlockNumber)

	// 20 s later: only lv moves.
	p2 := tr.refresh(t, p1, 1_020, `{"a":1}`, nil, nil, nil)
	m1, _ := extraSansLv(t, p1.Extra)
	m2, lv2 := extraSansLv(t, p2.Extra)
	assert.Equal(t, int64(1_020), lv2)
	assert.Equal(t, m1, m2, "only lv differs")
	assert.Equal(t, p1.Timestamp, p2.Timestamp, "an lv-only restamp is not activity")
	assert.Equal(t, p1.BlockNumber, p2.BlockNumber)
}

// A state change restamps lv at once and, for an opted-in hook, bumps Timestamp to the same time.
func TestLiveness_ChangeRestampsAndBumps(t *testing.T) {
	tr := newActivityTracker(t)
	p0 := tr.refresh(t, schedulePool(t, liveActiveHookAddr, 0), 1_000, `{"a":1}`, nil, nil, nil)

	p1 := tr.refresh(t, p0, 1_005, `{"a":2}`, nil, nil, nil)
	_, lv := extraSansLv(t, p1.Extra)
	assert.Equal(t, int64(1_005), lv)
	assert.Equal(t, int64(1_005), p1.Timestamp)
	assert.Equal(t, uint64(stubBlock), p1.BlockNumber)
}

// Liveness without the activity opt-in stamps lv but leaves Timestamp alone.
func TestLiveness_WithoutActivityKeepsTimestamp(t *testing.T) {
	tr := newActivityTracker(t)
	p0 := tr.refresh(t, schedulePool(t, liveHookAddr, 7), 1_000, `{"a":1}`, nil, nil, nil)
	p1 := tr.refresh(t, p0, 1_005, `{"a":2}`, nil, nil, nil)
	_, lv := extraSansLv(t, p1.Extra)
	assert.Equal(t, int64(1_005), lv)
	assert.Equal(t, int64(7), p1.Timestamp)
}

// Hooks without liveness never emit lv.
func TestLiveness_PlainHooksNeverEmitLv(t *testing.T) {
	for _, addr := range []common.Address{plainHookAddr, optedHookAddr} {
		tr := newActivityTracker(t)
		p := tr.refresh(t, schedulePool(t, addr, 0), 1_000, `{"a":1}`, nil, nil, nil)
		p = tr.refresh(t, p, 5_000, `{"a":2}`, nil, nil, nil)
		assert.NotContains(t, p.Extra, `"lv"`)
	}
}

// The tracker hands the hook the stored lv and the simulator hands it Extra lv.
func TestLiveness_HookParamTrackedAt(t *testing.T) {
	tr := newActivityTracker(t)
	p0 := tr.refresh(t, schedulePool(t, liveActiveHookAddr, 0), 1_000, `{"a":1}`, nil, nil, nil)

	capturedTrackedAt = -1
	tr.refresh(t, p0, 1_005, `{"a":1}`, nil, nil, nil)
	assert.Equal(t, int64(1_000), capturedTrackedAt, "tracker: old lv")

	simPool := entity.Pool{
		Address: "0xpool", SwapFee: 3000, Reserves: entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: "0x1", Decimals: 18}, {Address: "0x2", Decimals: 18}},
		StaticExtra: `{"hooks":"` + liveActiveHookAddr.Hex() + `","fee":3000,"tS":60}`,
		Extra:       `{"liquidity":1,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,"ticks":[],"hX":{"a":1},"lv":1000}`,
	}
	capturedTrackedAt = -1
	_, err := NewPoolSimulator(simPool, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1_000), capturedTrackedAt, "simulator: Extra lv")
}

// T7: an Extra written before lv existed decodes in both structs with lv 0; one with lv keeps it.
func TestExtraLvCompat(t *testing.T) {
	const legacy = `{"liquidity":1,"sqrtPriceX96":1,"tickSpacing":1,"tick":0,"ticks":[],"hX":{"a":1}}`
	withLv := legacy[:len(legacy)-1] + `,"lv":1234}`
	for extra, want := range map[string]int64{legacy: 0, withLv: 1234} {
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(extra), &e))
		var u ExtraU256
		require.NoError(t, json.Unmarshal([]byte(extra), &u))
		assert.Equal(t, want, e.TrackedAt)
		assert.Equal(t, want, u.TrackedAt)
		assert.JSONEq(t, `{"a":1}`, string(e.HookExtra))
		assert.JSONEq(t, `{"a":1}`, string(u.HookExtra))
	}
	out, err := json.Marshal(Extra{})
	require.NoError(t, err)
	assert.NotContains(t, string(out), "lv", "lv is omitted when 0")
}
