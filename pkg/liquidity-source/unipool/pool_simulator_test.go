package unipool

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// ---------------------------------------------------------------------------
// Test helpers / fixtures
// ---------------------------------------------------------------------------

const (
	tokenA = "0x000000000000000000000000000000000000000a"
	tokenB = "0x000000000000000000000000000000000000000b"

	// "non lexico" pair (token0 > token1) to verify we don't assume ordering.
	tokenC = "0x00000000000000000000000000000000000000ff"
	tokenD = "0x0000000000000000000000000000000000000011"
)

// makeExtra builds a default Extra (all VRs == reserves, no fees, no spread
// guard, no borrowed). Tests override fields as needed.
func makeExtra(reserve0, reserve1 *big.Int) Extra {
	return Extra{
		Reserve0:              new(big.Int).Set(reserve0),
		Reserve1:              new(big.Int).Set(reserve1),
		VirtualReserve0In:     new(big.Int).Set(reserve0),
		VirtualReserve0Out:    new(big.Int).Set(reserve0),
		VirtualReserve1In:     new(big.Int).Set(reserve1),
		VirtualReserve1Out:    new(big.Int).Set(reserve1),
		LastUpdateTimestamp:   uint64(time.Now().Unix()),
		PriceDecay:            300,
		FeeLpBps:              0,
		FeePoolBps:            0,
		TotalBorrowed0:        big.NewInt(0),
		TotalBorrowed1:        big.NewInt(0),
		SwapPriceToleranceBps: math.MaxUint16, // disabled
	}
}

// makePool wraps the entity.Pool boilerplate and instantiates a simulator.
func makePool(t *testing.T, token0, token1 string, extra Extra) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	ep := entity.Pool{
		Address:  "0xpool",
		Exchange: "unipool",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Extra: string(extraBytes),
	}
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// solidityAmountOut ports UniPoolPairSwap.getAmountOut for cross-checking.
//
//	amountInWithFee = amountIn * (BPS - totalFee)
//	amountOut       = (reserveOut * amountInWithFee) / (reserveIn * BPS + amountInWithFee)
func solidityAmountOut(amountIn, reserveIn, reserveOut *big.Int, totalFeeBps uint64) *big.Int {
	bps := big.NewInt(int64(bpsDivisor))
	netBps := new(big.Int).Sub(bps, new(big.Int).SetUint64(totalFeeBps))
	amountInWithFee := new(big.Int).Mul(amountIn, netBps)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, bps), amountInWithFee)
	if denominator.Sign() == 0 {
		return big.NewInt(0)
	}
	numerator := new(big.Int).Mul(reserveOut, amountInWithFee)
	return new(big.Int).Quo(numerator, denominator)
}

// pow10 returns 10**n as a *big.Int.
func pow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

// ---------------------------------------------------------------------------
// A. CalcAmountOut basique
// ---------------------------------------------------------------------------

func TestCalcAmountOut_A1_Token0ToToken1_EqualReserves(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// add a small fee so amountOut < amountIn even at equal reserves
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16) // 0.01e18
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.TokenAmountOut)

	out := res.TokenAmountOut.Amount
	assert.Equal(t, 1, out.Sign(), "amountOut must be > 0")
	assert.True(t, out.Cmp(amountIn) < 0, "amountOut must be < amountIn (fees)")

	// Cross-check against the Solidity formula.
	want := solidityAmountOut(amountIn, r, r, 30)
	assert.Equal(t, 0, out.Cmp(want), "want %s got %s", want, out)

	// Fee is reported as zero (port note: pool only tracks Fee at zero today).
	assert.Equal(t, 0, res.Fee.Amount.Sign())
	assert.Equal(t, tokenB, res.TokenAmountOut.Token)
	assert.Equal(t, int64(defaultGas), res.Gas)
}

func TestCalcAmountOut_A2_Token1ToToken0_Symmetry(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)

	res01, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	// Fresh sim because UpdateBalance is not called between calls — but reserves
	// are symmetric so the two directions must yield the exact same amountOut.
	sim2 := makePool(t, tokenA, tokenB, extra)
	res10, err := sim2.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenB, Amount: amountIn},
		TokenOut:      tokenA,
	})
	require.NoError(t, err)

	assert.Equal(t, 0, res01.TokenAmountOut.Amount.Cmp(res10.TokenAmountOut.Amount),
		"symmetric reserves must give the same out: 0->1=%s 1->0=%s",
		res01.TokenAmountOut.Amount, res10.TokenAmountOut.Amount)
}

