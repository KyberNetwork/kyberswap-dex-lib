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

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	const (
		token0 = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		token1 = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	)

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
			// swapping token1 → token0 (wrong direction for this unidirectional pool)
			name:   "wrong direction",
			maxFee: 10_000, halfAmount: 5_000_000, reserve: 10_000_000_000, scaleNum: "1", scaleDen: "1",
			tokenIn: token1, tokenOut: token0,
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
		})
	}
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
	assert.Equal(t, "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3", pm.TargetRouter)
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
