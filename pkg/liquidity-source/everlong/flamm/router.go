package everlongflamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// MMRouter / MMRouterLib (src/core/mm/MMRouter.sol, MMRouterLib.sol; c104 router 0x19A9b39E…6bB4 linking
// MMRouterLib 0x167740F8…8591): one pool's record -- loan assets, venues, priorities, pin -- and the multi-venue
// bodies the pool settles through: the funding plan and its ceiling, fund / repayCascade / supplyCascade /
// reclaim, and the aggregate views. Recognized positions are min(actual, managed); a venue unreadable past its
// account's grace recognizes nothing and carries its debt at the 1.05x haircut. Then the FLAMMSwapLib settlement
// legs that drive the Router (payLoan, takeLoan, releaseExcess, settleSell / the buy branch of execute) and the
// single-venue MMRouter entries the pool's pro-rata flows use.
//
// Router methods write through as they go and may leave a half-applied state behind an error; callers run them
// on clone() and keep the result only on success, as a reverted transaction leaves no trace. The repay snapshot
// (EIP-1153 transient storage on chain) is carried in TransientRepay across the calls of one transaction, clones
// included, and callers drop it with endTransaction when that transaction ends. mmSettleSell and mmSettleBuy are
// whole swap transactions and do both themselves.
//
// Router storage the tracker must read (verified against the views on a Base fork): _pools at slot 1; a
// PoolRecord's venues array at keccak(pool . 1) + 3; each Venue is 6 slots with managedCollateral at +4 and
// managedSupplyShares at +5 (the only venue fields no view exposes).

var (
	mmQuarantineCountWad = new(uint256.Int).Add(uWad, uint256.NewInt(5e16)) // WAD + QUARANTINE_HAIRCUT_WAD
	mmMaxUint256         = new(uint256.Int).Set(big256.UMax)
)

// mmLoan is MMRouterLib.Loan (the account addresses are the venues' own business here).
type mmLoan struct {
	Decimals      uint8       `json:"decimals"`
	LoanScale     uint256.Int `json:"loanScale"`
	DebtCap       uint256.Int `json:"debtCap"`
	SupplyCap     uint256.Int `json:"supplyCap"`
	BorrowEnabled bool        `json:"borrowEnabled"`
	Retired       bool        `json:"retired"`
}

// mmVenue is MMRouterLib.Venue with the financing account's Morpho market behind it.
type mmVenue struct {
	Morpho              mmVenueMarket `json:"morpho"`
	Kind                uint8         `json:"kind"`
	LoanIndex           uint8         `json:"loanIndex"`
	LltvWad             uint256.Int   `json:"lltvWad"`
	BorrowEnabled       bool          `json:"borrowEnabled"`
	SupplyEnabled       bool          `json:"supplyEnabled"`
	Retired             bool          `json:"retired"`
	DebtCap             uint256.Int   `json:"debtCap"`
	SupplyCap           uint256.Int   `json:"supplyCap"`
	MaxBorrowRateWad    uint256.Int   `json:"maxBorrowRateWad"`
	ManagedCollateral   uint256.Int   `json:"managedCollateral"`
	ManagedSupplyShares uint256.Int   `json:"managedSupplyShares"`
}

// mmRouter is one pool's MMRouterLib.PoolRecord plus the Router-wide globalPaused.
type mmRouter struct {
	GlobalPaused   bool        `json:"globalPaused"`
	PinLtvWad      uint256.Int `json:"pinLtvWad"`
	SafetyGapWad   uint256.Int `json:"safetyGapWad"`
	OracleBandWad  uint256.Int `json:"oracleBandWad"`
	MaxDrawnAssets uint8       `json:"maxDrawnAssets"`
	Loans          []mmLoan    `json:"loans"`
	Venues         []mmVenue   `json:"venues"`
	BorrowOrder    []uint16    `json:"borrowOrder"`
	SupplyOrder    []uint16    `json:"supplyOrder"`
	WithdrawOrder  []uint16    `json:"withdrawOrder"`
	RepayOrder     []uint16    `json:"repayOrder"`
	// TransientRepay is the per-transaction repay snapshot (transient storage), cleared by endTransaction; never
	// persisted.
	TransientRepay map[uint16]mmRepaySnapshot `json:"-" msgpack:"-"`
}

// clone deep-copies everything a settlement writes (loans and venues); the priority orders are never written.
func (p *mmRouter) clone() *mmRouter {
	q := *p
	q.Loans = append([]mmLoan(nil), p.Loans...)
	q.Venues = append([]mmVenue(nil), p.Venues...)
	if p.TransientRepay != nil {
		q.TransientRepay = make(map[uint16]mmRepaySnapshot, len(p.TransientRepay))
		for k, v := range p.TransientRepay {
			q.TransientRepay[k] = v
		}
	}
	return &q
}

// ------------------------------------------------------------------ reads

// mmRead is MMRouterLib.Read.
type mmRead struct {
	Readable             bool
	Collateral           uint256.Int
	RecognizedCollateral uint256.Int
	Supplied             uint256.Int
	RecognizedSupplied   uint256.Int
	Debt                 uint256.Int
}

// read is MMRouterLib.read: tryPosition, and for a readable venue the recognized figures -- collateral
// min(actual, managed), supply the full valuation when every share is managed, else the managed shares valued.
func (p *mmRouter) read(v *mmVenue, now uint64) (mmRead, error) {
	var r mmRead
	t, err := v.Morpho.tryPosition(now)
	if err != nil {
		return r, err
	}
	r.Readable, r.Collateral, r.Supplied, r.Debt = t.Readable, t.Collateral, t.Supplied, t.Debt
	if !r.Readable {
		return r, nil
	}
	r.RecognizedCollateral = *minU(&r.Collateral, &v.ManagedCollateral)
	managed := minU(&t.SupplyShares, &v.ManagedSupplyShares)
	if managed.Eq(&t.SupplyShares) {
		r.RecognizedSupplied = r.Supplied
	} else if r.RecognizedSupplied, err = v.Morpho.supplySharesToAssets(managed, now); err != nil {
		return r, err
	}
	return r, nil
}

