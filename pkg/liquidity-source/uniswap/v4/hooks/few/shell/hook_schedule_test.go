package fewshell

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const trackedAt = int64(1_800_000_000)

// withLv returns p with Extra "lv" set, as the tracker stamps it.
func withLv(t testing.TB, p entity.Pool, lv int64) entity.Pool {
	t.Helper()
	var extra map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	raw, err := json.Marshal(lv)
	require.NoError(t, err)
	extra["lv"] = raw
	out, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(out)
	return p
}

// The pool is tracked again one interval after the last read, with no event.
func TestNextTrackAt_OneIntervalAfterTheLastRead(t *testing.T) {
	assert.Equal(t, trackedAt+30, newHook().NextTrackAt(&uniswapv4.HookParam{TrackedAt: trackedAt}))
	assert.Equal(t, 30*time.Second, newHook().LivenessInterval(),
		"the tracker stamps lv at the interval NextTrackAt adds")
}

// A pool with no lv yet is due at once, and the time is not 0 (0 means no schedule: pool-service would drop the pool).
func TestNextTrackAt_NoStampIsDueAndNotZero(t *testing.T) {
	got := newHook().NextTrackAt(&uniswapv4.HookParam{})
	assert.NotZero(t, got)
	assert.Less(t, got, time.Now().Unix())
}

// The tracker reports the hook's schedule for a stored pool: lv read from Extra, hX untouched, no RPC.
func TestNextUpdateAt_FromTheStoredLv(t *testing.T) {
	tracker := uniswapv4.NewPoolTracker(&uniswapv4.Config{ChainID: valueobject.ChainIDEthereum}, nil, nil)
	scheduled := poolpkg.IPoolTrackerWithSchedule(tracker)

	p := newPool(t, liveExtra(), livePF)
	assert.Equal(t, int64(30), scheduled.NextUpdateAt(p), "no lv yet: due at once, never 0")

	p = withLv(t, p, trackedAt)
	assert.Equal(t, trackedAt+30, scheduled.NextUpdateAt(p))

	// hX does not carry the clock: the stamp is outside it.
	var extra struct {
		HX json.RawMessage `json:"hX"`
	}
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.NotContains(t, string(extra.HX), "lv")
}
