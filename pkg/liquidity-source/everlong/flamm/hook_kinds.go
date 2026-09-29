package everlongflamm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// A pool's hooks as the tracker reads them and the settlement calls them, one implementation per registered kind
// (hook_registry.go). A kind supplies its tracker reads and state build (hookKindSpec) and its quote steps and
// post-fill commit (the role ports below). The pool state holds, per role, the kind as a tag beside that kind's
// concrete state (poolHooks), and every step resolves the tag to its port: no interface is stored, because an
// interface-typed field survives Kyber's msgpack encoder only when its concrete type is registered with the
// encoder, by hand, in the shared pkg/msgpack/register_types.go (the generated register_pool_types.gen.go registers
// simulators alone). Unregistered, the encoder still writes the value -- as a bare array, with no type tag -- and
// the decoder panics on it ("reflect.Set: value of type []interface {} is not assignable to ..."), so every kind
// added here would owe an entry in another package, whose absence fails at runtime and not at build time.

// errHookUnported refuses a role whose kind the state carries no port for.
var errHookUnported = fmt.Errorf("%w: hook kind has no port", ErrInvalidProfile)

// swapHookPort is a swap-role hook on the swap path: spot, the fee role's previewFeeWad / executeFeeWad and the
// invariant role's previewExactIn / executeExactIn (FLAMMSwapLib.sol:84, :87, :149, :187, :197), with the book the
// fill materialised, which the execution commits.
type swapHookPort interface {
	spot() (uint256.Int, error)
	previewFeeWad(ctx *swapContext) (uint256.Int, error)
	fill(ctx *swapContext, feeWad *uint256.Int) (hookFillResult, hookBook, error)
	commit(b *hookBook)
}

// levBookSource is what EverlongLeverageHook reads from the swap hook it is bound to: EverlongHook.bookFor and
// reservationPriceWad (EverlongLeverageHook.sol:74, :77, the two reads of frame()).
type levBookSource interface {
	bookFor(ctx *poolContext) (hookBook, error)
	reservationPrice() uint256.Int
}

// leverageHookPort is a leverage-role hook's ILeverageInvariantHook.previewLever / executeLever
// (FLAMMLeverLib.sol:100, :128, :183) over the pool's swap hook.
type leverageHookPort interface {
	previewLever(ctx *leverContext, swap swapHookPort) (*levFill, error)
}

// spreadHookPort is a spread-role hook's answer to FLAMMLeverLib._spread's staticcall (FLAMMLeverLib.sol:163): the
// answer at now, whether it stays live through now + margin with a ppm below PPM, and the staleness deadline a live
// answer carries (lastSet + maxAge; ok false when it carries none).
type spreadHookPort interface {
	spreadPpm(now uint64) (bool, uint256.Int)
	liveThrough(now, margin uint64) bool
	expiry(now uint64) (lastSet, maxAge uint256.Int, ok bool)
}

// swapBoundKind is a kind that quotes on the pool's swap hook, and names the swap kinds it can read.
type swapBoundKind interface {
	quotesOn(swap hookKind) bool
}

// quotesOnSwapKind reports whether a leverage kind's spec can quote on the swap kind swap. A spec that is no
// swapBoundKind names no swap kind at all, so it quotes on none: the rule fails closed, as the settlement path's own
// assertion does (previewLever, errHookUnported). Every leverage-role spec implements it (TestRegistryWellFormed).
func quotesOnSwapKind(lev hookKindSpec, swap hookKind) bool {
	sb, ok := lev.(swapBoundKind)
	return ok && sb.quotesOn(swap)
}

var (
	_ swapHookPort     = (*hookState)(nil)
	_ levBookSource    = (*hookState)(nil)
	_ leverageHookPort = everlongLeverageV1{}
	_ spreadHookPort   = (*spreadHookState)(nil)
	_ swapBoundKind    = everlongLeverageV1{}
)