// mmCounted is MMRouterLib._counted: readable debt as is, an unreadable venue's at the 1.05x flat allowance, ceiled.
func mmCounted(r *mmRead) (uint256.Int, error) {
	if r.Readable {
		return r.Debt, nil
	}
	return mmMulDivOZUp(&r.Debt, mmQuarantineCountWad, uWad)
}

// loanAt is MMRouterLib.loanAt.
func (p *mmRouter) loanAt(idx uint8) (*mmLoan, error) {
	if int(idx) >= len(p.Loans) {
		return nil, ErrBadLoanIndex
	}
	return &p.Loans[idx], nil
}

// mmPositions is MMRouter.positions: per loan asset recognized collateral, supply and counted debt, and the
// total recognized collateral (the pool's posted poolAsset).
type mmPositions struct {
	Coll, Sup, Debt []uint256.Int
	TotalColl       uint256.Int
}

// positions is MMRouterLib.positions over the live (non-retired) venues.
func (p *mmRouter) positions(now uint64) (mmPositions, error) {
	m := len(p.Loans)
	out := mmPositions{Coll: make([]uint256.Int, m), Sup: make([]uint256.Int, m), Debt: make([]uint256.Int, m)}
	for i := range p.Venues {
		v := &p.Venues[i]
		if v.Retired {
			continue
		}
		r, err := p.read(v, now)
		if err != nil {
			return out, err
		}
		c, err := mmCounted(&r)
		if err != nil {
			return out, err
		}
		k := v.LoanIndex
		out.Coll[k].Add(&out.Coll[k], &r.RecognizedCollateral)
		out.Sup[k].Add(&out.Sup[k], &r.RecognizedSupplied)
		out.Debt[k].Add(&out.Debt[k], &c)
		out.TotalColl.Add(&out.TotalColl, &r.RecognizedCollateral)
	}
	return out, nil
}

// aggregate is MMRouterLib.aggregate for loan asset idx: (recognized collateral, recognized supply, counted debt).
func (p *mmRouter) aggregate(idx uint8, now uint64) (uint256.Int, uint256.Int, uint256.Int, error) {
	var coll, sup, debt uint256.Int
	for i := range p.Venues {
		v := &p.Venues[i]
		if v.Retired || v.LoanIndex != idx {
			continue
		}
		r, err := p.read(v, now)
		if err != nil {
			return coll, sup, debt, err
		}
		c, err := mmCounted(&r)
		if err != nil {
			return coll, sup, debt, err
		}
		coll.Add(&coll, &r.RecognizedCollateral)
		sup.Add(&sup, &r.RecognizedSupplied)
		debt.Add(&debt, &c)
	}
	return coll, sup, debt, nil
}

// position is MMRouter.position: loanAt's bound check, then aggregate.
func (p *mmRouter) position(idx uint8, now uint64) (uint256.Int, uint256.Int, uint256.Int, error) {
	if _, err := p.loanAt(idx); err != nil {
		return uint256.Int{}, uint256.Int{}, uint256.Int{}, err
	}
	return p.aggregate(idx, now)
}

// mmQuarantine is MMRouter.quarantine.
type mmQuarantine struct {
	Any        bool
	FrozenDebt uint256.Int
	FrozenColl uint256.Int
}

// quarantine is MMRouterLib.quarantine: the unreadable venues of loan asset idx, their counted debt and the
// collateral the quarantine de-recognised (the same min(actual, managed) a readable read uses).
func (p *mmRouter) quarantine(idx uint8, now uint64) (mmQuarantine, error) {
	var q mmQuarantine
	for i := range p.Venues {
		v := &p.Venues[i]
		if v.Retired || v.LoanIndex != idx {
			continue
		}
		r, err := p.read(v, now)
		if err != nil {
			return q, err
		}
		if r.Readable {
			continue
		}
		q.Any = true
		c, err := mmCounted(&r)
		if err != nil {
			return q, err
		}
		q.FrozenDebt.Add(&q.FrozenDebt, &c)
		q.FrozenColl.Add(&q.FrozenColl, minU(&r.Collateral, &v.ManagedCollateral))
	}
	return q, nil
}

// drawn is MMRouterLib.drawn: the loan assets carrying debt (readable or not) as a count and a bitmask.
func (p *mmRouter) drawn(now uint64) (uint8, uint8, error) {
	var count, mask uint8
	for i := range p.Venues {
		v := &p.Venues[i]
		bit := uint8(1) << v.LoanIndex
		if v.Retired || mask&bit != 0 {
			continue
		}
		r, err := p.read(v, now)
		if err != nil {
			return count, mask, err
		}
		if !r.Debt.IsZero() {
			mask |= bit
			count++
		}
	}
	return count, mask, nil
}

// drawable is MMRouterLib.drawable: a one-asset pool always; else an asset already drawn or a free draw slot.
func (p *mmRouter) drawable(idx uint8, now uint64) (bool, error) {
	if len(p.Loans) == 1 {
		return true, nil
	}
	count, mask, err := p.drawn(now)
	if err != nil {
		return false, err
	}
	return mask&(uint8(1)<<idx) != 0 || count < p.MaxDrawnAssets, nil
}

// loanDebtRoom is MMRouterLib.loanDebtRoom.
func (p *mmRouter) loanDebtRoom(l *mmLoan, idx uint8, now uint64) (uint256.Int, error) {
	if l.DebtCap.IsZero() {
		return *mmMaxUint256, nil
	}
	_, _, debt, err := p.aggregate(idx, now)
	var z uint256.Int
	if err == nil && l.DebtCap.Gt(&debt) {
		z.Sub(&l.DebtCap, &debt)
	}
	return z, err
}

// loanSupplyRoom is MMRouterLib.loanSupplyRoom.
func (p *mmRouter) loanSupplyRoom(l *mmLoan, idx uint8, now uint64) (uint256.Int, error) {
	if l.SupplyCap.IsZero() {
		return *mmMaxUint256, nil
	}
	_, sup, _, err := p.aggregate(idx, now)
	var z uint256.Int
	if err == nil && l.SupplyCap.Gt(&sup) {
		z.Sub(&l.SupplyCap, &sup)
	}
	return z, err
}

