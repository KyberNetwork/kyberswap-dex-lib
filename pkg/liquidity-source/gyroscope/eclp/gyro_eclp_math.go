package gyroeclp

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
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

	NUMBER_1E36 *big.Int
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

type calcGiven func(*big.Int, *params, *derivedParams, *vector2) (*big.Int, error)

func init() {
	GyroECLPMath = &gyroECLPMath{
		ONEHALF: big.NewInt(0.5e18),
		ONE:     big.NewInt(1e18),
		ONE_XP:  integer.TenPow(38),

		_ROTATION_VECTOR_NORM_ACCURACY:    big.NewInt(1e3),
		_MAX_STRETCH_FACTOR:               integer.TenPow(26),
		_DERIVED_TAU_NORM_ACCURACY_XP:     integer.TenPow(23),
		_MAX_INV_INVARIANT_DENOMINATOR_XP: integer.TenPow(43),
		_DERIVED_DSQ_NORM_ACCURACY_XP:     integer.TenPow(23),

		_MAX_BALANCES:  integer.TenPow(34),
		_MAX_INVARIANT: integer.TenPow(37),

		NUMBER_1E36: integer.TenPow(36),
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

	err = g.checkAssetBounds(params, derived, invariant, balInNewU256.ToBig(), ixIn)
	if err != nil {
		return nil, err
	}

	balOutNew, err := calcGive(amountIn.ToBig(), params, derived, invariant)
	if err != nil {
		return nil, err
	}

	balOutNewU256, err := math.GyroFixedPoint.Sub(
		balances[ixOut],
		uint256.MustFromBig(balOutNew),
	)
	if err != nil {
		return nil, err
	}

	return balOutNewU256, nil
}

func (g *gyroECLPMath) calculateInvariantWithError(
	balances []*uint256.Int,
	params *params,
	derived *derivedParams,
) (*big.Int, *big.Int, error) {
	x, y := balances[0].ToBig(), balances[1].ToBig()

	xPlusY, err := math.NewSignedFixedPointCalculator(x).
		Add(y).
		Result()
	if err != nil {
		return nil, nil, err
	}
	if xPlusY.Cmp(g._MAX_BALANCES) > 0 {
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

	if sqrt.Cmp(integer.Zero()) > 0 {
		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(integer.One()).
			DivUpMagU(new(big.Int).Mul(integer.Two(), sqrt)).
			Result()
		if err != nil {
			return nil, nil, err
		}

	} else {
		if errValue.Cmp(integer.Zero()) > 0 {
			errValueU256, err := math.GyroPoolMath.Sqrt(
				uint256.MustFromBig(errValue),
				uint256.NewInt(5),
			)
			if err != nil {
				return nil, nil, err
			}

			errValue = errValueU256.ToBig()

		} else {
			errValue = integer.TenPow(9)
		}
	}

	var t *big.Int
	{
		t, err = math.NewSignedFixedPointCalculator(params.Lambda).
			MulUpMagU(xPlusY).
			Result()
		if err != nil {
			return nil, nil, err
		}
	}
	errValue = new(big.Int).Mul(
		new(big.Int).Add(
			new(big.Int).Add(
				new(big.Int).Quo(
					t, g.ONE_XP,
				),
				errValue,
			),
			integer.One(),
		),
		big.NewInt(20),
	)

	var mulDenominator *big.Int
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

		u = new(big.Int).Quo(
			new(big.Int).Mul(
				new(big.Int).Mul(
					t,
					new(big.Int).Quo(
						new(big.Int).Mul(params.Lambda, params.Lambda),
						g.NUMBER_1E36,
					),
				),
				big.NewInt(40),
			),
			g.ONE_XP,
		)

		errValue, err = math.NewSignedFixedPointCalculator(errValue).
			Add(u).
			Add(integer.One()).
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
	if t.Cmp(g._MAX_INVARIANT) > 0 {
		return nil, nil, ErrMaxInvariantExceeded
	}

	return invariant, errValue, nil
}

func (g *gyroECLPMath) calcAChiAChiInXp(p *params, d *derivedParams) (*big.Int, error) {
	dSq3, err := math.NewSignedFixedPointCalculator(d.DSq).
		MulXpU(d.DSq).
		MulXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(p.Lambda).
		MulUpMagUWith(
			math.NewSignedFixedPointCalculator(new(big.Int).Mul(integer.Two(), d.U)).
				MulXpU(d.V).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(d.U).
				Add(integer.One()).
				MulXpU(new(big.Int).Add(d.U, integer.One())).
				DivXpU(dSq3).
				MulUpMagU(p.Lambda).
				MulUpMagU(p.Lambda),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(d.V).
				MulXpU(d.V).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.W).
		DivUpMagU(p.Lambda).
		Add(d.Z).
		Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termXp).
				MulXpU(termXp).
				DivXpU(dSq3),
		).Result()
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (g *gyroECLPMath) calcAtAChi(x, y *big.Int, p *params, d *derivedParams) (*big.Int, error) {
	dSq2, err := math.NewSignedFixedPointCalculator(d.DSq).
		MulXpU(d.DSq).
		Result()
	if err != nil {
		return nil, err
	}

	termXp, err := math.NewSignedFixedPointCalculator(d.W).
		DivDownMagU(p.Lambda).
		Add(d.Z).
		DivDownMagU(p.Lambda).
		DivXpU(dSq2).
		Result()
	if err != nil {
		return nil, err
	}

	val, err := math.NewSignedFixedPointCalculator(x).
		MulDownMagU(p.C).
		SubWith(
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
		AddWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(p.Lambda).
				MulDownMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
			math.NewSignedFixedPointCalculator(termNp).
				MulDownXpToNpUWith(
					math.NewSignedFixedPointCalculator(d.V).
						DivXpU(dSq2),
				),
		).Result()
	if err != nil {
		return nil, err
	}

	termNp, err = math.NewSignedFixedPointCalculator(x).
		MulDownMagU(p.S).
		AddWith(
			math.NewSignedFixedPointCalculator(y).
				MulDownMagU(p.C),
		).Result()
	if err != nil {
		return nil, err
	}

	val, err = math.NewSignedFixedPointCalculator(val).
		AddWith(
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

func (g *gyroECLPMath) calcSpotPrice0in1(
	balances []*big.Int,
	params *params,
	derived *derivedParams,
	invariant *big.Int,
) (*big.Int, error) {
	r := &vector2{X: invariant, Y: invariant}

	var ab *vector2
	{
		x, err := g.virtualOffset0(params, derived, r)
		if err != nil {
			return nil, err
		}

		y, err := g.virtualOffset1(params, derived, r)
		if err != nil {
			return nil, err
		}

		ab = &vector2{X: x, Y: y}
	}

	var vec *vector2
	{
		x, err := math.NewSignedFixedPointCalculator(balances[0]).
			Sub(ab.X).
			Result()
		if err != nil {
			return nil, err
		}

		y, err := math.NewSignedFixedPointCalculator(balances[1]).
			Sub(ab.Y).
			Result()
		if err != nil {
			return nil, err
		}

		vec = &vector2{X: x, Y: y}
	}

	vec, err := g.mulA(params, vec)
	if err != nil {
		return nil, err
	}

	var pc *vector2
	{
		x, err := math.NewSignedFixedPointCalculator(vec.X).
			DivDownMagU(vec.Y).
			Result()
		if err != nil {
			return nil, err
		}

		pc = &vector2{X: x, Y: g.ONE}
	}

	var pgx *big.Int
	{
		t2, err := g.mulA(params, &vector2{X: g.ONE, Y: integer.Zero()})
		if err != nil {
			return nil, err
		}

		pgx, err = g.scalarProd(pc, t2)
		if err != nil {
			return nil, err
		}
	}

	var t *big.Int
	{
		t2, err := g.mulA(params, &vector2{X: integer.Zero(), Y: g.ONE})
		if err != nil {
			return nil, err
		}

		t, err = g.scalarProd(pc, t2)
		if err != nil {
			return nil, err
		}
	}

	px, err := math.NewSignedFixedPointCalculator(pgx).
		DivDownMagU(t).
		Result()
	if err != nil {
		return nil, err
	}

	return px, nil
}

func (g *gyroECLPMath) checkAssetBounds(
	params *params,
	derived *derivedParams,
	invariant *vector2,
	newBal *big.Int,
	assetIndex int,
) error {
	if assetIndex == 0 {
		xPlus, err := g.maxBalances0(params, derived, invariant)
		if err != nil {
			return err
		}

		if newBal.Cmp(g._MAX_BALANCES) > 0 || newBal.Cmp(xPlus) > 0 {
			return ErrAssetBoundsExceeded
		}

		return nil
	}

	yPlus, err := g.maxBalances1(params, derived, invariant)
	if err != nil {
		return err
	}

	if newBal.Cmp(g._MAX_BALANCES) > 0 || newBal.Cmp(yPlus) > 0 {
		return ErrAssetBoundsExceeded
	}

	return nil
}

func (g *gyroECLPMath) calcXpXpDivLambdaLambda(
	x *big.Int,
	r *vector2,
	lambda *big.Int,
	s *big.Int,
	c *big.Int,
	tauBeta *vector2,
	dSq *big.Int,
) (*big.Int, error) {
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
	if termXp.Cmp(integer.Zero()) > 0 {
		a, err := math.NewSignedFixedPointCalculator(sqVars.Y).
			MulUpMagU(new(big.Int).Mul(integer.Two(), s)).
			Result()
		if err != nil {
			return nil, err
		}

		a, err = math.NewSignedFixedPointCalculator(a).
			MulUpMagU(c).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(termXp).
					Add(big.NewInt(7)),
			).Result()
		if err != nil {
			return nil, err
		}

		q.A = a
	} else {
		a, err := math.NewSignedFixedPointCalculator(r.Y).
			MulDownMagU(r.Y).
			MulDownMagU(new(big.Int).Mul(integer.Two(), s)).
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

	if tauBeta.X.Cmp(integer.Zero()) < 0 {
		b, err := math.NewSignedFixedPointCalculator(r.X).
			MulUpMagU(x).
			MulUpMagU(new(big.Int).Mul(integer.Two(), c)).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(integer.Zero()).
					SubWith(
						math.NewSignedFixedPointCalculator(tauBeta.X).
							DivXpU(dSq),
					).Add(big.NewInt(3)),
			).Result()
		if err != nil {
			return nil, err
		}

		q.B = b
	} else {
		b, err := math.NewSignedFixedPointCalculator(new(big.Int).Neg(r.Y)).
			MulDownMagU(x).
			MulDownMagU(new(big.Int).Mul(integer.Two(), c)).
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
		Add(big.NewInt(7)).
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

	q.C, err = math.NewSignedFixedPointCalculator(new(big.Int).Neg(r.Y)).
		MulDownMagU(x).
		MulDownMagU(new(big.Int).Mul(integer.Two(), s)).
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
			q.B.Cmp(integer.Zero()) > 0,

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
			q.A.Cmp(integer.Zero()) > 0,

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
		Add(big.NewInt(7)).
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
	lambda *big.Int,
	x *big.Int,
	s *big.Int,
	c *big.Int,
	r *vector2,
	ab *vector2,
	tauBeta *vector2,
	dSq *big.Int,
) (*big.Int, error) {
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
			xp.Cmp(integer.Zero()) > 0,

			math.NewSignedFixedPointCalculator(new(big.Int).Sub(integer.Zero(), xp)).
				MulDownMagU(s).
				MulDownMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.Y).
						DivXpU(dSq),
				),

			math.NewSignedFixedPointCalculator(new(big.Int).Sub(integer.Zero(), xp)).
				MulUpMagU(s).
				MulUpMagU(c).
				MulUpXpToNpUWith(
					math.NewSignedFixedPointCalculator(lamBar.X).
						DivXpU(dSq).
						Add(integer.One()),
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
				Add(integer.One()),
		).Add(integer.One()).
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
	q.C = new(big.Int).Neg(q.C)

	q.C, err = math.NewSignedFixedPointCalculator(q.C).
		AddWith(
			math.NewSignedFixedPointCalculator(r.Y).
				MulDownMagU(r.Y).
				MulDownXpToNpU(sTerm.Y),
		).Result()
	if err != nil {
		return nil, err
	}

	if q.C.Cmp(integer.Zero()) > 0 {
		qC, _ := uint256.FromBig(q.C)

		qC, err = math.GyroPoolMath.Sqrt(qC, uint256.NewInt(5))
		if err != nil {
			return nil, err
		}

		q.C = qC.ToBig()
	} else {
		q.C = integer.Zero()
	}

	if q.B.Cmp(q.C) > 0 {
		q.A, err = math.NewSignedFixedPointCalculator(q.B).
			Sub(q.C).
			MulUpXpToNpUWith(
				math.NewSignedFixedPointCalculator(g.ONE_XP).
					DivXpU(sTerm.Y).
					Add(integer.One()),
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
	x *big.Int,
	params *params,
	d *derivedParams,
	r *vector2,
) (*big.Int, error) {
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
	y *big.Int,
	params *params,
	d *derivedParams,
	r *vector2,
) (*big.Int, error) {
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
		&vector2{X: new(big.Int).Neg(d.TauAlpha.X), Y: d.TauAlpha.Y},
		d.DSq)
	if err != nil {
		return nil, err
	}

	return x, nil
}
