package algebrav1

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// port https://github.com/cryptoalgebra/AlgebraV1/blob/dfebf532a27803dafcbf2ba49724740bd6220505/src/core/contracts/libraries/AdaptiveFee.sol

// / @notice Calculates fee based on formula:
// / baseFee + sigmoidVolume(sigmoid1(volatility, volumePerLiquidity) + sigmoid2(volatility, volumePerLiquidity))
// / maximum value capped by baseFee + alpha1 + alpha2
func getFee(
	volatility *big.Int,
	volumePerLiquidity *big.Int,
	config *FeeConfiguration,
) uint16 {
	sumOfSigmoids := sigmoid(volatility, config.Gamma1, config.Alpha1, big.NewInt(int64(config.Beta1))) +
		sigmoid(volatility, config.Gamma2, config.Alpha2, big.NewInt(int64(config.Beta2)))

	// safe since alpha1 + alpha2 + baseFee _must_ be <= type(uint16).max
	return uint16(config.BaseFee +
		sigmoid(volumePerLiquidity, config.VolumeGamma, uint16(sumOfSigmoids), big.NewInt(int64(config.VolumeBeta))),
	)
}

// / @notice calculates α / (1 + e^( (β-x) / γ))
// / that is a sigmoid with a maximum value of α, x-shifted by β, and stretched by γ
// / @dev returns uint256 for fuzzy testing. Guaranteed that the result is not greater than alpha
func sigmoid(
	x *big.Int,
	g uint16,
	alpha uint16,
	beta *big.Int,
) (res uint16) {
	alphaBI := big.NewInt(int64(alpha))
	_6g := big.NewInt(6 * int64(g))
	g8 := new(big.Int).Exp(big.NewInt(int64(g)), big.NewInt(8), nil)
	if x.Cmp(beta) > 0 {
		x = new(big.Int).Sub(x, beta)
		if x.Cmp(_6g) >= 0 {
			return alpha // so x < 19 bits
		}
		// < 128 bits (8*16)
		ex := exp(x, g, g8) // < 155 bits
		resBI := new(big.Int).Div(
			new(big.Int).Mul(alphaBI, ex),
			new(big.Int).Add(g8, ex),
		) // in worst case: (16 + 155 bits) / 155 bits
		// so res <= alpha
		res = uint16(resBI.Int64())
	} else {
		x = new(big.Int).Sub(beta, x)
		if x.Cmp(_6g) >= 0 {
			return 0 // so x < 19 bits
		}
		// < 128 bits (8*16)
		ex := new(big.Int).Add(g8, exp(x, g, g8)) // < 156 bits
		resBI := new(big.Int).Div(
			new(big.Int).Mul(alphaBI, g8),
			ex,
		) // in worst case: (16 + 128 bits) / 156 bits
		// g8 <= ex, so res <= alpha
		res = uint16(resBI.Int64())
	}
	return
}

// / @notice calculates e^(x/g) * g^8 in a series, since (around zero):
// / e^x = 1 + x + x^2/2 + ... + x^n/n! + ...
// / e^(x/g) = 1 + x/g + x^2/(2*g^2) + ... + x^(n)/(g^n * n!) + ...
func exp(
	x *big.Int,
	_g uint16,
	gHighestDegree *big.Int,
) *big.Int {
	// calculating:
	// g**8 + x * g**7 + (x**2 * g**6) / 2 + (x**3 * g**5) / 6 + (x**4 * g**4) / 24 + (x**5 * g**3) / 120 + (x**6 * g^2) / 720 + x**7 * g / 5040 + x**8 / 40320

	// x**8 < 152 bits (19*8) and g**8 < 128 bits (8*16)
	// so each summand < 152 bits and res < 155 bits
	xLowestDegree := x
	res := gHighestDegree // g**8

	g := big.NewInt(int64(_g))

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**7
	res = new(big.Int).Add(res,
		new(big.Int).Mul(xLowestDegree, gHighestDegree),
	)

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**6
	xLowestDegree = new(big.Int).Mul(xLowestDegree, x)   // x**2
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(xLowestDegree, gHighestDegree), bignumber.Two))

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**5
	xLowestDegree = new(big.Int).Mul(xLowestDegree, x)   // x**3
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(xLowestDegree, gHighestDegree), bignumber.Six))

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**4
	xLowestDegree = new(big.Int).Mul(xLowestDegree, x)   // x**4
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(xLowestDegree, gHighestDegree), big.NewInt(24)))

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**3
	xLowestDegree = new(big.Int).Mul(xLowestDegree, x)   // x**5
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(xLowestDegree, gHighestDegree), big.NewInt(120)))

	gHighestDegree = new(big.Int).Div(gHighestDegree, g) // g**2
	xLowestDegree = new(big.Int).Mul(xLowestDegree, x)   // x**6
	res = new(big.Int).Add(res, new(big.Int).Div(new(big.Int).Mul(xLowestDegree, gHighestDegree), big.NewInt(720)))

	xLowestDegree = new(big.Int).Mul(xLowestDegree, x) // x**7
	res = new(big.Int).Add(res,
		new(big.Int).Add(
			new(big.Int).Div(
				new(big.Int).Mul(xLowestDegree, g),
				big.NewInt(5040),
			),
			new(big.Int).Div(
				new(big.Int).Mul(xLowestDegree, x),
				big.NewInt(40320),
			),
		),
	)
	return res
}
