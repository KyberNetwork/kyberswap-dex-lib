package ghost

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	token0 = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	token1 = "0xdac17f958d2ee523a2206206994597c13d831ec7"
)

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// makePool builds a pool whose zeroToOne (token0->token1) and oneToZero (token1->token0)
// directions use the SAME fee curve/reserve/scale — convenient for tests that only exercise
// one direction. makeAsymmetricPool below covers the case where the two directions differ.
func makePool(maxFee, halfAmount, reserve int64, scaleNum, scaleDen string) entity.Pool {
	return makeAsymmetricPool(
		maxFee, halfAmount, reserve, scaleNum, scaleDen,
		maxFee, halfAmount, reserve, scaleNum, scaleDen,
	)
}

func makeAsymmetricPool(
	fwdMaxFee, fwdHalfAmount, fwdReserve int64, fwdScaleNum, fwdScaleDen string,
	bwdMaxFee, bwdHalfAmount, bwdReserve int64, bwdScaleNum, bwdScaleDen string,
) entity.Pool {
	se := StaticExtra{
		ZeroToOne: DirectionStatic{
			SourceRouter:     "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F",
			TargetRouter:     "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3",
			LocalDomain:      1,
			ScaleNumerator:   fwdScaleNum,
			ScaleDenominator: fwdScaleDen,
		},
		OneToZero: DirectionStatic{
			SourceRouter:     "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3",
			TargetRouter:     "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F",
			LocalDomain:      1,
			ScaleNumerator:   bwdScaleNum,
			ScaleDenominator: bwdScaleDen,
		},
	}
	ex := Extra{
		ZeroToOne: DirectionExtra{
			MaxFee:     big.NewInt(fwdMaxFee),
			HalfAmount: big.NewInt(fwdHalfAmount),
			Reserve:    big.NewInt(fwdReserve),
		},
		OneToZero: DirectionExtra{
			MaxFee:     big.NewInt(bwdMaxFee),
			HalfAmount: big.NewInt(bwdHalfAmount),
			Reserve:    big.NewInt(bwdReserve),
		},
	}

	return entity.Pool{
		Address:  "ghost_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
		Exchange: "ghost",
		Type:     DexType,
		Reserves: []string{"10000000000", "10000000000"},
		Tokens: []*entity.PoolToken{
			{Address: token0, Symbol: "USDC", Decimals: 6, Swappable: true},
			{Address: token1, Symbol: "USDT", Decimals: 6, Swappable: true},
		},
		StaticExtra: mustMarshal(se),
		Extra:       mustMarshal(ex),
	}
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		maxFee, halfAmount, reserve int64
		scaleNum, scaleDen          string
		tokenIn, tokenOut           string // default to token0 -> token1 when empty
		amountIn                    *big.Int

		wantOut *big.Int // expected output amount on success
		wantFee *big.Int // expected fee amount; nil skips the assertion
		errIs   error    // when set, expect this sentinel via errors.Is
		wantErr bool     // when true (and errIs nil), expect any error
	}{
		{
			// maxFee=10000 (1 cent), halfAmount=5_000_000 (5 USDC)
			// inverseFee(1_000_000): truncated principal = 999_000, +1 dust recovery → 999_001
			// fee = calcFee(999_001) = 999_001 * 10_000 / 10_000_000 = 999
			// amountOut = 999_001 (scale 1:1), dust = 1_000_000 - (999_001 + 999) = 0
			name:   "linear region",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(1_000_000), wantOut: big.NewInt(999_001), wantFee: big.NewInt(999),
		},
		{
			// inverseFee(20_000_000): cappedPrincipal = 20_000_000 - 10_000 = 19_990_000
			// 19_990_000 >= 2 * 5_000_000 → capped region
			// principal = 19_990_000, fee = 10_000
			name:   "cap region",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(20_000_000), wantOut: big.NewInt(19_990_000), wantFee: big.NewInt(10_000),
		},
		{
			// scaleNumerator=2, scaleDenominator=1 means output is 2x the principal
			// amountIn=1_000_000, principal=999_001 (with dust recovery), fee=999 → amountOut = 999_001 * 2 / 1 = 1_998_002
			name:   "with scale",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "2", scaleDen: "1",
			amountIn: big.NewInt(1_000_000), wantOut: big.NewInt(1_998_002), wantFee: big.NewInt(999),
		},
		{
			// maxFee=0, halfAmount=0 → fee=0, principal passes through unchanged
			name:   "zero fee",
			maxFee: 0, halfAmount: 0, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(1_000_000), wantOut: big.NewInt(1_000_000), wantFee: big.NewInt(0),
		},
		{
			// inverseFee(5_000_000): truncated principal = 4_995_004, +1 dust recovery → 4_995_005
			// fee = calcFee(4_995_005) = 4_995_005 * 10_000 / 10_000_000 = 4_995
			// dust = 5_000_000 - (4_995_005 + 4_995) = 0
			name:   "exactly at halfAmount",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(5_000_000), wantOut: big.NewInt(4_995_005), wantFee: big.NewInt(4_995),
		},
		{
			// Linear region where amountIn divides evenly (amountIn = 1001 * 1000):
			// principal = 1_001_000 * 10_000_000 / 10_010_000 = 1_000_000 exactly, no division dust.
			// The +1 recovery check fails (1_000_001 + 1_000 = 1_001_001 > amountIn), so principal
			// is NOT bumped — exercises the conditional dust-recovery path that does not fire.
			// fee = calcFee(1_000_000) = 1_000, dust = 1_001_000 - (1_000_000 + 1_000) = 0
			name:   "linear region no dust recovery",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(1_001_000), wantOut: big.NewInt(1_000_000), wantFee: big.NewInt(1_000),
		},
		{
			// reserve=500_000, amountOut would be ~999_000 > 500_000
			name:   "insufficient reserve",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 500_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(1_000_000), errIs: ErrInsufficientLiquidity,
		},
		{
			name:   "zero amount",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			amountIn: big.NewInt(0), wantErr: true,
		},
		{
			// swapping token1 → token0 (oneToZero direction), using the same symmetric curve.
			name:   "oneToZero direction",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			tokenIn: token1, tokenOut: token0,
			amountIn: big.NewInt(1_000_000), wantOut: big.NewInt(999_001), wantFee: big.NewInt(999),
		},
		{
			name:   "wrong token pair",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			tokenIn: token1, tokenOut: "0x0000000000000000000000000000000000dead",
			amountIn: big.NewInt(1_000_000), errIs: ErrInvalidToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokenIn, tokenOut := tc.tokenIn, tc.tokenOut
			if tokenIn == "" {
				tokenIn = token0
			}
			if tokenOut == "" {
				tokenOut = token1
			}

			sim, err := NewPoolSimulator(makePool(tc.maxFee, tc.halfAmount, tc.reserve, tc.scaleNum, tc.scaleDen))
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: poolPkg.TokenAmount{Token: tokenIn, Amount: tc.amountIn},
					TokenOut:      tokenOut,
				})
			})

			if tc.errIs != nil {
				assert.ErrorIs(t, err, tc.errIs)
				return
			}
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantOut, result.TokenAmountOut.Amount)
			assert.Equal(t, DefaultGas, result.Gas)
			if tc.wantFee != nil {
				assert.Zerof(t, tc.wantFee.Cmp(result.Fee.Amount), "fee: want %s got %s", tc.wantFee, result.Fee.Amount)
			}

			si, ok := result.SwapInfo.(SwapInfo)
			require.True(t, ok)
			assert.GreaterOrEqual(t, si.TotalFeeBps, int64(0))
		})
	}
}