func TestCalcAmountOut_A3_DustAmountIn(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := big.NewInt(1)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})

	// With reserve = 1e18 and amountIn = 1, the numerator after the fee bps mul
	// is 9970 while the denominator is ~1e22 -> integer division rounds to 0.
	// The simulator should return ErrInsufficientOutputAmount in that case.
	if err == nil {
		assert.NotNil(t, res)
		assert.Equal(t, 1, res.TokenAmountOut.Amount.Sign())
	} else {
		assert.ErrorIs(t, err, ErrInsufficientOutputAmount)
	}
}

func TestCalcAmountOut_A4_HugeAmountIn(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	// amountIn = 10x reserve — well below uint256 overflow but huge for the curve.
	amountIn := new(big.Int).Mul(r, big.NewInt(10))
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	out := res.TokenAmountOut.Amount
	// amountOut must be strictly less than the (effective) reserveOut.
	assert.True(t, out.Cmp(r) < 0, "amountOut (%s) must stay below reserveOut (%s)", out, r)
	// And clearly close to reserveOut: at 10x reserveIn we expect well over 50%.
	half := new(big.Int).Rsh(r, 1)
	assert.True(t, out.Cmp(half) > 0, "amountOut should be most of reserveOut, got %s", out)
}

// ---------------------------------------------------------------------------
// B. Erreurs attendues
// ---------------------------------------------------------------------------

func TestCalcAmountOut_B1_UnknownTokenIn(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeef", Amount: big.NewInt(1)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_B2_UnknownTokenOut(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: big.NewInt(1)},
		TokenOut:      "0xdeadbeef",
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_B3_ZeroAmountIn(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: big.NewInt(0)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_B3b_NegativeAmountIn(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: big.NewInt(-1)},
		TokenOut:      tokenB,
	})
	// uint256.FromBig either flags overflow or the Sign() <= 0 branch trips.
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_B4_AmountInAboveReserve_GetsCaught(t *testing.T) {
	t.Parallel()
	// amountIn > reserveIn AND most of the output is borrowed. The combination
	// pushes the curve output above the available real cap and must be caught.
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.TotalBorrowed1 = new(big.Int).Sub(r, big.NewInt(1)) // cap = 1 wei
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := new(big.Int).Mul(r, big.NewInt(10)) // 10x reserveIn
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.Error(t, err)
	// Either of the two swap-liquidity error paths is acceptable; both express
	// "trade can't physically settle".
	assert.True(t,
		err == ErrInsufficientSwapLiquidity || err == ErrInsufficientLiquidity,
		"want swap or liquidity error, got %v", err)
}

func TestCalcAmountOut_B5_FeeAtOrAboveBPS(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 6000
	extra.FeePoolBps = 4000 // sum == 10_000 == BPS
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrFeeExceedsMax)
}

func TestCalcAmountOut_B5b_FeeWayAboveBPS(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 9000
	extra.FeePoolBps = 9000 // sum > 10_000
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrFeeExceedsMax)
}

// ---------------------------------------------------------------------------
// C. Virtual reserve decay (projectVirtualReserves)
// ---------------------------------------------------------------------------

func TestProjectVR_C1_ElapsedZero_ReturnsStored(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	// stored VRs different from reserves so we can tell them apart
	extra.VirtualReserve0In = pow10(20)
	extra.VirtualReserve0Out = pow10(19)
	extra.VirtualReserve1In = pow10(17)
	extra.VirtualReserve1Out = pow10(16)
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000 // in the future
	sim := makePool(t, tokenA, tokenB, extra)

	v0In, v0Out, v1In, v1Out := sim.projectVirtualReserves(uint64(time.Now().Unix()))
	assert.Equal(t, 0, v0In.ToBig().Cmp(pow10(20)))
	assert.Equal(t, 0, v0Out.ToBig().Cmp(pow10(19)))
	assert.Equal(t, 0, v1In.ToBig().Cmp(pow10(17)))
	assert.Equal(t, 0, v1Out.ToBig().Cmp(pow10(16)))
}