// minLltv is MMRouterLib.minLltv: over every live venue that may borrow or still carries debt.
func (p *mmRouter) minLltv(now uint64) (uint256.Int, error) {
	var lltv uint256.Int
	for i := range p.Venues {
		v := &p.Venues[i]
		if v.Retired {
			continue
		}
		if !v.BorrowEnabled {
			r, err := p.read(v, now)
			if err != nil {
				return lltv, err
			}
			if r.Debt.IsZero() {
				continue
			}
		}
		if lltv.IsZero() || v.LltvWad.Lt(&lltv) {
			lltv = v.LltvWad
		}
	}
	return lltv, nil
}

func (p *mmRouter) scaleOf(v *mmVenue) *uint256.Int {
	return &p.Loans[v.LoanIndex].LoanScale
}

// maxDebtAtPin is MMRouterLib.maxDebtAtPin: mulDiv(mulDiv(collateral, price, scale), pin, WAD).
func (p *mmRouter) maxDebtAtPin(v *mmVenue, collateral, priceWad *uint256.Int) (uint256.Int, error) {
	x, err := mmMulDivOZ(collateral, priceWad, p.scaleOf(v))
	if err != nil {
		return x, err
	}
	return mmMulDivOZ(&x, &p.PinLtvWad, uWad)
}

// requiredCollateral is MMRouterLib.requiredCollateral: mulDivUp(mulDivUp(debt, WAD, pin), scale, price).
func (p *mmRouter) requiredCollateral(v *mmVenue, debt, priceWad *uint256.Int) (uint256.Int, error) {
	value, err := mmMulDivOZUp(debt, uWad, &p.PinLtvWad)
	if err != nil {
		return value, err
	}
	return mmMulDivOZUp(&value, p.scaleOf(v), priceWad)
}

// bandOk is MMRouterLib.bandOk: the market oracle readable, and (with a band set) within the band of the
// PriceFeed cross lifted to the oracle's 1e36 scale; a zero cross fails a banded check.
func (p *mmRouter) bandOk(v *mmVenue, priceWad *uint256.Int) (bool, error) {
	if !v.Morpho.OracleOk {
		return false, nil
	}
	if p.OracleBandWad.IsZero() {
		return true, nil
	}
	if priceWad.IsZero() {
		return false, nil
	}
	expected, err := mmMulDivOZ(priceWad, mmOraclePriceScale, p.scaleOf(v))
	if err != nil {
		return false, err
	}
	price := &v.Morpho.OraclePrice
	var dev uint256.Int
	if price.Gt(&expected) {
		dev.Sub(price, &expected)
	} else {
		dev.Sub(&expected, price)
	}
	tol, err := mmMulDivOZ(&expected, &p.OracleBandWad, uWad)
	if err != nil {
		return false, err
	}
	return !dev.Gt(&tol), nil
}

// supplyRoom is MMRouterLib.supplyRoom.
func (p *mmRouter) supplyRoom(v *mmVenue, now uint64) (uint256.Int, error) {
	var z uint256.Int
	rd, err := p.read(v, now)
	if err != nil || !rd.Readable {
		return z, err
	}
	if v.SupplyCap.IsZero() {
		return *mmMaxUint256, nil
	}
	if v.SupplyCap.Gt(&rd.RecognizedSupplied) {
		z.Sub(&v.SupplyCap, &rd.RecognizedSupplied)
	}
	return z, nil
}

// managedShares is MMRouterLib.managedShares.
func (p *mmRouter) managedShares(v *mmVenue) uint256.Int {
	return *minU(&v.Morpho.Position.SupplyShares, &v.ManagedSupplyShares)
}

// recognizedSupplied is MMRouterLib.recognizedSupplied (reverts IrmUnreadable past the grace).
func (p *mmRouter) recognizedSupplied(v *mmVenue, now uint64) (uint256.Int, error) {
	shares := p.managedShares(v)
	return v.Morpho.supplySharesToAssets(&shares, now)
}

// freeCollateral is MMRouterLib.freeCollateral: collateral above the posting law at the pin, capped by the
// recognized figure; all of it for a debt-free venue, nothing for an indebted one at an unchecked or out-of-band
// cross.
func (p *mmRouter) freeCollateral(v *mmVenue, priceWad *uint256.Int, now uint64) (uint256.Int, error) {
	var z uint256.Int
	rd, err := p.read(v, now)
	if err != nil || !rd.Readable {
		return z, err
	}
	if rd.Debt.IsZero() {
		return rd.RecognizedCollateral, nil
	}
	if priceWad.IsZero() {
		return z, nil
	}
	ok, err := p.bandOk(v, priceWad)
	if err != nil || !ok {
		return z, err
	}
	required, err := p.requiredCollateral(v, &rd.Debt, priceWad)
	if err != nil {
		return z, err
	}
	if rd.Collateral.Gt(&required) {
		z.Sub(&rd.Collateral, &required)
	}
	return *minU(&z, &rd.RecognizedCollateral), nil
}

// reclaimable is MMRouter.reclaimable: every live venue's free collateral at its loan asset's cross.
func (p *mmRouter) reclaimable(priceWads []uint256.Int, now uint64) (uint256.Int, error) {
	var total uint256.Int
	if len(priceWads) != len(p.Loans) {
		return total, ErrInvalidConfig
	}
	for i := range p.Venues {
		v := &p.Venues[i]
		if v.Retired {
			continue
		}
		f, err := p.freeCollateral(v, &priceWads[v.LoanIndex], now)
		if err != nil {
			return total, err
		}
		total.Add(&total, &f)
	}
	return total, nil
}

// venuePosition is MMRouter.venuePosition: (collateral, recognized, supplied, recognized, readable ? debt : 0).
func (p *mmRouter) venuePosition(id uint16, now uint64) ([5]uint256.Int, error) {
	var out [5]uint256.Int
	if int(id) >= len(p.Venues) {
		return out, ErrBadVenueId
	}
	r, err := p.read(&p.Venues[id], now)
	if err != nil {
		return out, err
	}
	out[0], out[1], out[2], out[3] = r.Collateral, r.RecognizedCollateral, r.Supplied, r.RecognizedSupplied
	if r.Readable {
		out[4] = r.Debt
	}
	return out, nil
}