// hookBinding is what a hook reports it is bound to: POOL() for every kind, HOOK() for a leverage hook, LOAN_SCALE()
// for a swap or leverage hook and genesisStrategyHash() for a swap hook. The lister and every refresh read it back
// and compare it with the hook's entry.
type hookBinding struct {
	Pool      common.Address
	Hook      common.Address
	LoanScale uint256.Int
	Genesis   common.Hash
}

// hookKindSpec is a hook kind as the lister and the tracker see it.
type hookKindSpec interface {
	// role is the hooks() role the kind fills.
	role() hookRole
	// listingReads adds, to the listing's view round, the reads of what binds a hook of the kind (into b).
	listingReads(p *readPlan, hook common.Address, b *hookBinding)
	// refreshReads adds, to a refresh's view round, the reads of the hook's state (into r) and of its binding (b).
	refreshReads(p *readPlan, hook common.Address, r *flammReads, b *hookBinding)
	// bound reports whether b is what e's hook reports on pool, whose invariant hook is invariant.
	bound(e *hookEntry, b *hookBinding, pool, invariant common.Address) bool
	// build sets the kind's role in s from r, whose pool part s is already built from.
	build(r *flammReads, s *flammState) error
	// dependency reports whether the hook emits events that move a quote with none from the pool, which makes it a
	// refresh trigger (pool_tracker.go resolveDependencies).
	dependency() bool
}

var hookKindSpecs = [hookKindCount]hookKindSpec{
	hookKindEverlongSwapV1:     everlongSwapV1{},
	hookKindEverlongLeverageV1: everlongLeverageV1{},
	hookKindEverlongSpreadV1:   everlongSpreadV1{},
}

// hookKindSpecOf is k's spec, or nil for a kind with none.
func hookKindSpecOf(k hookKind) hookKindSpec {
	if k < hookKindCount {
		return hookKindSpecs[k]
	}
	return nil
}

// poolHooks is the pool's hook set in the state: the listed addresses and, per role, the kind the registry names
// with that kind's state (a nil state for a stateless kind or an empty role).
type poolHooks struct {
	Addrs    [7]common.Address `json:"addrs"`
	Swap     swapHookSlot      `json:"swap"`
	Leverage leverageHookSlot  `json:"leverage"`
	Spread   spreadHookSlot    `json:"spread"`
}

// clone deep-copies the kinds' states: an execution commits the swap hook's book in place.
func (h *poolHooks) clone() poolHooks {
	c := *h
	if h.Swap.EverlongSwap != nil {
		s := *h.Swap.EverlongSwap
		c.Swap.EverlongSwap = &s
	}
	if h.Spread.EverlongSpread != nil {
		s := *h.Spread.EverlongSpread
		c.Spread.EverlongSpread = &s
	}
	return c
}

func (h *poolHooks) hasLeverage() bool { return h.Leverage.Kind != hookKindNone }

func (h *poolHooks) hasSpread() bool { return h.Spread.Kind != hookKindNone }

// matches refuses a state whose hook set is not want, the listing's as the registry resolves it: the same addresses
// and, per role, the kind the registry names with a port for it.
func (h *poolHooks) matches(want *poolHookSet) error {
	if h.Addrs != want.Addrs {
		return fmt.Errorf("%w: state hook set", ErrInvalidProfile)
	}
	if h.Swap.Kind != want.kind(roleSwap) || h.Leverage.Kind != want.kind(roleLeverage) ||
		h.Spread.Kind != want.kind(roleSpread) {
		return fmt.Errorf("%w: state hook kinds", ErrInvalidProfile)
	}
	if _, err := h.Swap.port(); err != nil {
		return err
	}
	if h.hasLeverage() {
		if _, err := h.Leverage.port(); err != nil {
			return err
		}
	}
	if h.hasSpread() {
		if _, err := h.Spread.port(); err != nil {
			return err
		}
	}
	return nil
}

