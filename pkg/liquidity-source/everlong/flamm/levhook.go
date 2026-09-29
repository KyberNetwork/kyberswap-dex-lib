package everlongflamm

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// Port of EverlongLeverageHook (src/hooks/everlong/lev/EverlongLeverageHook.sol @ c104 80abd43, deployed on
// Base at 0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3): the leverage venue's frame over the swap hook's book
// and its exact fill. The hook is stateless, so executeLever == previewLever and levQuote serves both.
//
// Reverts are reproduced in Solidity's evaluation order. Custom errors map to the package sentinels; a
// checked-arithmetic Panic(0x11) and OpenZeppelin 4.8 Math.mulDiv's bare `require(denominator > prod1)`
// map to errPanicArithmetic and errMulDivOverflow (errors.go).
//
// levFrame and levFill words are freshly allocated and never alias the context, the book or a package
// constant, so callers may mutate them (TestAliasingReturnedWords).

var (
	levCrCeilingWad      = uint256.NewInt(2_200_000_000_000_000_000) // CR_CEILING_WAD = 2.2e18
	levMaxConcessionPpm  = uint256.NewInt(1_010_000)                 // PPM + LEV_MAX_CONCESSION_PPM
	levPriceBandUpperWad = uint256.NewInt(2_000_000_000_000_000_000) // WAD * PRICE_BAND_NUM
	levPriceBandLowerWad = uint256.NewInt(500_000_000_000_000_000)   // WAD / PRICE_BAND_NUM
)

// levInt256SignBit is bit 255 of a word, as seen in its top limb.
const levInt256SignBit = uint64(1) << 63

// levFill mirrors IFLAMMLeverage.LeverFill.
type levFill struct {
	AmountInUsed  *uint256.Int
	GrossOut      *uint256.Int
	VirtualLegL18 *uint256.Int
	CrAfterWad    *uint256.Int
}

// levBook carries what frame() consumes from the swap hook: the four legs of EverlongHook.bookFor(ctx) (the
// book already rescaled pro rata to the context's gross poolAsset) and EverlongHook.reservationPriceWad()
// (N18 per base unit, times WAD). Kappa and x are not read. The integrator fills it from the swap hook's
// bookFor port for the same PoolContext the fill is quoted on.
type levBook struct {
	Rs                  *uint256.Int
	Is                  *uint256.Int
	Rv                  *uint256.Int
	Iv                  *uint256.Int
	ReservationPriceWad *uint256.Int
}

// levFrame mirrors EverlongLeverageHook.Frame: cv and S in L18, V in poolAsset base units, D signed and
// XAnchor the frozen curve's anchor on (cv, D) when D > 0.
type levFrame struct {
	Cv      *uint256.Int
	D       *int256.Int
	V       *uint256.Int
	S       *uint256.Int
	XAnchor *uint256.Int
}

func levCheckedAdd(x, y *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).AddOverflow(x, y)
	if overflow {
		return nil, errPanicArithmetic
	}
	return z, nil
}

func levCheckedMul(x, y *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).MulOverflow(x, y)
	if overflow {
		return nil, errPanicArithmetic
	}
	return z, nil
}

// levMulDiv is Math.mulDiv (OZ 4.8) with its revert: floor(x*y/d), or the bare require on overflow.
func levMulDiv(x, y, d *uint256.Int) (*uint256.Int, error) {
	z, overflow := mulDiv(x, y, d)
	if overflow {
		return nil, errMulDivOverflow
	}
	return z, nil
}

// levMulDivRoundUp is Math.mulDiv(..., Rounding.Up): the floor's revert first, then the checked `+= 1`.
func levMulDivRoundUp(x, y, d *uint256.Int) (*uint256.Int, error) {
	z, err := levMulDiv(x, y, d)
	if err != nil {
		return nil, err
	}
	if !new(uint256.Int).MulMod(x, y, d).IsZero() {
		if z.Eq(maxUint256) {
			return nil, errPanicArithmetic
		}
		z.AddUint64(z, 1)
	}
	return z, nil
}

// levSignedAddChecked and levSignedSubChecked are int256 `+` and `-` under checked arithmetic, on the
// two's-complement words (int256(uint256) is a reinterpretation, never a revert).
func levSignedAddChecked(x, y *uint256.Int) (*uint256.Int, error) {
	z := new(uint256.Int).Add(x, y)
	if (x[3]^z[3])&(y[3]^z[3])&levInt256SignBit != 0 {
		return nil, errPanicArithmetic
	}
	return z, nil
}