func TestProjectVR_C2_ElapsedExceedsDecay_EqualsReserves(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), new(big.Int).Mul(pow10(18), big.NewInt(2))
	extra := makeExtra(r0, r1)
	extra.VirtualReserve0In = pow10(20)
	extra.VirtualReserve0Out = pow10(19)
	extra.VirtualReserve1In = pow10(17)
	extra.VirtualReserve1Out = pow10(16)
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) - 10_000 // far in the past
	extra.PriceDecay = 300
	sim := makePool(t, tokenA, tokenB, extra)

	v0In, v0Out, v1In, v1Out := sim.projectVirtualReserves(uint64(time.Now().Unix()))
	assert.Equal(t, 0, v0In.ToBig().Cmp(r0))
	assert.Equal(t, 0, v0Out.ToBig().Cmp(r0))
	assert.Equal(t, 0, v1In.ToBig().Cmp(r1))
	assert.Equal(t, 0, v1Out.ToBig().Cmp(r1))
}

func TestProjectVR_C3_PartialInterpolation(t *testing.T) {
	t.Parallel()
	r0 := big.NewInt(1_000_000)
	r1 := big.NewInt(4_000_000)
	extra := makeExtra(r0, r1)
	extra.VirtualReserve0In = big.NewInt(2_000_000)
	extra.VirtualReserve0Out = big.NewInt(500_000)
	extra.VirtualReserve1In = big.NewInt(8_000_000)
	extra.VirtualReserve1Out = big.NewInt(1_000_000)
	extra.PriceDecay = 200
	// Set lastUpdate so that elapsed == 100 at "now".
	now := uint64(time.Now().Unix())
	extra.LastUpdateTimestamp = now - 100
	sim := makePool(t, tokenA, tokenB, extra)

	v0In, v0Out, v1In, v1Out := sim.projectVirtualReserves(now)

	// formula: VR = (stored * (decay-elapsed) + reserve * elapsed) / decay
	// elapsed=100, decay=200, diff=100
	// vr0In = (2_000_000*100 + 1_000_000*100) / 200 = 300_000_000/200 = 1_500_000
	assert.Equal(t, "1500000", v0In.ToBig().String())
	// vr0Out = (500_000*100 + 1_000_000*100)/200 = 150_000_000/200 = 750_000
	assert.Equal(t, "750000", v0Out.ToBig().String())
	// vr1In = (8_000_000*100 + 4_000_000*100)/200 = 1_200_000_000/200 = 6_000_000
	assert.Equal(t, "6000000", v1In.ToBig().String())
	// vr1Out = (1_000_000*100 + 4_000_000*100)/200 = 500_000_000/200 = 2_500_000
	assert.Equal(t, "2500000", v1Out.ToBig().String())
}

// priceDecay == 0 with positive elapsed mirrors the contract's `elapsed >= decay`
// branch: virtual reserves snap to the real reserves immediately.
func TestProjectVR_C4_DecayZero_Elapsed_SnapsToReal(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	extra.VirtualReserve0In = pow10(20)
	extra.VirtualReserve0Out = pow10(19)
	extra.VirtualReserve1In = pow10(17)
	extra.VirtualReserve1Out = pow10(16)
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) - 5_000
	extra.PriceDecay = 0
	sim := makePool(t, tokenA, tokenB, extra)

	v0In, v0Out, v1In, v1Out := sim.projectVirtualReserves(uint64(time.Now().Unix()))
	assert.Equal(t, 0, v0In.ToBig().Cmp(r0))
	assert.Equal(t, 0, v0Out.ToBig().Cmp(r0))
	assert.Equal(t, 0, v1In.ToBig().Cmp(r1))
	assert.Equal(t, 0, v1Out.ToBig().Cmp(r1))
}

