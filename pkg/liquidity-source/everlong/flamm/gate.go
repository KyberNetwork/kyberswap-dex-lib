package everlongflamm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// FLAMMGateLib (src/core/flamm/FLAMMGateLib.sol): the credit gate `U <= ltv * gross` over per-loan-asset exposure
// (never netted across assets), the sell room, the hook frame, the NAV and posting laws, and the entry / exit
// gates. Pure functions over a gateBook (the pool's tracked physical and liquid plus the Router's aggregates)
// and a gatePool (the ledger, dials and the feed's peeks). The signed int256 arithmetic of netL18 / netPW / the
// anchor is carried as sign-magnitude (gateInt) with int256's own semantics: explicit uint256 -> int256 casts
// wrap and every signed add / sub / negation is checked. Validated wei-exact against FLAMMGateLib at 80abd43
// (gate_math.json.gz, edges/gate_edges.json.gz and gate_int_edges.json.gz including the wrap region) and through the
// fork settlements.

var (
	gateMonotoneSlackWad     = uint256.NewInt(1e9)  // FLAMMStore.MONOTONE_SLACK_WAD
	gateReleaseHysteresis    = uint256.NewInt(1e17) // FLAMMStore.RELEASE_HYSTERESIS_WAD
	gateWadPlusSlack         = new(uint256.Int).Add(uWad, gateMonotoneSlackWad)
	gateFeatureSupplyLending = uint256.NewInt(1 << 3) // FLAMMStore.FEATURE_SUPPLY_LENDING
)

// gateInt is an int256 as sign and magnitude; zero is never negative and the value always lies in
// [-2^255, 2^255 - 1].
type gateInt struct {
	Neg bool
	Abs uint256.Int
}

// gateIntMinAbs is |type(int256).min| = 2^255.
var gateIntMinAbs = new(uint256.Int).Lsh(uOne, 255)

// gateIntCast is the explicit `int256(x)` of a uint256: a two's-complement reinterpretation, unchecked, so
// x >= 2^255 reads as x - 2^256.
func gateIntCast(x *uint256.Int) gateInt {
	var z gateInt
	if x.Lt(gateIntMinAbs) {
		z.Abs = *x
		return z
	}
	z.Neg = true
	z.Abs.Neg(x) // 2^256 - x, in [1, 2^255]
	return z
}

func gateIntDiff(a, b *uint256.Int) gateInt {
	var z gateInt
	if a.Lt(b) {
		z.Neg = true
		z.Abs.Sub(b, a)
	} else {
		z.Abs.Sub(a, b)
	}
	return z
}

// positive reports int256 > 0.
func (x *gateInt) positive() bool { return !x.Neg && !x.Abs.IsZero() }

// gateIntChecked is Solidity's checked int256 result: Panic(0x11) outside [-2^255, 2^255 - 1].
func gateIntChecked(z gateInt) (gateInt, error) {
	if z.Neg && z.Abs.Gt(gateIntMinAbs) || !z.Neg && !z.Abs.Lt(gateIntMinAbs) {
		return gateInt{}, errPanicArithmetic
	}
	return z, nil
}

// add is the checked int256 `x + y`.
func (x gateInt) add(y gateInt) (gateInt, error) {
	if x.Neg != y.Neg {
		if x.Neg {
			return gateIntChecked(gateIntDiff(&y.Abs, &x.Abs))
		}
		return gateIntChecked(gateIntDiff(&x.Abs, &y.Abs))
	}
	z := gateInt{Neg: x.Neg}
	if _, overflow := z.Abs.AddOverflow(&x.Abs, &y.Abs); overflow {
		return gateInt{}, errPanicArithmetic
	}
	return gateIntChecked(z)
}

// sub is the checked int256 `x - y`; y = type(int256).min is valid here (the flipped magnitude never escapes).
func (x gateInt) sub(y gateInt) (gateInt, error) {
	if !y.Abs.IsZero() {
		y.Neg = !y.Neg
	}
	return x.add(y)
}

