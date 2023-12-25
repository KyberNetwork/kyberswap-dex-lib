package gyroeclp

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
)

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

func (g *gyroECLPMath) scalarProd(t1 *vector2, t2 *vector2) (*big.Int, error) {
	return math.NewSignedFixedPointCalculator(t1.X).
		MulDownMag(t2.X).
		AddWith(
			math.NewSignedFixedPointCalculator(t1.Y).MulDownMag(t2.Y),
		).Result()
}

func (g *gyroECLPMath) scalarProdXp(t1, t2 *vector2) (*big.Int, error) {
	return math.NewSignedFixedPointCalculator(t1.X).
		MulXp(t2.X).
		AddWith(
			math.NewSignedFixedPointCalculator(t1.Y).MulDownMag(t2.Y),
		).Result()
}

func (g *gyroECLPMath) mulA(params *params, tp *vector2) (*vector2, error) {
	x, err := math.NewSignedFixedPointCalculator(params.C).
		MulDownMagU(tp.X).
		DivDownMagU(params.Lambda).
		SubWith(
			math.NewSignedFixedPointCalculator(params.S).
				MulDownMag(tp.Y).
				DivDownMag(params.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	y, err := math.NewSignedFixedPointCalculator(params.S).
		MulDownMag(tp.X).
		AddWith(
			math.NewSignedFixedPointCalculator(params.C).
				MulDownMag(tp.Y),
		).Result()
	if err != nil {
		return nil, err
	}

	return &vector2{X: x, Y: y}, nil
}

func (g *gyroECLPMath) virtualOffset0(p *params, d *derivedParams, r *vector2) (*big.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauBeta.X, d.DSq)
	if err != nil {
		return nil, err
	}

	a, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauBeta.X.Cmp(integer.Zero()) > 0,

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

func (g *gyroECLPMath) virtualOffset1(p *params, d *derivedParams, r *vector2) (*big.Int, error) {
	termXp, err := math.SignedFixedPoint.DivXpU(d.TauAlpha.X, d.DSq)
	if err != nil {
		return nil, err
	}

	b, err := math.NewSignedFixedPointCalculator(nil).
		TernaryWith(
			d.TauAlpha.X.Cmp(integer.Zero()) < 0,

			math.NewSignedFixedPointCalculator(r.X).
				MulDownMagU(p.Lambda).
				MulUpMagU(p.S).
				MulUpXpToNp(new(big.Int).Neg(termXp)),

			math.NewSignedFixedPointCalculator(new(big.Int).Neg(r.Y)).
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

func (g *gyroECLPMath) maxBalances0(p *params, d *derivedParams, r *vector2) (*big.Int, error) {
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
					termXp2.Cmp(integer.Zero()) > 0,

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

func (g *gyroECLPMath) maxBalances1(p *params, d *derivedParams, r *vector2) (*big.Int, error) {
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
					termXp2.Cmp(integer.Zero()) > 0,

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

func (g *gyroECLPMath) calcMinAtxAChiySqPlusAtxSq(x, y *big.Int, p *params, d *derivedParams) (*big.Int, error) {
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
				MulDownMag(new(big.Int).Mul(p.C, integer.Two())).
				MulDownMag(p.S),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.U).
		MulXpU(d.U).
		AddWith(
			math.NewSignedFixedPointCalculator(new(big.Int).Mul(integer.Two(), d.U)).
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

	val, err := math.NewSignedFixedPointCalculator(integer.Zero()).
		Sub(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termNp).
				Sub(big.NewInt(9)).
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

func (g *gyroECLPMath) calc2AtxAtyAChixAChiy(x, y *big.Int, p *params, d *derivedParams) (*big.Int, error) {
	termNp, err := math.NewSignedFixedPointCalculator(x).
		MulUpMagU(x).
		SubWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(y),
		).
		MulDownMagU(new(big.Int).Mul(integer.Two(), p.C)).
		MulDownMagU(p.S).
		Result()
	if err != nil {
		return nil, err
	}

	xy, err := math.NewSignedFixedPointCalculator(y).
		MulDownMagU(new(big.Int).Mul(integer.Two(), x)).
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

func (g *gyroECLPMath) calcMinAtyAChixSqPlusAtySq(x, y *big.Int, p *params, d *derivedParams) (*big.Int, error) {
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
				MulUpMagU(new(big.Int).Mul(p.S, integer.Two())).
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
			math.NewSignedFixedPointCalculator(new(big.Int).Mul(integer.Two(), d.Z)).
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

	val, err := math.NewSignedFixedPointCalculator(integer.Zero()).
		Sub(termNp).
		MulDownXpToNpU(termXp).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termNp).
				Sub(big.NewInt(9)).
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

func (g *gyroECLPMath) calcInvariantSqrt(x, y *big.Int, p *params, d *derivedParams) (*big.Int, *big.Int, error) {
	var val *big.Int
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
		a   *big.Int
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
	errValue := new(big.Int).Quo(a, g.ONE_XP)

	{
		valU256, _ := uint256.FromBig(val)

		b, err := math.GyroPoolMath.Sqrt(valU256, uint256.NewInt(5))
		if err != nil {
			return nil, nil, err
		}

		a = b.ToBig()
	}

	val, err = math.NewSignedFixedPointCalculator(nil).
		Ternary(
			val.Cmp(integer.Zero()) > 0,

			a,

			integer.Zero(),
		).Result()
	if err != nil {
		return nil, nil, err
	}

	return val, errValue, nil
}