// priceDecay == 0 AND elapsed == 0: contract's early-return path takes precedence.
func TestProjectVR_C4b_DecayZero_NoElapsed_KeepsStored(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	extra.VirtualReserve0In = pow10(20)
	extra.VirtualReserve0Out = pow10(19)
	extra.VirtualReserve1In = pow10(17)
	extra.VirtualReserve1Out = pow10(16)
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000 // now <= lastUpdate
	extra.PriceDecay = 0
	sim := makePool(t, tokenA, tokenB, extra)

	v0In, v0Out, v1In, v1Out := sim.projectVirtualReserves(uint64(time.Now().Unix()))
	assert.Equal(t, 0, v0In.ToBig().Cmp(pow10(20)))
	assert.Equal(t, 0, v0Out.ToBig().Cmp(pow10(19)))
	assert.Equal(t, 0, v1In.ToBig().Cmp(pow10(17)))
	assert.Equal(t, 0, v1Out.ToBig().Cmp(pow10(16)))
}

// ---------------------------------------------------------------------------
// D. Effective reserves (MEV protection)
// ---------------------------------------------------------------------------

func TestEffectiveReserves_D1_VRInAboveReserveIn(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// Make VR0In strictly larger than reserve0 -> effIn = VR0In (token0 in).
	extra.VirtualReserve0In = new(big.Int).Mul(r, big.NewInt(2))
	// VRs frozen for the test (elapsed=0).
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	// Curve uses effIn = 2e18 instead of 1e18, so amountOut must match.
	want := solidityAmountOut(amountIn, new(big.Int).Mul(r, big.NewInt(2)), r, 0)
	assert.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(want),
		"want %s got %s", want, res.TokenAmountOut.Amount)

	// And clearly smaller than a swap done on plain reserves.
	wantPlain := solidityAmountOut(amountIn, r, r, 0)
	assert.True(t, res.TokenAmountOut.Amount.Cmp(wantPlain) < 0)
}

func TestEffectiveReserves_D2_VROutBelowReserveOut(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// Token0 in, Token1 out: effOut = min(VR1Out, reserve1). Make VR1Out half.
	extra.VirtualReserve1Out = new(big.Int).Rsh(r, 1)
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000 // freeze VRs
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	half := new(big.Int).Rsh(r, 1)
	want := solidityAmountOut(amountIn, r, half, 0)
	assert.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(want),
		"want %s got %s", want, res.TokenAmountOut.Amount)
}

func TestEffectiveReserves_D3_VREqualsReserve_NoMEVImpact(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r) // all VRs == reserves
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	want := solidityAmountOut(amountIn, r, r, 30)
	assert.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(want))
}

// ---------------------------------------------------------------------------
// E. Borrowed cap
// ---------------------------------------------------------------------------

func TestBorrowedCap_E1_AmountOutBelowAvailable_OK(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// Borrow some of token1: 10% of reserve1.
	extra.TotalBorrowed1 = new(big.Int).Div(r, big.NewInt(10))
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)
	assert.True(t, res.TokenAmountOut.Amount.Sign() > 0)
}

func TestBorrowedCap_E2_AmountOutExceedsCap_Errors(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// Borrow 99.9999...% of token1 reserve. Any positive output trips the cap.
	extra.TotalBorrowed1 = new(big.Int).Sub(r, big.NewInt(1))
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrInsufficientSwapLiquidity)
}

func TestBorrowedCap_E3_BorrowedEqualsReserve_Errors(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.TotalBorrowed1 = new(big.Int).Set(r) // borrowed == reserve
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestBorrowedCap_E3b_BorrowedAboveReserve_Errors(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.TotalBorrowed1 = new(big.Int).Add(r, big.NewInt(1))
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// ---------------------------------------------------------------------------
// F. validateSpreads
// ---------------------------------------------------------------------------

func TestSpread_F1_DisabledSentinel_NeverErrors(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// VRs absurdly far from reserves: would normally trip spreads.
	extra.VirtualReserve0In = new(big.Int).Mul(r, big.NewInt(1000))
	extra.VirtualReserve1Out = new(big.Int).Div(r, big.NewInt(1000))
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000 // freeze
	extra.SwapPriceToleranceBps = math.MaxUint16                   // disabled
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.NoError(t, err)
}

func TestSpread_F2_VRsAlignedWithReserves_NoError(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r) // VR_*=reserves
	extra.SwapPriceToleranceBps = 100
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(15)},
		TokenOut:      tokenB,
	})
	assert.NoError(t, err)
}