// neg is the checked int256 `-x`: only type(int256).min reverts.
func (x gateInt) neg() (gateInt, error) {
	if !x.Abs.IsZero() {
		x.Neg = !x.Neg
	}
	return gateIntChecked(x)
}

// gateLeg is FLAMMGateLib.Leg: one loan asset's position (native units) and frame. `PriceWad` is its checked
// poolAsset cross (zero: unchecked); `CrossWad` is q_i, numeraire value per L18 of the asset (WAD for asset 0).
type gateLeg struct {
	Liquid   uint256.Int
	Supplied uint256.Int
	Debt     uint256.Int
	Scale    uint256.Int
	PriceWad uint256.Int
	CrossWad uint256.Int
}

// gateBook is FLAMMGateLib.Book.
type gateBook struct {
	Physical uint256.Int
	Posted   uint256.Int
	Legs     []gateLeg
}

// gateLoanCfg is FLAMMStore.LoanCfg (FLAMMStore.sol:208): one loan asset's binding, swap policy and liquid.
type gateLoanCfg struct {
	Token            common.Address `json:"token"`
	Scale            uint256.Int    `json:"scale"`            // 10**(18 - decimals)
	SwapPriceBandWad uint256.Int    `json:"swapPriceBandWad"` // the fill's band around the checked cross
	FeeFloorWad      uint256.Int    `json:"feeFloorWad"`      // this asset's own fee floor
	MaxSwapNotional  uint256.Int    `json:"maxSwapNotional"`  // native; zero is uncapped
	ReserveTarget    uint256.Int    `json:"reserveTarget"`    // liquid kept back from supplyCascade
	Liquid           uint256.Int    `json:"liquid"`           // tracked unlent custody
}

// gatePool is the pool ledger and dials the gate and settlement read, plus the feed's peeks at the snapshot:
// PriceWad[i] is peekCross(poolAsset, loan_i) (zero when not ok) and CrossWad[i] is WAD for i == 0, else
// Math.mulDiv(usd_i, WAD, usd_0) when both USD peeks are ok and usd_0 != 0, else zero (FLAMMGateLib.priceIn).
type gatePool struct {
	Physical       uint256.Int   `json:"physical"`
	Loans          []gateLoanCfg `json:"loans"`
	LtvWad         uint256.Int   `json:"ltvWad"`
	PhiWad         uint256.Int   `json:"phiWad"`
	RoomEpsilonWad uint256.Int   `json:"roomEpsilonWad"`
	Features       uint256.Int   `json:"features"`
	PriceWad       []uint256.Int `json:"priceWad"`
	CrossWad       []uint256.Int `json:"crossWad"`
}

func (p *gatePool) clone() *gatePool {
	q := *p
	q.Loans = append([]gateLoanCfg(nil), p.Loans...)
	q.PriceWad = append([]uint256.Int(nil), p.PriceWad...)
	q.CrossWad = append([]uint256.Int(nil), p.CrossWad...)
	return &q
}

func gateMul(x, y *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.MulOverflow(x, y); overflow {
		return z, errPanicArithmetic
	}
	return z, nil
}

func gateAdd(x, y *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.AddOverflow(x, y); overflow {
		return z, errPanicArithmetic
	}
	return z, nil
}

// ------------------------------------------------------------------ pure law

// gateNetL18 is FLAMMGateLib.netL18: int256(debt*scale) - int256((supplied+liquid)*scale), positive when a
// net borrower. The casts wrap and the subtraction is checked.
func gateNetL18(L *gateLeg) (gateInt, error) {
	d, err := gateMul(&L.Debt, &L.Scale)
	if err != nil {
		return gateInt{}, err
	}
	a, err := gateAdd(&L.Supplied, &L.Liquid)
	if err != nil {
		return gateInt{}, err
	}
	if a, err = gateMul(&a, &L.Scale); err != nil {
		return gateInt{}, err
	}
	return gateIntCast(&d).sub(gateIntCast(&a))
}

