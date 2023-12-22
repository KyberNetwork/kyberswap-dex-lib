package gyroeclp

import "math/big"

var GyroECLPMath *gyroECLPMath

type gyroECLPMath struct {
	ONEHALF *big.Int
	ONE     *big.Int
	ONE_XP  *big.Int

	_ROTATION_VECTOR_NORM_ACCURACY    *big.Int
	_MAX_STRETCH_FACTOR               *big.Int
	_DERIVED_TAU_NORM_ACCURACY_XP     *big.Int
	_MAX_INV_INVARIANT_DENOMINATOR_XP *big.Int
	_DERIVED_DSQ_NORM_ACCURACY_XP     *big.Int

	_MAX_BALANCES  *big.Int
	_MAX_INVARIANT *big.Int
}

type (
	params struct {
		Alpha  *big.Int
		Beta   *big.Int
		C      *big.Int
		S      *big.Int
		Lambda *big.Int
	}

	vector2 struct {
		X *big.Int
		Y *big.Int
	}

	derivedParams struct {
		TauAlpha *vector2
		TauBeta  *vector2
		U        *big.Int
		V        *big.Int
		W        *big.Int
		Z        *big.Int
		DSq      *big.Int
	}

	qParams struct {
		A *big.Int
		B *big.Int
		C *big.Int
	}
)

// func (g *gyroECLPMath) scalarProd(t1 *vector2, t2 *vector2) (*big.Int, error) {

// }