// venueHealth is MMRouter.venueHealth: maxDebt*lltv/pin*WAD/debt, max for an unreadable or debt-free venue.
func (p *mmRouter) venueHealth(id uint16, priceWad *uint256.Int, now uint64) (uint256.Int, error) {
	if int(id) >= len(p.Venues) {
		return uint256.Int{}, ErrBadVenueId
	}
	v := &p.Venues[id]
	r, err := p.read(v, now)
	if err != nil {
		return uint256.Int{}, err
	}
	if !r.Readable || r.Debt.IsZero() {
		return *mmMaxUint256, nil
	}
	maxDebt, err := p.maxDebtAtPin(v, &r.Collateral, priceWad)
	if err != nil {
		return maxDebt, err
	}
	h, err := gateMul(&maxDebt, &v.LltvWad)
	if err != nil {
		return h, err
	}
	h.Div(&h, &p.PinLtvWad)
	if h, err = gateMul(&h, uWad); err != nil {
		return h, err
	}
	return *h.Div(&h, &r.Debt), nil
}

// ------------------------------------------------------------------ the funding plan

// mmPlan is MMRouterLib.Plan, indexed by venue id.
type mmPlan struct {
	WithdrawTake []uint256.Int
	BorrowSlice  []uint256.Int
	Post         []uint256.Int
	Remaining    uint256.Int
}

// slice is MMRouterLib._slice: one venue's borrow slice and the collateral it must post. The venue must be in
// band and readable; the slice is bounded by the cash this plan has not already withdrawn, the venue debt cap
// and the collateral it could hold at the pin; the rate ceiling is priced against the post-withdrawal supply
// (a slice that fails it is dropped whole, which is why the ceiling is not monotone in the collateral); if the
// posting law needs more than is available, the slice shrinks to what the available collateral carries.
func (p *mmRouter) slice(v *mmVenue, want, collAvail, cashUsed, priceWad *uint256.Int, now uint64) (uint256.Int, uint256.Int, error) {
	var zero uint256.Int
	ok, err := p.bandOk(v, priceWad)
	if err != nil || !ok {
		return zero, zero, err
	}
	rd, err := p.read(v, now)
	if err != nil || !rd.Readable {
		return zero, zero, err
	}
	debt, coll := &rd.Debt, &rd.Collateral
	cash := v.Morpho.freeLiquidity()
	if cash.Gt(cashUsed) {
		cash.Sub(&cash, cashUsed)
	} else {
		cash.Clear()
	}
	sl := *minU(want, &cash)
	if !v.DebtCap.IsZero() {
		var room uint256.Int
		if v.DebtCap.Gt(debt) {
			room.Sub(&v.DebtCap, debt)
		}
		if room.Lt(&sl) {
			sl = room
		}
	}
	collPlus, err := gateAdd(coll, collAvail)
	if err != nil {
		return zero, zero, err
	}
	maxDebt, err := p.maxDebtAtPin(v, &collPlus, priceWad)
	if err != nil {
		return zero, zero, err
	}
	var byColl uint256.Int
	if maxDebt.Gt(debt) {
		byColl.Sub(&maxDebt, debt)
	}
	if byColl.Lt(&sl) {
		sl = byColl
	}
	if sl.IsZero() {
		return zero, zero, nil
	}
	if !v.MaxBorrowRateWad.IsZero() {
		ok, rate, err := v.Morpho.borrowRateAfter(&sl, cashUsed, now)
		if err != nil {
			return zero, zero, err
		}
		if !ok || rate.Gt(&v.MaxBorrowRateWad) {
			return zero, zero, nil
		}
	}
	total, err := gateAdd(debt, &sl)
	if err != nil {
		return zero, zero, err
	}
	need, err := p.requiredCollateral(v, &total, priceWad)
	if err != nil {
		return zero, zero, err
	}
	var post uint256.Int
	if need.Gt(coll) {
		post.Sub(&need, coll)
	}
	if post.Gt(collAvail) {
		post = *collAvail
		collPlus, err = gateAdd(coll, &post)
		if err != nil {
			return zero, zero, err
		}
		if maxDebt, err = p.maxDebtAtPin(v, &collPlus, priceWad); err != nil {
			return zero, zero, err
		}
		sl.Clear()
		if maxDebt.Gt(debt) {
			sl.Sub(&maxDebt, debt)
		}
		if sl.IsZero() {
			return zero, zero, nil
		}
	}
	return sl, post, nil
}

// buildPlan is MMRouterLib.buildPlan, the plan the quote and the execution share: the asset's own recognized
// supply withdrawn first (withdraw order, capped by each market's cash), then -- unless paused, the asset
// borrow-disabled or outside the drawn set -- borrow slices in borrow order under the asset debt cap, with the
// posted collateral drawn from `collateralIn`.
func (p *mmRouter) buildPlan(idx uint8, assets, collateralIn, priceWad *uint256.Int, now uint64) (mmPlan, error) {
	n := len(p.Venues)
	plan := mmPlan{WithdrawTake: make([]uint256.Int, n), BorrowSlice: make([]uint256.Int, n),
		Post: make([]uint256.Int, n), Remaining: *assets}
	for _, id := range p.WithdrawOrder {
		if plan.Remaining.IsZero() {
			break
		}
		v := &p.Venues[id]
		if v.LoanIndex != idx {
			continue
		}
		rd, err := p.read(v, now)
		if err != nil {
			return plan, err
		}
		cash := v.Morpho.freeLiquidity()
		avail := minU(&rd.RecognizedSupplied, &cash)
		take := *minU(&plan.Remaining, avail)
		plan.WithdrawTake[id] = take
		plan.Remaining.Sub(&plan.Remaining, &take)
	}
	if plan.Remaining.IsZero() || p.GlobalPaused {
		return plan, nil
	}
	l, err := p.loanAt(idx)
	if err != nil {
		return plan, err
	}
	if !l.BorrowEnabled {
		return plan, nil
	}
	if ok, err := p.drawable(idx, now); err != nil || !ok {
		return plan, err
	}
	debtRoom, err := p.loanDebtRoom(l, idx, now)
	if err != nil {
		return plan, err
	}
	collAvail := *collateralIn
	for _, id := range p.BorrowOrder {
		if plan.Remaining.IsZero() || debtRoom.IsZero() {
			break
		}
		v := &p.Venues[id]
		if v.LoanIndex != idx || !v.BorrowEnabled {
			continue
		}
		want := minU(&plan.Remaining, &debtRoom)
		sl, post, err := p.slice(v, want, &collAvail, &plan.WithdrawTake[id], priceWad, now)
		if err != nil {
			return plan, err
		}
		if sl.IsZero() {
			continue
		}
		plan.BorrowSlice[id] = sl
		plan.Post[id] = post
		plan.Remaining.Sub(&plan.Remaining, &sl)
		collAvail.Sub(&collAvail, &post)
		debtRoom.Sub(&debtRoom, &sl)
	}
	return plan, nil
}

