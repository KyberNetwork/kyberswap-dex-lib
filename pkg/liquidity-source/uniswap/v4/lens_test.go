package uniswapv4

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	lens := NewPoolTracker(&Config{
		LensConfig: ephemeral.LensConfig{UseLens: true,
			LensGuard: ephemeral.GuardConfig{CooldownSecs: 30}}}, nil, nil)
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

// TestWithStateTakesTheHooksExchange: the hook names the pool's exchange, and the pool withState
// returns, bootstrapped or read through the lens, is under it.
func TestWithStateTakesTheHooksExchange(t *testing.T) {
	sqrtPriceOne, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
	tracker := &PoolTracker{config: &Config{}}
	p := entity.Pool{Address: "0xpool", Exchange: DexType}
	result := &FetchRPCResult{Liquidity: big.NewInt(0), TickSpacing: 60,
		Slot0: Slot0Data{SqrtPriceX96: sqrtPriceOne, Tick: big.NewInt(0)}}

	got, err := tracker.withState(t.Context(), p, result, &BaseHook{Exchange: "uniswap-v4-hooked"},
		&HookParam{Cfg: tracker.config, Pool: &p}, nil, poolpkg.GetNewPoolStateParams{}, snapshotState(&p), 0, 0,
		logger.WithFields(nil))
	require.NoError(t, err)
	assert.Equal(t, "uniswap-v4-hooked", got.Exchange)
}