// gateNetPW is FLAMMGateLib.netPW: the net in poolAsset-WAD, the borrower side ceiled and the surplus floored;
// a positioned leg at an unchecked feed reverts PriceUnchecked. `int256(ceilDiv(uint256(n)*WAD, p))` and
// `-int256((uint256(-n)*WAD) / p)`: both casts wrap, both negations are checked.
func gateNetPW(L *gateLeg) (gateInt, error) {
	n, err := gateNetL18(L)
	if err != nil {
		return n, err
	}
	if L.PriceWad.IsZero() {
		if !n.Abs.IsZero() {
			return n, ErrPriceUnchecked
		}
		return gateInt{}, nil
	}
	if !n.Neg {
		w, err := gateMul(&n.Abs, uWad)
		if err != nil {
			return n, err
		}
		return gateIntCast(divCeil(&w, &L.PriceWad)), nil
	}
	m, err := n.neg()
	if err != nil {
		return n, err
	}
	w, err := gateMul(&m.Abs, uWad)
	if err != nil {
		return n, err
	}
	var q uint256.Int
	q.Div(&w, &L.PriceWad)
	return gateIntCast(&q).neg()
}

// gateExposurePW is FLAMMGateLib.exposurePW: U = sum of max(0, netPW_i).
func gateExposurePW(b *gateBook) (uint256.Int, error) {
	var u uint256.Int
	for i := range b.Legs {
		n, err := gateNetPW(&b.Legs[i])
		if err != nil {
			return u, err
		}
		if n.positive() {
			if u, err = gateAdd(&u, &n.Abs); err != nil {
				return u, err
			}
		}
	}
	return u, nil
}

// gateGross is FLAMMGateLib.gross: physical + posted.
func gateGross(b *gateBook) (uint256.Int, error) {
	return gateAdd(&b.Physical, &b.Posted)
}

// gateBoundPW is FLAMMGateLib.boundPW: gross * ltv.
func gateBoundPW(gross, ltvWad *uint256.Int) (uint256.Int, error) {
	return gateMul(gross, ltvWad)
}

// gateBoundOf is FLAMMGateLib.boundOf: mulDiv(gross*price, ltv, WAD).
func gateBoundOf(gross, priceWad, ltvWad *uint256.Int) (uint256.Int, error) {
	c, err := gateMul(gross, priceWad)
	if err != nil {
		return c, err
	}
	return mmMulDivOZ(&c, ltvWad, uWad)
}

// gateLift is FLAMMGateLib.lift: mulDiv(head, WAD, WAD - mulDiv(ltv, phi, WAD)).
func gateLift(head, ltvWad, phiWad *uint256.Int) (uint256.Int, error) {
	lp, err := mmMulDivOZ(ltvWad, phiWad, uWad)
	if err != nil {
		return lp, err
	}
	if lp.Gt(uWad) {
		return lp, errPanicArithmetic
	}
	var den uint256.Int
	den.Sub(uWad, &lp)
	return mmMulDivOZ(head, uWad, &den)
}

// gateRoomWad is FLAMMGateLib.roomWad: (ltv*C - u)^+ / (1 - ltv*phi), a surplus u adding to the headroom
// (`uint256(-u)`, the negation checked).
func gateRoomWad(u gateInt, gross, priceWad, ltvWad, phiWad *uint256.Int) (uint256.Int, error) {
	ltvC, err := gateBoundOf(gross, priceWad, ltvWad)
	if err != nil {
		return ltvC, err
	}
	var headroom uint256.Int
	if !u.Neg {
		if ltvC.Gt(&u.Abs) {
			headroom.Sub(&ltvC, &u.Abs)
		}
	} else {
		m, err := u.neg()
		if err != nil {
			return headroom, err
		}
		if headroom, err = gateAdd(&ltvC, &m.Abs); err != nil {
			return headroom, err
		}
	}
	return gateLift(&headroom, ltvWad, phiWad)
}