// fundingCeiling is MMRouter.fundingCeiling: what a max-size plan would raise, `type(uint256).max - remaining`.
func (p *mmRouter) fundingCeiling(idx uint8, collateralIn, priceWad *uint256.Int, now uint64) (uint256.Int, error) {
	plan, err := p.buildPlan(idx, mmMaxUint256, collateralIn, priceWad, now)
	if err != nil {
		return uint256.Int{}, err
	}
	var z uint256.Int
	return *z.Sub(mmMaxUint256, &plan.Remaining), nil
}

// ------------------------------------------------------------------ execution

// requireRate is MMRouterLib.requireRate: the execution-side ceiling with a ZERO supply delta (the plan's
// withdrawals have already settled on the market).
func (p *mmRouter) requireRate(v *mmVenue, assets *uint256.Int, now uint64) error {
	if v.MaxBorrowRateWad.IsZero() {
		return nil
	}
	ok, rate, err := v.Morpho.borrowRateAfter(assets, uZero, now)
	if err != nil {
		return err
	}
	if !ok || rate.Gt(&v.MaxBorrowRateWad) {
		return ErrRateCeiling
	}
	return nil
}

// requireBorrowable is MMRouterLib.requireBorrowable, in its revert order. `debt + assets` is a checked add
// evaluated only where Solidity evaluates it: behind `debtCap != 0` (the && short-circuits), and again for the
// pin comparison after the liquidity check, before maxDebtAtPin (via-IR evaluates binary operands left to right).
func (p *mmRouter) requireBorrowable(v *mmVenue, assets, priceWad *uint256.Int, now uint64) error {
	if p.GlobalPaused {
		return ErrGlobalPaused
	}
	if !v.BorrowEnabled || !p.Loans[v.LoanIndex].BorrowEnabled {
		return ErrVenueDisabled
	}
	if ok, err := p.drawable(v.LoanIndex, now); err != nil {
		return err
	} else if !ok {
		return ErrDrawOutsideDrawnSet
	}
	if ok, err := p.bandOk(v, priceWad); err != nil {
		return err
	} else if !ok {
		return ErrOracleBand
	}
	debt, err := v.Morpho.debtOf(now)
	if err != nil {
		return err
	}
	if !v.DebtCap.IsZero() {
		total, err := gateAdd(&debt, assets)
		if err != nil {
			return err
		}
		if total.Gt(&v.DebtCap) {
			return ErrDebtCapExceeded
		}
	}
	if err := p.requireRate(v, assets, now); err != nil {
		return err
	}
	cash := v.Morpho.freeLiquidity()
	if cash.Lt(assets) {
		return ErrInsufficientLiquidity
	}
	total, err := gateAdd(&debt, assets)
	if err != nil {
		return err
	}
	maxDebt, err := p.maxDebtAtPin(v, &v.Morpho.Position.Collateral, priceWad)
	if err != nil {
		return err
	}
	if total.Gt(&maxDebt) {
		return ErrUnhealthy
	}
	return nil
}

// withdrawSupplied is MMRouterLib.withdrawSupplied: max withdraws the managed shares, else an asset amount
// bounded by the recognized supply (a zero amount refuses before the supply is valued: the || short-circuits);
// the managed shares shrink by what burned (floored at zero).
func (p *mmRouter) withdrawSupplied(v *mmVenue, assets *uint256.Int, now uint64) (uint256.Int, error) {
	var withdrawn, burned uint256.Int
	var err error
	if assets.Eq(mmMaxUint256) {
		shares := p.managedShares(v)
		if shares.IsZero() {
			return withdrawn, nil
		}
		withdrawn, burned, err = v.Morpho.accountWithdraw(uZero, &shares, now)
	} else {
		if assets.IsZero() {
			return withdrawn, ErrInsufficientLiquidity
		}
		rs, rerr := p.recognizedSupplied(v, now)
		if rerr != nil {
			return withdrawn, rerr
		}
		if assets.Gt(&rs) {
			return withdrawn, ErrInsufficientLiquidity
		}
		withdrawn, burned, err = v.Morpho.accountWithdraw(assets, uZero, now)
	}
	if err != nil {
		return withdrawn, err
	}
	if burned.Gt(&v.ManagedSupplyShares) {
		v.ManagedSupplyShares.Clear()
	} else {
		v.ManagedSupplyShares.Sub(&v.ManagedSupplyShares, &burned)
	}
	return withdrawn, nil
}

// fund is MMRouterLib.fund: build the plan (any remainder reverts InsufficientLiquidity), drain every withdraw
// leg, then per borrow leg post its collateral, re-check it with requireBorrowable and borrow.
func (p *mmRouter) fund(idx uint8, assets, collateralIn, priceWad *uint256.Int, now uint64) (withdrawn, borrowed, posted uint256.Int, err error) {
	plan, err := p.buildPlan(idx, assets, collateralIn, priceWad, now)
	if err != nil {
		return
	}
	if !plan.Remaining.IsZero() {
		err = ErrInsufficientLiquidity
		return
	}
	for _, id := range p.WithdrawOrder {
		take := &plan.WithdrawTake[id]
		if take.IsZero() {
			continue
		}
		w, werr := p.withdrawSupplied(&p.Venues[id], take, now)
		if werr != nil {
			err = werr
			return
		}
		withdrawn.Add(&withdrawn, &w)
	}
	for _, id := range p.BorrowOrder {
		sl := &plan.BorrowSlice[id]
		if sl.IsZero() {
			continue
		}
		v := &p.Venues[id]
		if post := &plan.Post[id]; !post.IsZero() {
			if err = v.Morpho.accountSupplyCollateral(post); err != nil {
				return
			}
			v.ManagedCollateral.Add(&v.ManagedCollateral, post)
			posted.Add(&posted, post)
		}
		if err = p.requireBorrowable(v, sl, priceWad, now); err != nil {
			return
		}
		if err = v.Morpho.accountBorrow(sl, now); err != nil {
			return
		}
		borrowed.Add(&borrowed, sl)
	}
	return
}