// swapHookSlot is the swap role: its kind and that kind's state.
type swapHookSlot struct {
	Kind         hookKind   `json:"kind"`
	EverlongSwap *hookState `json:"everlongSwap,omitempty"`
}

func (s *swapHookSlot) port() (swapHookPort, error) {
	if s.Kind == hookKindEverlongSwapV1 && s.EverlongSwap != nil {
		return s.EverlongSwap, nil
	}
	return nil, errHookUnported
}

// leverageHookSlot is the leverage role: its kind (the ported kind is stateless).
type leverageHookSlot struct {
	Kind hookKind `json:"kind"`
}

func (s *leverageHookSlot) port() (leverageHookPort, error) {
	if s.Kind == hookKindEverlongLeverageV1 {
		return everlongLeverageV1{}, nil
	}
	return nil, errHookUnported
}

// spreadHookSlot is the spread role: its kind and that kind's state.
type spreadHookSlot struct {
	Kind           hookKind         `json:"kind"`
	EverlongSpread *spreadHookState `json:"everlongSpread,omitempty"`
}

func (s *spreadHookSlot) port() (spreadHookPort, error) {
	if s.Kind == hookKindEverlongSpreadV1 && s.EverlongSpread != nil {
		return s.EverlongSpread, nil
	}
	return nil, errHookUnported
}

// spreadPpm is what FLAMMLeverLib._spread's staticcall gets at now (FLAMMLeverLib.sol:163-165). An empty spread role
// answers nothing: the staticcall to address(0) succeeds with empty data, which does not decode.
func (s *spreadHookSlot) spreadPpm(now uint64) (bool, uint256.Int, error) {
	if s.Kind == hookKindNone {
		return false, uint256.Int{}, nil
	}
	p, err := s.port()
	if err != nil {
		return false, uint256.Int{}, err
	}
	ok, ppm := p.spreadPpm(now)
	return ok, ppm, nil
}

// everlongSwapV1 is EverlongHook (hook.go, fee.go, almcurve.go).
type everlongSwapV1 struct{}

func (everlongSwapV1) role() hookRole { return roleSwap }

func (everlongSwapV1) listingReads(p *readPlan, hook common.Address, b *hookBinding) {
	p.add("hook.POOL", hook, &hookABI, "POOL", nil, readAddr(&b.Pool))
	p.add("hook.LOAN_SCALE", hook, &hookABI, "LOAN_SCALE", nil, readWord(&b.LoanScale))
	p.add("hook.genesisStrategyHash", hook, &hookABI, "genesisStrategyHash", nil, readHash(&b.Genesis))
}

