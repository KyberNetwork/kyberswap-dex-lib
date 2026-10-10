package btr

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
		{500, 503},   // below the first sample the first segment extends back: 999 - ceil(500*991/1000)
		{1, 8},
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
