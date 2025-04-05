package math

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

type gyroECLPMath struct{}

type (
	ECLParams struct {
		Alpha  *int256.Int
		Beta   *int256.Int
		C      *int256.Int
		S      *int256.Int
		Lambda *int256.Int
	}

	Vector2 struct {
		X *int256.Int
		Y *int256.Int
	}

	ECLDerivedParams struct {
		TauAlpha *Vector2
		TauBeta  *Vector2
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

type calcGiven func(*int256.Int, *ECLParams, *ECLDerivedParams, *Vector2) (*int256.Int, error)

func (g *gyroECLPMath) CalcOutGivenIn(
	balances []*uint256.Int,
	amountIn *uint256.Int,
	tokenInIsToken0 bool,
	params *ECLParams,
	derived *ECLDerivedParams,
	invariant *Vector2,
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

	balOutNewI256, err := calcGive(balInNew, params, derived, invariant)
	if err != nil {
		return nil, err
	}
	balOutNew, err := math.SafeCast.ToUint256(balOutNewI256)
	if err != nil {
		return nil, err
	}

	out, err := math.GyroFixedPoint.Sub(
		balances[ixOut],
		balOutNew,
	)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (g *gyroECLPMath) CalculateInvariantWithError(
	balances []*uint256.Int,
	params *ECLParams,
	derived *ECLDerivedParams,
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
	if xPlusY.Gt(IMaxBalances) {
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

	if sqrt.IsPositive() {
		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(i1).
			DivUpMagU(new(int256.Int).Mul(I2, sqrt)).
			Result()
		if err != nil {
			return nil, nil, err
		}

	} else {
		if errValue.IsPositive() {
			t, err := math.SafeCast.ToUint256(errValue)
			if err != nil {
				return nil, nil, err
			}

			errValueU256, err := math.GyroPoolMath.Sqrt(
				t,
				U5,
			)
			if err != nil {
				return nil, nil, err
			}

			errValue, err = math.SafeCast.ToInt256(errValueU256)
			if err != nil {
				return nil, nil, err
			}

		} else {
			errValue = i1e9
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
					t, i1e38,
				),
				errValue,
			),
			i1,
		),
		i20,
	)

	var mulDenominator *int256.Int
	{
		t, err := g.calcAChiAChiInXp(params, derived)
		if err != nil {
			return nil, nil, err
		}

		mulDenominator, err = math.NewSignedFixedPointCalculator(i1e38).
			DivXpUWith(
				math.NewSignedFixedPointCalculator(t).
					Sub(i1e38),
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
						i1e36,
					),
				),
				i40,
			),
			i1e38,
		)

		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(u).
			Add(i1).
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
	if t.Gt(IMaxInvariant) {
		return nil, nil, ErrMaxInvariantExceeded
	}

	return invariant, errValue, nil
}