func TestSpread_F3_VRsFarFromReserves_TripsSpread(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	// Token0 in -> effIn = max(VR0In, r0). Inflate VR0In by 10x to force a huge
	// gap between post-update reserveIn and the spread reference.
	extra.VirtualReserve0In = new(big.Int).Mul(r, big.NewInt(10))
	extra.LastUpdateTimestamp = uint64(time.Now().Unix()) + 10_000 // freeze
	extra.SwapPriceToleranceBps = 100                              // 1%
	sim := makePool(t, tokenA, tokenB, extra)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrExcessiveSpread)
}

func TestSpread_F4_ToleranceZero_RevertsOnAnySwap(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r) // perfectly aligned reserves & VRs
	extra.SwapPriceToleranceBps = 0
	sim := makePool(t, tokenA, tokenB, extra)

	// Any non-trivial swap makes new reserves drift from the (unchanged) VRs.
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenOut:      tokenB,
	})
	assert.ErrorIs(t, err, ErrExcessiveSpread)
}

// ---------------------------------------------------------------------------
// G. CloneState + UpdateBalance
// ---------------------------------------------------------------------------

func TestCloneState_G1_IndependentReserves(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	sim := makePool(t, tokenA, tokenB, extra)

	cloned := sim.CloneState().(*PoolSimulator)

	// Mutate clone via UpdateBalance.
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenA, Amount: pow10(16)},
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: pow10(15)},
	})

	// Original must be untouched.
	assert.Equal(t, 0, sim.reserve0.ToBig().Cmp(r),
		"original reserve0 mutated: got %s", sim.reserve0.ToBig())
	assert.Equal(t, 0, sim.reserve1.ToBig().Cmp(r),
		"original reserve1 mutated: got %s", sim.reserve1.ToBig())
	// And the parent slice too.
	assert.Equal(t, 0, sim.Info.Reserves[0].Cmp(r))
	assert.Equal(t, 0, sim.Info.Reserves[1].Cmp(r))

	// Clone must have moved.
	assert.NotEqual(t, 0, cloned.reserve0.ToBig().Cmp(r))
	assert.NotEqual(t, 0, cloned.reserve1.ToBig().Cmp(r))
}

func TestUpdateBalance_G2_PostUpdateAffectsNextCalc(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: res1.TokenAmountOut.Amount},
	})

	// After the swap reserve0 grew and reserve1 shrank.
	assert.True(t, sim.reserve0.ToBig().Cmp(r) > 0)
	assert.True(t, sim.reserve1.ToBig().Cmp(r) < 0)
	// Info.Reserves mirror must follow.
	assert.Equal(t, 0, sim.Info.Reserves[0].Cmp(sim.reserve0.ToBig()))
	assert.Equal(t, 0, sim.Info.Reserves[1].Cmp(sim.reserve1.ToBig()))

	// Second swap of same amount yields strictly less because reserveIn went up
	// and reserveOut went down.
	res2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)
	assert.True(t, res2.TokenAmountOut.Amount.Cmp(res1.TokenAmountOut.Amount) < 0,
		"second swap should yield less: r1=%s r2=%s",
		res1.TokenAmountOut.Amount, res2.TokenAmountOut.Amount)
}

func TestUpdateBalance_G2b_UnknownTokenSilentlyIgnored(t *testing.T) {
	t.Parallel()
	r := pow10(18)
	sim := makePool(t, tokenA, tokenB, makeExtra(r, r))
	before0, before1 := new(big.Int).Set(sim.reserve0.ToBig()), new(big.Int).Set(sim.reserve1.ToBig())

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xdeadbeef", Amount: big.NewInt(1)},
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: big.NewInt(1)},
	})

	assert.Equal(t, 0, sim.reserve0.ToBig().Cmp(before0))
	assert.Equal(t, 0, sim.reserve1.ToBig().Cmp(before1))
}

// ---------------------------------------------------------------------------
// H. JSON round-trip via NewPoolSimulator
// ---------------------------------------------------------------------------

