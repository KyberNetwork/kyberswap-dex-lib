package integral

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
)

// TestHasState: only a liquidity the tracker wrote, 0 included, marks a pool as having its state;
// the factory's empty Extra and one carrying only lensDue do not.
func TestHasState(t *testing.T) {
	t.Parallel()

	for extra, want := range map[string]bool{
		``:                        false,
		`{}`:                      false,
		`{"lensDue":7}`:           false,
		`{"liq":null,"ticks":[]}`: false,
		`{"liq":"0","gS":{"price":"1","tick":0}}`:  true,
		`{"liq":"12345","ticks":[],"tS":60}`:       true,
		`{"liq":12345,"ticks":[],"tS":60,"tP":{}}`: true,
	} {
		require.Equal(t, want, hasState(entity.Pool{Extra: extra}), extra)
	}
}

// TestWithoutState: a reset pool loses its liquidity and ticks, so it reads as without state and the
// simulator skips it, keeps the plugin state, and records the block a snapshot must reach, never
// lowering one already recorded.
func TestWithoutState(t *testing.T) {
	t.Parallel()

	p := entity.Pool{Extra: `{"liq":"5","gS":{"price":"79228162514264337593543950336","tick":3},` +
		`"ticks":[{"Index":-60,"LiquidityGross":"5","LiquidityNet":"5"}],"tS":60,"tP":{"3":{"init":true,"ts":9}},` +
		`"lensDue":120}`}

	reset := withoutState(p, 100)
	require.False(t, hasState(reset))
	require.EqualValues(t, 120, tickspkg.LensDue(reset))
	require.EqualValues(t, 150, tickspkg.LensDue(withoutState(p, 150)))

	var extra map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(reset.Extra), &extra))
	require.NotContains(t, extra, "liq")
	require.NotContains(t, extra, "ticks")
	require.JSONEq(t, `{"price":"79228162514264337593543950336","tick":3}`, string(extra["gS"]))
	require.JSONEq(t, `{"3":{"init":true,"ts":9}}`, string(extra["tP"]))

	_, err := NewPoolSimulator(entity.Pool{Extra: reset.Extra, StaticExtra: `{}`, Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{{Address: "a"}, {Address: "b"}}})
	require.ErrorIs(t, err, ErrTicksEmpty)
}

// TestResetPoolStateNeedsTheLens: without the lens nothing can read a pool whole, so it is kept.
func TestResetPoolStateNeedsTheLens(t *testing.T) {
	t.Parallel()

	p := entity.Pool{Extra: `{"liq":"5","ticks":[]}`}
	got, ok := (&PoolTracker{config: &Config{}}).ResetPoolState(p, 10)
	require.False(t, ok)
	require.Equal(t, p, got)

	got, ok = (&PoolTracker{config: &Config{
		UseLens: true}}).ResetPoolState(p, 10)
	require.True(t, ok)
	require.False(t, hasState(got))
}
