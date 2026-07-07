package twamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestCalculateNextSqrtRatio(t *testing.T) {
	t.Parallel()
	oneE18 := uint256.NewInt(1e18)
	shift32 := big256.U2Pow32
	tokenSaleRate := new(uint256.Int).Mul(oneE18, shift32)

	testCases := [...]struct {
		description    string
		sqrtRatio      *uint256.Int
		liquidity      *uint256.Int
		token0SaleRate *uint256.Int
		token1SaleRate *uint256.Int
		timeElapsed    uint32
		fee            uint64
		expected       *uint256.Int
	}{
		{
			description:    "zero_liquidity_price_eq_sale_ratio",
			sqrtRatio:      big256.U0,
			liquidity:      big256.U0,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    0,
			fee:            0,
			expected:       big256.New("340282366920938463463374607431768211456"),
		},
		{
			description:    "large_exponent_price_sqrt_ratio",
			sqrtRatio:      big256.U2Pow128,
			liquidity:      big256.U1,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(uint256.Int).Mul(uint256.NewInt(1980), tokenSaleRate),
			timeElapsed:    1,
			fee:            0,
			expected:       big256.New("15141609448466370575828005229206655991808"),
		},
		{
			description:    "low_liquidity_same_sale_ratio",
			sqrtRatio:      new(uint256.Int).Mul(big256.U2, big256.U2Pow128),
			liquidity:      big256.U1,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       big256.New("340282366920938463463374607431768211456"),
		},
		{
			description:    "low_liquidity_token0_gt_token1",
			sqrtRatio:      big256.U2Pow128,
			liquidity:      big256.U1,
			token0SaleRate: new(uint256.Int).Mul(big256.U2, tokenSaleRate),
			token1SaleRate: tokenSaleRate,
			timeElapsed:    16,
			fee:            0,
			expected:       big256.New("240615969168004511545033772477625056927"),
		},
		{
			description:    "low_liquidity_token1_gt_token0",
			sqrtRatio:      big256.U2Pow128,
			liquidity:      big256.U1,
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(uint256.Int).Mul(big256.U2, tokenSaleRate),
			timeElapsed:    16,
			fee:            0,
			expected:       big256.New("481231938336009023090067544951314448384"),
		},
		{
			description:    "high_liquidity_same_sale_rate",
			sqrtRatio:      new(uint256.Int).Mul(big256.U2, big256.U2Pow128),
			liquidity:      new(uint256.Int).Mul(uint256.NewInt(1_000_000), oneE18),
			token0SaleRate: tokenSaleRate,
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       big256.New("680563712996817890757827685335626524191"),
		},
		{
			description:    "high_liquidity_token0_gt_token1",
			sqrtRatio:      big256.U2Pow128,
			liquidity:      new(uint256.Int).Mul(uint256.NewInt(1_000_000), oneE18),
			token0SaleRate: new(uint256.Int).Mul(big256.U2, tokenSaleRate),
			token1SaleRate: tokenSaleRate,
			timeElapsed:    1,
			fee:            0,
			expected:       big256.New("340282026639252118183347287047607050305"),
		},
		{
			description:    "high_liquidity_token1_gt_token0",
			sqrtRatio:      big256.U2Pow128,
			liquidity:      new(uint256.Int).Mul(uint256.NewInt(1_000_000), oneE18),
			token0SaleRate: tokenSaleRate,
			token1SaleRate: new(uint256.Int).Mul(big256.U2, tokenSaleRate),
			timeElapsed:    1,
			fee:            0,
			expected:       big256.New("340282707202965090089453576058304747105"),
		},
		{
			description:    "round_in_direction_of_price",
			sqrtRatio:      big256.New("481231811499356508086519009265716982182"),
			liquidity:      big256.New("70710696755630728101718334"),
			token0SaleRate: big256.New("10526880627450980392156862745"),
			token1SaleRate: big256.New("10526880627450980392156862745"),
			timeElapsed:    2040,
			fee:            0,
			expected:       big256.New("481207752340104468493822013619596511452"),
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
