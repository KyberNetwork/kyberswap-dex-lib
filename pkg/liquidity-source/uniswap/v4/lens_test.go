package uniswapv4

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// TestHasState: the pool factory fills in the price and tick from the Initialize event, which must
// not read as state, or a backfilled pool would never get its ticks.
func TestHasState(t *testing.T) {
	for extra, want := range map[string]bool{
		``: false,
		`{"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":10,"tick":0}`:         false,
		`{"liquidity":0,"sqrtPriceX96":79228162514264337593543950336,"tick":0,"ticks":[]}`: true,
		`{"liquidity":5,"sqrtPriceX96":79228162514264337593543950336,"tick":0,"ticks":[]}`: true,
	} {
		assert.Equal(t, want, tickspkg.HasState(entity.Pool{Extra: extra}), extra)
	}
}

// TestStatelessPoolsAreScheduled: a pool whose lens read the guard declined comes back as soon as
// the guard would take it, at once if it was not tried, after the cooldown if it was, since nothing
// else may update it for a long time; a pool with state, or a tracker without the lens, schedules
// nothing of its own.
func TestStatelessPoolsAreScheduled(t *testing.T) {
	const noHook = `{"hooks":"0x0000000000000000000000000000000000000000"}`
	fromFactory := entity.Pool{Extra: `{"sqrtPriceX96":79228162514264337593543950336,"tick":0}`, StaticExtra: noHook}
	tracked := entity.Pool{Extra: `{"liquidity":5,"sqrtPriceX96":79228162514264337593543950336}`, StaticExtra: noHook}

	lens := NewPoolTracker(&Config{UseLens: true, LensGuard: ephemeral.GuardConfig{CooldownSecs: 30}}, nil, nil)
	before := NowFn()
	at := lens.NextUpdateAt(fromFactory)
	assert.GreaterOrEqual(t, at, before)
	assert.LessOrEqual(t, at, NowFn(), "a pool not tried yet is due at once")

	_, _ = lens.lensGuard.Try(t.Context(), fromFactory.Address,
		func(context.Context) error { return errors.New("failed") })
	at = lens.NextUpdateAt(fromFactory)
	assert.GreaterOrEqual(t, at, before+30)
	assert.LessOrEqual(t, at, NowFn()+30, "a pool just tried waits out its cooldown")
	assert.Zero(t, lens.NextUpdateAt(tracked))

	assert.Zero(t, NewPoolTracker(&Config{}, nil, nil).NextUpdateAt(fromFactory))
}