func TestNewPoolSimulator_H1_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	r0 := pow10(18)
	r1 := new(big.Int).Mul(pow10(18), big.NewInt(3))
	extra := Extra{
		Reserve0:              r0,
		Reserve1:              r1,
		VirtualReserve0In:     pow10(20),
		VirtualReserve0Out:    pow10(19),
		VirtualReserve1In:     pow10(17),
		VirtualReserve1Out:    pow10(16),
		LastUpdateTimestamp:   1_700_000_000,
		PriceDecay:            299,
		FeeLpBps:              30,
		FeePoolBps:            5,
		TotalBorrowed0:        pow10(10),
		TotalBorrowed1:        pow10(11),
		SwapPriceToleranceBps: 250,
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  "0xabc",
		Exchange: "unipool",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: tokenA, Swappable: true},
			{Address: tokenB, Swappable: true},
		},
		BlockNumber: 42,
		Extra:       string(extraBytes),
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	assert.Equal(t, "0xabc", sim.Info.Address)
	assert.Equal(t, "unipool", sim.Info.Exchange)
	assert.Equal(t, DexType, sim.Info.Type)
	assert.Equal(t, uint64(42), sim.Info.BlockNumber)
	require.Len(t, sim.Info.Tokens, 2)
	assert.Equal(t, tokenA, sim.Info.Tokens[0])
	assert.Equal(t, tokenB, sim.Info.Tokens[1])

	assert.Equal(t, 0, sim.reserve0.ToBig().Cmp(r0))
	assert.Equal(t, 0, sim.reserve1.ToBig().Cmp(r1))
	assert.Equal(t, 0, sim.vr0In.ToBig().Cmp(pow10(20)))
	assert.Equal(t, 0, sim.vr0Out.ToBig().Cmp(pow10(19)))
	assert.Equal(t, 0, sim.vr1In.ToBig().Cmp(pow10(17)))
	assert.Equal(t, 0, sim.vr1Out.ToBig().Cmp(pow10(16)))
	assert.Equal(t, uint64(1_700_000_000), sim.lastUpdateTimestamp)
	assert.Equal(t, uint64(299), sim.priceDecay)
	assert.Equal(t, uint64(30), sim.feeLpBps.Uint64())
	assert.Equal(t, uint64(5), sim.feePoolBps.Uint64())
	assert.Equal(t, 0, sim.totalBorrowed0.ToBig().Cmp(pow10(10)))
	assert.Equal(t, 0, sim.totalBorrowed1.ToBig().Cmp(pow10(11)))
	assert.Equal(t, uint16(250), sim.swapPriceToleranceBps)
	assert.Equal(t, int64(defaultGas), sim.gas)

	// Info.Reserves slice mirrors the typed reserves.
	require.Len(t, sim.Info.Reserves, 2)
	assert.Equal(t, 0, sim.Info.Reserves[0].Cmp(r0))
	assert.Equal(t, 0, sim.Info.Reserves[1].Cmp(r1))
}

func TestNewPoolSimulator_H1b_InvalidJSON_Errors(t *testing.T) {
	t.Parallel()
	_, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{{Address: tokenA}, {Address: tokenB}},
		Extra:  "not-json",
	})
	assert.Error(t, err)
}

func TestNewPoolSimulator_H1c_NullBigIntsTolerated(t *testing.T) {
	t.Parallel()
	// fromBig() treats nil as zero. Hand-craft a JSON blob where every big.Int
	// field is JSON `null` to verify NewPoolSimulator accepts that.
	extraJSON := `{
		"reserve0": null,
		"reserve1": null,
		"vr0In": null,
		"vr0Out": null,
		"vr1In": null,
		"vr1Out": null,
		"lastUpdateTs": 1700000000,
		"priceDecay": 299,
		"feeLpBps": 0,
		"feePoolBps": 0,
		"totalBorrowed0": null,
		"totalBorrowed1": null,
		"swapPriceToleranceBps": 65535
	}`
	sim, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{{Address: tokenA}, {Address: tokenB}},
		Type:   DexType,
		Extra:  extraJSON,
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), sim.reserve0.Uint64())
	assert.Equal(t, uint64(0), sim.reserve1.Uint64())
	assert.Equal(t, uint64(0), sim.totalBorrowed0.Uint64())
	assert.Equal(t, uint64(0), sim.totalBorrowed1.Uint64())
}

