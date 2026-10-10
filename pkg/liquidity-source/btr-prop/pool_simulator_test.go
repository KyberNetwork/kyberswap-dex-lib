package btrprop

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	usdc = "0x754704bc059f8c67012fed69bc8a327a5aafb603"
	wmon = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	// Live Monad core flag word (2026-10-08): SWAP_ENABLED|LIABILITY_SWAP|
	// DEPOSIT_GATED|SWAP_GATED|COOP_ENABLED = 3632.
	liveCoreFlags uint16 = 3632
)

func pts(v ...uint64) []Point {
	res := make([]Point, len(v)/2)
	for i := range res {
		res[i] = Point{In: *uint256.NewInt(v[2*i]), Out: *uint256.NewInt(v[2*i+1])}
	}
	return res
}

// newSim builds a USDC/WMON simulator with one USDC->WMON ladder.
func newSim(t *testing.T, flags uint16, ladder []Point) *PoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{Flags: []uint16{flags, flags}, Ladders: []Ladder{{From: 0, To: 1, Points: ladder}}})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: CreditPoolCore, Exchange: DexType, Type: DexType, Extra: string(extra),
		Tokens:   []*entity.PoolToken{{Address: usdc, Swappable: true}, {Address: wmon, Swappable: true}},
		Reserves: entity.PoolReserves{"1000000", "1000000"},
	}})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, in int64) (*big.Int, error) {
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(in)}, TokenOut: wmon,
	})
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

func TestGate(t *testing.T) {
	cases := []struct {
		name  string
		flags uint16
		want  error
	}{
		{"live core is SWAP_GATED", liveCoreFlags, ErrBTRGatedLane},
		{"clean", SwapEnabledBit, nil},
		{"halted", SwapEnabledBit | HaltBit, ErrBTRGated},
		{"swap disabled", 0, ErrBTRGated},
		{"gated only", SwapEnabledBit | SwapGatedBit, ErrBTRGatedLane},
		{"feed halt", SwapEnabledBit | 4, ErrBTRGated},
	}
	for _, c := range cases {
		require.Equal(t, c.want, gate(c.flags), c.name)
	}
}

func TestSelector(t *testing.T) {
	// swap_qe selector is suffix-mined; the encoder hard-codes the ABI signature.
	require.Equal(t, "01dc30b8", hex.EncodeToString(poolABI.Methods["swap_qe"].ID))
}

func TestInterpolate(t *testing.T) {
	ladder := pts(1000, 999, 2000, 1990)
	var res uint256.Int
	for _, c := range []struct {
		in   uint64
		want uint64
	}{
		{2000, 1990}, // exact sample
		{1500, 1494}, // 999 + (1990-999)*500/1000 = 1494, floor
		{500, 499},   // below the first sample: min(extension 999-ceil(500*991/1000)=503, proportional 499)
		{1, 0},
	} {
		require.NoError(t, interpolate(ladder, uint256.NewInt(c.in), &res))
		require.Equal(t, c.want, res.Uint64(), c.in)
	}
	// A fixed cost makes the head convex: the extension must not go negative.
	require.NoError(t, interpolate(pts(1000, 100, 2000, 1000), uint256.NewInt(100), &res))
	require.True(t, res.IsZero())
	// Past the last sample the pool caps output but still takes the input.
	require.Equal(t, ErrBTRAmountTooLarge, interpolate(ladder, uint256.NewInt(2001), &res))
}

// The live curve is concave, so the first segment extended backwards sits above
// it at dust (+145 bps USDC>WMON, +101 bps WMON>USDC measured) and only the line
// from the origin stays below. Without the min() this fails at every dust size.
func TestInterpolateDustNotAboveConcaveCurve(t *testing.T) {
	const b = 1_000_000
	exact := func(x uint64) uint64 { return x * b / (x + b) } // concave, exact(0) = 0
	var ladder []Point
	for _, x := range []uint64{100_000, 200_000, 400_000} {
		ladder = append(ladder, Point{In: *uint256.NewInt(x), Out: *uint256.NewInt(exact(x))})
	}
	var res uint256.Int
	for _, x := range []uint64{1, 7, 100, 1000, 9999, 50_000, 99_999} {
		require.NoError(t, interpolate(ladder, uint256.NewInt(x), &res))
		require.LessOrEqual(t, res.Uint64(), exact(x), "overquote at dust x=%d", x)
	}
}

func TestCalcAmountOutGated(t *testing.T) {
	sim := newSim(t, liveCoreFlags, pts(1000, 999))
	_, err := calc(sim, 100)
	require.Equal(t, ErrBTRGatedLane, err)
	require.True(t, sim.GetMetaInfo("", "").(PoolMeta).Gated)
}

