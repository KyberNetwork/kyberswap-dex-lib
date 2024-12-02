package integral

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// / @notice Calculates fee based on formula:
// / baseFee + sigmoid1(volatility) + sigmoid2(volatility)
// / maximum value capped by baseFee + alpha1 + alpha2
func getFee(volatility *big.Int, config FeeConfiguration) uint16 {
	// normalize for 15 sec interval
	normalizedVolatility := new(big.Int).Div(volatility, FIFTEEN)

	sumOfSigmoids := new(big.Int).Add(
		sigmoid(normalizedVolatility, config.Gamma1, config.Alpha1, big.NewInt(int64(config.Beta1))),
		sigmoid(normalizedVolatility, config.Gamma2, config.Alpha2, big.NewInt(int64(config.Beta2))),
	)

	result := new(big.Int).Add(
		new(big.Int).SetUint64(uint64(config.BaseFee)),
		sumOfSigmoids,
	)

	if result.Cmp(MAX_UINT16) > 0 { // should always be true
		panic("Fee calculation exceeded uint16 max value")
	}

	return uint16(result.Uint64())
}

// / @notice calculates α / (1 + e^( (β-x) / γ))
// / that is a sigmoid with a maximum value of α, x-shifted by β, and stretched by γ
// / @dev returns uint256 for fuzzy testing. Guaranteed that the result is not greater than alpha
func sigmoid(x *big.Int, g uint16, alpha uint16, beta *big.Int) *big.Int {
	// Determine if x is greater than beta
	comparison := x.Cmp(beta)

	var res *big.Int
	if comparison > 0 {
		// x > beta case
		x.Sub(x, beta)

		// If x >= 6*g, return alpha
		sixTimesG := new(big.Int).Mul(bignumber.Six, big.NewInt(int64(g)))
		if x.Cmp(sixTimesG) >= 0 {
			return new(big.Int).SetUint64(uint64(alpha))
		}

		g4 := new(big.Int).Exp(big.NewInt(int64(g)), big.NewInt(4), nil)
		ex := expXg4(x, g, g4)

		// (alpha * ex) / (g4 + ex)
		numerator := new(big.Int).Mul(big.NewInt(int64(alpha)), ex)
		denominator := new(big.Int).Add(g4, ex)
		res = new(big.Int).Div(numerator, denominator)
	} else {
		// x <= beta case
		x.Sub(beta, x)

		// If x >= 6*g, return 0
		sixTimesG := new(big.Int).Mul(bignumber.Six, big.NewInt(int64(g)))
		if x.Cmp(sixTimesG) >= 0 {
			return bignumber.ZeroBI
		}

		g4 := new(big.Int).Exp(big.NewInt(int64(g)), big.NewInt(4), nil)
		ex := new(big.Int).Add(g4, expXg4(x, g, g4))

		// (alpha * g4) / ex
		numerator := new(big.Int).Mul(big.NewInt(int64(alpha)), g4)
		res = new(big.Int).Div(numerator, ex)
	}

	return res
}

func expXg4(x *big.Int, g uint16, gHighestDegree *big.Int) *big.Int {
	var closestValue *big.Int

	gBig := big.NewInt(int64(g))

	// Predefined e values multiplied by 10^20
	xdg := new(big.Int).Div(x, gBig)
	switch xdg.Uint64() {
	case 0:
		closestValue = CLOSEST_VALUE_0
	case 1:
		closestValue = CLOSEST_VALUE_1
	case 2:
		closestValue = CLOSEST_VALUE_2
	case 3:
		closestValue = CLOSEST_VALUE_3
	case 4:
		closestValue = CLOSEST_VALUE_4
	default:
		closestValue = CLOSEST_VALUE_DEFAULT
	}

	x.Mod(x, gBig)

	gDiv2 := new(big.Int).Div(gBig, bignumber.Two)
	if x.Cmp(gDiv2) >= 0 {
		// (x - closestValue) >= 0.5, so closestValue := closestValue * e^0.5
		x.Sub(x, gDiv2)
		closestValue.Mul(closestValue, E_HALF_MULTIPLIER).Div(closestValue, E_MULTIPLIER_BIG)
	}

	// After calculating the closestValue x/g is <= 0.5, so that the series in the neighborhood of zero converges with sufficient speed
	xLowestDegree := new(big.Int).Set(x)
	res := new(big.Int).Set(gHighestDegree) // g**4

	gHighestDegree.Div(gHighestDegree, gBig)                      // g**3
	res.Add(res, new(big.Int).Mul(xLowestDegree, gHighestDegree)) // g**4 + x*g**3

	gHighestDegree.Div(gHighestDegree, gBig) // g**2
	xLowestDegree.Mul(xLowestDegree, x)      // x**2
	res.Add(res, new(big.Int).Div(
		new(big.Int).Mul(xLowestDegree, gHighestDegree),
		bignumber.Two,
	))

	gHighestDegree.Div(gHighestDegree, gBig) // g
	xLowestDegree.Mul(xLowestDegree, x)      // x**3
	res.Add(res, new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(xLowestDegree, new(big.Int).Mul(gBig, bignumber.Four)),
			new(big.Int).Mul(xLowestDegree, x),
		),
		TWENTYFOUR,
	))

	res.Mul(res, closestValue).Div(res, E_MULTIPLIER_BIG)

	return res
}
