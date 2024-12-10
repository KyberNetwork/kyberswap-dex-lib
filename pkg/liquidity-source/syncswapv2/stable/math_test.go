package syncswapv2stable

import (
	"testing"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestUint256Refactor(t *testing.T) {
	func(A, s, dp, d *uint256.Int) {
		num := new(uint256.Int)
		den := new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Mul(A, s),
					new(uint256.Int).Mul(uint256.NewInt(2), dp),
				), d),
			new(uint256.Int).Add(
				new(uint256.Int).Mul(new(uint256.Int).Sub(A, uint256.NewInt(1)), d),
				new(uint256.Int).Mul(uint256.NewInt(3), dp),
			),
		), num.Mul(A, s).Add(
			num,
			new(uint256.Int).Mul(constant.Two, dp),
		).Mul(num, d).Div(
			num,
			den.Sub(A, constant.One).Mul(den, d).Add(
				den,
				new(uint256.Int).Mul(constant.Three, dp),
			),
		), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5))

	func(x, adjustedReserveIn, MaxFee, swapFee, tokenInPrecisionMultiplier *uint256.Int) {
		amountIn := new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					MaxFee,
					new(uint256.Int).Sub(x, adjustedReserveIn),
				),
				new(uint256.Int).Sub(MaxFee, swapFee),
			),
			uint256.NewInt(1),
		), tokenInPrecisionMultiplier),
			amountIn.Sub(x, adjustedReserveIn).Mul(amountIn, MaxFee).Div(
				amountIn,
				new(uint256.Int).Sub(MaxFee, swapFee),
			).Add(amountIn, constant.One).Div(amountIn, tokenInPrecisionMultiplier), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6))

	func(d, xp0, xp1 *uint256.Int) {
		var dp = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(d, d), xp0), d), xp1), uint256.NewInt(4)),
			dp.Set(d).Mul(dp, d).Div(dp, xp0).Mul(dp, d).Div(dp, xp1).Div(dp, constant.Four), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4))
}