// gateHeadOf is FLAMMGateLib.headOf: the credit-zero headroom of leg idx in L18 of that asset -- the standing
// bound less every asset's exposure, plus this asset's own surplus (another asset's surplus never counts).
func gateHeadOf(b *gateBook, idx int, u, ltvWad *uint256.Int) (uint256.Int, error) {
	gross, err := gateGross(b)
	if err != nil {
		return gross, err
	}
	bound, err := gateBoundPW(&gross, ltvWad)
	if err != nil {
		return bound, err
	}
	var headPW uint256.Int
	if bound.Gt(u) {
		headPW.Sub(&bound, u)
	}
	n, err := gateNetPW(&b.Legs[idx])
	if err != nil {
		return headPW, err
	}
	if n.Neg {
		m, err := n.neg()
		if err != nil {
			return headPW, err
		}
		if headPW, err = gateAdd(&headPW, &m.Abs); err != nil {
			return headPW, err
		}
	}
	return mmMulDivOZ(&headPW, &b.Legs[idx].PriceWad, uWad)
}

// gateStructuralDistWad is FLAMMGateLib.structuralDistWad: 1 - ltv/lltv with the quotient ceiled.
func gateStructuralDistWad(ltvWad, lltvWad *uint256.Int) uint256.Int {
	var z uint256.Int
	if lltvWad.IsZero() || !ltvWad.Lt(lltvWad) {
		return z
	}
	q, _ := mmMulDivOZUp(ltvWad, uWad, lltvWad) // < WAD
	return *z.Sub(uWad, &q)
}

// gateRequiredPosted is FLAMMGateLib.requiredPosted: mulDivUp(debt*scale, WAD, ltv*price) poolAsset units.
func gateRequiredPosted(grossDebt, loanScale, ltvWad, priceWad *uint256.Int) (uint256.Int, error) {
	if grossDebt.IsZero() {
		return uint256.Int{}, nil
	}
	num, err := gateMul(grossDebt, loanScale)
	if err != nil {
		return num, err
	}
	den, err := gateMul(ltvWad, priceWad)
	if err != nil {
		return den, err
	}
	return mmMulDivOZUp(&num, uWad, &den)
}

// gateRequiredPostedAll is FLAMMGateLib.requiredPostedAll: the posting law over every indebted asset.
func gateRequiredPostedAll(b *gateBook, ltvWad *uint256.Int) (uint256.Int, error) {
	var need uint256.Int
	for i := range b.Legs {
		L := &b.Legs[i]
		if L.Debt.IsZero() {
			continue
		}
		if L.PriceWad.IsZero() {
			return need, ErrPriceUnchecked
		}
		r, err := gateRequiredPosted(&L.Debt, &L.Scale, ltvWad, &L.PriceWad)
		if err != nil {
			return need, err
		}
		if need, err = gateAdd(&need, &r); err != nil {
			return need, err
		}
	}
	return need, nil
}

// gateNavAt is FLAMMGateLib.navAt: physical + posted + sum floor(assets_i/p_i) - sum ceil(debt_i/p_i), floored at 0.
func gateNavAt(b *gateBook) (uint256.Int, error) {
	plus, err := gateGross(b)
	if err != nil {
		return plus, err
	}
	var minus uint256.Int
	for i := range b.Legs {
		L := &b.Legs[i]
		if L.PriceWad.IsZero() {
			if !L.Liquid.IsZero() || !L.Supplied.IsZero() || !L.Debt.IsZero() {
				return plus, ErrPriceUnchecked
			}
			continue
		}
		a, err := gateAdd(&L.Liquid, &L.Supplied)
		if err != nil {
			return plus, err
		}
		if a, err = gateMul(&a, &L.Scale); err != nil {
			return plus, err
		}
		a.Div(&a, &L.PriceWad)
		if plus, err = gateAdd(&plus, &a); err != nil {
			return plus, err
		}
		d, err := gateMul(&L.Debt, &L.Scale)
		if err != nil {
			return plus, err
		}
		if minus, err = gateAdd(&minus, divCeil(&d, &L.PriceWad)); err != nil {
			return plus, err
		}
	}
	var z uint256.Int
	if plus.Gt(&minus) {
		z.Sub(&plus, &minus)
	}
	return z, nil
}

