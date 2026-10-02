package integral

import (
	"math"

	"github.com/holiman/uint256"
)

// @notice Calculates fee based on formula:
// baseFee + sigmoid1(volatility) + sigmoid2(volatility)
// maximum value capped by baseFee + alpha1 + alpha2
func getFee(volatility *uint256.Int, config *DynamicFeeConfig) uint16 {
	// normalize for 15 sec interval
	var normalizedVolatility uint256.Int
	normalizedVolatility.Div(volatility, uFIFTEEN)

	fee := uint64(config.BaseFee) +
		sigmoid(normalizedVolatility, config.Gamma1, config.Alpha1, config.Beta1) +
		sigmoid(normalizedVolatility, config.Gamma2, config.Alpha2, config.Beta2)
	if fee > math.MaxUint16 { // should not happen
		panic("Fee calculation exceeded uint16 max value")
	}

	return uint16(fee)
}

// @notice calculates α / (1 + e^( (β-x) / γ))
// that is a sigmoid with a maximum value of α, x-shifted by β, and stretched by γ
// @dev x is taken by value as it is modified. Guaranteed that the result is not greater than alpha
func sigmoid(x uint256.Int, gU16 uint16, alpha uint16, beta uint32) uint64 {
	g := uint64(gU16)
	g4 := g * g * g * g
	var numerator, tmp uint256.Int
	if x.CmpUint64(uint64(beta)) > 0 {
		x.SubUint64(&x, uint64(beta))
		// If x >= 6*g, return alpha
		if x.CmpUint64(6*g) >= 0 {
			return uint64(alpha)
		}

		ex := expXg4(&x, g)

		// (alpha * ex) / (g4 + ex)
		numerator.Mul(numerator.SetUint64(uint64(alpha)), &ex)
		return numerator.Div(&numerator, tmp.AddUint64(&ex, g4)).Uint64()
	}

	x.Sub(tmp.SetUint64(uint64(beta)), &x)
	if x.CmpUint64(6*g) >= 0 {
		return 0
	}

	ex := expXg4(&x, g)
	ex.AddUint64(&ex, g4)

	// (alpha * g4) / ex
	numerator.Mul(numerator.SetUint64(uint64(alpha)), tmp.SetUint64(g4))
	return numerator.Div(&numerator, &ex).Uint64()
}

// expXg4 calculates e^(x/g) * g^4 in a series. x is modified.
func expXg4(x *uint256.Int, g uint64) uint256.Int {
	var tmp uint256.Int

	gU := uint256.NewInt(g)

	// Predefined e values multiplied by 10^20
	var closestValue *uint256.Int
	switch tmp.Div(x, gU).Uint64() {
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
		var halved uint256.Int // never write to the shared CLOSEST_VALUE_* constants
		closestValue = halved.Mul(closestValue, E_HALF_MULTIPLIER)
		closestValue.Div(closestValue, E_MULTIPLIER_BIG)
	}

	// After calculating the closestValue x/g is <= 0.5, so that the series in the neighborhood of zero converges with sufficient speed
	xLowestDegree := *x
	var res uint256.Int
	res.SetUint64(g * g * g * g) // g**4

	res.Add(&res, tmp.Mul(&xLowestDegree, tmp.SetUint64(g*g*g))) // g**4 + x*g**3

	xLowestDegree.Mul(&xLowestDegree, x) // x**2
	res.Add(&res, tmp.Div(
		tmp.Mul(&xLowestDegree, tmp.SetUint64(g*g)),
		uTWO,
	)) // g**4 + x * g**3 + (x**2 * g**2) / 2, res < 71

	xLowestDegree.Mul(&xLowestDegree, x) // x**3
	res.Add(&res, tmp.Div(
		tmp.Add(
			tmp.Mul(&xLowestDegree, tmp.SetUint64(g*4)),
			xLowestDegree.Mul(&xLowestDegree, x),
		),
		uTWENTYFOUR,
	)) // g^4 + x * g^3 + (x^2 * g^2)/2 + x^3(g*4 + x)/24, res < 73

	res.Mul(&res, closestValue).Div(&res, E_MULTIPLIER_BIG)

	return res
}
