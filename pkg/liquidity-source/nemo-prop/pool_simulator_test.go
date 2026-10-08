package nemoprop

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func quote(sim pool.IPoolSimulator, dir int, amountIn *big.Int) (*big.Int, error) {
	res, err := quoteResult(sim, dir, amountIn, nil)
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

func quoteResult(sim pool.IPoolSimulator, dir int, amountIn *big.Int,
	limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	tokens := sim.GetTokens()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[dir], Amount: amountIn},
		TokenOut:      tokens[1-dir],
		Limit:         limit,
	})
}

func quote1000(sim pool.IPoolSimulator) int64 {
	out, err := quote(sim, 0, big.NewInt(1000))
	if err != nil {
		return -1
	}
	return out.Int64()
}

func newSim(t *testing.T, p entity.Pool) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	return sim
}

var reserves1e9 = entity.PoolReserves{"1000000000", "1000000000"}

func TestPoolSimulator_QuotesPoolStateLadders(t *testing.T) {
	t.Parallel()
	sim := newSim(t, pooledState(t, linearLadders(2, 0.5), reserves1e9, 0, 1000, 5000, 10))
	assert.Equal(t, int64(2000), quote1000(sim))
	out, err := quote(sim, 1, big.NewInt(1000))
	require.NoError(t, err)
	assert.Equal(t, int64(500), out.Int64())
	assert.Equal(t, pool.MetaInfo{ApprovalAddress: hexAddr(testProxy), BlockNumber: 1001}, sim.GetMetaInfo("", ""))
	assert.Equal(t, int64(defaultGas), sim.Gas)
}

// Routers decode cached simulators; a decoded one quotes the same, with no
// connection or other live handle to restore.
func TestPoolSimulator_MsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	sim := newSim(t, pooledState(t, linearLadders(2, 0.5), reserves1e9, 0, 1000, 5000, 10))

	require.NoError(t, msgpack.RegisterConcreteType(&PoolSimulator{}))
	var buf bytes.Buffer
	en := msgpack.NewEncoder(&buf)
	en.IncludeUnexported(true)
	en.SetForceAsArray(true)
	require.NoError(t, en.Encode(map[string]pool.IPoolSimulator{"p": sim}))

	de := msgpack.NewDecoder(&buf)
	de.IncludeUnexported(true)
	de.SetForceAsArray(true)
	var decoded map[string]pool.IPoolSimulator
	require.NoError(t, de.Decode(&decoded))
	assert.Equal(t, int64(2000), quote1000(decoded["p"]))
	assert.Equal(t, sim.GetMetaInfo("", ""), decoded["p"].GetMetaInfo("", ""))
}

// Past the fresh window a snapshot's outputs decay at DecayBps per second,
// fixed when the simulator is built.
func TestPoolSimulator_DecaysPastFreshWindow(t *testing.T) {
	t.Parallel()
	// 2s past a 1s window at 50%/s: outputs scale by 1/(1 + 0.5*2) = 1/2. Test runtime (and
	// ms truncation) adds staleness, so bound the decay by the measured build time.
	start := time.Now()
	sim := newSim(t, pooledState(t, linearLadders(2, 0.5), reserves1e9, 3*time.Second, 1000, 60_000, 5000))
	maxStale := 2 + time.Since(start).Seconds() + 0.001
	got := quote1000(sim)
	assert.LessOrEqual(t, got, int64(1000))
	assert.GreaterOrEqual(t, got, int64(2000/(1+0.5*maxStale))-1)

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, got, quote1000(sim), "decay is fixed at build, quotes don't read the clock")
}