// gateContext is FLAMMGateLib.context: the poolAsset side native, the loan side aggregated into the numeraire
// (N18, loan asset 0's value) with liquid and supplied floored and debt ceiled; an idle leg at an unchecked
// cross contributes nothing, a positioned one reverts PriceUnchecked.
func gateContext(b *gateBook, priceWad *uint256.Int, priceTs uint64, supply *uint256.Int) (poolContext, error) {
	c := poolContext{PhysicalPoolAsset: b.Physical, PostedPoolAsset: b.Posted, ShareSupply: *supply,
		PriceWad: *priceWad, PriceTs: priceTs, LoanCount: uint8(len(b.Legs))}
	for i := range b.Legs {
		L := &b.Legs[i]
		if L.CrossWad.IsZero() {
			if !L.Liquid.IsZero() || !L.Supplied.IsZero() || !L.Debt.IsZero() {
				return c, ErrPriceUnchecked
			}
			continue
		}
		for _, f := range [3]struct {
			dst *uint256.Int
			v   *uint256.Int
			up  bool
		}{{&c.LiquidLoanAsset, &L.Liquid, false}, {&c.SuppliedLoanAsset, &L.Supplied, false}, {&c.DebtLoanAsset, &L.Debt, true}} {
			n18, err := gateMul(f.v, &L.Scale)
			if err != nil {
				return c, err
			}
			var q uint256.Int
			if f.up {
				q, err = mmMulDivOZUp(&n18, &L.CrossWad, uWad)
			} else {
				q, err = mmMulDivOZ(&n18, &L.CrossWad, uWad)
			}
			if err != nil {
				return c, err
			}
			if *f.dst, err = gateAdd(f.dst, &q); err != nil {
				return c, err
			}
		}
	}
	return c, nil
}

// gateRoomNative is FLAMMGateLib.roomNative: lift(headOf) less the epsilon shave (a checked subtraction), in
// native units of asset idx.
func gateRoomNative(p *gatePool, b *gateBook, idx int, u *uint256.Int) (uint256.Int, error) {
	head, err := gateHeadOf(b, idx, u, &p.LtvWad)
	if err != nil {
		return head, err
	}
	raw, err := gateLift(&head, &p.LtvWad, &p.PhiWad)
	if err != nil {
		return raw, err
	}
	shave, err := mmMulDivOZ(&raw, &p.RoomEpsilonWad, uWad)
	if err != nil {
		return raw, err
	}
	if shave.Gt(&raw) {
		return raw, errPanicArithmetic
	}
	raw.Sub(&raw, &shave)
	if b.Legs[idx].Scale.IsZero() {
		return raw, errPanicDivZero
	}
	return *raw.Div(&raw, &b.Legs[idx].Scale), nil
}

// ------------------------------------------------------------------ storage composites

// gateRouterReads are the two Router views the storage-reading gate functions make (IMMRouter.positions and
// IMMRouter.quarantine); *mmRouter answers them from the tracked venues.
type gateRouterReads interface {
	positions(now uint64) (mmPositions, error)
	quarantine(idx uint8, now uint64) (mmQuarantine, error)
}