// refreshReads reads the storage the swap path reads (hookReads) and the binding; LOAN_SCALE is read once for both.
func (everlongSwapV1) refreshReads(p *readPlan, hook common.Address, r *flammReads, b *hookBinding) {
	h := &hookReads{}
	r.Hook = h
	p.add("hook.params", hook, &hookABI, "params", nil, func(vals []any) error {
		hp, err := tupleOf[abiHookParams](vals[0])
		if err != nil || hp.AWad == nil {
			return errReadDecode
		}
		if h.AWad, err = wordOf(hp.AWad); err != nil {
			return err
		}
		f := hp.Tuning.Fee
		for i, v := range []uint64{f.MidFeeWad, f.OutFeeWad, f.GammaWad, f.SigmaRefWad, f.VolBetaWad, f.VolMinWad,
			f.VolMaxWad, f.DirSkewWad} {
			h.Fee[i].SetUint64(v)
		}
		h.InvSkewKappaWad.SetUint64(hp.Tuning.InvSkewKappaWad)
		h.InvSkewBandWad.SetUint64(hp.Tuning.InvSkewBandWad)
		return nil
	})
	p.add("hook.support", hook, &hookABI, "support", nil, func(vals []any) error {
		sup, err := tupleOf[abiSupport](vals[0])
		if err != nil {
			return err
		}
		return wordsInto([]any{sup.AWad, sup.XLo, sup.XHi, sup.YHi}, &h.Support[0], &h.Support[1], &h.Support[2],
			&h.Support[3])
	})
	for _, w := range []struct {
		method string
		dst    *uint256.Int
	}{{"anchorSqrtX96", &h.AnchorSqrtX96}, {"reservationPriceWad", &h.ReservationPriceWad}, {"kappa", &h.Kappa},
		{"xWad", &h.XWad}, {"reserveStable", &h.ReserveStable}, {"idleStable", &h.IdleStable},
		{"reserveVolatile", &h.ReserveVolatile}, {"idleVolatile", &h.IdleVolatile}, {"rvWad", &h.RvWad}} {
		p.add("hook."+w.method, hook, &hookABI, w.method, nil, readWord(w.dst))
	}
	p.add("hook.LOAN_SCALE", hook, &hookABI, "LOAN_SCALE", nil, func(vals []any) error {
		if err := wordsInto(vals, &h.LoanScale); err != nil {
			return err
		}
		b.LoanScale = h.LoanScale
		return nil
	})
	p.add("hook.POOL", hook, &hookABI, "POOL", nil, readAddr(&b.Pool))
	p.add("hook.genesisStrategyHash", hook, &hookABI, "genesisStrategyHash", nil, readHash(&b.Genesis))
}

func (everlongSwapV1) bound(e *hookEntry, b *hookBinding, pool, _ common.Address) bool {
	scale := e.loanScale()
	return b.Pool == pool && b.Genesis == e.GenesisStrategyHash && b.LoanScale.Eq(&scale)
}

// build sets the swap role to the hook's storage; its LOAN_SCALE must be loan asset 0's.
func (everlongSwapV1) build(r *flammReads, s *flammState) error {
	h := r.Hook
	if h == nil {
		return errReadsInconsistent
	}
	st := &hookState{AWad: h.AWad, AnchorSqrtX96: h.AnchorSqrtX96, ReservationPriceWad: h.ReservationPriceWad,
		Kappa: h.Kappa, XWad: h.XWad, ReserveStable: h.ReserveStable, IdleStable: h.IdleStable,
		ReserveVolatile: h.ReserveVolatile, IdleVolatile: h.IdleVolatile, RvWad: h.RvWad,
		InvSkewKappaWad: h.InvSkewKappaWad, InvSkewBandWad: h.InvSkewBandWad, LoanScale: h.LoanScale,
		Support: almSupport{AWad: h.Support[0], XLo: h.Support[1], XHi: h.Support[2], YHi: h.Support[3]},
		Fee: feeParams{MidFeeWad: h.Fee[0], OutFeeWad: h.Fee[1], GammaWad: h.Fee[2], SigmaRefWad: h.Fee[3],
			VolBetaWad: h.Fee[4], VolMinWad: h.Fee[5], VolMaxWad: h.Fee[6], DirSkewWad: h.Fee[7]}}
	if !h.LoanScale.Eq(&s.Pool.Loans[0].Scale) {
		return errReadsInconsistent
	}
	s.Hooks.Swap = swapHookSlot{Kind: hookKindEverlongSwapV1, EverlongSwap: st}
	return nil
}

// dependency: a keeper's setFeeRow / _setTuning changes every fee and emits only the hook's TuningChanged.
func (everlongSwapV1) dependency() bool { return true }

// reservationPrice is EverlongHook.reservationPriceWad().
func (s *hookState) reservationPrice() uint256.Int { return s.ReservationPriceWad }

// everlongLeverageV1 is EverlongLeverageHook (levhook.go, levcurve.go). It is stateless and quotes on the book of
// the swap hook it is bound to (HOOK()), so it carries nothing in the state.
type everlongLeverageV1 struct{}

func (everlongLeverageV1) role() hookRole { return roleLeverage }