// repayCascade is MMRouterLib.repayCascade: repay order over the asset's readable indebted venues; each leg is
// min(remaining, debt) and counts what the account actually repaid (a remainder goes back to the pool).
func (p *mmRouter) repayCascade(idx uint8, assets *uint256.Int, now uint64) (uint256.Int, error) {
	var repaid uint256.Int
	if _, err := p.loanAt(idx); err != nil {
		return repaid, err
	}
	remaining := *assets
	for _, id := range p.RepayOrder {
		if remaining.IsZero() {
			break
		}
		v := &p.Venues[id]
		if v.LoanIndex != idx {
			continue
		}
		rd, err := p.read(v, now)
		if err != nil {
			return repaid, err
		}
		if !rd.Readable || rd.Debt.IsZero() {
			continue
		}
		pay := *minU(&remaining, &rd.Debt)
		if err := p.snapshotForProportional(v, id, now); err != nil {
			return repaid, err
		}
		got, err := v.Morpho.accountRepay(&pay, now)
		if err != nil {
			return repaid, err
		}
		repaid.Add(&repaid, &got)
		remaining.Sub(&remaining, &got)
	}
	return repaid, nil
}

// supplyCascade is MMRouterLib.supplyCascade: nothing while paused; supply order over the asset's
// supply-enabled venues, each take bounded by the venue's and the asset's remaining supply room.
func (p *mmRouter) supplyCascade(idx uint8, assets *uint256.Int, now uint64) (uint256.Int, error) {
	var supplied uint256.Int
	if p.GlobalPaused {
		return supplied, nil
	}
	l, err := p.loanAt(idx)
	if err != nil {
		return supplied, err
	}
	remaining := *assets
	assetRoom, err := p.loanSupplyRoom(l, idx, now)
	if err != nil {
		return supplied, err
	}
	for _, id := range p.SupplyOrder {
		if remaining.IsZero() || assetRoom.IsZero() {
			break
		}
		v := &p.Venues[id]
		if v.LoanIndex != idx || !v.SupplyEnabled {
			continue
		}
		room, err := p.supplyRoom(v, now)
		if err != nil {
			return supplied, err
		}
		if room.Gt(&assetRoom) {
			room = assetRoom
		}
		take := *minU(&remaining, &room)
		if take.IsZero() {
			continue
		}
		shares, err := v.Morpho.accountSupply(&take, now)
		if err != nil {
			return supplied, err
		}
		v.ManagedSupplyShares.Add(&v.ManagedSupplyShares, &shares)
		supplied.Add(&supplied, &take)
		remaining.Sub(&remaining, &take)
		assetRoom.Sub(&assetRoom, &take)
	}
	return supplied, nil
}

// reclaim is MMRouterLib.reclaim: REVERSE borrow order, each venue giving up its free collateral. The strict
// entry (MMRouter.reclaim) reverts InsufficientCollateral on a shortfall; reclaimBestEffort returns what it got.
func (p *mmRouter) reclaim(assets *uint256.Int, priceWads []uint256.Int, strict bool, now uint64) (uint256.Int, error) {
	var got uint256.Int
	if len(priceWads) != len(p.Loans) {
		return got, ErrInvalidConfig
	}
	remaining := *assets
	for i := len(p.BorrowOrder); i > 0 && !remaining.IsZero(); i-- {
		v := &p.Venues[p.BorrowOrder[i-1]]
		free, err := p.freeCollateral(v, &priceWads[v.LoanIndex], now)
		if err != nil {
			return got, err
		}
		take := *minU(&remaining, &free)
		if take.IsZero() {
			continue
		}
		if err := v.Morpho.accountWithdrawCollateral(&take, now); err != nil {
			return got, err
		}
		if take.Gt(&v.ManagedCollateral) {
			return got, errPanicArithmetic
		}
		v.ManagedCollateral.Sub(&v.ManagedCollateral, &take)
		got.Add(&got, &take)
		remaining.Sub(&remaining, &take)
	}
	if strict && !got.Eq(assets) {
		return got, ErrInsufficientCollateral
	}
	return got, nil
}

// ------------------------------------------------------------------ FLAMMSwapLib settlement legs

// mmPayLoan is FLAMMSwapLib.payLoan: pay `net` of loan asset idx from tracked liquid, funding the shortfall
// through fund with the whole physical poolAsset offered as collateral; the posted collateral leaves physical and
// the withdrawn + borrowed cash lands in liquid.
func mmPayLoan(pool *gatePool, r *mmRouter, idx uint8, priceWad, net *uint256.Int, now uint64) error {
	cfg := &pool.Loans[idx]
	if net.Gt(&cfg.Liquid) {
		var rest uint256.Int
		rest.Sub(net, &cfg.Liquid)
		coll := pool.Physical
		withdrawn, borrowed, posted, err := r.fund(idx, &rest, &coll, priceWad, now)
		if err != nil {
			return err
		}
		pool.Physical.Sub(&pool.Physical, &posted)
		cfg.Liquid.Add(&cfg.Liquid, &withdrawn)
		cfg.Liquid.Add(&cfg.Liquid, &borrowed)
	}
	if net.Gt(&cfg.Liquid) {
		return errPanicArithmetic
	}
	cfg.Liquid.Sub(&cfg.Liquid, net)
	return nil
}