func levSignedSubChecked(x, y *uint256.Int) (*uint256.Int, error) {
	z := new(uint256.Int).Sub(x, y)
	if (x[3]^y[3])&(x[3]^z[3])&levInt256SignBit != 0 {
		return nil, errPanicArithmetic
	}
	return z, nil
}

// levFrameFor is frame(ctx): v = rv + iv, s = rs + is_, cv = s + floor(v*reservationPriceWad/WAD),
// D = int256(s) + int256(debt) - int256(supplied + liquid), and xAnchor = anchorAndBase(cv, D, WAD, RATIO)
// when D > 0 (zero otherwise).
func levFrameFor(ctx *poolContext, book *levBook) (*levFrame, error) {
	f := &levFrame{XAnchor: new(uint256.Int)}
	var err error
	if f.V, err = levCheckedAdd(book.Rv, book.Iv); err != nil {
		return nil, err
	}
	if f.S, err = levCheckedAdd(book.Rs, book.Is); err != nil {
		return nil, err
	}
	vValue, err := levMulDiv(f.V, book.ReservationPriceWad, uWad)
	if err != nil {
		return nil, err
	}
	if f.Cv, err = levCheckedAdd(f.S, vValue); err != nil {
		return nil, err
	}
	credit, err := levCheckedAdd(&ctx.SuppliedLoanAsset, &ctx.LiquidLoanAsset)
	if err != nil {
		return nil, err
	}
	d, err := levSignedAddChecked(f.S, &ctx.DebtLoanAsset)
	if err != nil {
		return nil, err
	}
	if d, err = levSignedSubChecked(d, credit); err != nil {
		return nil, err
	}
	f.D = (*int256.Int)(d)
	if f.D.Sign() > 0 {
		f.XAnchor, _ = levAnchorAndBase(f.Cv, d, uWad, levLeverageRatioWad)
	}
	return f, nil
}

