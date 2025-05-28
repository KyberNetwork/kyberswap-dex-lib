package twamm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalculateNextSqrtRatio(t *testing.T) {
	t.Parallel()
	oneE18 := bignum.NewBig("1_000_000_000_000_000_000") // 10^18
	shift32 := math.TwoPow32                             // 2^32 = 4294967296
	tokenSaleRate := new(big.Int).Mul(oneE18, shift32)   // 10^18 * 2^32

	testCases := [...]struct {
		description    string
		sqrtRatio      *big.Int
		liquidity      *big.Int
		token0SaleRate *big.Int
		token1SaleRate *big.Int
		timeElapsed    uint32
		fee            uint64
		expected       *big.Int
	}{
		{
			description:    "zero_liquidity_price_eq_sale_ratio",
			sqrtRatio:      bignum.ZeroBI,
			liquidity:      bignum.ZeroBI,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    0,
			fee:            0,
			expected:       bignum.NewBig("340282366920938463463374607431768211456"),
		},
		{
			description:    "large_exponent_price_sqrt_ratio",
			sqrtRatio:      math.TwoPow128,
			liquidity:      bignum.One,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(big.Int).Mul(big.NewInt(1980), tokenSaleRate),
			timeElapsed:    1,
			fee:            0,
			expected:       bignum.NewBig("15141609448466370575828005229206655991808"),
		},
		{
			description:    "low_liquidity_same_sale_ratio",
			sqrtRatio:      new(big.Int).Mul(bignum.Two, math.TwoPow128),
			liquidity:      bignum.One,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       bignum.NewBig("340282366920938463463374607431768211456"),
		},
		{
			description:    "low_liquidity_token0_gt_token1",
			sqrtRatio:      math.TwoPow128,
			liquidity:      bignum.One,
			token0SaleRate: new(big.Int).Mul(bignum.Two, tokenSaleRate),
			token1SaleRate: tokenSaleRate,
			timeElapsed:    16,
			fee:            0,
			expected:       bignum.NewBig("240615969168004511545033772477625056927"),
		},
		{
			description:    "low_liquidity_token1_gt_token0",
			sqrtRatio:      math.TwoPow128,
			liquidity:      bignum.One,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(big.Int).Mul(bignum.Two, tokenSaleRate),
			timeElapsed:    16,
			fee:            0,
			expected:       bignum.NewBig("481231938336009023090067544951314448384"),
		},
		{
			description:    "high_liquidity_same_sale_rate",
			sqrtRatio:      new(big.Int).Mul(bignum.Two, math.TwoPow128),
			liquidity:      new(big.Int).Mul(big.NewInt(1_000_000), oneE18),
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       bignum.NewBig("680563712996817890757827685335626524191"),
		},
		{
			description:    "high_liquidity_token0_gt_token1",
			sqrtRatio:      math.TwoPow128,
			liquidity:      new(big.Int).Mul(big.NewInt(1_000_000), oneE18),
			token0SaleRate: new(big.Int).Mul(bignum.Two, tokenSaleRate),
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       bignum.NewBig("340282026639252118183347287047607050305"),
		},
		{
			description:    "high_liquidity_token1_gt_token0",
			sqrtRatio:      math.TwoPow128,
			liquidity:      new(big.Int).Mul(big.NewInt(1_000_000), oneE18),
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(big.Int).Mul(bignum.Two, tokenSaleRate),
			timeElapsed:    1,
			fee:            0,
			expected:       bignum.NewBig("340282707202965090089453576058304747105"),
		},
		{
			description:    "round_in_direction_of_price",
			sqrtRatio:      bignum.NewBig("481231811499356508086519009265716982182"),
			liquidity:      bignum.NewBig("70710696755630728101718334"),
			token0SaleRate: bignum.NewBig("10526880627450980392156862745"),
			token1SaleRate: bignum.NewBig("10526880627450980392156862745"),
			timeElapsed:    2040,
			fee:            0,
			expected:       bignum.NewBig("481207752340104468493822013619596511452"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			require.Equal(t, testCase.expected, CalculateNextSqrtRatio(
				testCase.sqrtRatio,
				testCase.liquidity,
				testCase.token0SaleRate,
				testCase.token1SaleRate,
				testCase.timeElapsed,
				testCase.fee,
			))
		})
	}
}
