package v3

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// TestStatelessPoolsAreScheduled: a pool created from its factory log whose lens read the guard
// declined is due again as soon as the guard would take it; any other pool needs no scheduled
// update.
func TestStatelessPoolsAreScheduled(t *testing.T) {
	t.Parallel()

	tr := NewPoolTracker(&Config{
		LensConfig: ephemeral.LensConfig{UseLens: true,
			LensGuard: ephemeral.GuardConfig{CooldownSecs: 30}}}, nil, nil)
	before := time.Now().Unix()
	at := tr.NextUpdateAt(entity.Pool{Extra: `{}`})
	assert.GreaterOrEqual(t, at, before)
	assert.LessOrEqual(t, at, time.Now().Unix(), "a pool not tried yet is due at once")

	_, _ = tr.lensGuard.Try(t.Context(), "", func(context.Context) error { return nil })
	at = tr.NextUpdateAt(entity.Pool{Extra: `{}`})
	assert.GreaterOrEqual(t, at, before+30, "a pool just tried waits out its cooldown")
	assert.Zero(t, tr.NextUpdateAt(entity.Pool{Extra: `{"liquidity":0,"sqrtPriceX96":1}`}))

	assert.Zero(t, NewPoolTracker(&Config{}, nil, nil).NextUpdateAt(entity.Pool{Extra: `{}`}))
}

// TestResetPoolState: with the lens a reset pool is one without state, read whole on its next
// update; without it the pool is kept, as nothing could read it whole.
func TestResetPoolState(t *testing.T) {
	t.Parallel()

	p := entity.Pool{Extra: `{"unlocked":true,"liquidity":7,"sqrtPriceX96":1,"ticks":[]}`}
	lens := NewPoolTracker(&Config{LensConfig: ephemeral.LensConfig{UseLens: true}}, nil, nil)
	got, ok := lens.ResetPoolState(p, 100)
	assert.True(t, ok)
	assert.False(t, tickspkg.HasState(got))
	assert.Equal(t, uint64(100), tickspkg.LensDue(got))
	assert.NotZero(t, lens.NextUpdateAt(got), "it is due for its lens read")

	got, ok = NewPoolTracker(&Config{}, nil, nil).ResetPoolState(p, 100)
	assert.False(t, ok)
	assert.Equal(t, p, got)
}

// TestWithStateAddsUnderlyingTokensOnce: a pool from its factory log holds its LP tokens only, and
// the first state it gets, from logs or from the lens, puts their underlying tokens in front; a
// later one leaves them.
func TestWithStateAddsUnderlyingTokensOnce(t *testing.T) {
	t.Parallel()

	weth, usdc := common.HexToAddress("0x1111"), common.HexToAddress("0x2222")
	p := entity.Pool{Tokens: []*entity.PoolToken{
		{Address: "0x5994258ec80cc6853e2b6f047ec6d213fe89b24b", Decimals: 18},
		{Address: "0x91f70f89915f8e5fc9fdd8078685067a49cc6c28", Decimals: 6},
	}}
	rpcState := &FetchRPCResult{
		Liquidity:        big.NewInt(9),
		Slot0:            Slot0{SqrtPriceX96: big.NewInt(1), Tick: big.NewInt(-5), Unlocked: true},
		Reserves:         [2]*big.Int{big.NewInt(3), big.NewInt(4)},
		UnderlyingTokens: []common.Address{weth, usdc},
		Vaults:           [2]Vault{{ExchangeRate: uint256.NewInt(1)}, {ExchangeRate: uint256.NewInt(1)}},
	}
	ticks := []Tick{{Index: -10, LiquidityGross: big.NewInt(9), LiquidityNet: big.NewInt(9)},
		{Index: 10, LiquidityGross: big.NewInt(9), LiquidityNet: big.NewInt(-9)}}

	got, err := withState(p, rpcState, StaticExtra{TickSpacing: 10}, ticks)
	require.NoError(t, err)
	require.Len(t, got.Tokens, 4)
	assert.Equal(t, "0x0000000000000000000000000000000000001111", got.Tokens[0].Address)
	assert.Equal(t, 6, int(got.Tokens[1].Decimals))
	assert.Equal(t, entity.PoolReserves{"3", "4", "1", "1"}, got.Reserves)
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &staticExtra))
	assert.True(t, staticExtra.NeedScanUnderlying)
	assert.True(t, tickspkg.HasState(got))

	again, err := withState(got, rpcState, staticExtra, ticks)
	require.NoError(t, err)
	assert.Len(t, again.Tokens, 4, "the underlying tokens were added twice")
}

// TestTicksFromLensNeedsTheLens: without the lens the stored ticks are refetched as before.
func TestTicksFromLensNeedsTheLens(t *testing.T) {
	t.Parallel()

	tr := NewPoolTracker(&Config{}, nil, nil)
	tbp := tickspkg.TicksBasedPool{Ticks: map[int]tickspkg.Tick{1: {TickIdx: 1}}}
	assert.False(t, tr.ticksFromLens(t.Context(), &tbp, 100))
	assert.Len(t, tbp.Ticks, 1)
}