// ---------------------------------------------------------------------------
// I. Token mapping symetry
// ---------------------------------------------------------------------------

func TestTokenMapping_I1_LexicoOrder(t *testing.T) {
	t.Parallel()
	// token0 < token1 (tokenA < tokenB lexically by hex).
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)
	out01 := res.TokenAmountOut.Amount
	want01 := solidityAmountOut(amountIn, r, r, 30)
	assert.Equal(t, 0, out01.Cmp(want01))
}

func TestTokenMapping_I1b_ReverseOrder(t *testing.T) {
	t.Parallel()
	// token0 > token1 (tokenC = 0x..ff > tokenD = 0x..11).
	r := pow10(18)
	extra := makeExtra(r, r)
	extra.FeeLpBps = 30
	sim := makePool(t, tokenC, tokenD, extra)

	amountIn := pow10(16)

	// Swap "token0" (tokenC) -> "token1" (tokenD): indexIn=0, indexOut=1.
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenC, Amount: amountIn},
		TokenOut:      tokenD,
	})
	require.NoError(t, err)
	want := solidityAmountOut(amountIn, r, r, 30)
	assert.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(want),
		"out != want despite reversed lexico order")

	// And the reverse direction must produce the same number too (symmetric reserves).
	sim2 := makePool(t, tokenC, tokenD, extra)
	res2, err := sim2.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenD, Amount: amountIn},
		TokenOut:      tokenC,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(res2.TokenAmountOut.Amount))

	// Verify the simulator maps indices the way the entity ordered them
	// (no implicit sort).
	assert.Equal(t, 0, sim.GetTokenIndex(tokenC), "tokenC must remain at index 0")
	assert.Equal(t, 1, sim.GetTokenIndex(tokenD), "tokenD must remain at index 1")
}

// ---------------------------------------------------------------------------
// Misc helpers exercised by the simulator
// ---------------------------------------------------------------------------

func TestHelpers_MaxMin256(t *testing.T) {
	t.Parallel()
	a := uint256.NewInt(10)
	b := uint256.NewInt(20)
	assert.Equal(t, b, max256(a, b))
	assert.Equal(t, b, max256(b, a))
	assert.Equal(t, a, min256(a, b))
	assert.Equal(t, a, min256(b, a))
	// Equal case: max picks `a` (>=), min picks `a` (<=).
	c := uint256.NewInt(10)
	assert.Equal(t, a, max256(a, c))
	assert.Equal(t, a, min256(a, c))
}

func TestHelpers_PoolFeeNetIn(t *testing.T) {
	t.Parallel()
	// 1_000 in @ 5 bps fee -> 1_000 * (10_000-5) / 10_000 = 999 (floor).
	got := poolFeeNetIn(uint256.NewInt(1000), uint256.NewInt(5))
	assert.Equal(t, uint64(999), got.Uint64())
	// Zero pool fee is a no-op.
	got = poolFeeNetIn(uint256.NewInt(1000), uint256.NewInt(0))
	assert.Equal(t, uint64(1000), got.Uint64())
}

func TestHelpers_GetAmountOut_MatchesSolidity(t *testing.T) {
	t.Parallel()
	in := uint256.NewInt(125_224_746)
	rIn := uint256.MustFromDecimal("10089138480746")
	rOut := uint256.MustFromDecimal("10066716097576")
	totalFee := uint256.NewInt(30) // 0.3%

	got := getAmountOut(in, rIn, rOut, totalFee)
	want := solidityAmountOut(in.ToBig(), rIn.ToBig(), rOut.ToBig(), 30)
	assert.Equal(t, 0, got.ToBig().Cmp(want),
		"port mismatch: want %s got %s", want, got.ToBig())
}

// ---------------------------------------------------------------------------
// J. CalcAmountIn (exact-out routing)
// ---------------------------------------------------------------------------

