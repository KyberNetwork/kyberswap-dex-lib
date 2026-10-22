package cl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// fromFactory is a pool as the pool factory builds it from an Initialize event: a price and a
// tick, no liquidity and no ticks.
var fromFactory = entity.Pool{
	Address: "0xd0949ad5874cee7991ccd4cc2c5f7f80a7be4a220689cf5947ccb12a946fa50d",
	Extra:   `{"liquidity":null,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":10,"tick":0,"ticks":null}`,
}

// TestFactoryPoolHasNoState: the factory's Extra must not read as state, or a pool created from
// its Initialize event, or first seen in a Swap, would never be read whole.
func TestFactoryPoolHasNoState(t *testing.T) {
	assert.False(t, tickspkg.HasState(fromFactory))
}

// TestStatelessPoolsAreScheduled: a pool whose lens read the guard declined comes back as soon as
// the guard would take it, at once if it was not tried, after the cooldown if it was; a pool with
// state, or a tracker without the lens, schedules nothing.
func TestStatelessPoolsAreScheduled(t *testing.T) {
	tracked := entity.Pool{Extra: `{"liquidity":5,"sqrtPriceX96":79228162514264337593543950336}`}

	tracker := NewPoolTracker(&Config{UseLens: true, LensGuard: ephemeral.GuardConfig{CooldownSecs: 30}}, nil, nil)
	before := time.Now().Unix()
	at := tracker.NextUpdateAt(fromFactory)
	assert.GreaterOrEqual(t, at, before)
	assert.LessOrEqual(t, at, time.Now().Unix(), "a pool not tried yet is due at once")

	_, _ = tracker.lensGuard.Try(t.Context(), fromFactory.Address,
		func(context.Context) error { return errors.New("failed") })
	at = tracker.NextUpdateAt(fromFactory)
	assert.GreaterOrEqual(t, at, before+30)
	assert.LessOrEqual(t, at, time.Now().Unix()+30, "a pool just tried waits out its cooldown")
	assert.Zero(t, tracker.NextUpdateAt(tracked))

	assert.Zero(t, NewPoolTracker(&Config{}, nil, nil).NextUpdateAt(fromFactory))
}

// TestResetPoolState: with the lens a reset drops what the logs built and keeps the hook's state,
// due a snapshot at the block asked; without it nothing could read the pool whole, so it is kept.
func TestResetPoolState(t *testing.T) {
	p := entity.Pool{Extra: `{"liquidity":5,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":10,"tick":3,` +
		`"ticks":[{"index":-10,"liquidityGross":5,"liquidityNet":5},{"index":10,"liquidityGross":5,"liquidityNet":-5}],` +
		`"lensBlock":90,"hX":{"k":1}}`}

	kept, ok := NewPoolTracker(&Config{}, nil, nil).ResetPoolState(p, 100)
	assert.False(t, ok)
	assert.Equal(t, p, kept)

	reset, ok := NewPoolTracker(&Config{UseLens: true}, nil, nil).ResetPoolState(p, 100)
	require.True(t, ok)
	assert.False(t, tickspkg.HasState(reset))
	assert.EqualValues(t, 100, tickspkg.LensDue(reset))
	var extra map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(reset.Extra), &extra))
	assert.NotContains(t, extra, "ticks")
	assert.NotContains(t, extra, "lensBlock")
	assert.JSONEq(t, `{"k":1}`, string(extra["hX"]), "the hook's state is not the logs' to drop")
}
