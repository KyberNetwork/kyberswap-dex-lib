package liquidityparty

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// fixtureExtra is a real snapshot of the mainnet 3-token pool 0x1270…fdA6 (κ = 2^64 = 1.0),
// captured via TestIntegration_PoolTracker. Used for deterministic, RPC-free unit tests.
const fixtureExtra = `{
	"kappa": 18446744073709551616,
	"eSigmaQ": 55645345147174899802,
	"q": [17906861502704967074, 18693449564602851413, 19049174469333789893],
	"bases": [3333710, 2003578391006337, 51969649724560855],
	"fees": [40, 250, 1250],
	"killed": false
}`

var fixtureReserves = entity.PoolReserves{"3236142", "2030374111081453", "53666865044452501"}

func newFixturePool(t *testing.T, extra string, reserves entity.PoolReserves) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x1270da05cf1d047763ceefde25a4a5438b26fda6",
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0xaa", Swappable: true},
			{Address: "0xbb", Swappable: true},
			{Address: "0xcc", Swappable: true},
		},
		Reserves:    reserves,
		Extra:       extra,
		BlockNumber: 25460504,
	})
	require.NoError(t, err)
	return sim
}

func (p *PoolSimulator) tokenAt(i int) string { return p.Info.Tokens[i] }

func TestPoolSimulator_CalcAmountOut_Basic(t *testing.T) {
	sim := newFixturePool(t, fixtureExtra, fixtureReserves)

	// 0.001 token0 (base_0 ≈ 3.3M so ~300 internal ulps) → some token1 out.
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.tokenAt(0), Amount: big.NewInt(1000)},
		TokenOut:      sim.tokenAt(1),
	})
	require.NoError(t, err)
	require.NotNil(t, res.TokenAmountOut)
	require.Equal(t, 1, res.TokenAmountOut.Amount.Sign(), "output must be positive")
	require.Equal(t, sim.tokenAt(1), res.TokenAmountOut.Token)
	require.Equal(t, sim.tokenAt(1), res.Fee.Token, "fee is charged on the output token")
	require.GreaterOrEqual(t, res.Fee.Amount.Sign(), 0)

	// SwapInfo must be populated for UpdateBalance.
	si, ok := res.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.Equal(t, 0, si.TokenInIndex)
	require.Equal(t, 1, si.TokenOutIndex)
	require.Equal(t, 1, si.DeltaInternal.Sign())
	require.Equal(t, 1, si.GrossInternal.Sign())
}

func TestPoolSimulator_CalcAmountOut_PurityAndClone(t *testing.T) {
	sim := newFixturePool(t, fixtureExtra, fixtureReserves)
	amt := big.NewInt(1_000_000)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.tokenAt(0), Amount: amt},
		TokenOut:      sim.tokenAt(2),
	}

	// Repeated quoting is deterministic and does not mutate state.
	r1, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	r2, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, r1.TokenAmountOut.Amount, r2.TokenAmountOut.Amount)

	// Clone, apply the swap on the clone: original stays unchanged.
	clone := sim.CloneState()
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.tokenAt(0), Amount: amt},
		TokenAmountOut: pool.TokenAmount{Token: sim.tokenAt(2), Amount: r1.TokenAmountOut.Amount},
		SwapInfo:       r1.SwapInfo,
	})

	rOrig, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, r1.TokenAmountOut.Amount, rOrig.TokenAmountOut.Amount, "original must be untouched by clone update")

	// After consuming input on the clone, the same quote moves (q-vector shifted, price impact).
	rClone, err := clone.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotEqual(t, r1.TokenAmountOut.Amount, rClone.TokenAmountOut.Amount, "clone state must evolve after UpdateBalance")

	// Reserves updated: token0 +amountIn, token2 -amountOut.
	origRes := sim.GetReserves()
	cloneRes := clone.GetReserves()
	require.Equal(t, 0, new(big.Int).Sub(cloneRes[0], origRes[0]).Cmp(amt))
	require.Equal(t, 0, new(big.Int).Add(new(big.Int).Sub(cloneRes[2], origRes[2]), r1.TokenAmountOut.Amount).Sign())
}

func TestPoolSimulator_ExactOut_RoundTrip(t *testing.T) {
	sim := newFixturePool(t, fixtureExtra, fixtureReserves)

	// Want ~1e9 of token2 out, paying token1.
	want := big.NewInt(1_000_000_000)
	in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: sim.tokenAt(2), Amount: want},
		TokenIn:        sim.tokenAt(1),
	})
	require.NoError(t, err)
	require.Equal(t, 1, in.TokenAmountIn.Amount.Sign())
	require.Equal(t, sim.tokenAt(1), in.TokenAmountIn.Token)

	// Feeding that input to exact-in must yield at least the requested output (conservative rounding).
	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.tokenAt(1), Amount: in.TokenAmountIn.Amount},
		TokenOut:      sim.tokenAt(2),
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(want), 0,
		"exact-in of the exact-out amountIn must cover the requested output")
}

func TestPoolSimulator_Errors(t *testing.T) {
	sim := newFixturePool(t, fixtureExtra, fixtureReserves)

	// Same token.
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.tokenAt(0), Amount: big.NewInt(1000)},
		TokenOut:      sim.tokenAt(0),
	})
	require.ErrorIs(t, err, ErrSameToken)

	// Unknown token.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeef", Amount: big.NewInt(1000)},
		TokenOut:      sim.tokenAt(1),
	})
	require.ErrorIs(t, err, ErrInvalidToken)

	// Zero input floors to zero internal delta → too small.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.tokenAt(0), Amount: big.NewInt(0)},
		TokenOut:      sim.tokenAt(1),
	})
	require.ErrorIs(t, err, ErrTooSmall)

	// Killed pool rejects all swaps.
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(fixtureExtra), &extra))
	extra.Killed = true
	killedBytes, _ := json.Marshal(&extra)
	killed := newFixturePool(t, string(killedBytes), fixtureReserves)
	_, err = killed.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: killed.tokenAt(0), Amount: big.NewInt(1_000_000)},
		TokenOut:      killed.tokenAt(1),
	})
	require.ErrorIs(t, err, ErrPoolKilled)
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	sim := newFixturePool(t, fixtureExtra, fixtureReserves)
	meta, ok := sim.GetMetaInfo(sim.tokenAt(2), sim.tokenAt(0)).(Meta)
	require.True(t, ok)
	require.Equal(t, 2, meta.TokenInIndex)
	require.Equal(t, 0, meta.TokenOutIndex)
}