// solidityAmountIn mirrors UniPoolPairSwap.getAmountIn:
//
//	amountIn = ⌈(reserveIn * amountOut * BPS) / ((reserveOut - amountOut) * (BPS - totalFee))⌉
func solidityAmountIn(amountOut, reserveIn, reserveOut *big.Int, totalFeeBps uint64) *big.Int {
	bps := big.NewInt(int64(bpsDivisor))
	num := new(big.Int).Mul(reserveIn, amountOut)
	num.Mul(num, bps)
	den := new(big.Int).Sub(reserveOut, amountOut)
	den.Mul(den, new(big.Int).Sub(bps, new(big.Int).SetUint64(totalFeeBps)))
	if den.Sign() == 0 {
		return big.NewInt(0)
	}
	// ceil division
	one := big.NewInt(1)
	tmp := new(big.Int).Sub(den, one)
	tmp.Add(tmp, num)
	return tmp.Quo(tmp, den)
}

func TestCalcAmountIn_J1_Basic(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	extra.FeeLpBps = 25
	extra.FeePoolBps = 5
	sim := makePool(t, tokenA, tokenB, extra)

	amountOut := pow10(16) // 0.01 of reserve1
	res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: amountOut},
		TokenIn:        tokenA,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Cross-check against the Solidity formula port.
	expected := solidityAmountIn(amountOut, r0, r1, 30)
	assert.Equal(t, 0, res.TokenAmountIn.Amount.Cmp(expected),
		"want %s got %s", expected, res.TokenAmountIn.Amount)
}

// J2: round-trip — feed CalcAmountOut's result back into CalcAmountIn and
// verify we recover the original amountIn (modulo the +1 wei ceil overhead).
func TestCalcAmountIn_J2_RoundTrip(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	extra.FeeLpBps = 25
	extra.FeePoolBps = 5
	sim := makePool(t, tokenA, tokenB, extra)

	amountIn := pow10(16)
	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: amountIn},
		TokenOut:      tokenB,
	})
	require.NoError(t, err)

	back, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: out.TokenAmountOut.Amount},
		TokenIn:        tokenA,
	})
	require.NoError(t, err)

	// Round-trip property: amountOut = floor(f(X)) loses fractional output, so
	// the minimum input ceil(g(amountOut)) needed to PRODUCE that floored
	// output is ≤ the original X. (If amountOut were the exact non-floored
	// value, back would equal X. With floor, back can be at most X, and at
	// least X minus a small wei budget.)
	diff := new(big.Int).Sub(amountIn, back.TokenAmountIn.Amount)
	assert.True(t, diff.Sign() >= 0,
		"back must not exceed amountIn (rounding makes it ≤): diff=%s", diff)
	assert.True(t, diff.Cmp(big.NewInt(2)) <= 0,
		"rounding slack must be tiny: diff=%s", diff)
}

func TestCalcAmountIn_J3_InvalidToken(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xdeadbeef", Amount: pow10(15)},
		TokenIn:        tokenA,
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountIn_J4_AmountOutNonPositive(t *testing.T) {
	t.Parallel()
	sim := makePool(t, tokenA, tokenB, makeExtra(pow10(18), pow10(18)))
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: big.NewInt(0)},
		TokenIn:        tokenA,
	})
	require.ErrorIs(t, err, ErrInvalidAmountOut)
}

func TestCalcAmountIn_J5_AmountOutAboveEffOut(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	sim := makePool(t, tokenA, tokenB, makeExtra(r0, r1))
	// Request the entire reserveOut: must reject (curve asymptotes here).
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: r1},
		TokenIn:        tokenA,
	})
	require.ErrorIs(t, err, ErrInsufficientSwapLiquidity)
}

func TestCalcAmountIn_J6_BorrowedCap(t *testing.T) {
	t.Parallel()
	r0, r1 := pow10(18), pow10(18)
	extra := makeExtra(r0, r1)
	// 99% of token1 is locked in loans.
	extra.TotalBorrowed1 = new(big.Int).Sub(r1, big.NewInt(100))
	sim := makePool(t, tokenA, tokenB, extra)

	// Trying to pull out 1000 wei of token1 -> only 100 wei are spendable.
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenB, Amount: big.NewInt(1000)},
		TokenIn:        tokenA,
	})
	require.ErrorIs(t, err, ErrInsufficientSwapLiquidity)
}
