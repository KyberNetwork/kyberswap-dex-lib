package integral

import (
	"github.com/holiman/uint256"
)

// / @notice Calculates fee based on formula:
// / baseFee + sigmoid1(volatility) + sigmoid2(volatility)
// / maximum value capped by baseFee + alpha1 + alpha2
func getFee(volatility *uint256.Int, config *DynamicFeeConfig) uint16 {
	// normalize for 15 sec interval
	normalizedVolatility := new(uint256.Int).Div(volatility, uFIFTEEN)

	sumOfSigmoids := new(uint256.Int).Add(
		sigmoid(normalizedVolatility, config.Gamma1, config.Alpha1, config.Beta1),
		sigmoid(normalizedVolatility, config.Gamma2, config.Alpha2, config.Beta2),
	)

	result := normalizedVolatility.Add(
		normalizedVolatility.SetUint64(uint64(config.BaseFee)),
		sumOfSigmoids,
	)

	if result.BitLen() > 15 { // should not happen
		panic("Fee calculation exceeded uint16 max value")
	}

	return uint16(result.Uint64())
}

// / @notice calculates α / (1 + e^( (β-x) / γ))
// / that is a sigmoid with a maximum value of α, x-shifted by β, and stretched by γ
// / @dev returns uint256 for fuzzy testing. Guaranteed that the result is not greater than alpha
func sigmoid(x *uint256.Int, gU16 uint16, alpha uint16, beta uint32) *uint256.Int {
	x.SubUint64(x, uint64(beta))
	g := uint64(gU16)
	g4 := g*g*g*g
	var tmp, res uint256.Int
	if x.Sign() > 0 {
		// If x >= 6*g, return alpha
		if x.CmpUint64(6*g) >= 0 {
			return tmp.SetUint64(uint64(alpha))
		}

		ex := expXg4(x, g)

		// (alpha * ex) / (g4 + ex)
		numerator := res.Mul(res.SetUint64(uint64(alpha)), ex)
		denominator := tmp.AddUint64(ex, g4)
		return res.Div(numerator, denominator)
	} else {
		x.Abs(x)

		if x.CmpUint64(6*g) >= 0 {
			return uZERO
		}

		ex := expXg4(x, g)
		ex.AddUint64(ex, g4)

		// (alpha * g4) / ex
		numerator := res.Mul(res.SetUint64(uint64(alpha)), tmp.SetUint64(g4))
		return res.Div(numerator, ex)
	}
}

func expXg4(x *uint256.Int, g uint64) *uint256.Int {
	var closestValue *uint256.Int
	var tmp uint256.Int

	gU := uint256.NewInt(g)

	// Predefined e values multiplied by 10^20
	xdg := tmp.Div(x, gU)
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

	x.Mod(x, gU)

	if x.CmpUint64(g/2) >= 0 {
		// (x - closestValue) >= 0.5, so closestValue := closestValue * e^0.5
		x.SubUint64(x, g/2)
		closestValue.Mul(closestValue, E_HALF_MULTIPLIER).Div(closestValue, E_MULTIPLIER_BIG)
	}

	// After calculating the closestValue x/g is <= 0.5, so that the series in the neighborhood of zero converges with sufficient speed
	var xLowestDegree uint256.Int
	xLowestDegree.Set(x)
	res := uint256.NewInt(g*g*g*g) // g**4

	res.Add(res, tmp.Mul(&xLowestDegree, tmp.SetUint64(g*g*g))) // g**4 + x*g**3

	xLowestDegree.Mul(&xLowestDegree, x) // x**2
	res.Add(res, tmp.Div(
		tmp.Mul(&xLowestDegree, tmp.SetUint64(g*g)),
		uTWO,
	)) // g**4 + x * g**3 + (x**2 * g**2) / 2, res < 71

	xLowestDegree.Mul(&xLowestDegree, x) // x**3
	res.Add(res, tmp.Div(
		tmp.Add(
			tmp.Mul(&xLowestDegree, tmp.SetUint64(g*4)),
			xLowestDegree.Mul(&xLowestDegree, x),
		),
		uTWENTYFOUR,
	)) // g^4 + x * g^3 + (x^2 * g^2)/2 + x^3(g*4 + x)/24, res < 73

	res.Mul(res, closestValue).Div(res, E_MULTIPLIER_BIG)

	return res
}