func (g *gyroECLPMath) calcAChiAChiInXp(p *ECLParams, d *ECLDerivedParams) (*int256.Int, error) {
	dSq3, err := math.NewSignedFixedPointCalculator(d.DSq).
		MulXpU(d.DSq).
		MulXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(p.Lambda).
		MulUpMagUWith(
			math.NewSignedFixedPointCalculator(I2).
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
				AddNormal(i1).
				MulXpU(new(int256.Int).Add(d.U, i1)).
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

func (g *gyroECLPMath) calcAtAChi(x, y *int256.Int, p *ECLParams, d *ECLDerivedParams) (*int256.Int, error) {
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
		MulDownXpToNpU(termXp).
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

func (g *gyroECLPMath) virtualOffset0(p *ECLParams, d *ECLDerivedParams, r *Vector2) (*int256.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauBeta.X, d.DSq)
	if err != nil {
		return nil, err
	}

	a, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauBeta.X.IsPositive(),

			math.NewSignedFixedPointCalculator(r.X).
				MulUpMagU(p.Lambda).
				MulUpMagU(p.C).
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
		AddNormalWith(
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

func (g *gyroECLPMath) virtualOffset1(p *ECLParams, d *ECLDerivedParams, r *Vector2) (*int256.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauAlpha.X, d.DSq)
	if err != nil {
		return nil, err
	}

	b, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauAlpha.X.IsNegative(),

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
		AddNormalWith(
			math.NewSignedFixedPointCalculator(r.X).
				MulUpMagU(p.C).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(d.TauAlpha.Y).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (g *gyroECLPMath) maxBalances0(p *ECLParams, d *ECLDerivedParams, r *Vector2) (*int256.Int, error) {
	termXp1, err := math.NewSignedFixedPointCalculator(d.TauBeta.X).
		SubNormal(d.TauAlpha.X).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp2, err := math.NewSignedFixedPointCalculator(d.TauBeta.Y).
		SubNormal(d.TauAlpha.Y).
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
		AddNormalWith(
			math.NewSignedFixedPointCalculator(nil).
				TernaryWith(
					termXp2.IsPositive(),

					math.NewSignedFixedPointCalculator(r.Y).
						MulDownMagU(p.S),

					math.NewSignedFixedPointCalculator(r.X).
						MulUpMagU(p.S),
				).
				MulDownXpToNpU(termXp2),
		).Result()
	if err != nil {
		return nil, err
	}

	return xp, nil
}

func (g *gyroECLPMath) maxBalances1(p *ECLParams, d *ECLDerivedParams, r *Vector2) (*int256.Int, error) {
	termXp1, err := math.NewSignedFixedPointCalculator(d.TauBeta.X).
		SubNormal(d.TauAlpha.X).
		DivXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp2, err := math.NewSignedFixedPointCalculator(d.TauAlpha.Y).
		SubNormal(d.TauBeta.Y).
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
		AddNormalWith(
			math.NewSignedFixedPointCalculator(nil).
				TernaryWith(
					termXp2.IsPositive(),

					math.NewSignedFixedPointCalculator(r.Y).
						MulDownMagU(p.C),

					math.NewSignedFixedPointCalculator(r.X).
						MulUpMagU(p.C),
				).
				MulDownXpToNpU(termXp2),
		).Result()
	if err != nil {
		return nil, err
	}

	return yp, nil
}

func (g *gyroECLPMath) calcMinAtxAChiySqPlusAtxSq(x, y *int256.Int, p *ECLParams, d *ECLDerivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		MulUpMagU(p.C).
		MulUpMagU(p.C).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulUpMagU(y).
				MulUpMagU(p.S).
				MulUpMagU(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		SubNormalWith(
			math.NewSignedFixedPointCalculator(x).
				MulDownMagU(y).
				MulDownMagU(new(int256.Int).Mul(p.C, I2)).
				MulDownMagU(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.U).
		MulXpU(d.U).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(new(int256.Int).Mul(I2, d.U)).
				MulXpU(d.V).
				DivDownMagU(p.Lambda),
		).
		AddNormalWith(
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

	val, err := math.NewSignedFixedPointCalculator(i0).
		Sub(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(termNp).
				Sub(i9).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(i1e38).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calc2AtxAtyAChixAChiy(x, y *int256.Int, p *ECLParams, d *ECLDerivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulDownMagU(x).
		SubNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulUpMagU(y),
		).
		MulDownMagU(new(int256.Int).Mul(I2, p.C)).
		MulDownMagU(p.S).
		Result()
	if err != nil {
		return nil, err
	}

	xy, err := math.NewSignedFixedPointCalculator(y).
		MulDownMagU(new(int256.Int).Mul(I2, x)).
		Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(xy).
				MulDownMagU(p.C).
				MulDownMagU(p.C),
		).
		SubNormalWith(
			math.NewSignedFixedPointCalculator(xy).
				MulDownMagU(p.S).
				MulDownMagU(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.Z).
		MulXpU(d.U).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.V).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.U).
				AddNormalWith(
					math.NewSignedFixedPointCalculator(d.Z).
						MulXpU(d.V),
				).
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

	val, err := math.NewSignedFixedPointCalculator(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcMinAtyAChixSqPlusAtySq(x, y *int256.Int, p *ECLParams, d *ECLDerivedParams) (*int256.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		MulUpMagU(p.S).
		MulUpMagU(p.S).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(y).
				MulUpMagU(y).
				MulUpMagU(p.C).
				MulUpMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(termNp).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(x).
				MulUpMagU(y).
				MulUpMagU(new(int256.Int).Mul(p.S, I2)).
				MulUpMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.Z).
		MulXpU(d.Z).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(d.W).
				MulXpU(d.W).
				DivDownMagU(p.Lambda).
				DivDownMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err = math.NewSignedFixedPointCalculator(termXp).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(new(int256.Int).Mul(I2, d.Z)).
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

	val, err := math.NewSignedFixedPointCalculator(i0).
		SubNormal(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(termNp).
				SubNormal(i9).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(i1e38).
						DivXpU(d.DSq),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcInvariantSqrt(x, y *int256.Int, p *ECLParams, d *ECLDerivedParams) (*int256.Int, *int256.Int, error) {
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

		val = new(int256.Int).Add(l, r)
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
	val.Add(val, a)

	{
		a, err = math.NewSignedFixedPointCalculator(x).
			MulUpMagU(x).
			AddNormalWith(
				math.NewSignedFixedPointCalculator(y).
					MulUpMagU(y),
			).Result()
		if err != nil {
			return nil, nil, err
		}
	}
	errValue := new(int256.Int).Quo(a, i1e38)

	{
		valU256, err := math.SafeCast.ToUint256(val)
		if err != nil {
			return nil, nil, err
		}

		b, err := math.GyroPoolMath.Sqrt(valU256, U5)
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
			val.IsPositive(),

			a,

			i0,
		).Result()
	if err != nil {
		return nil, nil, err
	}

	return val, errValue, nil
}

func (g *gyroECLPMath) checkAssetBounds(
	params *ECLParams,
	derived *ECLDerivedParams,
	invariant *Vector2,
	newBal *int256.Int,
	assetIndex int,
) error {
	if assetIndex == 0 {
		xPlus, err := g.maxBalances0(params, derived, invariant)
		if err != nil {
			return err
		}

		if newBal.Gt(IMaxBalances) || newBal.Gt(xPlus) {
			return ErrAssetBoundsExceeded
		}

		return nil
	}

	yPlus, err := g.maxBalances1(params, derived, invariant)
	if err != nil {
		return err
	}

	if newBal.Gt(IMaxBalances) || newBal.Gt(yPlus) {
		return ErrAssetBoundsExceeded
	}

	return nil
}

func (g *gyroECLPMath) calcXpXpDivLambdaLambda(
	x *int256.Int,
	r *Vector2,
	lambda *int256.Int,
	s *int256.Int,
	c *int256.Int,
	tauBeta *Vector2,
	dSq *int256.Int,
) (*int256.Int, error) {
	var sqVars *Vector2
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

		sqVars = &Vector2{X: x, Y: y}
	}

	q := &qParams{}

	termXp, err := math.NewSignedFixedPointCalculator(tauBeta.X).
		MulXpU(tauBeta.Y).
		DivXpU(sqVars.X).
		Result()
	if err != nil {
		return nil, err
	}
	if termXp.IsPositive() {
		a, err := math.NewSignedFixedPointCalculator(sqVars.Y).
			MulUpMagU(new(int256.Int).Mul(I2, s)).
			Result()
		if err != nil {
			return nil, err
		}

		a, err = math.NewSignedFixedPointCalculator(a).
			MulUpMagU(c).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(termXp).
					AddNormal(i7),
			).Result()
		if err != nil {
			return nil, err
		}

		q.A = a
	} else {
		a, err := math.NewSignedFixedPointCalculator(r.Y).
			MulDownMagU(r.Y).
			MulDownMagU(new(int256.Int).Mul(I2, s)).
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

	if tauBeta.X.IsNegative() {
		b, err := math.NewSignedFixedPointCalculator(r.X).
			MulUpMagU(x).
			MulUpMagU(new(int256.Int).Mul(I2, c)).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(i0).
					SubNormalWith(
						math.NewSignedFixedPointCalculator(tauBeta.X).
							DivXpU(dSq),
					).
					Add(i3),
			).Result()
		if err != nil {
			return nil, err
		}

		q.B = b
	} else {
		b, err := math.NewSignedFixedPointCalculator(new(int256.Int).Neg(r.Y)).
			MulDownMagU(x).
			MulDownMagU(new(int256.Int).Mul(I2, c)).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(tauBeta.X).
					DivXpU(dSq),
			).Result()
		if err != nil {
			return nil, err
		}

		q.B = b
	}

	q.A.Add(q.A, q.B)

	termXp, err = math.NewSignedFixedPointCalculator(tauBeta.Y).
		MulXpU(tauBeta.Y).
		DivXpU(sqVars.X).
		AddNormal(i7).
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
		MulDownMagU(new(int256.Int).Mul(I2, s)).
		MulUpXpToNpUWith(
			math.NewSignedFixedPointCalculator(tauBeta.Y).
				DivXpU(dSq),
		).Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(q.B).
		AddNormal(q.C).
		AddNormalWith(
			math.NewSignedFixedPointCalculator(x).
				MulUpMagU(x),
		).Result()
	if err != nil {
		return nil, err
	}

	q.B, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			q.B.IsPositive(),

			math.NewSignedFixedPointCalculator(q.B).
				DivUpMagU(lambda),

			math.NewSignedFixedPointCalculator(q.B).
				DivDownMagU(lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	q.A, err = math.NewSignedFixedPointCalculator(q.A).
		AddNormal(q.B).
		Result()
	if err != nil {
		return nil, err
	}

	q.A, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			q.A.IsPositive(),

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
		Add(i7).
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
		AddNormal(q.A).
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
	r *Vector2,
	ab *Vector2,
	tauBeta *Vector2,
	dSq *int256.Int,
) (*int256.Int, error) {
	lamBar := &Vector2{}
	{
		x, err := math.NewSignedFixedPointCalculator(i1e38).
			SubNormalWith(
				math.NewSignedFixedPointCalculator(i1e38).
					DivDownMagU(lambda).
					DivDownMagU(lambda),
			).Result()
		if err != nil {
			return nil, err
		}

		lamBar.X = x
	}
	{
		y, err := math.NewSignedFixedPointCalculator(i1e38).
			SubNormalWith(
				math.NewSignedFixedPointCalculator(i1e38).
					DivUpMagU(lambda).
					DivUpMagU(lambda),
			).Result()
		if err != nil {
			return nil, err
		}

		lamBar.Y = y
	}

	var err error
	q := &qParams{}
	xp := new(int256.Int).Sub(x, ab.X)
	q.B, err = math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			xp.IsPositive(),

			math.NewSignedFixedPointCalculator(new(int256.Int).Neg(xp)).
				MulDownMagU(s).
				MulDownMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.Y).
						DivXpU(dSq),
				),

			math.NewSignedFixedPointCalculator(new(int256.Int).Neg(xp)).
				MulUpMagU(s).
				MulUpMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.X).
						DivXpU(dSq).
						AddNormal(i1),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	sTerm := &Vector2{}
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
				AddNormal(i1),
		).
		AddNormal(i1).
		Result()
	if err != nil {
		return nil, err
	}

	sTerm.X, err = math.NewSignedFixedPointCalculator(i1e38).
		SubNormal(sTerm.X).
		Result()
	if err != nil {
		return nil, err
	}
	sTerm.Y, err = math.NewSignedFixedPointCalculator(i1e38).
		SubNormal(sTerm.Y).
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
		AddNormalWith(
			math.NewSignedFixedPointCalculator(r.Y).
				MulDownMagU(r.Y).
				MulDownXpToNpU(sTerm.Y),
		).Result()
	if err != nil {
		return nil, err
	}

	if q.C.IsPositive() {
		qC, err := math.SafeCast.ToUint256(q.C)
		if err != nil {
			return nil, err
		}

		qC, err = math.GyroPoolMath.Sqrt(qC, U5)
		if err != nil {
			return nil, err
		}

		q.C, err = math.SafeCast.ToInt256(qC)
		if err != nil {
			return nil, err
		}
	} else {
		q.C = i0
	}

	if q.B.Gt(q.C) {
		q.A, err = math.NewSignedFixedPointCalculator(q.B).
			SubNormal(q.C).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(i1e38).
					DivXpU(sTerm.Y).
					AddNormal(i1),
			).Result()
	} else {
		q.A, err = math.NewSignedFixedPointCalculator(q.B).
			SubNormal(q.C).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(i1e38).
					DivXpU(sTerm.X),
			).Result()
	}
	if err != nil {
		return nil, err
	}

	return math.NewSignedFixedPointCalculator(q.A).
		AddNormal(ab.Y).
		Result()
}

func (g *gyroECLPMath) calcYGivenX(
	x *int256.Int,
	params *ECLParams,
	d *ECLDerivedParams,
	r *Vector2,
) (*int256.Int, error) {
	ab := &Vector2{}
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
	params *ECLParams,
	d *ECLDerivedParams,
	r *Vector2,
) (*int256.Int, error) {
	ba := &Vector2{}
	{
		x, err := g.virtualOffset1(params, d, r)
		if err != nil {
			return nil, err
		}

		ba.X = x
	}
	{
		y, err := g.virtualOffset0(params, d, r)
		if err != nil {
			return nil, err
		}

		ba.Y = y
	}

	x, err := g.solveQuadraticSwap(
		params.Lambda,
		y,
		params.C,
		params.S,
		r,
		ba,
		&Vector2{X: new(int256.Int).Neg(d.TauAlpha.X), Y: d.TauAlpha.Y},
		d.DSq,
	)
	if err != nil {
		return nil, err
	}

	return x, nil
}