func (everlongLeverageV1) listingReads(p *readPlan, hook common.Address, b *hookBinding) {
	p.add("levHook.HOOK", hook, &levHookABI, "HOOK", nil, readAddr(&b.Hook))
	p.add("levHook.POOL", hook, &levHookABI, "POOL", nil, readAddr(&b.Pool))
	p.add("levHook.LOAN_SCALE", hook, &levHookABI, "LOAN_SCALE", nil, readWord(&b.LoanScale))
}

// refreshReads is the binding alone: the hook has no storage of its own.
func (k everlongLeverageV1) refreshReads(p *readPlan, hook common.Address, _ *flammReads, b *hookBinding) {
	k.listingReads(p, hook, b)
}

func (everlongLeverageV1) bound(e *hookEntry, b *hookBinding, pool, invariant common.Address) bool {
	scale := e.loanScale()
	return b.Hook == invariant && b.Pool == pool && b.LoanScale.Eq(&scale)
}

func (everlongLeverageV1) build(_ *flammReads, s *flammState) error {
	s.Hooks.Leverage = leverageHookSlot{Kind: hookKindEverlongLeverageV1}
	return nil
}

func (everlongLeverageV1) dependency() bool { return false }

func (everlongLeverageV1) quotesOn(swap hookKind) bool { return swap == hookKindEverlongSwapV1 }

// previewLever is the hook's _quote(ctx) (previewLever and executeLever alike, the hook being stateless) on the swap
// hook's bookFor(ctx.pool) and reservationPriceWad.
func (everlongLeverageV1) previewLever(ctx *leverContext, swap swapHookPort) (*levFill, error) {
	src, ok := swap.(levBookSource)
	if !ok {
		return nil, errHookUnported
	}
	hb, err := src.bookFor(&ctx.Pool)
	if err != nil {
		return nil, err
	}
	rp := src.reservationPrice()
	return levQuote(ctx, &levBook{Rs: &hb.Rs, Is: &hb.Is, Rv: &hb.Rv, Iv: &hb.Iv, ReservationPriceWad: &rp})
}

// everlongSpreadV1 is LeverageSpreadHook (spreadHookState).
type everlongSpreadV1 struct{}

func (everlongSpreadV1) role() hookRole { return roleSpread }

func (everlongSpreadV1) listingReads(p *readPlan, hook common.Address, b *hookBinding) {
	p.add("spreadHook.POOL", hook, &spreadHookABI, "POOL", nil, readAddr(&b.Pool))
}

// refreshReads reads the post core's staticcall reads (spreadHookState) and the binding.
func (everlongSpreadV1) refreshReads(p *readPlan, hook common.Address, r *flammReads, b *hookBinding) {
	sp := &spreadHookState{}
	r.Spread = sp
	p.add("spreadHook.spread", hook, &spreadHookABI, "spread", nil, readWord(&sp.Spread))
	p.add("spreadHook.maxSpreadAge", hook, &spreadHookABI, "maxSpreadAge", nil, readWord(&sp.MaxSpreadAge))
	p.add("spreadHook.lastSetTs", hook, &spreadHookABI, "lastSetTs", nil, readWord(&sp.LastSetTs))
	p.add("spreadHook.POOL", hook, &spreadHookABI, "POOL", nil, readAddr(&b.Pool))
}

func (everlongSpreadV1) bound(_ *hookEntry, b *hookBinding, pool, _ common.Address) bool {
	return b.Pool == pool
}

func (everlongSpreadV1) build(r *flammReads, s *flammState) error {
	if r.Spread == nil {
		return errReadsInconsistent
	}
	sp := *r.Spread
	s.Hooks.Spread = spreadHookSlot{Kind: hookKindEverlongSpreadV1, EverlongSpread: &sp}
	return nil
}

// dependency: a keeper's spread post emits only the hook's SpreadSet.
func (everlongSpreadV1) dependency() bool { return true }
