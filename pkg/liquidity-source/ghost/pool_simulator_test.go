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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func makePool(
	maxFee, halfAmount, reserve int64,
	scaleNum, scaleDen string,
) entity.Pool {
	se := StaticExtra{
		SourceRouter:     "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F",
		TargetRouter:     "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3",
		LocalDomain:      1,
		ScaleNumerator:   scaleNum,
		ScaleDenominator: scaleDen,
	}
	ex := Extra{
		MaxFee:     big.NewInt(maxFee),
		HalfAmount: big.NewInt(halfAmount),
		Reserve:    big.NewInt(reserve),
	}

	return entity.Pool{
		Address:  "0xa9c9a8fb36ce3e5ffbac3757da7141262723541f:0xeb1b48b238e15a62e1858a601b6bffdf41163ae3",
		Exchange: "ghost",
		Type:     DexType,
		Reserves: []string{"10000000000", "10000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Symbol: "USDC", Decimals: 6, Swappable: true},
			{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Symbol: "USDT", Decimals: 6, Swappable: true},
		},
		StaticExtra: mustMarshal(se),
		Extra:       mustMarshal(ex),
	}
}

func TestCalcAmountOut_LinearRegion(t *testing.T) {
	t.Parallel()

	// maxFee=10000 (1 cent), halfAmount=5_000_000 (5 USDC)
	// inverseFee(1_000_000): principal = 1_000_000 * 10_000_000 / 10_010_000 = 999_000
	// fee = calcFee(999_000) = 999_000 * 10_000 / 10_000_000 = 999
	// amountOut = 999_000 (scale 1:1), dust = 1_000_000 - (999_000 + 999) = 1
	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(1_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(999_000), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(999), result.Fee.Amount)
	assert.Equal(t, DefaultGas, result.Gas)
}

func TestCalcAmountOut_CapRegion(t *testing.T) {
	t.Parallel()

	// maxFee=10000, halfAmount=5_000_000
	// inverseFee(20_000_000): cappedPrincipal = 20_000_000 - 10_000 = 19_990_000
	// 19_990_000 >= 2 * 5_000_000 → capped region
	// principal = 19_990_000, fee = 10_000
	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(20_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(19_990_000), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(10_000), result.Fee.Amount)
}

func TestCalcAmountOut_WithScale(t *testing.T) {
	t.Parallel()

	// scaleNumerator=2, scaleDenominator=1 means output is 2x the principal
	// amountIn=1_000_000, principal=999_000, fee=999
	// amountOut = 999_000 * 2 / 1 = 1_998_000
	p := makePool(10_000, 5_000_000, 10_000_000_000, "2", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(1_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1_998_000), result.TokenAmountOut.Amount)
}

func TestCalcAmountOut_InsufficientReserve(t *testing.T) {
	t.Parallel()

	// reserve=500_000, amountOut would be ~999_000 > 500_000
	p := makePool(10_000, 5_000_000, 500_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	_, err = testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(1_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_ZeroAmount(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	_, err = testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(0),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	assert.Error(t, err)
}

func TestCalcAmountOut_WrongDirection(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// Try swapping token1 → token0 (wrong direction for this unidirectional pool)
	_, err = testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: big.NewInt(1_000_000),
			},
			TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		})
	})

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_ZeroFee(t *testing.T) {
	t.Parallel()

	// maxFee=0, halfAmount=0 → fee=0
	p := makePool(0, 0, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(1_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1_000_000), result.TokenAmountOut.Amount)
	assert.Equal(t, 0, result.Fee.Amount.Sign())
}

func TestCalcAmountOut_ExactlyAtHalfAmount(t *testing.T) {
	t.Parallel()

	// maxFee=10000, halfAmount=5_000_000
	// inverseFee(5_000_000): principal = 5_000_000 * 10_000_000 / 10_010_000 = 4_995_004
	// fee = calcFee(4_995_004) = 4_995_004 * 10_000 / 10_000_000 = 4_995
	// dust = 5_000_000 - (4_995_004 + 4_995) = 1
	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(5_000_000),
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(4_995_004), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(4_995), result.Fee.Amount)
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: big.NewInt(1_000_000),
		},
		TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
	})
	require.NoError(t, err)

	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  poolPkg.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: big.NewInt(1_000_000)},
		TokenAmountOut: *result.TokenAmountOut,
	})

	expectedReserve := new(big.Int).Sub(big.NewInt(10_000_000), result.TokenAmountOut.Amount)
	assert.Equal(t, expectedReserve, sim.reserve.ToBig())
}

func TestCanSwapDirections(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// CanSwapTo: given output token (token1), returns valid input tokens
	from := sim.CanSwapTo("0xdac17f958d2ee523a2206206994597c13d831ec7")
	assert.Equal(t, []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, from)

	// CanSwapTo: given wrong output token (token0), returns nil
	from = sim.CanSwapTo("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	assert.Nil(t, from)

	// CanSwapFrom: given input token (token0), returns valid output tokens
	to := sim.CanSwapFrom("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	assert.Equal(t, []string{"0xdac17f958d2ee523a2206206994597c13d831ec7"}, to)

	// CanSwapFrom: given wrong input token (token1), returns nil
	to = sim.CanSwapFrom("0xdac17f958d2ee523a2206206994597c13d831ec7")
	assert.Nil(t, to)
}

func TestGetMetaInfo(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	meta := sim.GetMetaInfo("", "")
	pm, ok := meta.(PoolMeta)
	require.True(t, ok)

	assert.Equal(t, "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F", pm.SourceRouter)
	// bytes32(uint256(uint160(0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3)))
	assert.Equal(t, "0x000000000000000000000000eb1b48b238e15a62e1858a601b6bffdf41163ae3", pm.TargetRouterBytes32)
}

func TestCalculateLimit(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	limits := sim.CalculateLimit()
	require.NotNil(t, limits)
	assert.Equal(t, big.NewInt(10_000_000_000), limits["0xdac17f958d2ee523a2206206994597c13d831ec7"])
}

func TestCloneState(t *testing.T) {
	t.Parallel()

	p := makePool(10_000, 5_000_000, 10_000_000, "1", "1")
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	clone := sim.CloneState().(*PoolSimulator)
	clone.reserve = uint256.NewInt(1)
	assert.Equal(t, "10000000", sim.reserve.Dec())
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