// levQuote is _quote(ctx), the body of previewLever and executeLever.
//
// Up: dq = max(D, ceil(cv*WAD/2.2e18)) is the CR-ceiling-clamped debt the curve prices on,
// dCv = floor(cv*in/v) the collateral delta and dS = ceil(s*in/v) the virtual leg the join mints. The gross
// is leverageQuote(cv, dq, WAD, RATIO, spread, dCv) capped at floor(dCv*(PPM-spread)/PPM) and must exceed dS.
//
// Down: in18 < D, collOut = deleverageQuote(cv, D, WAD, RATIO, spread, in18) in (0, cv), bounded by
// floor(in18*1.01); the taker receives volOut = floor(v*collOut/cv) and dSBurn = floor(s*collOut/cv) of the
// virtual leg is burnt, which in18 must exceed.
//
// Both sides assert the anchor and the 2x value band on the post-fill (cv, D) and report
// crAfter = floor(gavAfter*WAD/dAfter), gavAfter the book at the pool's feed scaled by the fill.
func levQuote(ctx *leverContext, book *levBook) (*levFill, error) {
	f, err := levFrameFor(&ctx.Pool, book)
	if err != nil {
		return nil, err
	}
	if f.V.IsZero() || f.Cv.IsZero() || f.D.Sign() <= 0 {
		return nil, ErrFrameUnquotable
	}
	d := (*uint256.Int)(f.D)
	vAtFeed, err := levCheckedMul(f.V, &ctx.Pool.PriceWad)
	if err != nil {
		return nil, err
	}
	gavAtFeed, err := levCheckedAdd(f.S, vAtFeed)
	if err != nil {
		return nil, err
	}

	if ctx.Up {
		dq, err := levMulDivRoundUp(f.Cv, uWad, levCrCeilingWad)
		if err != nil {
			return nil, err
		}
		if dq.Lt(d) {
			dq = d
		}
		dCv, err := levMulDiv(f.Cv, &ctx.AmountIn, f.V)
		if err != nil {
			return nil, err
		}
		dS, err := levMulDivRoundUp(f.S, &ctx.AmountIn, f.V)
		if err != nil {
			return nil, err
		}
		gross, _, _ := levLeverageQuote(f.Cv, dq, uWad, levLeverageRatioWad, &ctx.SpreadPpm, dCv)
		// The clamped debt describes a book that does not exist: never pay more than the delta is worth at
		// its own mark, net of the spread.
		if ctx.SpreadPpm.Gt(uPpm) {
			return nil, errPanicArithmetic
		}
		capped, err := levMulDiv(dCv, new(uint256.Int).Sub(uPpm, &ctx.SpreadPpm), uPpm)
		if err != nil {
			return nil, err
		}
		if gross.Gt(capped) {
			gross = capped
		}
		if !gross.Gt(dS) {
			return nil, ErrNothingToFill
		}
		dAfter, err := levCheckedAdd(d, gross)
		if err != nil {
			return nil, err
		}
		cvAfter, err := levCheckedAdd(f.Cv, dCv)
		if err != nil {
			return nil, err
		}
		if err = levAssertAnchorAndBand(f.XAnchor, cvAfter, dAfter); err != nil {
			return nil, err
		}
		vAfter, err := levCheckedAdd(f.V, &ctx.AmountIn)
		if err != nil {
			return nil, err
		}
		gavAfter, err := levMulDiv(gavAtFeed, vAfter, f.V)
		if err != nil {
			return nil, err
		}
		crAfter, err := levMulDiv(gavAfter, uWad, dAfter)
		if err != nil {
			return nil, err
		}
		return &levFill{AmountInUsed: new(uint256.Int).Set(&ctx.AmountIn), GrossOut: gross, VirtualLegL18: dS,
			CrAfterWad: crAfter}, nil
	}

	in18 := &ctx.AmountIn
	if !in18.Lt(d) {
		return nil, ErrNothingToFill
	}
	collOut, _, _ := levDeleverageQuote(f.Cv, d, uWad, levLeverageRatioWad, &ctx.SpreadPpm, in18)
	if collOut.IsZero() || !collOut.Lt(f.Cv) {
		return nil, ErrNothingToFill
	}
	// Bounded, not banned: near target the curve pays a small, honest premium; far from it, it must not.
	concession, err := levMulDiv(in18, levMaxConcessionPpm, uPpm)
	if err != nil {
		return nil, err
	}
	if collOut.Gt(concession) {
		return nil, ErrLevValueLeak
	}
	volOut := levMulDivFloor(f.V, collOut, f.Cv) // collOut < cv: cannot overflow
	dSBurn := levMulDivFloor(f.S, collOut, f.Cv) // likewise
	if volOut.IsZero() || !in18.Gt(dSBurn) {
		return nil, ErrNothingToFill
	}
	dAfterDown := new(uint256.Int).Sub(d, in18)
	cvAfterDown := new(uint256.Int).Sub(f.Cv, collOut)
	if err = levAssertAnchorAndBand(f.XAnchor, cvAfterDown, dAfterDown); err != nil {
		return nil, err
	}
	gavAfterDown := levMulDivFloor(gavAtFeed, cvAfterDown, f.Cv) // cvAfterDown < cv: cannot overflow
	crAfter, err := levMulDiv(gavAfterDown, uWad, dAfterDown)
	if err != nil {
		return nil, err
	}
	return &levFill{AmountInUsed: new(uint256.Int).Set(in18), GrossOut: volOut, VirtualLegL18: dSBurn,
		CrAfterWad: crAfter}, nil
}

// levAssertAnchorAndBand is _assertAnchorAndBand: the post-fill anchor must not fall below the frame's
// (FillValueDrop, also on a zero baseX) and floor(baseX*WAD/cvAfter) must lie in [WAD/2, 2*WAD]
// (FillPriceBand).
func levAssertAnchorAndBand(xAnchorBefore, cvAfter, dAfter *uint256.Int) error {
	xAnchorAfter, baseXAfter := levAnchorAndBase(cvAfter, dAfter, uWad, levLeverageRatioWad)
	if xAnchorAfter.Lt(xAnchorBefore) || baseXAfter.IsZero() {
		return ErrFillValueDrop
	}
	internalValueAfter, err := levMulDiv(baseXAfter, uWad, cvAfter)
	if err != nil {
		return err
	}
	if internalValueAfter.Gt(levPriceBandUpperWad) || internalValueAfter.Lt(levPriceBandLowerWad) {
		return ErrFillPriceBand
	}
	return nil
}