// mmTakeLoan is FLAMMSwapLib.takeLoan: `used` lands in liquid, repays the asset's counted debt through
// repayCascade, and the surplus above the reserve target is lent through supplyCascade when lending is on.
func mmTakeLoan(pool *gatePool, r *mmRouter, idx uint8, used *uint256.Int, now uint64) error {
	cfg := &pool.Loans[idx]
	cfg.Liquid.Add(&cfg.Liquid, used)
	_, _, debt, err := r.position(idx, now)
	if err != nil {
		return err
	}
	if pay := minU(&debt, &cfg.Liquid); !pay.IsZero() {
		repaid, err := r.repayCascade(idx, pay, now)
		if err != nil {
			return err
		}
		cfg.Liquid.Sub(&cfg.Liquid, &repaid)
	}
	if cfg.Liquid.Gt(&cfg.ReserveTarget) && !new(uint256.Int).And(&pool.Features, gateFeatureSupplyLending).IsZero() {
		var excess uint256.Int
		excess.Sub(&cfg.Liquid, &cfg.ReserveTarget)
		supplied, err := r.supplyCascade(idx, &excess, now)
		if err != nil {
			return err
		}
		cfg.Liquid.Sub(&cfg.Liquid, &supplied)
	}
	return nil
}

// mmReleaseExcess is FLAMMSwapLib.releaseExcess: posted collateral above the posting law, released best-effort
// once the excess is at least RELEASE_HYSTERESIS of posted; a Router revert is swallowed (its writes undone).
func mmReleaseExcess(pool *gatePool, r *mmRouter, now uint64) error {
	b, err := gatePriced(pool, r, now)
	if err != nil {
		return err
	}
	for i := range b.Legs {
		if !b.Legs[i].Debt.IsZero() && b.Legs[i].PriceWad.IsZero() {
			return nil
		}
	}
	need, err := gateRequiredPostedAll(&b, &pool.LtvWad)
	if err != nil {
		return err
	}
	if !b.Posted.Gt(&need) {
		return nil
	}
	var excess uint256.Int
	excess.Sub(&b.Posted, &need)
	frac, err := mmMulDivOZ(&excess, uWad, &b.Posted)
	if err != nil {
		return err
	}
	if frac.Lt(gateReleaseHysteresis) {
		return nil
	}
	trial := r.clone()
	got, err := trial.reclaim(&excess, gatePriceWads(&b), false, now)
	if err != nil {
		return nil
	}
	*r = *trial
	pool.Physical.Add(&pool.Physical, &got)
	return nil
}

// mmSettleSell is FLAMMSwapLib.settleSell as one swap transaction: run on clones, written to the pool and router
// only on success, with the transient repay snapshots ending with the transaction.
func mmSettleSell(pool *gatePool, r *mmRouter, idx uint8, priceWad, used, net *uint256.Int, now uint64) error {
	pc, rc := pool.clone(), r.clone()
	if err := mmSettleSellLegs(pc, rc, idx, priceWad, used, net, now); err != nil {
		return err
	}
	rc.endTransaction()
	*pool, *r = *pc, *rc
	return nil
}

// mmSettleSellLegs is the body of FLAMMSwapLib.settleSell, writing through: `used` poolAsset in, `net` of loan
// asset idx paid out at the leg's checked cross `priceWad`, then assertGate.
func mmSettleSellLegs(pc *gatePool, rc *mmRouter, idx uint8, priceWad, used, net *uint256.Int, now uint64) error {
	pc.Physical.Add(&pc.Physical, used)
	if err := mmPayLoan(pc, rc, idx, priceWad, net, now); err != nil {
		return err
	}
	return gateAssertGate(pc, rc, now)
}

// mmSettleBuy is the buy branch of FLAMMSwapLib.execute as one swap transaction: run on clones, written to the
// pool and router only on success, with the transient repay snapshots ending with the transaction.
func mmSettleBuy(pool *gatePool, r *mmRouter, idx uint8, used, net *uint256.Int, now uint64) error {
	pc, rc := pool.clone(), r.clone()
	if err := mmSettleBuyLegs(pc, rc, idx, used, net, now); err != nil {
		return err
	}
	rc.endTransaction()
	*pool, *r = *pc, *rc
	return nil
}

// mmSettleBuyLegs is the buy branch of FLAMMSwapLib.execute, writing through: anchor, settleBuy (takeLoan;
// reclaim the payout beyond physical, strictly, at the pool's price vector; pay `net` poolAsset; releaseExcess),
// then the entry gate against the anchor.
func mmSettleBuyLegs(pc *gatePool, rc *mmRouter, idx uint8, used, net *uint256.Int, now uint64) error {
	u0, gross0, q0, err := gateAnchor(pc, rc, now)
	if err != nil {
		return err
	}
	if err := mmTakeLoan(pc, rc, idx, used, now); err != nil {
		return err
	}
	if pc.Physical.Lt(net) {
		var short uint256.Int
		short.Sub(net, &pc.Physical)
		got, err := rc.reclaim(&short, pc.PriceWad, true, now)
		if err != nil {
			return err
		}
		pc.Physical.Add(&pc.Physical, &got)
	}
	if pc.Physical.Lt(net) {
		return errPanicArithmetic
	}
	pc.Physical.Sub(&pc.Physical, net)
	if err := mmReleaseExcess(pc, rc, now); err != nil {
		return err
	}
	return gateAssertEntryGate(pc, rc, now, u0, gross0, q0)
}

// ------------------------------------------------------------------ MMRouter single-venue entries

// mmRepaySnapshot is the (debt, collateral) MMRouterLib.snapshotForProportional parks in transient storage before
// a repay, read back by a proportional withdrawCollateral in the same transaction.
type mmRepaySnapshot struct {
	Debt uint256.Int
	Coll uint256.Int
}

// mmPackRepaySnapshot is the tstore / tload round trip of the snapshot: packed = (debt << 128) | coll, read back
// as (packed >> 128, packed & type(uint128).max). The unchecked shift drops a debt's bits at and above 2^128.
func mmPackRepaySnapshot(debt, coll *uint256.Int) mmRepaySnapshot {
	var packed uint256.Int
	packed.Lsh(debt, 128)
	packed.Or(&packed, coll)
	var s mmRepaySnapshot
	s.Debt.Rsh(&packed, 128)
	s.Coll.And(&packed, mmMaxUint128)
	return s
}

// snapshotForProportional is MMRouterLib.snapshotForProportional; the snapshot lives in TransientRepay until
// endTransaction.
func (p *mmRouter) snapshotForProportional(v *mmVenue, id uint16, now uint64) error {
	debt, err := v.Morpho.debtOf(now)
	if err != nil {
		return err
	}
	if p.TransientRepay == nil {
		p.TransientRepay = make(map[uint16]mmRepaySnapshot)
	}
	p.TransientRepay[id] = mmPackRepaySnapshot(&debt, &v.Morpho.Position.Collateral)
	return nil
}

