package nightlyprop

import (
	"math/big"
	"testing"
	"testing/synctest"

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
	checkRange := func(name string, in, out common.Address, wantMax string) {
		t.Run(name, func(t *testing.T) {
			minIn, maxIn, err := sim.AmountInRange(hexAddress(in), hexAddress(out), nil)
			require.NoError(t, err)
			require.Nil(t, minIn)
			require.Equal(t, wantMax, maxIn.String())

			atMax := uint256.MustFromBig(maxIn)
			wantOut, err := f.poolQuote(a.WethUsdc, in, atMax)
			require.NoError(t, err)
			require.Positive(t, wantOut.Sign())
			got, err := quote(sim, hexAddress(in), atMax, hexAddress(out))
			require.NoError(t, err)
			require.Equal(t, wantOut, got.TokenAmountOut.Amount)

			aboveMax := new(uint256.Int).AddUint64(atMax, 1)
			_, err = quote(sim, hexAddress(in), aboveMax, hexAddress(out))
			require.ErrorIs(t, err, math.ErrInsufficientLiquidity)
			input, err := evmABI.Pack("quoteSwapExactIn", in, aboveMax.ToBig())
			require.NoError(t, err)
			revert, err := f.call(common.Address{}, a.WethUsdc, input)
			require.ErrorIs(t, err, vm.ErrExecutionReverted)
			require.Len(t, revert, 4+common.HashLength)
			require.ErrorIs(t, fuzzSwapErrors[[4]byte(revert[:4])], math.ErrInsufficientLiquidity)
			require.Equal(t, big.NewInt(1), new(big.Int).SetBytes(revert[4:]))
		})
	}

	checkRange("buy across two asks", a.Usdc, a.Weth, "40040000000")
	checkRange("sell after an empty bid", a.Weth, a.Usdc, "10005002501250625313")

	buy := evmUnits(25_000, 6)
	f.fuzzMustCall(t, a.Partner, a.Usdc, "mint", a.Partner, buy.ToBig())
	f.fuzzMustCall(t, a.Partner, a.Usdc, "approve", a.Router, buy.ToBig())
	f.fuzzSwap(t, sim, true, buy)
	// Rounding in the partly filled ask leaves 15,040.000001 USDC of buy capacity.
	checkRange("buy after spending 25000 USDC", a.Usdc, a.Weth, "15040000001")
	checkRange("sell after spending 25000 USDC", a.Weth, a.Usdc, "22497199751643493029")

	sell := evmUnits(5, 18)
	f.fuzzMustCall(t, a.Partner, a.Weth, "approve", a.Router, sell.ToBig())
	f.fuzzSwap(t, sim, false, sell)
	checkRange("buy after selling 5 WETH", a.Usdc, a.Weth, "25049992199")
	checkRange("sell after selling 5 WETH", a.Weth, a.Usdc, "17497199751768638710")
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
	extra.Price.MidPrice = 50_000_000 // The midpoint is half a quote token per base token.
	liquidity := new(uint256.Int).Lsh(uint256.NewInt(1), 254)
	extra.Pool.Levels = []math.Level{
		simLevel(-1_000, 0, uint256.Int{}, *liquidity),
		simLevel(0, 1_000, uint256.Int{}, *liquidity),
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

func TestAmountInRange_Uint256BuyCeiling(t *testing.T) {
	extra := simExtra(t, 0)
	extra.Pool.BaseDecimals, extra.Pool.QuoteDecimals = 0, 0
	extra.Price.MidPrice = 300_000_000
	liquidity := new(uint256.Int).Lsh(uint256.NewInt(1), 253)
	extra.Pool.LowestLevelID = 0
	extra.Pool.Levels = []math.Level{
		simLevel(-1_000, 1_000, *liquidity, uint256.Int{}),
		simLevel(1_000, 3_000, *liquidity, uint256.Int{}),
		simLevel(3_000, 5_000, *liquidity, uint256.Int{}),
	}
	sim := newSim(t, extra, false)
	_, maxIn, err := sim.AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
	require.NoError(t, err)
	require.Equal(t, new(uint256.Int).SetAllOne().ToBig(), maxIn)

	// A full first ask costs exactly three quote units per base unit.
	firstAskCost := new(uint256.Int).Mul(liquidity, uint256.NewInt(3))
	fill, err := quote(sim, arbitrumUsdc, firstAskCost, arbitrumWeth)
	require.NoError(t, err)
	require.Equal(t, liquidity.ToBig(), fill.TokenAmountOut.Amount)

	// At the reported ceiling, the last ask is only partly filled and its price calculation overflows.
	_, err = quote(sim, arbitrumUsdc, uint256.MustFromBig(maxIn), arbitrumWeth)
	require.ErrorIs(t, err, math.ErrOverflow)
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
}

func TestAmountInRange_PriceAge(t *testing.T) {
	const capacity = 60_070_000_000 // USDC base units
	for _, tc := range []struct {
		name       string
		ageMs      uint64
		staleCheck bool
		wantMax    int64
	}{
		{"fresh", 0, true, capacity},
		{"at expiry", 30_000, true, capacity},
		{"one millisecond past expiry", 30_001, true, 0},
		{"age check disabled", 30_001, false, capacity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				sim := newSim(t, simExtra(t, nowMs()-tc.ageMs), tc.staleCheck)
				minIn, maxIn, err := sim.AmountInRange(arbitrumUsdc, arbitrumWeth, nil)
				require.NoError(t, err)
				require.Nil(t, minIn)
				require.Equal(t, big.NewInt(tc.wantMax), maxIn)
			})
		})
	}
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