func TestCalcAmountOutBounds(t *testing.T) {
	sim := newSim(t, SwapEnabledBit, pts(1000, 999, 2000, 1990))
	out, err := calc(sim, 1500)
	require.NoError(t, err)
	require.Equal(t, "1494", out.String())

	_, err = calc(sim, 2001)
	require.Equal(t, ErrBTRAmountTooLarge, err)
	_, err = calc(sim, 0)
	require.Equal(t, ErrBTRInvalidAmount, err)
	// Dust below the fixed cost fills nothing.
	_, err = calc(newSim(t, SwapEnabledBit, pts(1000, 100, 2000, 1000)), 100)
	require.Equal(t, ErrBTRNoFill, err)

	// The reverse pair has no ladder.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wmon, Amount: big.NewInt(1000)}, TokenOut: usdc,
	})
	require.Equal(t, ErrBTRNoRoute, err)
}

// A second swap on the same pair must price along the curve, and a clone must
// not leak into the original: split routes would otherwise reuse the best
// segment for every leg.
func TestUpdateBalanceContinuesCurve(t *testing.T) {
	sim := newSim(t, SwapEnabledBit, pts(1000, 990, 2000, 1900)) // slope drops from 0.99 to 0.91
	first, err := calc(sim, 1000)
	require.NoError(t, err)
	whole, err := calc(sim, 2000)
	require.NoError(t, err)

	cloned := sim.CloneState().(*PoolSimulator)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: usdc, Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: wmon, Amount: first},
	})
	second, err := calc(cloned, 1000)
	require.NoError(t, err)
	require.Equal(t, whole.Int64(), first.Int64()+second.Int64(), "two halves must equal one whole swap")
	require.Less(t, second.Int64(), first.Int64())

	again, err := calc(sim, 1000)
	require.NoError(t, err)
	require.Equal(t, first, again, "UpdateBalance on the clone changed the original")
	require.Equal(t, "1000000", sim.Info.Reserves[0].String())
	require.Equal(t, "1001000", cloned.Info.Reserves[0].String())

	_, err = calc(cloned, 1001) // 1000 used + 1001 exceeds the last sample
	require.Equal(t, ErrBTRAmountTooLarge, err)
}

func TestNewPoolSimulatorRejectsUntrackedPool(t *testing.T) {
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Tokens: []*entity.PoolToken{{Address: usdc}, {Address: wmon}}, Extra: "{}",
	}})
	require.Equal(t, ErrBTRBadState, err)
}

// Swaps on different pairs that share a token draw on the same inventory. A
// route that sells A>C and then B>C must price the B>C leg after the C already
// taken, not from the first sample: on the live core, splitting USDC>WMON over
// USDC>cbBTC>WMON paths was quoted 2.5% above what the chain delivered.
func TestUpdateBalanceSharesOutputToken(t *testing.T) {
	const c = "0x0555e30da8f98308edb960aa94c0db47230d2b9c"
	extra, err := json.Marshal(Extra{
		Flags: []uint16{SwapEnabledBit, SwapEnabledBit, SwapEnabledBit},
		Ladders: []Ladder{
			{From: 0, To: 2, Points: pts(1000, 990, 2000, 1900)},
			{From: 1, To: 2, Points: pts(500, 1000, 1000, 1900)},
		},
	})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: CreditPoolCore, Exchange: DexType, Type: DexType, Extra: string(extra),
		Tokens:   []*entity.PoolToken{{Address: usdc}, {Address: wmon}, {Address: c}},
		Reserves: entity.PoolReserves{"1", "1", "1"},
	}})
	require.NoError(t, err)
	quoteBC := func(s *PoolSimulator) string {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wmon, Amount: big.NewInt(100)}, TokenOut: c,
		})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount.String()
	}
	require.Equal(t, "200", quoteBC(sim)) // below the first sample: min(1000-ceil(400*900/500)=280, 1000*100/500=200)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: usdc, Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: c, Amount: big.NewInt(990)},
	})
	// 990 of C is gone: B>C restarts at invert(990) = 495, so 100 more input
	// yields f(595)-f(495) = 1171-990 = 181.
	require.Equal(t, "181", quoteBC(sim))
}

// invert must never map an output to a smaller input than interpolate needs: a
// round-down would let a later leg start earlier on the curve than the earlier legs left it.
func TestInvertRoundsAgainstTrader(t *testing.T) {
	// The second ladder has a positive backward intercept (300), so below the first
	// sample the proportional line binds instead of the extension.
	for _, ladder := range [][]Point{pts(500, 1000, 1000, 1900, 2000, 3000), pts(1000, 900, 2000, 1500)} {
		for _, y := range []uint64{1, 100, 400, 899, 900, 999, 1000, 1001, 1500, 1900, 2500, 2999} {
			if y > ladder[len(ladder)-1].Out.Uint64() {
				continue
			}
			var x, back uint256.Int
			invert(ladder, uint256.NewInt(y), &x)
			require.NoError(t, interpolate(ladder, &x, &back))
			require.GreaterOrEqual(t, back.Uint64(), y, "y=%d x=%d", y, x.Uint64())
		}
	}
	var x uint256.Int
	invert(pts(500, 1000, 1000, 1900, 2000, 3000), uint256.NewInt(3000), &x)
	require.Equal(t, uint64(2000), x.Uint64(), "at or above the last sample returns the last input")
}