func TestCalcAmountOut_AsymmetricDirections(t *testing.T) {
	t.Parallel()

	// ZeroToOne (token0->token1) and oneToZero (token1->token0) use deliberately different fee
	// curves, mirroring that each direction is an independent on-chain call with its own
	// market-maker-set curve.
	p := makeAsymmetricPool(
		10_000, 5_000_000, 10_000_000_000, "1", "1", // zeroToOne: 0.1%-ish
		20_000, 5_000_000, 10_000_000_000, "1", "1", // oneToZero: double the fee
	)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	zeroToOne, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(1_000_000)},
		TokenOut:      token1,
	})
	require.NoError(t, err)

	oneToZero, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: token1, Amount: big.NewInt(1_000_000)},
		TokenOut:      token0,
	})
	require.NoError(t, err)

	assert.True(t, oneToZero.Fee.Amount.Cmp(zeroToOne.Fee.Amount) > 0,
		"oneToZero fee %s should exceed zeroToOne fee %s given its higher maxFee",
		oneToZero.Fee.Amount, zeroToOne.Fee.Amount)
}

func TestTotalFeeBps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		principal, fee  uint64
		wantTotalFeeBps int64
	}{
		{"linear region", 999_001, 999, 1000},
		{"cap region", 19_990_000, 10_000, 501},
		{"zero fee", 1_000_000, 0, 0},
		{"zero principal", 0, 100, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := totalFeeBps(uint256.NewInt(tc.principal), uint256.NewInt(tc.fee))
			assert.Equal(t, tc.wantTotalFeeBps, got)

			if tc.principal == 0 {
				return
			}

			// Verify the on-chain recovery formula (mirroring executeGhost) never recovers more
			// than the quoted principal — otherwise transferRemoteTo would pull more than
			// amountIn and revert.
			amountIn := new(big.Int).Add(new(big.Int).SetUint64(tc.principal), new(big.Int).SetUint64(tc.fee))
			denom := big.NewInt(GhostFeeDenominator + got)
			num := new(big.Int).Add(amountIn, big.NewInt(1))
			num.Mul(num, big.NewInt(GhostFeeDenominator))
			recovered := new(big.Int).Div(num, denom)
			if new(big.Int).Mod(num, denom).Sign() != 0 {
				recovered.Add(recovered, big.NewInt(1))
			}
			recovered.Sub(recovered, big.NewInt(1))

			assert.True(t, recovered.Cmp(new(big.Int).SetUint64(tc.principal)) <= 0,
				"recovered principal %s must be <= quoted principal %d", recovered, tc.principal)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(1_000_000)},
		TokenOut:      token1,
	})
	require.NoError(t, err)

	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: *result.TokenAmountOut,
	})

	expectedReserve := new(big.Int).Sub(big.NewInt(10_000_000), result.TokenAmountOut.Amount)
	assert.Equal(t, expectedReserve, sim.zeroToOne.reserve.ToBig())
	// OneToZero direction's reserve is untouched by a zeroToOne swap.
	assert.Equal(t, "10000000", sim.oneToZero.reserve.Dec())
}