// gateBookOf is FLAMMGateLib.book: tracked physical and liquid with the Router's per-asset aggregates, price-free.
func gateBookOf(p *gatePool, r gateRouterReads, now uint64) (gateBook, error) {
	var b gateBook
	pos, err := r.positions(now)
	if err != nil {
		return b, err
	}
	b.Physical = p.Physical
	b.Posted = pos.TotalColl
	b.Legs = make([]gateLeg, len(p.Loans))
	for i := range p.Loans {
		if i >= len(pos.Sup) {
			return b, errPanicIndex
		}
		b.Legs[i] = gateLeg{Liquid: p.Loans[i].Liquid, Supplied: pos.Sup[i], Debt: pos.Debt[i], Scale: p.Loans[i].Scale}
	}
	return b, nil
}

// gatePriced is FLAMMGateLib.priced: the book with every leg's cross and numeraire frame.
func gatePriced(p *gatePool, r gateRouterReads, now uint64) (gateBook, error) {
	b, err := gateBookOf(p, r, now)
	if err != nil {
		return b, err
	}
	for i := range b.Legs {
		b.Legs[i].PriceWad = p.PriceWad[i]
		if i == 0 {
			b.Legs[i].CrossWad = *uWad
		} else {
			b.Legs[i].CrossWad = p.CrossWad[i]
		}
	}
	return b, nil
}

// gatePriceWads is FLAMMGateLib.priceWads / priceVector: the per-asset crosses reclaim takes.
func gatePriceWads(b *gateBook) []uint256.Int {
	a := make([]uint256.Int, len(b.Legs))
	for i := range b.Legs {
		a[i] = b.Legs[i].PriceWad
	}
	return a
}

// gateAssertGate is FLAMMGateLib.assertGate: exposurePW > gross*ltv reverts LedgerBoundBreached.
func gateAssertGate(p *gatePool, r gateRouterReads, now uint64) error {
	b, err := gatePriced(p, r, now)
	if err != nil {
		return err
	}
	return gateAssertBound(&b, &p.LtvWad)
}

func gateAssertBound(b *gateBook, ltvWad *uint256.Int) error {
	u, err := gateExposurePW(b)
	if err != nil {
		return err
	}
	gross, err := gateGross(b)
	if err != nil {
		return err
	}
	bound, err := gateBoundPW(&gross, ltvWad)
	if err != nil {
		return err
	}
	if u.Gt(&bound) {
		return ErrLedgerBoundBreached
	}
	return nil
}

// gateAnchor is FLAMMGateLib.anchor: the READABLE venues' signed per-asset exposure (a quarantined venue's
// counted debt subtracted), the gross poolAsset, and whether any venue is quarantined.
func gateAnchor(p *gatePool, r gateRouterReads, now uint64) ([]gateInt, uint256.Int, bool, error) {
	b, err := gateBookOf(p, r, now)
	if err != nil {
		return nil, uint256.Int{}, false, err
	}
	q, u, err := gateReadableU(r, &b, now)
	if err != nil {
		return nil, uint256.Int{}, false, err
	}
	gross, err := gateGross(&b)
	return u, gross, q, err
}

func gateReadableU(r gateRouterReads, b *gateBook, now uint64) (bool, []gateInt, error) {
	quarantined := false
	u := make([]gateInt, len(b.Legs))
	for i := range b.Legs {
		qr, err := r.quarantine(uint8(i), now)
		if err != nil {
			return false, nil, err
		}
		if qr.Any {
			quarantined = true
		}
		n, err := gateNetL18(&b.Legs[i])
		if err != nil {
			return false, nil, err
		}
		f, err := gateMul(&qr.FrozenDebt, &b.Legs[i].Scale)
		if err != nil {
			return false, nil, err
		}
		if u[i], err = n.sub(gateIntCast(&f)); err != nil {
			return false, nil, err
		}
	}
	return quarantined, u, nil
}

