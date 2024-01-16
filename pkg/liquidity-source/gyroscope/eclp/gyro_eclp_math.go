package gyroeclp

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
)

var GyroECLPMath *gyroECLPMath

var (
	ErrAssetBoundsExceeded  = errors.New("ASSET_BOUNDS_EXCEEDED")
	ErrMaxAssetsExceeded    = errors.New("MAX_ASSETS_EXCEEDED")
	ErrMaxInvariantExceeded = errors.New("MAX_INVARIANT_EXCEEDED")
)

type gyroECLPMath struct {
	ONEHALF *int256.Int
	ONE     *int256.Int
	ONE_XP  *int256.Int

	_ROTATION_VECTOR_NORM_ACCURACY    *int256.Int
	_MAX_STRETCH_FACTOR               *int256.Int
	_DERIVED_TAU_NORM_ACCURACY_XP     *int256.Int
	_MAX_INV_INVARIANT_DENOMINATOR_XP *int256.Int
	_DERIVED_DSQ_NORM_ACCURACY_XP     *int256.Int

	_MAX_BALANCES  *int256.Int
	_MAX_INVARIANT *int256.Int

	NUMBER_1E36 *int256.Int

	_number_0          *int256.Int
	_number_1          *int256.Int
	_number_2          *int256.Int
	_int256_number_3   *int256.Int
	_int256_number_7   *int256.Int
	_number_9          *int256.Int
	_number_10         *int256.Int
	_int256_number_20  *int256.Int
	_number_40         *int256.Int
	_int256_number_1e9 *int256.Int

	_uint256_number_5 *uint256.Int
}

type (
	params struct {
		Alpha  *int256.Int
		Beta   *int256.Int
		C      *int256.Int
		S      *int256.Int
		Lambda *int256.Int
	}

	vector2 struct {
		X *int256.Int
		Y *int256.Int
	}

	derivedParams struct {
		TauAlpha *vector2
		TauBeta  *vector2
		U        *int256.Int
		V        *int256.Int
		W        *int256.Int
		Z        *int256.Int
		DSq      *int256.Int
	}

	qParams struct {
		A *int256.Int
		B *int256.Int
		C *int256.Int
	}
)

type calcGiven func(*int256.Int, *params, *derivedParams, *vector2) (*int256.Int, error)

func init() {
	number_0 := int256.NewInt(0)
	number_1 := int256.NewInt(1)
	number_2 := int256.NewInt(2)
	int256_number_3 := int256.NewInt(3)
	int256_number_7 := int256.NewInt(7)
	number_9 := int256.NewInt(9)
	number_10 := int256.NewInt(10)
	int256_number_20 := int256.NewInt(20)
	number_40 := int256.NewInt(40)
	int256_number_1e9 := new(int256.Int).Pow(number_10, 9)

	uint256_number_5 := uint256.NewInt(5)

	GyroECLPMath = &gyroECLPMath{
		ONEHALF: int256.NewInt(0.5e18),
		ONE:     int256.NewInt(1e18),
		ONE_XP:  new(int256.Int).Pow(number_10, 38),

		_ROTATION_VECTOR_NORM_ACCURACY:    int256.NewInt(1e3),
		_MAX_STRETCH_FACTOR:               new(int256.Int).Pow(number_10, 26),
		_DERIVED_TAU_NORM_ACCURACY_XP:     new(int256.Int).Pow(number_10, 23),
		_MAX_INV_INVARIANT_DENOMINATOR_XP: new(int256.Int).Pow(number_10, 43),
		_DERIVED_DSQ_NORM_ACCURACY_XP:     new(int256.Int).Pow(number_10, 23),

		_MAX_BALANCES:  new(int256.Int).Pow(number_10, 34),
		_MAX_INVARIANT: new(int256.Int).Pow(number_10, 37),

		NUMBER_1E36: new(int256.Int).Pow(number_10, 36),

		_number_0:          number_0,
		_number_1:          number_1,
		_number_2:          number_2,
		_int256_number_3:   int256_number_3,
		_int256_number_7:   int256_number_7,
		_number_9:          number_9,
		_number_10:         number_10,
		_int256_number_20:  int256_number_20,
		_number_40:         number_40,
		_int256_number_1e9: int256_number_1e9,

		_uint256_number_5: uint256_number_5,
	}
}