func TestCanSwapDirections(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// Both directions are swappable: CanSwapTo(token1) means token0 can swap to it, and
	// CanSwapTo(token0) means token1 can swap to it.
	from := sim.CanSwapTo(token1)
	assert.Equal(t, []string{token0}, from)

	from = sim.CanSwapTo(token0)
	assert.Equal(t, []string{token1}, from)

	to := sim.CanSwapFrom(token0)
	assert.Equal(t, []string{token1}, to)

	to = sim.CanSwapFrom(token1)
	assert.Equal(t, []string{token0}, to)
}

func TestGetMetaInfo(t *testing.T) {
	t.Parallel()

	p := makeAsymmetricPool(
		10_000, 5_000_000, 10_000_000_000, "1", "1",
		20_000, 5_000_000, 10_000_000_000, "1", "1",
	)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	zeroToOneMeta, ok := sim.GetMetaInfo(token0, token1).(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F", zeroToOneMeta.SourceRouter)
	assert.Equal(t, "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3", zeroToOneMeta.TargetRouter)

	oneToZeroMeta, ok := sim.GetMetaInfo(token1, token0).(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3", oneToZeroMeta.SourceRouter)
	assert.Equal(t, "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F", oneToZeroMeta.TargetRouter)
}

func TestCalculateLimit(t *testing.T) {
	t.Parallel()

	p := makeAsymmetricPool(
		10_000, 5_000_000, 10_000_000_000, "1", "1",
		20_000, 5_000_000, 20_000_000_000, "1", "1",
	)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	limits := sim.CalculateLimit()
	require.NotNil(t, limits)
	assert.Equal(t, big.NewInt(10_000_000_000), limits[token1]) // zeroToOne reserve
	assert.Equal(t, big.NewInt(20_000_000_000), limits[token0]) // oneToZero reserve
}

func TestCalcAmountOut_SharedSwapLimit(t *testing.T) {
	t.Parallel()

	// Pool-local reserve is generous, but a shared inventory limit (as if another ghost pool
	// already drew down the same targetRouter/token vault) caps token1 much lower.
	sim, err := NewPoolSimulator(makePool(10_000, 5_000_000, 10_000_000_000, "1", "1"))
	require.NoError(t, err)

	limit := swaplimit.NewInventory(DexType, map[string]*big.Int{token1: big.NewInt(500_000)})

	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(1_000_000)},
		TokenOut:      token1,
		Limit:         limit,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
	assert.Nil(t, result)

	// Under the shared limit, the swap succeeds and depletes the shared inventory, not just
	// the pool-local reserve.
	result, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(300_000)},
		TokenOut:      token1,
		Limit:         limit,
	})
	require.NoError(t, err)

	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(300_000)},
		TokenAmountOut: *result.TokenAmountOut,
		SwapLimit:      limit,
	})

	assert.Equal(t, new(big.Int).Sub(big.NewInt(500_000), result.TokenAmountOut.Amount), limit.GetLimit(token1))
}

func TestCloneState(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	clone := sim.CloneState().(*PoolSimulator)
	clone.zeroToOne.reserve = uint256.NewInt(1)
	assert.Equal(t, "10000000", sim.zeroToOne.reserve.Dec())
}

func TestCloneState_InfoReservesIndependent(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	clone := sim.CloneState().(*PoolSimulator)
	clone.Info.Reserves[0] = big.NewInt(1)
	assert.Equal(t, "10000000000", sim.Info.Reserves[0].String())
}

func TestCalcFee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		amount     uint64
		maxFee     uint64
		halfAmount uint64
		expected   uint64
	}{
		{"below cap", 1_000_000, 10_000, 5_000_000, 1000},
		{"at cap boundary", 10_000_000, 10_000, 5_000_000, 10_000},
		{"above cap", 20_000_000, 10_000, 5_000_000, 10_000},
		{"at halfAmount", 5_000_000, 10_000, 5_000_000, 5_000},
		{"small amount", 100, 10_000, 5_000_000, 0},
		{"zero maxFee", 1_000_000, 0, 5_000_000, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fee := calcFee(uint256.NewInt(tc.amount), uint256.NewInt(tc.maxFee), uint256.NewInt(tc.halfAmount))
			assert.Equal(t, uint256.NewInt(tc.expected), fee)
		})
	}
}