// gateAssertEntryGate is FLAMMGateLib.assertEntryGate: the absolute bound, else no drawn asset's u/gross may
// worsen (per asset, 1e-9 slack). A quarantined anchor restores one frame on both sides: the frozen collateral
// is added back to both grosses and the anchor's frozen-debt subtraction undone.
func gateAssertEntryGate(p *gatePool, r gateRouterReads, now uint64, u0 []gateInt, gross0 uint256.Int, quarantined bool) error {
	b, err := gatePriced(p, r, now)
	if err != nil {
		return err
	}
	g1, err := gateGross(&b)
	if err != nil {
		return err
	}
	n := len(b.Legs)
	var fDebt []uint256.Int
	if quarantined {
		fDebt = make([]uint256.Int, n)
		var fColl uint256.Int
		for i := 0; i < n; i++ {
			qr, err := r.quarantine(uint8(i), now)
			if err != nil {
				return err
			}
			fDebt[i] = qr.FrozenDebt
			if fColl, err = gateAdd(&fColl, &qr.FrozenColl); err != nil {
				return err
			}
		}
		if g1, err = gateAdd(&g1, &fColl); err != nil {
			return err
		}
		if gross0, err = gateAdd(&gross0, &fColl); err != nil {
			return err
		}
	}
	u, err := gateExposurePW(&b)
	if err != nil {
		return err
	}
	bound, err := gateBoundPW(&g1, &p.LtvWad)
	if err != nil {
		return err
	}
	if !u.Gt(&bound) {
		return nil
	}
	for i := 0; i < n; i++ {
		u1, err := gateNetL18(&b.Legs[i])
		if err != nil {
			return err
		}
		if !u1.positive() {
			continue
		}
		base := u0[i]
		if quarantined {
			f, err := gateMul(&fDebt[i], &b.Legs[i].Scale)
			if err != nil {
				return err
			}
			if base, err = base.add(gateIntCast(&f)); err != nil {
				return err
			}
		}
		if !base.positive() {
			return ErrLedgerBoundBreached
		}
		if err := gateRequireNotWorsened(&base.Abs, &gross0, &u1.Abs, &g1, ErrLedgerBoundBreached); err != nil {
			return err
		}
	}
	return nil
}

// gateAssertExitNotWorsened is FLAMMGateLib.assertExitNotWorsened: price-free, per asset, over readable venues.
func gateAssertExitNotWorsened(p *gatePool, r gateRouterReads, now uint64, u0 []gateInt, gross0 uint256.Int) error {
	b, err := gateBookOf(p, r, now)
	if err != nil {
		return err
	}
	_, u1, err := gateReadableU(r, &b, now)
	if err != nil {
		return err
	}
	g1, err := gateGross(&b)
	if err != nil {
		return err
	}
	for i := range u1 {
		if !u1[i].positive() {
			continue
		}
		if !u0[i].positive() {
			return ErrExitWorsensLedger
		}
		if err := gateRequireNotWorsened(&u0[i].Abs, &gross0, &u1[i].Abs, &g1, ErrExitWorsensLedger); err != nil {
			return err
		}
	}
	return nil
}

// gateRequireNotWorsened is FLAMMGateLib._requireNotWorsened: u1*g0 <= mulDiv(u0*g1, WAD + slack, WAD).
func gateRequireNotWorsened(u0, g0, u1, g1 *uint256.Int, fail error) error {
	if g0.IsZero() {
		return nil
	}
	lhs, err := gateMul(u1, g0)
	if err != nil {
		return err
	}
	x, err := gateMul(u0, g1)
	if err != nil {
		return err
	}
	rhs, err := mmMulDivOZ(&x, gateWadPlusSlack, uWad)
	if err != nil {
		return err
	}
	if lhs.Gt(&rhs) {
		return fail
	}
	return nil
}

// gateTotalAssets is FLAMMGateLib.totalAssets: navAt over the priced book.
func gateTotalAssets(p *gatePool, r gateRouterReads, now uint64) (uint256.Int, error) {
	b, err := gatePriced(p, r, now)
	if err != nil {
		return uint256.Int{}, err
	}
	return gateNavAt(&b)
}