func (g *gyroECLPMath) calcOutGivenIn(
	balances []*uint256.Int,
	amountIn *uint256.Int,
	tokenInIsToken0 bool,
	params *params,
	derived *derivedParams,
	invariant *vector2,
) (*uint256.Int, error) {
	var calcGive calcGiven
	var ixIn, ixOut int

	if tokenInIsToken0 {
		ixIn = 0
		ixOut = 1
		calcGive = g.calcYGivenX
	} else {
		ixIn = 1
		ixOut = 0
		calcGive = g.calcXGivenY
	}

	balInNewU256, err := math.GyroFixedPoint.Add(
		balances[ixIn],
		amountIn,
	)
	if err != nil {
		return nil, err
	}

	balInNew, err := math.SafeCast.ToInt256(balInNewU256)
	if err != nil {
		return nil, err
	}

	err = g.checkAssetBounds(params, derived, invariant, balInNew, ixIn)
	if err != nil {
		return nil, err
	}

	amountInI256, err := math.SafeCast.ToInt256(amountIn)
	if err != nil {
		return nil, err
	}
	balOutNew, err := calcGive(amountInI256, params, derived, invariant)
	if err != nil {
		return nil, err
	}
	balOutNewU256, err := math.SafeCast.ToUint256(balOutNew)
	if err != nil {
		return nil, err
	}

	out, err := math.GyroFixedPoint.Sub(
		balances[ixOut],
		balOutNewU256,
	)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (g *gyroECLPMath) calculateInvariantWithError(
	balances []*uint256.Int,
	params *params,
	derived *derivedParams,
) (*int256.Int, *int256.Int, error) {
	x, err := math.SafeCast.ToInt256(balances[0])
	if err != nil {
		return nil, nil, err
	}
	y, err := math.SafeCast.ToInt256(balances[1])
	if err != nil {
		return nil, nil, err
	}

	xPlusY, err := math.NewSignedFixedPointCalculator(x).
		Add(y).
		Result()
	if err != nil {
		return nil, nil, err
	}
	if xPlusY.Gt(g._MAX_BALANCES) {
		return nil, nil, ErrMaxAssetsExceeded
	}

	atAtChi, err := g.calcAtAChi(x, y, params, derived)
	if err != nil {
		return nil, nil, err
	}

	sqrt, errValue, err := g.calcInvariantSqrt(x, y, params, derived)
	if err != nil {
		return nil, nil, err
	}

	if sqrt.Gt(g._number_0) {
		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(g._number_1).
			DivUpMagU(new(int256.Int).Mul(g._number_2, sqrt)).
			Result()
		if err != nil {
			return nil, nil, err
		}

	} else {
		if errValue.Gt(g._number_0) {
			t, err := math.SafeCast.ToUint256(errValue)
			if err != nil {
				return nil, nil, err
			}

			errValueU256, err := math.GyroPoolMath.Sqrt(
				t,
				g._uint256_number_5,
			)
			if err != nil {
				return nil, nil, err
			}

			errValue, err = math.SafeCast.ToInt256(errValueU256)
			if err != nil {
				return nil, nil, err
			}

		} else {
			errValue = g._int256_number_1e9
		}
	}

	var t *int256.Int
	{
		t, err = math.NewSignedFixedPointCalculator(params.Lambda).
			MulUpMagU(xPlusY).
			Result()
		if err != nil {
			return nil, nil, err
		}
	}
	errValue = new(int256.Int).Mul(
		new(int256.Int).Add(
			new(int256.Int).Add(
				new(int256.Int).Quo(
					t, g.ONE_XP,
				),
				errValue,
			),
			g._number_1,
		),
		g._int256_number_20,
	)

	var mulDenominator *int256.Int
	{
		t, err := g.calcAChiAChiInXp(params, derived)
		if err != nil {
			return nil, nil, err
		}

		mulDenominator, err = math.NewSignedFixedPointCalculator(g.ONE_XP).
			DivXpUWith(
				math.NewSignedFixedPointCalculator(t).
					Sub(g.ONE_XP),
			).Result()
		if err != nil {
			return nil, nil, err
		}
	}

	invariant, err := math.NewSignedFixedPointCalculator(atAtChi).
		Add(sqrt).
		Sub(errValue).
		MulDownXpToNpU(mulDenominator).
		Result()
	if err != nil {
		return nil, nil, err
	}

	errValue, err = math.NewSignedFixedPointCalculator(errValue).
		MulUpXpToNpU(mulDenominator).
		Result()
	if err != nil {
		return nil, nil, err
	}

	{
		u, err := math.NewSignedFixedPointCalculator(invariant).
			MulUpXpToNpU(mulDenominator).
			Result()
		if err != nil {
			return nil, nil, err
		}

		u = new(int256.Int).Quo(
			new(int256.Int).Mul(
				new(int256.Int).Mul(
					u,
					new(int256.Int).Quo(
						new(int256.Int).Mul(params.Lambda, params.Lambda),
						g.NUMBER_1E36,
					),
				),
				g._number_40,
			),
			g.ONE_XP,
		)

		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(u).
			Add(g._number_1).
			Result()
		if err != nil {
			return nil, nil, err
		}
	}

	t, err = math.NewSignedFixedPointCalculator(invariant).
		Add(errValue).
		Result()
	if err != nil {
		return nil, nil, err
	}
	if t.Gt(g._MAX_INVARIANT) {
		return nil, nil, ErrMaxInvariantExceeded
	}

	return invariant, errValue, nil
}

func (g *gyroECLPMath) calcAChiAChiInXp(p *params, d *derivedParams) (*int256.Int, error) {
	dSq3, err := math.NewSignedFixedPointCalculator(d.DSq).
		MulXpU(d.DSq).
		MulXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(p.Lambda).
		MulUpMagUWith(
			math.NewSignedFixedPointCalculator(g._number_2).
				MulNormal(d.U).
				MulXpU(d.V).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(d.U).
				AddNormal(g._number_1).
				MulXpU(new(int256.Int).Add(d.U, g._number_1)).
				DivXpU(dSq3).
				MulUpMagU(p.Lambda).
				MulUpMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(d.V).
				MulXpU(d.V).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.W).
		DivUpMagU(p.Lambda).
		AddNormal(d.Z).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(termXp).
				MulXpU(termXp).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcAtAChi(x, y *int256.Int, p *params, d *derivedParams) (*int256.Int, error) {
	dSq2, err := math.NewSignedFixedPointCalculator(d.DSq).
		MulXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.W).
		DivDownMagU(p.Lambda).
		AddNormal(d.Z).
		DivDownMagU(p.Lambda).
		DivXpU(dSq2).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(x).
		MulDownMagU(p.C).
		SubNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(p.S),
		).
		MulDownXpToNpU(termXp). // FIXME: wrong here
		Result()
	if err != nil {
		return nil, err
	}

	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulDownMagU(p.Lambda).
		MulDownMagU(p.S).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(p.Lambda).
				MulDownMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(termNp).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(d.U).
						DivXpU(dSq2),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(x).
		MulDownMagU(p.S).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(termNp).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(d.V).
						DivXpU(dSq2),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) virtualOffset0(p *params, d *derivedParams, r *vector2) (*int256.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauBeta.X, d.DSq)
	if err != nil {
		return nil, err
	}

	a, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauBeta.X.Gt(g._number_0),

			math.NewSignedFixedPointCalculator(r.X).
				MulUpMagU(p.Lambda).MulUpMagU(p.C).
				MulUpXpToNpU(termXp),

			math.NewSignedFixedPointCalculator(r.Y).
				MulDownMagU(p.Lambda).
				MulDownMagU(p.C).
				MulUpXpToNpU(termXp),
		).Result()
	if err != nil {
		return nil, err
	}

	a, err = math.NewSignedFixedPointCalculator(a).
		AddWith(
			math.NewSignedFixedPointCalculator(r.X).
				MulUpMagU(p.S).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(d.TauBeta.Y).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (g *gyroECLPMath) virtualOffset1(p *params, d *derivedParams, r *vector2) (*int256.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauAlpha.X, d.DSq)
	if err != nil {
		return nil, err
	}

	b, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauAlpha.X.Lt(g._number_0),

			math.NewSignedFixedPointCalculator(r.X).
				MulDownMagU(p.Lambda).
				MulUpMagU(p.S).
				MulUpXpToNp(new(int256.Int).Neg(termXp)),

			math.NewSignedFixedPointCalculator(new(int256.Int).Neg(r.Y)).
				MulDownMagU(p.Lambda).
				MulDownMagU(p.S).
				MulUpXpToNpU(termXp),
		).Result()
	if err != nil {
		return nil, err
	}

	b, err = math.NewSignedFixedPointCalculator(b).
		AddWith(
			math.NewSignedFixedPointCalculator(r.X).
				MulUpMagU(p.C).
				MulUpXpToNpU(d.TauAlpha.Y).
				DivXpU(d.DSq),
		).Result()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (g *gyroECLPMath) maxBalances0(p *params, d *derivedParams, r *vector2) (*int256.Int, error) {
	termXp1, err := math.NewSignedFixedPointCalculator(d.TauBeta.X).
		Sub(d.TauAlpha.X).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp2, err := math.NewSignedFixedPointCalculator(d.TauBeta.Y).
		Sub(d.TauAlpha.Y).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	xp, err := math.NewSignedFixedPointCalculator(r.Y).
		MulDownMagU(p.Lambda).
		MulDownMagU(p.C).
		MulDownXpToNpU(termXp1).
		Result()
	if err != nil {
		return nil, err
	}

	xp, err = math.NewSignedFixedPointCalculator(xp).
		AddWith(
			math.NewSignedFixedPointCalculator(nil).
				TernaryWith(
					termXp2.Gt(g._number_0),

					math.NewSignedFixedPointCalculator(r.Y).
						MulDownMagU(p.S),

					math.NewSignedFixedPointCalculator(r.X).
						MulUpMagU(p.S).
						MulDownXpToNpU(termXp2),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return xp, nil
}

func (g *gyroECLPMath) maxBalances1(p *params, d *derivedParams, r *vector2) (*int256.Int, error) {
	termXp1, err := math.NewSignedFixedPointCalculator(d.TauBeta.X).
		Sub(d.TauAlpha.X).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp2, err := math.NewSignedFixedPointCalculator(d.TauAlpha.Y).
		Sub(d.TauBeta.Y).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	yp, err := math.NewSignedFixedPointCalculator(r.Y).
		MulDownMagU(p.Lambda).
		MulDownMagU(p.S).
		MulDownXpToNpU(termXp1).
		Result()
	if err != nil {
		return nil, err
	}

	yp, err = math.NewSignedFixedPointCalculator(yp).
		AddWith(
			math.NewSignedFixedPointCalculator(nil).
				TernaryWith(
					termXp2.Gt(g._number_0),

					math.NewSignedFixedPointCalculator(r.Y).
						MulDownMagU(p.C),

					math.NewSignedFixedPointCalculator(r.X).
						MulUpMagU(p.C).
						MulDownXpToNpU(termXp2),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return yp, nil
}

func (g *gyroECLPMath) calcMinAtxAChiySqPlusAtxSq(x, y *int256.Int, p *params, d *derivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		MulUpMagU(p.C).
		MulUpMagU(p.C).
		AddWith(
			math.NewSignedFixedPointCalculator(y).
				MulUpMagU(y).
				MulUpMagU(p.S).
				MulUpMagU(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		SubWith(
			math.NewSignedFixedPointCalculator(x).
				MulDownMag(y).
				MulDownMag(new(int256.Int).Mul(p.C, g._number_2)).
				MulDownMag(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.U).
		MulXpU(d.U).
		AddWith(
			math.NewSignedFixedPointCalculator(new(int256.Int).Mul(g._number_2, d.U)).
				MulXpU(d.V).
				DivDownMagU(p.Lambda),
		).
		AddWith(
			math.NewSignedFixedPointCalculator(d.V).
				MulXpU(d.V).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		DivXpUWith(
			math.NewSignedFixedPointCalculator(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(g._number_0).
		Sub(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termNp).
				Sub(g._number_9).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(g.ONE_XP).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calc2AtxAtyAChixAChiy(x, y *int256.Int, p *params, d *derivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		SubWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(y),
		).
		MulDownMagU(new(int256.Int).Mul(g._number_2, p.C)).
		MulDownMagU(p.S).
		Result()
	if err != nil {
		return nil, err
	}

	xy, err := math.NewSignedFixedPointCalculator(y).
		MulDownMagU(new(int256.Int).Mul(g._number_2, x)).
		Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		AddWith(
			math.NewSignedFixedPointCalculator(xy).
				MulDownMagU(p.C).
				MulDownMagU(p.C),
		).SubWith(
		math.NewSignedFixedPointCalculator(xy).
			MulDownMagU(p.S).
			MulDownMagU(p.S),
	).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.Z).
		MulXpU(d.U).
		AddWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.V).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		AddWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.U).
				AddWith(
					math.NewSignedFixedPointCalculator(d.Z).
						MulXpU(d.V),
				).DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		DivXpUWith(
			math.NewSignedFixedPointCalculator(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcMinAtyAChixSqPlusAtySq(x, y *int256.Int, p *params, d *derivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		MulUpMagU(p.S).
		MulUpMagU(p.S).
		AddWith(
			math.NewSignedFixedPointCalculator(y).
				MulUpMagU(y).
				MulUpMagU(p.C).
				MulUpMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		AddWith(
			math.NewSignedFixedPointCalculator(x).
				MulUpMagU(y).
				MulUpMagU(new(int256.Int).Mul(p.S, g._number_2)).
				MulUpMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.Z).
		MulXpU(d.Z).
		AddWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.W).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		AddWith(
			math.NewSignedFixedPointCalculator(new(int256.Int).Mul(g._number_2, d.Z)).
				MulXpU(d.W).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		DivXpUWith(
			math.NewSignedFixedPointCalculator(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq).
				MulXpU(d.DSq),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(g._number_0).
		Sub(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termNp).
				Sub(g._number_9).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(g.ONE_XP).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcInvariantSqrt(x, y *int256.Int, p *params, d *derivedParams) (*int256.Int, *int256.Int, error) {
	var val *int256.Int
	{
		l, err := g.calcMinAtxAChiySqPlusAtxSq(x, y, p, d)
		if err != nil {
			return nil, nil, err
		}

		r, err := g.calc2AtxAtyAChixAChiy(x, y, p, d)
		if err != nil {
			return nil, nil, err
		}

		val, err = math.NewSignedFixedPointCalculator(l).
			Add(r).Result()
		if err != nil {
			return nil, nil, err
		}
	}

	var (
		a   *int256.Int
		err error
	)
	{
		a, err = g.calcMinAtyAChixSqPlusAtySq(x, y, p, d)
		if err != nil {
			return nil, nil, err
		}
	}
	val, err = math.NewSignedFixedPointCalculator(val).
		Add(a).Result()
	if err != nil {
		return nil, nil, err
	}

	{
		a, err = math.NewSignedFixedPointCalculator(x).
			MulUpMagU(x).
			AddWith(
				math.NewSignedFixedPointCalculator(y).
					MulUpMagU(y),
			).Result()
		if err != nil {
			return nil, nil, err
		}
	}
	errValue := new(int256.Int).Quo(a, g.ONE_XP)

	{
		valU256, err := math.SafeCast.ToUint256(val)

		b, err := math.GyroPoolMath.Sqrt(valU256, uint256.NewInt(5))
		if err != nil {
			return nil, nil, err
		}

		a, err = math.SafeCast.ToInt256(b)
		if err != nil {
			return nil, nil, err
		}
	}

	val, err = math.NewSignedFixedPointCalculator(nil).
		Ternary(
			val.Gt(g._number_0),

			a,

			g._number_0,
		).Result()
	if err != nil {
		return nil, nil, err
	}

	return val, errValue, nil
}

func (g *gyroECLPMath) checkAssetBounds(
	params *params,
	derived *derivedParams,
	invariant *vector2,
	newBal *int256.Int,
	assetIndex int,
) error {
	if assetIndex == 0 {
		xPlus, err := g.maxBalances0(params, derived, invariant)
		if err != nil {
			return err
		}

		if newBal.Gt(g._MAX_BALANCES) || newBal.Gt(xPlus) {
			return ErrAssetBoundsExceeded
		}

		return nil
	}

	yPlus, err := g.maxBalances1(params, derived, invariant)
	if err != nil {
		return err
	}

	if newBal.Gt(g._MAX_BALANCES) || newBal.Gt(yPlus) {
		return ErrAssetBoundsExceeded
	}

	return nil
}

func (g *gyroECLPMath) calcXpXpDivLambdaLambda(
	x *int256.Int,
	r *vector2,
	lambda *int256.Int,
	s *int256.Int,
	c *int256.Int,
	tauBeta *vector2,
	dSq *int256.Int,
) (*int256.Int, error) {
	var sqVars *vector2
	{
		x, err := math.NewSignedFixedPointCalculator(dSq).
			MulXpU(dSq).
			Result()
		if err != nil {
			return nil, err
		}

		y, err := math.NewSignedFixedPointCalculator(r.X).
			MulUpMagU(r.X).
			Result()
		if err != nil {
			return nil, err
		}

		sqVars = &vector2{X: x, Y: y}
	}

	q := &qParams{}

	termXp, err := math.NewSignedFixedPointCalculator(tauBeta.X).
		MulXpU(tauBeta.Y).
		DivXpU(sqVars.X).
		Result()
	if err != nil {
		return nil, err
	}
	if termXp.Gt(g._number_0) {
		a, err := math.NewSignedFixedPointCalculator(sqVars.Y).
			MulUpMagU(new(int256.Int).Mul(g._number_2, s)).
			Result()
		if err != nil {
			return nil, err
		}

		a, err = math.NewSignedFixedPointCalculator(a).
			MulUpMagU(c).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(termXp).
					Add(g._int256_number_7),
			).Result()
		if err != nil {
			return nil, err
		}

		q.A = a
	} else {
		a, err := math.NewSignedFixedPointCalculator(r.Y).
			MulDownMagU(r.Y).
			MulDownMagU(new(int256.Int).Mul(g._number_2, s)).
			Result()
		if err != nil {
			return nil, err
		}

		a, err = math.NewSignedFixedPointCalculator(a).
			MulDownMagU(c).
			MulUpXpToNpU(termXp).
			Result()
		if err != nil {
			return nil, err
		}

		q.A = a
	}

	if tauBeta.X.Lt(g._number_0) {
		b, err := math.NewSignedFixedPointCalculator(r.X).
			MulUpMagU(x).
			MulUpMagU(new(int256.Int).Mul(g._number_2, c)).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(g._number_0).
					SubWith(
						math.NewSignedFixedPointCalculator(tauBeta.X).
							DivXpU(dSq),
					).Add(g._int256_number_3),
			).Result()
		if err != nil {
			return nil, err
		}

		q.B = b
	} else {
		b, err := math.NewSignedFixedPointCalculator(new(int256.Int).Neg(r.Y)).
			MulDownMagU(x).
			MulDownMagU(new(int256.Int).Mul(g._number_2, c)).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(tauBeta.X).
					DivXpU(dSq),
			).Result()
		if err != nil {
			return nil, err
		}

		q.B = b
	}

	termXp, err = math.NewSignedFixedPointCalculator(tauBeta.Y).
		MulXpU(tauBeta.Y).
		DivXpU(sqVars.X).
		Add(g._int256_number_7).
		Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(sqVars.Y).
		MulUpMagU(s).
		Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(q.B).
		MulUpMagU(s).
		MulUpXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	q.C, err = math.NewSignedFixedPointCalculator(new(int256.Int).Neg(r.Y)).
		MulDownMagU(x).
		MulDownMagU(new(int256.Int).Mul(g._number_2, s)).
		MulUpXpToNpUWith(
			math.NewSignedFixedPointCalculator(tauBeta.Y).
				DivXpU(dSq),
		).Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(q.B).
		Add(q.C).
		AddWith(
			math.NewSignedFixedPointCalculator(x).
				MulUpMagU(x),
		).Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			q.B.Gt(g._number_0),

			math.NewSignedFixedPointCalculator(q.B).
				DivUpMagU(lambda),

			math.NewSignedFixedPointCalculator(q.B).
				DivDownMagU(lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	q.A, err = math.NewSignedFixedPointCalculator(q.A).
		Add(q.B).
		Result()
	if err != nil {
		return nil, err
	}

	q.A, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			q.A.Gt(g._number_0),

			math.NewSignedFixedPointCalculator(q.A).
				DivUpMagU(lambda),

			math.NewSignedFixedPointCalculator(q.A).
				DivDownMagU(lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(tauBeta.X).
		MulXpU(tauBeta.X).
		DivXpU(sqVars.X).
		Add(g._int256_number_7).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(sqVars.Y).
		MulUpMagU(c).
		MulUpMagU(c).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		MulUpXpToNpU(termXp).
		Add(q.A).
		Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) solveQuadraticSwap(
	lambda *int256.Int,
	x *int256.Int,
	s *int256.Int,
	c *int256.Int,
	r *vector2,
	ab *vector2,
	tauBeta *vector2,
	dSq *int256.Int,
) (*int256.Int, error) {
	lamBar := &vector2{}
	{
		x, err := math.NewSignedFixedPointCalculator(g.ONE_XP).
			SubWith(
				math.NewSignedFixedPointCalculator(g.ONE_XP).
					DivDownMagU(lambda).
					DivDownMagU(lambda),
			).Result()
		if err != nil {
			return nil, err
		}

		lamBar.X = x
	}
	{
		y, err := math.NewSignedFixedPointCalculator(g.ONE_XP).
			SubWith(
				math.NewSignedFixedPointCalculator(g.ONE_XP).
					DivUpMagU(lambda).
					DivUpMagU(lambda),
			).Result()
		if err != nil {
			return nil, err
		}

		lamBar.Y = y
	}

	q := &qParams{}
	xp, err := math.SignedFixedPoint.Sub(x, ab.X)
	if err != nil {
		return nil, err
	}
	q.B, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			xp.Gt(g._number_0),

			math.NewSignedFixedPointCalculator(new(int256.Int).Sub(g._number_0, xp)).
				MulDownMagU(s).
				MulDownMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.Y).
						DivXpU(dSq),
				),

			math.NewSignedFixedPointCalculator(new(int256.Int).Sub(g._number_0, xp)).
				MulUpMagU(s).
				MulUpMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.X).
						DivXpU(dSq).
						Add(g._number_1),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	sTerm := &vector2{}
	sTerm.X, err = math.NewSignedFixedPointCalculator(lamBar.Y).
		MulDownMagU(s).
		MulDownMagU(s).
		DivXpU(dSq).
		Result()
	if err != nil {
		return nil, err
	}

	sTerm.Y, err = math.NewSignedFixedPointCalculator(lamBar.X).
		MulUpMagU(s).
		Result()
	if err != nil {
		return nil, err
	}

	sTerm.Y, err = math.NewSignedFixedPointCalculator(sTerm.Y).
		MulUpMagU(s).
		DivXpUWith(
			math.NewSignedFixedPointCalculator(dSq).
				Add(g._number_1),
		).Add(g._number_1).
		Result()
	if err != nil {
		return nil, err
	}

	sTerm.X, err = math.NewSignedFixedPointCalculator(g.ONE_XP).
		Sub(sTerm.X).
		Result()
	if err != nil {
		return nil, err
	}
	sTerm.Y, err = math.NewSignedFixedPointCalculator(g.ONE_XP).
		Sub(sTerm.Y).
		Result()
	if err != nil {
		return nil, err
	}

	q.C, err = g.calcXpXpDivLambdaLambda(x, r, lambda, s, c, tauBeta, dSq)
	if err != nil {
		return nil, err
	}
	q.C = new(int256.Int).Neg(q.C)

	q.C, err = math.NewSignedFixedPointCalculator(q.C).
		AddWith(
			math.NewSignedFixedPointCalculator(r.Y).
				MulDownMagU(r.Y).
				MulDownXpToNpU(sTerm.Y),
		).Result()
	if err != nil {
		return nil, err
	}

	if q.C.Gt(g._number_0) {
		qC, err := math.SafeCast.ToUint256(q.C)
		if err != nil {
			return nil, err
		}

		qC, err = math.GyroPoolMath.Sqrt(qC, uint256.NewInt(5))
		if err != nil {
			return nil, err
		}

		q.C, err = math.SafeCast.ToInt256(qC)
		if err != nil {
			return nil, err
		}
	} else {
		q.C = g._number_0
	}

	if q.B.Cmp(q.C) > 0 {
		q.A, err = math.NewSignedFixedPointCalculator(q.B).
			Sub(q.C).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(g.ONE_XP).
					DivXpU(sTerm.Y).
					Add(g._number_1),
			).Result()
	} else {
		q.A, err = math.NewSignedFixedPointCalculator(q.B).
			Sub(q.C).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(g.ONE_XP).
					DivXpU(sTerm.X),
			).Result()
	}
	if err != nil {
		return nil, err
	}

	return math.NewSignedFixedPointCalculator(q.A).
		Add(ab.Y).
		Result()
}

func (g *gyroECLPMath) calcYGivenX(
	x *int256.Int,
	params *params,
	d *derivedParams,
	r *vector2,
) (*int256.Int, error) {
	ab := &vector2{}
	{
		x, err := g.virtualOffset0(params, d, r)
		if err != nil {
			return nil, err
		}

		ab.X = x
	}
	{
		y, err := g.virtualOffset1(params, d, r)
		if err != nil {
			return nil, err
		}

		ab.Y = y
	}

	y, err := g.solveQuadraticSwap(params.Lambda, x, params.S, params.C, r, ab, d.TauBeta, d.DSq)
	if err != nil {
		return nil, err
	}

	return y, nil
}

func (g *gyroECLPMath) calcXGivenY(
	y *int256.Int,
	params *params,
	d *derivedParams,
	r *vector2,
) (*int256.Int, error) {
	ab := &vector2{}
	{
		x, err := g.virtualOffset1(params, d, r)
		if err != nil {
			return nil, err
		}

		ab.X = x
	}
	{
		y, err := g.virtualOffset0(params, d, r)
		if err != nil {
			return nil, err
		}

		ab.Y = y
	}

	x, err := g.solveQuadraticSwap(
		params.Lambda,
		y,
		params.C,
		params.S,
		r,
		ab,
		&vector2{X: new(int256.Int).Neg(d.TauAlpha.X), Y: d.TauAlpha.Y},
		d.DSq)
	if err != nil {
		return nil, err
	}

	return x, nil
}
