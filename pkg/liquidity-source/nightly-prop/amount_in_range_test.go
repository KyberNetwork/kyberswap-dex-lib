package nightlyprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestEVM_AmountInRange(t *testing.T) {
	f := loadEVMFixture(t)
	a := f.accounts
	sim := f.simulator(t, a.WethUsdc, a.Weth, a.Usdc)
	for _, tc := range []struct {
		name    string
		in, out common.Address
		maxIn   string
	}{
		{"buy across two asks", a.Usdc, a.Weth, "40040000000"},
		{"sell after an empty bid", a.Weth, a.Usdc, "10005002501250625313"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			minIn, maxIn, err := sim.AmountInRange(hexAddress(tc.in), hexAddress(tc.out), nil)
			require.NoError(t, err)
			require.Nil(t, minIn)
			require.Equal(t, tc.maxIn, maxIn.String())

			atMax := uint256.MustFromBig(maxIn)
			out, err := f.poolQuote(a.WethUsdc, tc.in, atMax)
			require.NoError(t, err)
			require.Positive(t, out.Sign())
			_, err = f.poolQuote(a.WethUsdc, tc.in, new(uint256.Int).AddUint64(atMax, 1))
			require.ErrorIs(t, err, vm.ErrExecutionReverted)
		})
	}
}

func TestAmountInRange_AfterFill(t *testing.T) {
	extra := simExtra(t, 0)
	extra.Price.BidSpread, extra.Price.AskSpread = 20_000, 10_000
	original := newSim(t, extra, false)
	sim := original.CloneState().(*PoolSimulator)
	fill, err := quote(sim, arbitrumUsdc, usdcUnits(t, 25_000), arbitrumWeth)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: fill.SwapInfo})
	// The buy crossed into level 1 and left the active level with both tokens.
	require.Equal(t, int16(1), sim.Extra.Pool.Position.LevelID)
	require.False(t, sim.Extra.Pool.Position.Fraction.IsLower())
	require.False(t, sim.Extra.Pool.Position.Fraction.IsUpper())

	for _, tc := range []struct {
		in, out     string
		originalMax string
		change      int
	}{
		{arbitrumUsdc, arbitrumWeth, "60670700000", -1},
		{arbitrumWeth, arbitrumUsdc, "10209186225765944197", 1},
	} {
		_, before, err := original.AmountInRange(tc.in, tc.out, nil)
		require.NoError(t, err)
		require.Equal(t, tc.originalMax, before.String())
		_, after, err := sim.AmountInRange(tc.in, tc.out, nil)
		require.NoError(t, err)
		require.Equal(t, tc.change, after.Cmp(before))

		maxIn := uint256.MustFromBig(after)
		drain, err := quote(sim, tc.in, maxIn, tc.out)
		require.NoError(t, err)
		_, err = quote(sim, tc.in, new(uint256.Int).AddUint64(maxIn, 1), tc.out)
		require.ErrorIs(t, err, math.ErrInsufficientLiquidity)

		drained := sim.CloneState().(*PoolSimulator)
		drained.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: drain.SwapInfo})
		_, empty, err := drained.AmountInRange(tc.in, tc.out, nil)
		require.NoError(t, err)
		require.Zero(t, empty.Sign())
	}
}

func TestAmountInRange_Uint256Limit(t *testing.T) {
	extra := simExtra(t, 0)
	extra.Pool.BaseDecimals, extra.Pool.QuoteDecimals = 0, 0
	extra.Price.MidPrice = 50_000_000 // Each quote token buys two base tokens.
	liquidity := new(uint256.Int).Lsh(uint256.NewInt(1), 254)
	extra.Pool.Levels = []math.Level{
		simLevel(0, 0, uint256.Int{}, *liquidity),
		simLevel(0, 0, uint256.Int{}, *liquidity),
	}
	extra.Pool.Position.Fraction = math.Upper
	sim := newSim(t, extra, false)
	_, maxIn, err := sim.AmountInRange(arbitrumWeth, arbitrumUsdc, nil)
	require.NoError(t, err)
	require.Equal(t, new(uint256.Int).SetAllOne().ToBig(), maxIn)
	_, err = quote(sim, arbitrumWeth, uint256.MustFromBig(maxIn), arbitrumUsdc)
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: arbitrumWeth, Amount: new(big.Int).Add(maxIn, big.NewInt(1))},
		TokenOut:      arbitrumUsdc,
	})
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestAmountInRange_Unavailable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Extra)
	}{
		{"router paused", func(e *Extra) { e.RouterPaused = true }},
		{"partner disabled", func(e *Extra) { e.PartnerRole = false }},
		{"no price", func(e *Extra) { e.PriceAvailable = false }},
		{"no levels", func(e *Extra) { e.Pool.Levels = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extra := simExtra(t, nowMs())
			tc.change(&extra)
			sim := newSim(t, extra, true)
			minIn, maxIn, err := sim.AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
			require.NoError(t, err)
			require.Nil(t, minIn)
			require.Zero(t, maxIn.Sign())
		})
	}

	stale := simExtra(t, 0)
	_, maxIn, err := newSim(t, stale, true).AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
	require.NoError(t, err)
	require.Zero(t, maxIn.Sign())
	_, maxIn, err = newSim(t, stale, false).AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
	require.NoError(t, err)
	require.Equal(t, "60070000000", maxIn.String())
}

func TestAmountInRange_Error(t *testing.T) {
	extra := simExtra(t, 0)
	// Draining the last ask would overflow its quote balance. Smaller buys still succeed.
	extra.Pool.Levels[2].Liquidity.QuoteAmount.SetAllOne()
	sim := newSim(t, extra, false)
	for _, tokens := range [][2]string{{"unknown", arbitrumWeth}, {arbitrumUsdc, "unknown"}, {arbitrumUsdc, arbitrumUsdc}} {
		_, _, err := sim.AmountInRange(tokens[0], tokens[1], nil)
		require.ErrorIs(t, err, ErrInvalidToken)
	}
	_, err := quote(sim, arbitrumUsdc, usdcUnits(t, 1), arbitrumWeth)
	require.NoError(t, err)
	minIn, maxIn, err := sim.AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
	require.ErrorIs(t, err, pool.ErrUnsupported)
	require.ErrorIs(t, err, math.ErrLiquidityOverflow)
	require.Nil(t, minIn)
	require.Nil(t, maxIn)
}