// endTransaction clears the transient repay snapshots, as EIP-1153 storage is at the end of a transaction: a
// proportional withdrawCollateral in a later transaction finds none and reverts NoRepaySnapshot.
func (p *mmRouter) endTransaction() {
	p.TransientRepay = nil
}

func (p *mmRouter) venueAt(id uint16) (*mmVenue, error) {
	if int(id) >= len(p.Venues) {
		return nil, ErrBadVenueId
	}
	return &p.Venues[id], nil
}

func (p *mmRouter) live(id uint16) (*mmVenue, error) {
	v, err := p.venueAt(id)
	if err == nil && v.Retired {
		return nil, ErrVenueIsRetired
	}
	return v, err
}

// postCollateral is MMRouter.postCollateral.
func (p *mmRouter) postCollateral(id uint16, assets *uint256.Int) error {
	v, err := p.live(id)
	if err != nil {
		return err
	}
	if assets.IsZero() {
		return ErrInvalidConfig
	}
	if err := v.Morpho.accountSupplyCollateral(assets); err != nil {
		return err
	}
	v.ManagedCollateral.Add(&v.ManagedCollateral, assets)
	return nil
}

// withdrawCollateral is MMRouterLib.withdrawCollateral: bounded by the recognized collateral; an indebted venue
// must stay no worse than its repay snapshot (proportional) or within the pin at an in-band cross.
func (p *mmRouter) withdrawCollateral(id uint16, assets, priceWad *uint256.Int, proportional bool, now uint64) error {
	v, err := p.venueAt(id)
	if err != nil {
		return err
	}
	recognized := minU(&v.Morpho.Position.Collateral, &v.ManagedCollateral)
	if assets.IsZero() || assets.Gt(recognized) {
		return ErrInsufficientCollateral
	}
	debt, err := v.Morpho.debtOf(now)
	if err != nil {
		return err
	}
	var rest uint256.Int
	rest.Sub(&v.Morpho.Position.Collateral, assets)
	if !debt.IsZero() {
		if proportional {
			snap := p.TransientRepay[id]
			if snap.Coll.IsZero() {
				return ErrNoRepaySnapshot
			}
			lhs, err := gateMul(&debt, &snap.Coll)
			if err != nil {
				return err
			}
			rhs, err := gateMul(&snap.Debt, &rest)
			if err != nil {
				return err
			}
			if lhs.Gt(&rhs) {
				return ErrUnhealthy
			}
		} else {
			if ok, err := p.bandOk(v, priceWad); err != nil {
				return err
			} else if !ok {
				return ErrOracleBand
			}
			maxDebt, err := p.maxDebtAtPin(v, &rest, priceWad)
			if err != nil {
				return err
			}
			if debt.Gt(&maxDebt) {
				return ErrUnhealthy
			}
		}
	}
	if err := v.Morpho.accountWithdrawCollateral(assets, now); err != nil {
		return err
	}
	v.ManagedCollateral.Sub(&v.ManagedCollateral, assets)
	return nil
}

// borrow is MMRouter.borrow: requireBorrowable, then the asset debt cap.
func (p *mmRouter) borrow(id uint16, assets, priceWad *uint256.Int, now uint64) error {
	v, err := p.live(id)
	if err != nil {
		return err
	}
	if assets.IsZero() {
		return ErrInvalidConfig
	}
	if err := p.requireBorrowable(v, assets, priceWad, now); err != nil {
		return err
	}
	room, err := p.loanDebtRoom(&p.Loans[v.LoanIndex], v.LoanIndex, now)
	if err != nil {
		return err
	}
	if assets.Gt(&room) {
		return ErrDebtCapExceeded
	}
	return v.Morpho.accountBorrow(assets, now)
}

// repay is MMRouter.repay: snapshot, then min(assets, debtOf) repaid into the account.
func (p *mmRouter) repay(id uint16, assets *uint256.Int, now uint64) (uint256.Int, error) {
	v, err := p.venueAt(id)
	if err != nil {
		return uint256.Int{}, err
	}
	if err := p.snapshotForProportional(v, id, now); err != nil {
		return uint256.Int{}, err
	}
	debt, err := v.Morpho.debtOf(now)
	if err != nil {
		return uint256.Int{}, err
	}
	return v.Morpho.accountRepay(minU(assets, &debt), now)
}

// supply is MMRouter.supply: requireSuppliable, the asset supply cap, then the managed shares grow by the mint.
func (p *mmRouter) supply(id uint16, assets *uint256.Int, now uint64) (uint256.Int, error) {
	v, err := p.live(id)
	if err != nil {
		return uint256.Int{}, err
	}
	if assets.IsZero() {
		return uint256.Int{}, ErrInvalidConfig
	}
	if p.GlobalPaused {
		return uint256.Int{}, ErrGlobalPaused
	}
	if !v.SupplyEnabled {
		return uint256.Int{}, ErrVenueDisabled
	}
	room, err := p.supplyRoom(v, now)
	if err != nil {
		return uint256.Int{}, err
	}
	if assets.Gt(&room) {
		return uint256.Int{}, ErrSupplyCapExceeded
	}
	if room, err = p.loanSupplyRoom(&p.Loans[v.LoanIndex], v.LoanIndex, now); err != nil {
		return uint256.Int{}, err
	}
	if assets.Gt(&room) {
		return uint256.Int{}, ErrSupplyCapExceeded
	}
	shares, err := v.Morpho.accountSupply(assets, now)
	if err != nil {
		return shares, err
	}
	v.ManagedSupplyShares.Add(&v.ManagedSupplyShares, &shares)
	return shares, nil
}

// withdrawSuppliedEntry is MMRouter.withdrawSupplied.
func (p *mmRouter) withdrawSuppliedEntry(id uint16, assets *uint256.Int, now uint64) (uint256.Int, error) {
	v, err := p.venueAt(id)
	if err != nil {
		return uint256.Int{}, err
	}
	return p.withdrawSupplied(v, assets, now)
}