func TestPoolSimulator_StaleSnapshots(t *testing.T) {
	t.Parallel()
	for name, p := range map[string]entity.Pool{
		"no snapshot":         testPool(t),
		"past max age":        pooledState(t, linearLadders(2, 0.5), reserves1e9, 6*time.Second, 1000, 5000, 10),
		"stale without decay": pooledState(t, linearLadders(2, 0.5), reserves1e9, 2*time.Second, 1000, 5000, 0),
	} {
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
		assert.ErrorIs(t, err, ladder.ErrStale, name)

		// Without StaleCheck it builds but quotes and delivers nothing: an aged-out
		// snapshot must never quote better than a younger, decayed one.
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err, name)
		_, err = quote(sim, 0, big.NewInt(1000))
		assert.ErrorIs(t, err, ladder.ErrNoQuote, name)
		for token, limit := range sim.CalculateLimit() {
			assert.Zero(t, limit.Sign(), "%s: %s limit", name, token)
		}
	}

	// Just under the maximum age it still quotes, decayed (3.9s past a 1s
	// window at 10 bps/s: 2000 / 1.0039).
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: pooledState(t, linearLadders(2, 0.5), reserves1e9, 4900*time.Millisecond, 1000, 5000, 10),
	})
	require.NoError(t, err)
	assert.InDelta(t, 1992, quote1000(sim), 2)
}

func TestExtra_DecayAt(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for name, tc := range map[string]struct {
		age      time.Duration
		extra    Extra
		scale    float64
		quotable bool
	}{
		"fresh":               {999 * time.Millisecond, Extra{FreshMs: 1000, MaxAgeMs: 5000, DecayBps: 10}, 1, true},
		"decayed":             {3 * time.Second, Extra{FreshMs: 1000, MaxAgeMs: 5000, DecayBps: 10}, 1 / 1.002, true},
		"max age":             {5 * time.Second, Extra{FreshMs: 1000, MaxAgeMs: 5000, DecayBps: 10}, 1, false},
		"stale without decay": {2 * time.Second, Extra{FreshMs: 1000, MaxAgeMs: 5000}, 1, false},
	} {
		tc.extra.PricedAtMs = now.Add(-tc.age).UnixMilli()
		scale, quotable := tc.extra.decayAt(now)
		assert.InDelta(t, tc.scale, scale, 1e-12, name)
		assert.Equal(t, tc.quotable, quotable, name)
	}
}

// A route's consumption never changes what other quotes see, and inventory
// consumed in one market lowers what other markets can deliver.
func TestPoolSimulator_RouteConsumptionAndSharedInventory(t *testing.T) {
	t.Parallel()
	usdc := entity.PoolReserves{"1500", "1000000000"}
	sim := newSim(t, pooledState(t, linearLadders(2, 0.5), usdc, 0, 1000, 5000, 0))
	btcState := pooledState(t, linearLadders(2, 0.5), usdc, 0, 1000, 5000, 0)
	btcState.Address, btcState.Tokens = btcPool(t).Address, btcPool(t).Tokens
	btc := newSim(t, btcState)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: hexAddr(testWETH), Amount: big.NewInt(2000)},
		TokenOut:      hexAddr(testUSDC),
	}
	testutil.TestCloneState(t, sim, params, nil)

	limits := map[string]*big.Int{}
	for _, s := range []*PoolSimulator{sim, btc} {
		for token, l := range s.CalculateLimit() {
			limits[token] = l
		}
	}
	inventory := swaplimit.NewInventory(DexType, limits)

	// Sell WETH for 1000 USDC through one market...
	res, err := quoteResult(sim, 1, big.NewInt(2000), inventory)
	require.NoError(t, err)
	require.Equal(t, int64(1000), res.TokenAmountOut.Amount.Int64())
	route := sim.CloneState()
	route.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
		SwapInfo: res.SwapInfo, SwapLimit: inventory,
	})

	// ...which leaves 500 deliverable USDC for the other market.
	_, err = quoteResult(btc, 1, big.NewInt(2000), inventory)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	res, err = quoteResult(btc, 1, big.NewInt(1000), inventory)
	require.NoError(t, err)
	assert.Equal(t, int64(500), res.TokenAmountOut.Amount.Int64())

	// The base simulator is untouched.
	out, err := quote(sim, 1, big.NewInt(2000))
	require.NoError(t, err)
	assert.Equal(t, int64(1000), out.Int64())
}
