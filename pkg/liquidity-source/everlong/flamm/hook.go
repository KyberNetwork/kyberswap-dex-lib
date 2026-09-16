package everlongflamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Wei-exact port of the swap path of EverlongHook.sol (c104 @ 80abd43,
// src/hooks/everlong/EverlongHook.sol): the lazy book rescale (_book), the fee role (_fee), the
// invariant role (_fill, _fillSell, _fillBuy, _swap, _maxInForGrossCap, _spotAt) and the book commit an
// executeExactIn writes. The curve's stable leg is the pool numeraire N18 and the volatile leg poolAsset
// base units; prices held here are the pool's priceWad times WAD. Validated against the live Base hook
// (storage overwritten per state) over testdata/hook_fill_grid.json.gz, and against the one real
// swap (tx 0x46c3cd72..., block 51302916) in testdata/hook_live_swap.json.

// hookKappaSeed is EverlongHook.KAPPA_SEED, the scale an empty accounted book is re-seeded at.
var hookKappaSeed = new(uint256.Int).Set(big256.TenPow(30))

// hookState is the EverlongHook storage the swap path reads, slot for slot.
type hookState struct {
	AWad                uint256.Int // _p.aWad, read by _spotAt
	Support             almSupport  // _sup, read by the curve solve and the seed
	AnchorSqrtX96       uint256.Int
	ReservationPriceWad uint256.Int
	Kappa               uint256.Int
	XWad                uint256.Int
	ReserveStable       uint256.Int
	IdleStable          uint256.Int
	ReserveVolatile     uint256.Int
	IdleVolatile        uint256.Int
	RvWad               uint256.Int
	Fee                 feeParams   // _p.tuning.fee
	InvSkewKappaWad     uint256.Int // _p.tuning.invSkewKappaWad
	InvSkewBandWad      uint256.Int // _p.tuning.invSkewBandWad
	LoanScale           uint256.Int // immutable LOAN_SCALE = 10**(18 - loanDecimals)
}

// hookBook is EverlongHook.Book: the book as an execution materialises it.
type hookBook struct {
	Kappa uint256.Int
	Rs    uint256.Int
	Is    uint256.Int
	Rv    uint256.Int
	Iv    uint256.Int
	X     uint256.Int
}

// hookFillResult is FillResult: the input consumed, the gross output, the fee retained on it, and the
// post-fill spot in the pool's price units. capEvals is not part of FillResult: it counts the curve solves
// _maxInForGrossCap ran for the fill, which the settlement's gas grows with (constant.go).
type hookFillResult struct {
	AmountInUsed uint256.Int
	GrossOut     uint256.Int
	FeeOut       uint256.Int
	SpotAfterWad uint256.Int
	capEvals     uint64
}

// bookFor is EverlongHook.bookFor / _book: the stored book rescaled lazily by gross/(rv+iv). An empty
// accounted book (rv + iv == 0) is first re-seeded at KAPPA_SEED — at the stored coordinate, or at
// WAD/2 when that coordinate holds no volatile — with idle cleared.
func (s *hookState) bookFor(ctx *poolContext) (hookBook, error) {
	b := hookBook{Kappa: s.Kappa, Rs: s.ReserveStable, Is: s.IdleStable, Rv: s.ReserveVolatile, Iv: s.IdleVolatile,
		X: s.XWad}
	var gross, acc uint256.Int
	if _, overflow := gross.AddOverflow(&ctx.PhysicalPoolAsset, &ctx.PostedPoolAsset); overflow {
		return b, errPanicArithmetic
	}
	if _, overflow := acc.AddOverflow(&b.Rv, &b.Iv); overflow {
		return b, errPanicArithmetic
	}
	var err error
	if acc.IsZero() {
		b.Kappa.Set(hookKappaSeed)
		if b.Rs, b.Rv, err = almReservesAt(&s.Support, &s.AnchorSqrtX96, hookKappaSeed, &b.X); err != nil {
			return b, err
		}
		if b.Rv.IsZero() {
			b.X.Set(almHalfWad)
			if b.Rs, b.Rv, err = almReservesAt(&s.Support, &s.AnchorSqrtX96, hookKappaSeed, almHalfWad); err != nil {
				return b, err
			}
		}
		b.Is.Clear()
		b.Iv.Clear()
		acc.Set(&b.Rv)
		if acc.IsZero() {
			return b, nil
		}
	}
	if gross.Eq(&acc) {
		return b, nil
	}
	if b.Kappa, err = almMulDiv(&b.Kappa, &gross, &acc); err != nil {
		return b, err
	}
	rv, err := almMulDiv(&b.Rv, &gross, &acc)
	if err != nil {
		return b, err
	}
	if rv.Gt(&gross) {
		rv.Set(&gross)
	}
	b.Rv = rv
	b.Iv.Sub(&gross, &rv)
	if b.Rs, err = almMulDiv(&b.Rs, &gross, &acc); err != nil {
		return b, err
	}
	b.Is, err = almMulDiv(&b.Is, &gross, &acc)
	return b, err
}

// spotAt is EverlongHook._spotAt: priceAtX(x, _p.aWad) * reservationPriceWad / WAD, the hook's WAD
// price (the pool's price units times WAD).
func (s *hookState) spotAt(x *uint256.Int) (uint256.Int, error) {
	p, err := almPriceAtX(x, &s.AWad)
	if err != nil {
		return p, err
	}
	return almMulDiv(&p, &s.ReservationPriceWad, uWad)
}

// spot is EverlongHook.spot: the spot at the stored coordinate in the pool's price units (the context
// is ignored on-chain and is not taken).
func (s *hookState) spot() (uint256.Int, error) {
	p, err := s.spotAt(&s.XWad)
	if err != nil {
		return p, err
	}
	return *p.Div(&p, uWad), nil
}

// previewFeeWad is EverlongHook.previewFeeWad / executeFeeWad (_fee): the fill fee on the book as the
// execution would materialise it. The pool's spotBefore argument is ignored on-chain and is not taken.
func (s *hookState) previewFeeWad(ctx *swapContext) (uint256.Int, error) {
	b, err := s.bookFor(&ctx.Pool)
	if err != nil {
		return uint256.Int{}, err
	}
	st := feeState{ReserveStable: b.Rs, ReserveVolatile: b.Rv, AnchorWad: s.ReservationPriceWad, RvWad: s.RvWad}
	if st.SpotWad, err = s.spotAt(&b.X); err != nil {
		return uint256.Int{}, err
	}
	return feeFillFee(&s.Fee, &st, !ctx.PoolAssetIn, &s.InvSkewKappaWad, &s.InvSkewBandWad)
}

// previewExactIn is EverlongHook.previewExactIn: the exact-input fill at feeWad without a commit.
func (s *hookState) previewExactIn(ctx *swapContext, feeWad *uint256.Int) (hookFillResult, error) {
	fr, _, err := s.fill(ctx, feeWad)
	return fr, err
}

// executeExactIn is EverlongHook.executeExactIn: the same fill, returning the post-state the hook stores.
// The book is committed (kappa, reserves, idle, x) only when the fill consumed input; otherwise the
// state is returned unchanged, lazy rescale included. The receiver is never mutated.
func (s *hookState) executeExactIn(ctx *swapContext, feeWad *uint256.Int) (hookFillResult, hookState, error) {
	post := *s
	fr, b, err := s.fill(ctx, feeWad)
	if err != nil {
		return fr, post, err
	}
	if !fr.AmountInUsed.IsZero() {
		post.commit(&b)
	}
	return fr, post, nil
}

// commit is EverlongHook._commit.
func (s *hookState) commit(b *hookBook) {
	s.Kappa, s.ReserveStable, s.IdleStable = b.Kappa, b.Rs, b.Is
	s.ReserveVolatile, s.IdleVolatile, s.XWad = b.Rv, b.Iv, b.X
}

// fill is EverlongHook._fill: a retracted book, a zero input or a fee at 100% fills nothing.
func (s *hookState) fill(ctx *swapContext, feeWad *uint256.Int) (hookFillResult, hookBook, error) {
	var fr hookFillResult
	b, err := s.bookFor(&ctx.Pool)
	if err != nil {
		return fr, b, err
	}
	if b.Kappa.IsZero() || ctx.AmountIn.IsZero() || !feeWad.Lt(uWad) {
		return fr, b, nil
	}
	if ctx.PoolAssetIn {
		return s.fillSell(ctx, feeWad, b)
	}
	return s.fillBuy(ctx, feeWad, b)
}

// hookNetOf is gross - mulDiv(gross, feeWad, WAD): the output after the fee haircut (feeWad < WAD).
func hookNetOf(gross, feeWad *uint256.Int) (uint256.Int, error) {
	haircut, err := almMulDiv(gross, feeWad, uWad)
	if err != nil {
		return haircut, err
	}
	var net uint256.Int
	net.Sub(gross, &haircut)
	return net, nil
}

// capFill is the shared cap step of _fillSell/_fillBuy: when the net output exceeds maxAmountOut, the
// input is re-solved as the largest whose gross stays within cap*WAD/(WAD - fee), and the net is then
// clamped to the cap. evals counts the bisection's solves. A buy never bisects: its maxAmountOut is the pool's
// gross poolAsset (FLAMMSwapLib.sol:151), the same gross _book clamps rv to (EverlongHook.sol:617), and _swap
// clamps the output to rv (:578), so net <= gross <= rv <= maxAmountOut.
func (s *hookState) capFill(ctx *swapContext, feeWad *uint256.Int, b *hookBook, stableIn bool) (gross, xAfter,
	used, net uint256.Int, evals uint64, err error) {
	gross, xAfter, unspent, err := s.swap(b, stableIn, &ctx.AmountIn)
	if err != nil {
		return
	}
	used.Sub(&ctx.AmountIn, &unspent)
	if net, err = hookNetOf(&gross, feeWad); err != nil || !net.Gt(&ctx.MaxAmountOut) {
		return
	}
	var keep uint256.Int
	keep.Sub(uWad, feeWad)
	capGross, err := almMulDiv(&ctx.MaxAmountOut, uWad, &keep)
	if err != nil {
		return
	}
	maxIn, evals, err := s.maxInForGrossCap(b, stableIn, &ctx.AmountIn, &capGross)
	if err != nil {
		return
	}
	if gross, xAfter, unspent, err = s.swap(b, stableIn, &maxIn); err != nil {
		return
	}
	used.Sub(&maxIn, &unspent)
	if net, err = hookNetOf(&gross, feeWad); err != nil {
		return
	}
	if net.Gt(&ctx.MaxAmountOut) {
		net.Set(&ctx.MaxAmountOut)
	}
	return
}

// fillSell is EverlongHook._fillSell (poolAsset in, N18 out). The fill is snapped to the loan asset's
// native grid: the taker is paid floor(net/LOAN_SCALE) native units, the reported gross and fee are
// grid multiples, and the unpaid residue gross - netNative*LOAN_SCALE is booked to idle stable.
func (s *hookState) fillSell(ctx *swapContext, feeWad *uint256.Int, b hookBook) (hookFillResult, hookBook,
	error) {
	var fr hookFillResult
	gross, xAfter, used, net, evals, err := s.capFill(ctx, feeWad, &b, false)
	if err != nil {
		return fr, b, err
	}
	fr.capEvals = evals
	var netNative, grossNative, paid, t uint256.Int
	netNative.Div(&net, &s.LoanScale)
	if used.IsZero() || gross.IsZero() || netNative.IsZero() {
		return fr, b, nil
	}
	grossNative.Div(&gross, &s.LoanScale)
	if _, overflow := b.Rv.AddOverflow(&b.Rv, &used); overflow {
		return fr, b, errPanicArithmetic
	}
	if b.Rs.Lt(&gross) {
		return fr, b, errPanicArithmetic
	}
	b.Rs.Sub(&b.Rs, &gross)
	paid.Mul(&netNative, &s.LoanScale) // netNative*LOAN_SCALE <= net
	t.Sub(&gross, &paid)
	if _, overflow := b.Is.AddOverflow(&b.Is, &t); overflow {
		return fr, b, errPanicArithmetic
	}
	b.X = xAfter
	fr.AmountInUsed = used
	fr.GrossOut.Mul(&grossNative, &s.LoanScale)
	fr.FeeOut.Sub(&grossNative, &netNative)
	fr.FeeOut.Mul(&fr.FeeOut, &s.LoanScale)
	spot, err := s.spotAt(&xAfter)
	if err != nil {
		return hookFillResult{}, b, err
	}
	fr.SpotAfterWad.Div(&spot, uWad)
	return fr, b, nil
}

// fillBuy is EverlongHook._fillBuy (N18 in, poolAsset out). The charge is the used input rounded UP to
// the native grid; the over-collected residue paid - used is booked to idle stable, and the fee is
// retained in kind as idle volatile.
func (s *hookState) fillBuy(ctx *swapContext, feeWad *uint256.Int, b hookBook) (hookFillResult, hookBook,
	error) {
	var fr hookFillResult
	gross, xAfter, usedL18, net, evals, err := s.capFill(ctx, feeWad, &b, true)
	if err != nil {
		return fr, b, err
	}
	fr.capEvals = evals
	if usedL18.IsZero() || gross.IsZero() || net.IsZero() {
		return fr, b, nil
	}
	// Math.ceilDiv (OZ 4.x): (a - 1) / b + 1 for a != 0.
	var usedNative, paidL18 uint256.Int
	usedNative.SubUint64(&usedL18, 1)
	usedNative.Div(&usedNative, &s.LoanScale)
	usedNative.AddUint64(&usedNative, 1)
	if _, overflow := paidL18.MulOverflow(&usedNative, &s.LoanScale); overflow {
		return fr, b, errPanicArithmetic
	}
	if _, overflow := b.Rs.AddOverflow(&b.Rs, &usedL18); overflow {
		return fr, b, errPanicArithmetic
	}
	if paidL18.Gt(&usedL18) {
		var residue uint256.Int
		residue.Sub(&paidL18, &usedL18)
		if _, overflow := b.Is.AddOverflow(&b.Is, &residue); overflow {
			return fr, b, errPanicArithmetic
		}
	}
	if b.Rv.Lt(&gross) {
		return fr, b, errPanicArithmetic
	}
	b.Rv.Sub(&b.Rv, &gross)
	var feeOut uint256.Int
	feeOut.Sub(&gross, &net)
	if _, overflow := b.Iv.AddOverflow(&b.Iv, &feeOut); overflow {
		return fr, b, errPanicArithmetic
	}
	b.X = xAfter
	fr.AmountInUsed, fr.GrossOut, fr.FeeOut = paidL18, gross, feeOut
	spot, err := s.spotAt(&xAfter)
	if err != nil {
		return hookFillResult{}, b, err
	}
	fr.SpotAfterWad.Div(&spot, uWad)
	return fr, b, nil
}

// swap is EverlongHook._swap: the curve solve, its gross output clamped to the deployed reserve of the
// output leg (rv for a stable-in fill, rs for a volatile-in fill).
func (s *hookState) swap(b *hookBook, stableIn bool, amountIn *uint256.Int) (gross, xAfter, unspent uint256.Int,
	err error) {
	gross, xAfter, unspent, err = almSwapExactInX96(&s.Support, &s.AnchorSqrtX96, &b.Kappa, &b.X, stableIn, amountIn)
	if err != nil {
		return
	}
	available := &b.Rs
	if stableIn {
		available = &b.Rv
	}
	if gross.Gt(available) {
		gross.Set(available)
	}
	return
}

// maxInForGrossCap is EverlongHook._maxInForGrossCap: the largest input in [0, hi] whose clamped gross
// stays within capGross, by a 64-step bisection on the monotone solve (lo + hi checked). It also returns the
// number of _swap solves the loop ran, min(64, about log2(hi)) (EverlongHook.sol:588-594).
func (s *hookState) maxInForGrossCap(b *hookBook, stableIn bool, hiIn, capGross *uint256.Int) (uint256.Int, uint64,
	error) {
	var lo, hi, mid uint256.Int
	var evals uint64
	hi.Set(hiIn)
	for i := 0; i < 64; i++ {
		if _, overflow := mid.AddOverflow(&lo, &hi); overflow {
			return lo, evals, errPanicArithmetic
		}
		mid.Rsh(&mid, 1)
		if mid.Eq(&lo) {
			break
		}
		evals++
		g, _, _, err := s.swap(b, stableIn, &mid)
		if err != nil {
			return lo, evals, err
		}
		if !g.Gt(capGross) {
			lo.Set(&mid)
		} else {
			hi.Set(&mid)
		}
	}
	return lo, evals, nil
}
