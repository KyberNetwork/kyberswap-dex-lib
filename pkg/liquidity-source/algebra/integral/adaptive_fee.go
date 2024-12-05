package integral

import (
	"github.com/holiman/uint256"
)

// / @notice Calculates fee based on formula:
// / baseFee + sigmoid1(volatility) + sigmoid2(volatility)
// / maximum value capped by baseFee + alpha1 + alpha2
func getFee(volatility *uint256.Int, config FeeConfiguration) uint16 {
	// normalize for 15 sec interval
	normalizedVolatility := new(uint256.Int).Div(volatility, uFIFTEEN)

	sumOfSigmoids := new(uint256.Int).Add(
		sigmoid(normalizedVolatility, config.Gamma1, config.Alpha1, uint256.NewInt(uint64(config.Beta1))),
		sigmoid(normalizedVolatility, config.Gamma2, config.Alpha2, uint256.NewInt(uint64(config.Beta2))),
	)

	result := new(uint256.Int).Add(
		new(uint256.Int).SetUint64(uint64(config.BaseFee)),
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
func sigmoid(x *uint256.Int, g uint16, alpha uint16, beta *uint256.Int) *uint256.Int {
	// Determine if x is greater than beta
	comparison := x.Cmp(beta)

	var res *uint256.Int
	if comparison > 0 {
		// x > beta case
		x.Sub(x, beta)

		// If x >= 6*g, return alpha
		sixTimesG := new(uint256.Int).Mul(uSIX, uint256.NewInt(uint64(g)))
		if x.Cmp(sixTimesG) >= 0 {
			return new(uint256.Int).SetUint64(uint64(alpha))
		}

		g4 := new(uint256.Int).Exp(uint256.NewInt(uint64(g)), uFOUR)
		ex := expXg4(x, g, g4)

		// (alpha * ex) / (g4 + ex)
		numerator := new(uint256.Int).Mul(uint256.NewInt(uint64(alpha)), ex)
		denominator := new(uint256.Int).Add(g4, ex)
		res = new(uint256.Int).Div(numerator, denominator)
	} else {
		// x <= beta case
		x.Sub(beta, x)

		// If x >= 6*g, return 0
		sixTimesG := new(uint256.Int).Mul(uSIX, uint256.NewInt(uint64(g)))
		if x.Cmp(sixTimesG) >= 0 {
			return uZERO
		}

		g4 := new(uint256.Int).Exp(uint256.NewInt(uint64(g)), uFOUR)
		ex := new(uint256.Int).Add(g4, expXg4(x, g, g4))

		// (alpha * g4) / ex
		numerator := new(uint256.Int).Mul(uint256.NewInt(uint64(alpha)), g4)
		res = new(uint256.Int).Div(numerator, ex)
	}

	return res
}

func expXg4(x *uint256.Int, g uint16, gHighestDegree *uint256.Int) *uint256.Int {
	var closestValue *uint256.Int

	gBig := uint256.NewInt(uint64(g))

	// Predefined e values multiplied by 10^20
	xdg := new(uint256.Int).Div(x, gBig)
	switch xdg.Uint64() {
	case 0:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_0)
	case 1:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_1)
	case 2:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_2)
	case 3:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_3)
	case 4:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_4)
	default:
		closestValue = uint256.MustFromBig(CLOSEST_VALUE_DEFAULT)
	}

	x.Mod(x, gBig)

	gDiv2 := new(uint256.Int).Div(gBig, uTWO)
	if x.Cmp(gDiv2) >= 0 {
		// (x - closestValue) >= 0.5, so closestValue := closestValue * e^0.5
		x.Sub(x, gDiv2)
		closestValue.Mul(closestValue, uint256.MustFromBig(E_HALF_MULTIPLIER)).Div(closestValue, uint256.MustFromBig(E_MULTIPLIER_BIG))
	}

	// After calculating the closestValue x/g is <= 0.5, so that the series in the neighborhood of zero converges with sufficient speed
	xLowestDegree := new(uint256.Int).Set(x)
	res := new(uint256.Int).Set(gHighestDegree) // g**4

	gHighestDegree.Div(gHighestDegree, gBig)                          // g**3
	res.Add(res, new(uint256.Int).Mul(xLowestDegree, gHighestDegree)) // g**4 + x*g**3

	gHighestDegree.Div(gHighestDegree, gBig) // g**2
	xLowestDegree.Mul(xLowestDegree, x)      // x**2
	res.Add(res, new(uint256.Int).Div(
		new(uint256.Int).Mul(xLowestDegree, gHighestDegree),
		uTWO,
	))

	gHighestDegree.Div(gHighestDegree, gBig) // g
	xLowestDegree.Mul(xLowestDegree, x)      // x**3
	res.Add(res, new(uint256.Int).Div(
		new(uint256.Int).Add(
			new(uint256.Int).Mul(xLowestDegree, new(uint256.Int).Mul(gBig, uFOUR)),
			new(uint256.Int).Mul(xLowestDegree, x),
		),
		uTWENTYFOUR,
	))

	res.Mul(res, closestValue).Div(res, uint256.MustFromBig(E_MULTIPLIER_BIG))

	return res
}
