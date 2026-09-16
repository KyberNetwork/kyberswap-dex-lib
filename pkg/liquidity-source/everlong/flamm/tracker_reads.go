package everlongflamm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// One refresh's reads (pool_tracker.go): the view getters flammReads is built from (state_reads.go), in one
// Multicall3 aggregate at the refresh block, with each hook read by its kind (hook_kinds.go), the wiring the listing
// pinned (for drift), the deployed views the attestation reproduces, and the storage words no view exposes, read at
// the same block. The shape of every read is the one testdata/gen/CoreE2EBase.sol dumps for the core end-to-end
// fixtures, so the tracker builds the state exactly as those fixtures were replayed.

// FLAMMStore's ERC-7201 namespace (FLAMMStore.sol) and the slots read or checked there: physicalPoolAsset (+12),
// pendingControllerHook | lastLeverSpreadPpm << 160 | hookSetExecutableAt << 192 (+22), features (+23),
// leverageHook (+24), spreadHook (+25), and the two delayed admission ceremonies, which are the struct's last four
// fields: pendingLoanHash (+30), pendingLoanAt (+31), pendingVenueHash (+32), pendingVenueAt (+33). Each of those
// four has a slot to itself (a bytes32 cannot share one, and a uint48 that follows one starts a new slot).
// The four offsets were confirmed on a Base fork: after the curator's scheduling call, addVenue wrote +32/+33 and
// addLoanAsset wrote +30/+31, and +32/+33 equal what pendingHookSet() reports as (venueHash, venueAt).
var flammStoreBase = common.HexToHash("0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500").Big()

const (
	slotPhysical     = 12
	slotSpreadWord   = 22
	slotFeatures     = 23
	slotLevHook      = 24
	slotSpreadHook   = 25
	slotPendingLoan  = 30
	slotPendingVenue = 32
)

func flammSlot(off int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(flammStoreBase, big.NewInt(off)))
}

// FLAMMFactory storage (FLAMMFactory.sol:48-57): the beacon's `_implementation` at slot 0, and
// `pendingImplementation` packed with `implementationExecutableAt` (bits 160..208) at slot 1. Both pending words
// are written and deleted together, and the pair is what Extra.ScheduledChangeAt reads, so both are compared with
// the views that report them (LayoutOk).
const (
	factorySlotImplementation = 0
	factorySlotPending        = 1
)

// MMRouter storage (MMRouter.sol:23-25, MMRouterLib.sol:39-75): globalPaused at slot 0 (byte 0), _pools at slot 1.
// PoolRecord at keccak(pool . 1): poolAsset | pinLtvWad << 160 | maxDrawnAssets << 224 (+0); safetyGapWad |
// oracleBandWad << 64 (+1); loans (+2); venues (+3); borrowOrder, supplyOrder, withdrawOrder, repayOrder (+4..+7),
// each uint16 array packed sixteen to a word. Loan i at keccak(record + 2) + 7i: token | decimals << 160 |
// borrowEnabled << 168 | retired << 176; loanScale; debtCap | supplyCap << 128; accounts[4]. Venue i at
// keccak(record + 3) + 6i: account; id; kind | loanIndex << 8 | lltvWad << 16 | borrowEnabled << 80 |
// supplyEnabled << 88 | retired << 96 | debtCap << 104; supplyCap | maxBorrowRateWad << 128; managedCollateral;
// managedSupplyShares. Every Router configuration word the port reads through a view is compared with its slot
// (LayoutOk), so a flag that only prices another state (a retired loan, a supply-disabled venue) is bound to the
// chain even where no probe reaches it.

func routerRecordSlot(pool common.Address) *big.Int {
	var key [64]byte
	copy(key[12:32], pool[:])
	key[63] = 1
	return new(big.Int).SetBytes(crypto.Keccak256(key[:]))
}

// routerArraySlot is element slot i of the dynamic array whose length is at slot.
func routerArraySlot(slot *big.Int, i int) *big.Int {
	arr := new(big.Int).SetBytes(crypto.Keccak256(common.BigToHash(slot).Bytes()))
	return arr.Add(arr, big.NewInt(int64(i)))
}

func routerVenueSlot(pool common.Address, i int) *big.Int {
	return routerArraySlot(new(big.Int).Add(routerRecordSlot(pool), big.NewInt(3)), 6*i)
}

func routerLoanSlot(pool common.Address, i int) *big.Int {
	return routerArraySlot(new(big.Int).Add(routerRecordSlot(pool), big.NewInt(2)), 7*i)
}

// storageField is bits [off, off+width) of a storage word.
func storageField(w common.Hash, off, width uint) uint256.Int {
	var z, mask uint256.Int
	z.SetBytes(w[:]).Rsh(&z, off)
	mask.Lsh(uOne, width).Sub(&mask, uOne)
	return *z.And(&z, &mask)
}

// readPlan pairs each call with its decoder.
type readPlan struct {
	calls []mcCall
	decs  []func(res mcResult) error
}

func (p *readPlan) add(name string, target common.Address, a *abi.ABI, method string, args []any,
	dec func(vals []any) error) {
	c := mcCall{Name: name, Target: target, ABI: a, Method: method, Args: args}
	p.calls = append(p.calls, c)
	p.decs = append(p.decs, func(res mcResult) error {
		vals, err := unpack(&c, res)
		if err != nil {
			return err
		}
		if err = dec(vals); err != nil {
			return fmt.Errorf("%w: %s: %w", errReadDecode, name, err)
		}
		return nil
	})
}

// try adds a call whose revert is an answer: dec sees vals == nil and the revert data.
func (p *readPlan) try(name string, target common.Address, a *abi.ABI, method string, args []any,
	dec func(vals []any, revert []byte) error) {
	c := mcCall{Name: name, Target: target, ABI: a, Method: method, Args: args, Optional: true}
	p.calls = append(p.calls, c)
	p.decs = append(p.decs, func(res mcResult) error {
		if !res.Ok {
			return dec(nil, res.Data)
		}
		vals, err := unpack(&c, res)
		if err != nil {
			return err
		}
		if err = dec(vals, nil); err != nil {
			return fmt.Errorf("%w: %s: %w", errReadDecode, name, err)
		}
		return nil
	})
}

// optional adds a call whose revert and whose answer that does not decode are both an absent value: dec runs only
// on outputs the chain answered with. Nothing the port prices is read this way. It reads the addresses the
// dependency set resolves through (pool_tracker.go), which a contract that is not the one the getter belongs to
// simply does not have, and the listing's first round over every candidate pool (pools_list_updater.go poolHead),
// whose three answers -- isPool, hooks() and priceFeed() -- gate admission before any read is addressed to a hook
// or a feed, and are then checked again by required reads in the view round pinned to the same block: hooks() and
// priceFeed() re-read and compared, and the pool's own factory() matched against the deployment.
func (p *readPlan) optional(name string, target common.Address, a *abi.ABI, method string, args []any,
	dec func(vals []any)) {
	c := mcCall{Name: name, Target: target, ABI: a, Method: method, Args: args, Optional: true}
	p.calls = append(p.calls, c)
	p.decs = append(p.decs, func(res mcResult) error {
		if vals, err := unpack(&c, res); err == nil {
			dec(vals)
		}
		return nil
	})
}

// run returns the block the aggregate ran at, also with an error the chain answered there (chainAnswered).
func (p *readPlan) run(ctx context.Context, rpc *mcRPC, block *big.Int) (uint64, error) {
	return p.runWith(ctx, rpc, block, callOpts{})
}

// runWith is run with the eth_call options of callOpts.
func (p *readPlan) runWith(ctx context.Context, rpc *mcRPC, block *big.Int, opts callOpts) (uint64, error) {
	b, res, err := rpc.aggregate(ctx, block, opts, p.calls)
	if err != nil {
		return b, err
	}
	for i := range res {
		if err = p.decs[i](res[i]); err != nil {
			return b, err
		}
	}
	return b, nil
}

// tupleOf converts a decoded tuple into T (abi.ConvertType panics on a shape mismatch).
func tupleOf[T any](v any) (out T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tuple shape: %v", r)
		}
	}()
	p, ok := abi.ConvertType(v, new(T)).(*T)
	if !ok {
		return out, errReadDecode
	}
	return *p, nil
}

func valAt[T any](vals []any, i int) (T, error) {
	var zero T
	if i >= len(vals) {
		return zero, errReadDecode
	}
	v, ok := vals[i].(T)
	if !ok {
		return zero, errReadDecode
	}
	return v, nil
}

// readAddr, readWord and readHash decode a view's first output into dst.
func readAddr(dst *common.Address) func([]any) error {
	return func(vals []any) (err error) {
		*dst, err = valAt[common.Address](vals, 0)
		return err
	}
}

func readWord(dst *uint256.Int) func([]any) error {
	return func(vals []any) error { return wordsInto(vals, dst) }
}

func readHash(dst *common.Hash) func([]any) error {
	return func(vals []any) error {
		h, err := valAt[[32]byte](vals, 0)
		*dst = h
		return err
	}
}

func wordAt(vals []any, i int) (uint256.Int, error) {
	if i >= len(vals) {
		return uint256.Int{}, errReadDecode
	}
	return wordOf(vals[i])
}

// wordsInto decodes vals[i] into the given destinations in order.
func wordsInto(vals []any, dst ...*uint256.Int) error {
	if len(vals) < len(dst) {
		return errReadDecode
	}
	for i, d := range dst {
		w, err := wordOf(vals[i])
		if err != nil {
			return err
		}
		*d = w
	}
	return nil
}

type abiHookSet struct {
	InvariantHook, FeeHook, RecenterHook, ControllerHook, LeverageHook, SpreadHook, LoanSwapHook common.Address
}

func (h abiHookSet) array() [7]common.Address {
	return [7]common.Address{h.InvariantHook, h.FeeHook, h.RecenterHook, h.ControllerHook, h.LeverageHook,
		h.SpreadHook, h.LoanSwapHook}
}

type abiDials struct {
	PhiWad, LtvWad, PhiMinWad, PhiMaxWad, LtvMinWad, LtvMaxWad, LtvMaxStepWad uint64
	LtvCooldownSec                                                            uint32
	MinStructDistWad                                                          uint64
}

type abiLimits struct {
	DepositCapPoolAsset *big.Int
	RoomEpsilonWad      uint64
	MaxPriceAgeSec      uint32
	FeeFloorWad         uint64
	FeeCapWad           uint64
	PerformanceFeeBp    uint16
	PeakSharePriceWad   *big.Int
	MaxPerformanceFeeBp uint16
}

type abiFeeParams struct {
	MidFeeWad, OutFeeWad, GammaWad, SigmaRefWad, VolBetaWad, VolMinWad, VolMaxWad, DirSkewWad uint64
}

type abiHookParams struct {
	AWad           *big.Int
	SpanUpWad      *big.Int
	SpanDnWad      *big.Int
	AnchorPriceWad *big.Int
	LoanDecimals   uint8
	Tuning         struct {
		Fee             abiFeeParams
		InvSkewKappaWad uint64
		InvSkewBandWad  uint64
		EmaHalfLife     uint64
		RvHalfLife      uint64
		StepDivisorWad  *big.Int
		InertiaWad      uint64
		InertiaMaxWad   uint64
	}
	Bounds struct {
		MinFeeWad, MaxFeeWad, MinEmaHalfLife, MaxEmaHalfLife, MaxRvHalfLife, MaxStepWad uint64
		MinStepDivisorWad                                                               *big.Int
	}
}

type abiSupport struct {
	AWad, XLo, XHi, YHi *big.Int
}

type abiFeedToken struct {
	Aggregator common.Address
	Heartbeat  uint32
	Scale      uint64
	Unit       uint64
	PegBandWad uint64
}

type abiLoanView struct {
	Token         common.Address
	Decimals      uint8
	LoanScale     *big.Int
	DebtCap       *big.Int
	SupplyCap     *big.Int
	BorrowEnabled bool
	Retired       bool
	Accounts      [4]common.Address
}

type abiVenueView struct {
	Account          common.Address
	Id               [32]byte
	Kind             uint8
	LoanIndex        uint8
	LltvWad          uint64
	BorrowEnabled    bool
	SupplyEnabled    bool
	Retired          bool
	DebtCap          *big.Int
	SupplyCap        *big.Int
	MaxBorrowRateWad uint64
}

type abiMarketParams struct {
	LoanToken, CollateralToken, Oracle, Irm common.Address
	Lltv                                    *big.Int
}

// abiPoolContext / abiLeverContext are IFLAMMHooks.PoolContext and IFLAMMLeverage.LeverContext, the argument
// LeverageSpreadHook.spreadPpm takes and ignores (LeverageSpreadHook.sol:75). The deadline round sends a zero one.
type abiPoolContext struct {
	PhysicalPoolAsset, PostedPoolAsset, LiquidLoanAsset, SuppliedLoanAsset, DebtLoanAsset, ShareSupply,
	PriceWad *big.Int
	PriceTs   *big.Int
	LoanCount uint8
}

type abiLeverContext struct {
	Pool             abiPoolContext
	Up               bool
	SpreadPpm        *big.Int
	AmountIn, MaxOut *big.Int
}

func zeroLeverContext() abiLeverContext {
	z := func() *big.Int { return new(big.Int) }
	return abiLeverContext{Pool: abiPoolContext{PhysicalPoolAsset: z(), PostedPoolAsset: z(), LiquidLoanAsset: z(),
		SuppliedLoanAsset: z(), DebtLoanAsset: z(), ShareSupply: z(), PriceWad: z(), PriceTs: z()},
		SpreadPpm: z(), AmountIn: z(), MaxOut: z()}
}

// clockAnswer is what the deployed views a deadline word decides answer at one clock: the PriceFeed's cross of the
// pair and each token's USD quote, the loan asset's peg, and the spread hook's post (the staticcall
// FLAMMLeverLib._spread makes, whatever the hook's kind).
type clockAnswer struct {
	At         uint64
	CrossOk    bool
	CrossPrice uint256.Int
	CrossTs    uint64
	UsdOk      [2]bool
	Usd        [2]uint256.Int
	UsdTs      [2]uint64
	PegOk      *bool  // nil: pegOk reverted
	PegErr     []byte // its revert data
	HasSpread  bool
	SpreadOk   bool
	SpreadPpm  uint256.Int
}

// readClock reads those views at block with only the block timestamp overridden to `at`, in one aggregate.
func readClock(ctx context.Context, rpc *mcRPC, se *StaticExtra, block *big.Int, at uint64) (clockAnswer, error) {
	spreadHook := se.Hooks[roleSlot[roleSpread]]
	out := clockAnswer{At: at, HasSpread: spreadHook != (common.Address{})}
	p := &readPlan{}
	p.add(fmt.Sprintf("feed.peekCross at %d", at), se.PriceFeed, &priceFeedABI, "peekCross",
		[]any{se.PoolAsset, se.LoanAsset}, func(vals []any) (err error) {
			if out.CrossOk, err = valAt[bool](vals, 0); err != nil {
				return err
			}
			if out.CrossPrice, err = wordAt(vals, 1); err != nil {
				return err
			}
			ts, err := wordAt(vals, 2)
			out.CrossTs = ts.Uint64()
			return err
		})
	for i, token := range [2]common.Address{se.PoolAsset, se.LoanAsset} {
		p.add(fmt.Sprintf("feed.peekUsd(%d) at %d", i, at), se.PriceFeed, &priceFeedABI, "peekUsd", []any{token},
			func(vals []any) (err error) {
				if out.UsdOk[i], err = valAt[bool](vals, 0); err != nil {
					return err
				}
				if out.Usd[i], err = wordAt(vals, 1); err != nil {
					return err
				}
				ts, err := wordAt(vals, 2)
				out.UsdTs[i] = ts.Uint64()
				return err
			})
	}
	p.try(fmt.Sprintf("feed.pegOk at %d", at), se.PriceFeed, &priceFeedABI, "pegOk", []any{se.LoanAsset},
		func(vals []any, revert []byte) error {
			if vals == nil {
				out.PegErr = revert
				return nil
			}
			ok, err := valAt[bool](vals, 0)
			out.PegOk = &ok
			return err
		})
	if out.HasSpread {
		p.add(fmt.Sprintf("spreadHook.spreadPpm at %d", at), spreadHook, &spreadHookABI, "spreadPpm",
			[]any{zeroLeverContext()}, func(vals []any) (err error) {
				if out.SpreadOk, err = valAt[bool](vals, 0); err != nil {
					return err
				}
				out.SpreadPpm, err = wordAt(vals, 1)
				return err
			})
	}
	if _, err := p.runWith(ctx, rpc, block, callOpts{time: at}); err != nil {
		return out, err
	}
	return out, nil
}

// snapshot is one round of reads: the state words, the wiring as read and the attestation views.
type snapshot struct {
	Block     uint64
	Timestamp uint64
	Reads     flammReads

	Asset, LoanAsset, Router, PriceFeed, Factory, Implementation common.Address
	Hooks                                                        [7]common.Address
	PendingControllerHook, PendingInvariantHook                  common.Address
	HookSetExecutableAt                                          uint64
	PendingVenueHash, PendingLoanHash                            common.Hash
	PendingVenueAt, PendingLoanAt                                uint64
	PendingImplementation                                        common.Address
	ImplementationExecutableAt                                   uint64
	LoanCount, RouterLoanCount, VenueCount                       uint256.Int
	RouterLoanToken                                              common.Address
	// Bindings is what each role's hook reported it is bound to (hook_kinds.go hookBinding).
	Bindings        [hookRoles]hookBinding
	SequencerFeed   common.Address
	FeedAggregators [2]common.Address
	VenueAccounts   []common.Address
	VenueIDs        []common.Hash

	PeekCrossOk    bool
	PeekCrossPrice uint256.Int
	PeekCrossTs    uint64
	PegOk          *bool
	LoanPosition   [3]uint256.Int
	Positions      *mmPositions
	TotalAssets    *uint256.Int
	TotalAssetsErr []byte
	VenuePositions [][5]uint256.Int
	TryPositions   []mmVenueRead
	BorrowRates    []uint256.Int
	// LayoutOk: the storage words read beside the views agree with them (FLAMMStore and MMRouter layouts).
	LayoutOk bool
}

// scheduledChangeAt is Extra.ScheduledChangeAt: the earliest executableAt of the four delayed changes that would
// move what a quote settles, at least 1 while any of them is pending and 0 when none is. Each is a (committed
// value, executableAt) pair, and either word marks it pending:
//
//   - the factory's pending implementation (FLAMMFactory.sol:150-152), which anyone may execute;
//   - the pool's pending hook set (FLAMMOpsLib.sol:259-261);
//   - a pending venue admission (FLAMMOpsLib.sol:384-392), which adds a funding leg to every settlement;
//   - a pending loan asset (FLAMMOpsLib.sol:523-529), which the quoted envelope refuses outright once admitted.
//
// The last two are curator-only, like the hook set; only the implementation is permissionless. Both words of each
// pair are written and deleted in the same statement, so a non-zero executableAt beside a zero committed value
// cannot happen on chain -- but an answer that zeroed only one of them would otherwise turn the guard off
// silently. The hook set's and the venue's executableAt are bound to their FLAMMStore words by the layout check
// and the factory's pair to slot 1 of the factory; the loan asset's pair has no view at all and is read from
// storage only.
func (s *snapshot) scheduledChangeAt() uint64 {
	var at uint64
	for _, c := range []struct {
		pending bool
		at      uint64
	}{{s.PendingImplementation != (common.Address{}) || s.ImplementationExecutableAt != 0,
		s.ImplementationExecutableAt},
		{s.PendingInvariantHook != (common.Address{}) || s.HookSetExecutableAt != 0, s.HookSetExecutableAt},
		{s.PendingVenueHash != (common.Hash{}) || s.PendingVenueAt != 0, s.PendingVenueAt},
		{s.PendingLoanHash != (common.Hash{}) || s.PendingLoanAt != 0, s.PendingLoanAt}} {
		if c.pending && (at == 0 || max(c.at, 1) < at) {
			at = max(c.at, 1)
		}
	}
	return at
}

// readSnapshot runs the refresh's view round at block (nil: latest) and the storage words at the block it ran at,
// for the hook set hooks (the listing's, resolved against the registry) and the venue set venues (the previous
// refresh's, or the one readVenueSet just read). A view round the chain answered with a revert or undecodable data
// returns the error with a snapshot holding only the block it ran at; a venue the pool no longer has is such an
// answer, and a venue it gained shows up as snapshot.VenueCount.
func readSnapshot(ctx context.Context, rpc *mcRPC, pool common.Address, se *StaticExtra, hooks *poolHookSet,
	venues []StaticVenue, block *big.Int) (*snapshot, error) {
	s := &snapshot{}
	r := &s.Reads
	nv := len(venues)
	r.Pool.Loans = make([]loanConfigReads, 1)
	r.Router.Loans = make([]routerLoanReads, 1)
	r.Router.Venues = make([]venueReads, nv)
	r.Feed.Tokens = make([]feedTokenReads, 2)
	s.VenueAccounts, s.VenueIDs = make([]common.Address, nv), make([]common.Hash, nv)
	s.VenuePositions, s.TryPositions, s.BorrowRates = make([][5]uint256.Int, nv), make([]mmVenueRead, nv),
		make([]uint256.Int, nv)
	p := &readPlan{}
	none := []any(nil)

	// ---- pool
	p.add("pool.asset", pool, &flammABI, "asset", none, readAddr(&s.Asset))
	p.add("pool.loanAsset", pool, &flammABI, "loanAsset", none, readAddr(&s.LoanAsset))
	p.add("pool.router", pool, &flammABI, "router", none, readAddr(&s.Router))
	p.add("pool.priceFeed", pool, &flammABI, "priceFeed", none, readAddr(&s.PriceFeed))
	p.add("pool.factory", pool, &flammABI, "factory", none, readAddr(&s.Factory))
	p.add("pool.paused", pool, &flammABI, "paused", none, func(vals []any) (err error) {
		r.Pool.Paused, err = valAt[bool](vals, 0)
		return err
	})
	p.add("pool.switches", pool, &flammABI, "switches", none, func(vals []any) (err error) {
		if r.Pool.Features, err = wordAt(vals, 0); err != nil {
			return err
		}
		r.Pool.LevPaused, err = valAt[bool](vals, 2)
		return err
	})
	p.add("pool.hooks", pool, &flammABI, "hooks", none, func(vals []any) error {
		h, err := tupleOf[abiHookSet](vals[0])
		s.Hooks, r.Pool.Hooks = h.array(), h.array()
		return err
	})
	p.add("pool.pendingHookSet", pool, &flammABI, "pendingHookSet", none, func(vals []any) error {
		h, err := tupleOf[abiHookSet](vals[0])
		if err != nil {
			return err
		}
		s.PendingControllerHook, s.PendingInvariantHook = h.ControllerHook, h.InvariantHook
		at, err := wordAt(vals, 1) // uint48
		if s.HookSetExecutableAt = at.Uint64(); err != nil {
			return err
		}
		// The same view reports the pool's other delayed admission, a scheduled venue (FLAMM.sol:85-100): the
		// hash the curator committed to and the second it becomes admissible. A venue changes the settlement's
		// funding legs, so its executableAt is a scheduled change like the hook set's.
		hash, err := valAt[[32]byte](vals, 2)
		if err != nil {
			return err
		}
		s.PendingVenueHash = hash
		vAt, err := wordAt(vals, 3) // uint48
		s.PendingVenueAt = vAt.Uint64()
		return err
	})
	p.add("pool.poolAssetPosition", pool, &flammABI, "poolAssetPosition", none, func(vals []any) error {
		return wordsInto(vals, &r.Pool.Physical, &r.Pool.Gross)
	})
	p.add("pool.totalSupply", pool, &flammABI, "totalSupply", none, readWord(&r.Pool.TotalSupply))
	p.add("pool.dials", pool, &flammABI, "dials", none, func(vals []any) error {
		d, err := tupleOf[abiDials](vals[0])
		r.Pool.PhiWad.SetUint64(d.PhiWad)
		r.Pool.LtvWad.SetUint64(d.LtvWad)
		return err
	})
	p.add("pool.limits", pool, &flammABI, "limits", none, func(vals []any) error {
		l, err := tupleOf[abiLimits](vals[0])
		r.Pool.RoomEpsilonWad.SetUint64(l.RoomEpsilonWad)
		r.Pool.FeeFloorWad.SetUint64(l.FeeFloorWad)
		r.Pool.FeeCapWad.SetUint64(l.FeeCapWad)
		return err
	})
	p.add("pool.loanCount", pool, &flammABI, "loanCount", none, readWord(&s.LoanCount))
	p.add("pool.loanConfig(0)", pool, &flammABI, "loanConfig", []any{uint8(0)}, func(vals []any) (err error) {
		l := &r.Pool.Loans[0]
		if l.Token, err = valAt[common.Address](vals, 0); err != nil {
			return err
		}
		if l.Decimals, err = valAt[uint8](vals, 1); err != nil {
			return err
		}
		return wordsInto(vals[2:], &l.SwapPriceBandWad, &l.FeeFloorWad, &l.MaxSwapNotional, &l.ReserveTarget,
			&l.Liquid)
	})
	p.add("pool.loanPosition", pool, &flammABI, "loanPosition", none, func(vals []any) error {
		return wordsInto(vals, &s.LoanPosition[0], &s.LoanPosition[1], &s.LoanPosition[2])
	})
	p.try("pool.totalAssets", pool, &flammABI, "totalAssets", none, func(vals []any, revert []byte) error {
		if vals == nil {
			s.TotalAssetsErr = append([]byte{}, revert...)
			return nil
		}
		w, err := wordAt(vals, 0)
		s.TotalAssets = &w
		return err
	})
	p.add("factory.implementation", se.Factory, &factoryABI, "implementation", none, readAddr(&s.Implementation))
	p.add("factory.pendingImplementation", se.Factory, &factoryABI, "pendingImplementation", none,
		readAddr(&s.PendingImplementation))
	p.add("factory.implementationExecutableAt", se.Factory, &factoryABI, "implementationExecutableAt", none,
		func(vals []any) error {
			at, err := wordAt(vals, 0) // uint48
			s.ImplementationExecutableAt = at.Uint64()
			return err
		})

	// ---- the hooks, each by its kind, in role order (addressed to the listed set; drift checks it against hooks())
	for role := roleSwap; role < hookRoles; role++ {
		if e := hooks.Entries[role]; e != nil {
			spec := hookKindSpecOf(e.Kind)
			if spec == nil {
				return nil, fmt.Errorf("%w: %s kind %s has no reads", errRPC, roleName[role], e.Kind)
			}
			spec.refreshReads(p, e.Address, r, &s.Bindings[role])
		}
	}

	// ---- price feed: its immutables, each token's config and each aggregator's latest round
	f := &r.Feed
	p.add("feed.SEQUENCER_FEED", se.PriceFeed, &priceFeedABI, "SEQUENCER_FEED", none, readAddr(&s.SequencerFeed))
	p.add("feed.SEQUENCER_GRACE", se.PriceFeed, &priceFeedABI, "SEQUENCER_GRACE", none, readWord(&f.SequencerGrace))
	f.SequencerFeed = se.SequencerFeed
	round := func(name string, agg common.Address, dst *roundReads) {
		if agg == (common.Address{}) {
			return // PriceFeed never calls a zero sequencer feed; a zero aggregator cannot be registered
		}
		p.try(name, agg, &aggregatorABI, "latestRoundData", none, func(vals []any, _ []byte) error {
			if vals == nil {
				*dst = roundReads{}
				return nil
			}
			if len(vals) < 4 {
				return errReadDecode
			}
			id, err := wordOf(vals[0])
			if err != nil {
				return err
			}
			a, err := signedOf(vals[1])
			if err != nil {
				return err
			}
			*dst = roundReads{Ok: true, RoundId: id, Answer: a}
			return wordsInto(vals[2:], &dst.StartedAt, &dst.UpdatedAt)
		})
	}
	round("sequencer.latestRoundData", se.SequencerFeed, &f.Sequencer)
	for i, token := range []common.Address{se.PoolAsset, se.LoanAsset} {
		t := &f.Tokens[i]
		t.Token = token
		p.try(fmt.Sprintf("feed.config(%d)", i), se.PriceFeed, &priceFeedABI, "config", []any{token},
			func(vals []any, _ []byte) error {
				if vals == nil {
					t.Known = false
					return nil
				}
				c, err := tupleOf[abiFeedToken](vals[0])
				t.Known = true
				s.FeedAggregators[i] = c.Aggregator
				t.Heartbeat.SetUint64(uint64(c.Heartbeat))
				t.Scale.SetUint64(c.Scale)
				t.Unit.SetUint64(c.Unit)
				t.PegBandWad.SetUint64(c.PegBandWad)
				return err
			})
		round(fmt.Sprintf("aggregator(%d).latestRoundData", i), se.Aggregators[i], &t.Round)
	}
	p.try("feed.peekCross", se.PriceFeed, &priceFeedABI, "peekCross", []any{se.PoolAsset, se.LoanAsset},
		func(vals []any, _ []byte) (err error) {
			if vals == nil {
				return errReadFailed // peekCross never reverts
			}
			if s.PeekCrossOk, err = valAt[bool](vals, 0); err != nil {
				return err
			}
			if s.PeekCrossPrice, err = wordAt(vals, 1); err != nil {
				return err
			}
			ts, err := wordAt(vals, 2) // uint48
			s.PeekCrossTs = ts.Uint64()
			return err
		})
	p.try("feed.pegOk", se.PriceFeed, &priceFeedABI, "pegOk", []any{se.LoanAsset}, func(vals []any, _ []byte) error {
		if vals == nil {
			return nil
		}
		ok, err := valAt[bool](vals, 0)
		s.PegOk = &ok
		return err
	})

	// ---- router, venues, Morpho, IRM, oracle
	rt := &r.Router
	p.add("router.globalPaused", se.Router, &routerABI, "globalPaused", none, func(vals []any) (err error) {
		rt.GlobalPaused, err = valAt[bool](vals, 0)
		return err
	})
	p.add("router.pin", se.Router, &routerABI, "pin", []any{pool}, func(vals []any) error {
		return wordsInto(vals, &rt.PinLtvWad, &rt.SafetyGapWad, &rt.OracleBandWad)
	})
	p.add("router.maxDrawnAssets", se.Router, &routerABI, "maxDrawnAssets", []any{pool}, func(vals []any) (err error) {
		rt.MaxDrawnAssets, err = valAt[uint8](vals, 0)
		return err
	})
	p.add("router.loanCount", se.Router, &routerABI, "loanCount", []any{pool}, readWord(&s.RouterLoanCount))
	p.add("router.venueCount", se.Router, &routerABI, "venueCount", []any{pool}, readWord(&s.VenueCount))
	p.add("router.loan(0)", se.Router, &routerABI, "loan", []any{pool, uint8(0)}, func(vals []any) error {
		l, err := tupleOf[abiLoanView](vals[0])
		if err != nil {
			return err
		}
		s.RouterLoanToken = l.Token
		rl := &rt.Loans[0]
		rl.Decimals, rl.BorrowEnabled, rl.Retired = l.Decimals, l.BorrowEnabled, l.Retired
		return wordsInto([]any{l.LoanScale, l.DebtCap, l.SupplyCap}, &rl.LoanScale, &rl.DebtCap, &rl.SupplyCap)
	})
	p.add("router.priorities", se.Router, &routerABI, "priorities", []any{pool}, func(vals []any) (err error) {
		for i, dst := range []*[]uint16{&rt.BorrowOrder, &rt.SupplyOrder, &rt.WithdrawOrder, &rt.RepayOrder} {
			if *dst, err = valAt[[]uint16](vals, i); err != nil {
				return err
			}
		}
		return nil
	})
	p.try("router.positions", se.Router, &routerABI, "positions", []any{pool}, func(vals []any, _ []byte) error {
		if vals == nil {
			return nil
		}
		var pos mmPositions
		var err error
		for i, dst := range []*[]uint256.Int{&pos.Coll, &pos.Sup, &pos.Debt} {
			if *dst, err = wordsOf(vals[i]); err != nil {
				return err
			}
		}
		if pos.TotalColl, err = wordAt(vals, 3); err != nil {
			return err
		}
		s.Positions = &pos
		return nil
	})
	for i := range venues {
		sv := &venues[i]
		v := &rt.Venues[i]
		v.Irm, v.MarketLltv, v.HasIrm = sv.Irm, sv.Lltv, sv.Irm != (common.Address{})
		p.add(fmt.Sprintf("router.venue(%d)", i), se.Router, &routerABI, "venue", []any{pool, uint16(i)},
			func(vals []any) error {
				vv, err := tupleOf[abiVenueView](vals[0])
				if err != nil {
					return err
				}
				s.VenueAccounts[i], s.VenueIDs[i] = vv.Account, vv.Id
				v.Kind, v.LoanIndex, v.BorrowEnabled, v.SupplyEnabled, v.Retired = vv.Kind, vv.LoanIndex,
					vv.BorrowEnabled, vv.SupplyEnabled, vv.Retired
				v.LltvWad.SetUint64(vv.LltvWad)
				v.MaxBorrowRateWad.SetUint64(vv.MaxBorrowRateWad)
				return wordsInto([]any{vv.DebtCap, vv.SupplyCap}, &v.DebtCap, &v.SupplyCap)
			})
		p.try(fmt.Sprintf("router.venuePosition(%d)", i), se.Router, &routerABI, "venuePosition",
			[]any{pool, uint16(i)}, func(vals []any, revert []byte) error {
				if vals == nil {
					return fmt.Errorf("%w: venuePosition reverted %x", errReadFailed, revert)
				}
				vp := &s.VenuePositions[i]
				return wordsInto(vals, &vp[0], &vp[1], &vp[2], &vp[3], &vp[4])
			})
		p.add(fmt.Sprintf("morpho.market(%d)", i), se.Morpho, &morphoABI, "market", []any{sv.MarketID},
			func(vals []any) error {
				m := &v.Market
				return wordsInto(vals, &m.TotalSupplyAssets, &m.TotalSupplyShares, &m.TotalBorrowAssets,
					&m.TotalBorrowShares, &m.LastUpdate, &m.Fee)
			})
		p.add(fmt.Sprintf("morpho.position(%d)", i), se.Morpho, &morphoABI, "position", []any{sv.MarketID, sv.Account},
			func(vals []any) error {
				return wordsInto(vals, &v.Position.SupplyShares, &v.Position.BorrowShares, &v.Position.Collateral)
			})
		if v.HasIrm {
			p.add(fmt.Sprintf("irm.rateAtTarget(%d)", i), sv.Irm, &irmABI, "rateAtTarget", []any{sv.MarketID},
				func(vals []any) (err error) {
					v.RateAtTarget, err = signedOf(vals[0])
					return err
				})
		}
		// borrowRateAfter(id, 0, 0) is the account's own borrowRateView over the live market: ok is the IRM
		// answering (a market without an IRM answers a zero rate).
		p.add(fmt.Sprintf("account.borrowRateAfter(%d)", i), sv.Account, &accountABI, "borrowRateAfter",
			[]any{sv.MarketID, new(big.Int), new(big.Int)}, func(vals []any) (err error) {
				if v.IrmReadable, err = valAt[bool](vals, 0); err != nil {
					return err
				}
				if !v.HasIrm {
					v.IrmReadable = true
				}
				s.BorrowRates[i], err = wordAt(vals, 1)
				return err
			})
		p.add(fmt.Sprintf("account.oraclePrice(%d)", i), sv.Account, &accountABI, "oraclePrice", []any{sv.MarketID},
			func(vals []any) (err error) {
				if v.OracleOk, err = valAt[bool](vals, 0); err != nil {
					return err
				}
				v.OraclePrice, err = wordAt(vals, 1)
				return err
			})
		p.try(fmt.Sprintf("oracle.price(%d)", i), sv.Oracle, &oracleABI, "price", none,
			func(vals []any, _ []byte) error {
				if vals == nil {
					v.OracleZero = false
					return nil
				}
				w, err := wordAt(vals, 0)
				v.OracleZero = w.IsZero()
				return err
			})
		p.add(fmt.Sprintf("account.tryPosition(%d)", i), sv.Account, &accountABI, "tryPosition", []any{sv.MarketID},
			func(vals []any) (err error) {
				tp := &s.TryPositions[i]
				if tp.Readable, err = valAt[bool](vals, 0); err != nil {
					return err
				}
				return wordsInto(vals[1:], &tp.Collateral, &tp.SupplyShares, &tp.Supplied, &tp.Debt)
			})
	}
	p.add("multicall.getCurrentBlockTimestamp", multicall3, &multicallABI, "getCurrentBlockTimestamp", none,
		func(vals []any) error {
			ts, err := wordAt(vals, 0)
			if err != nil || !ts.IsUint64() {
				return errReadDecode
			}
			s.Timestamp = ts.Uint64()
			return nil
		})

	b, err := p.run(ctx, rpc, block)
	if err != nil {
		if b != 0 && chainAnswered(err) {
			return &snapshot{Block: b}, err
		}
		return nil, err
	}
	s.Block, r.Block, r.Timestamp = b, b, s.Timestamp
	r.Pool.Asset, r.Pool.Router, r.Pool.PriceFeed = s.Asset, s.Router, s.PriceFeed
	for i := range venues {
		// OracleZero is only the zero answer oraclePrice folds into not-ok (CoreE2EBase._venueMorpho).
		v := &rt.Venues[i]
		v.OracleZero = !v.OracleOk && v.OracleZero
	}

	// ---- storage words at the same block, with the layout checked against the views
	reads := []storageRead{{pool, flammSlot(slotPhysical)}, {pool, flammSlot(slotSpreadWord)},
		{pool, flammSlot(slotFeatures)}, {pool, flammSlot(slotLevHook)}, {pool, flammSlot(slotSpreadHook)},
		{pool, flammSlot(slotPendingLoan)}, {pool, flammSlot(slotPendingLoan + 1)},
		{pool, flammSlot(slotPendingVenue)}, {pool, flammSlot(slotPendingVenue + 1)},
		{se.Factory, common.BigToHash(big.NewInt(factorySlotImplementation))},
		{se.Factory, common.BigToHash(big.NewInt(factorySlotPending))}}
	slot := func(account common.Address, at *big.Int) {
		reads = append(reads, storageRead{account, common.BigToHash(at)})
	}
	record := routerRecordSlot(pool)
	slot(se.Router, new(big.Int))
	for off := int64(0); off < 8; off++ {
		slot(se.Router, new(big.Int).Add(record, big.NewInt(off)))
	}
	orders := []*[]uint16{&rt.BorrowOrder, &rt.SupplyOrder, &rt.WithdrawOrder, &rt.RepayOrder}
	for k, order := range orders {
		for j := 0; j < (len(*order)+15)/16; j++ {
			slot(se.Router, routerArraySlot(new(big.Int).Add(record, big.NewInt(4+int64(k))), j))
		}
	}
	loan := routerLoanSlot(pool, 0)
	for off := int64(0); off < 3; off++ {
		slot(se.Router, new(big.Int).Add(loan, big.NewInt(off)))
	}
	venuesAt := len(reads)
	for i := range venues {
		base := routerVenueSlot(pool, i)
		for off := int64(0); off < 6; off++ {
			slot(se.Router, new(big.Int).Add(base, big.NewInt(off)))
		}
	}
	words, err := rpc.storageAt(ctx, new(big.Int).SetUint64(b), reads)
	if err != nil {
		return nil, err
	}
	var sw uint256.Int
	sw.SetBytes(words[1][:])
	var spread, execAt uint256.Int
	spread.Rsh(&sw, 160)
	execAt.Rsh(&sw, 192)
	r.Pool.LastLeverSpreadPpm.SetUint64(spread.Uint64() & 0xffffffff)
	layout := new(uint256.Int).SetBytes(words[0][:]).Eq(&r.Pool.Physical) &&
		common.BytesToAddress(words[1][12:]) == s.PendingControllerHook && execAt.Uint64() == s.HookSetExecutableAt &&
		new(uint256.Int).SetBytes(words[2][:]).Eq(&r.Pool.Features) &&
		common.BytesToAddress(words[3][:]) == s.Hooks[roleSlot[roleLeverage]] &&
		common.BytesToAddress(words[4][:]) == s.Hooks[roleSlot[roleSpread]]
	eq := func(w common.Hash, off, width uint, want uint64) bool {
		f := storageField(w, off, width)
		return f.IsUint64() && f.Uint64() == want
	}
	// The delayed loan-asset admission, which no view reports at all, and the venue admission beside the view that
	// does report it (pool.pendingHookSet). Both feed Extra.ScheduledChangeAt.
	loanAt := storageField(words[6], 0, 48)
	s.PendingLoanHash, s.PendingLoanAt = words[5], loanAt.Uint64()
	layout = layout && words[7] == s.PendingVenueHash && eq(words[8], 0, 48, s.PendingVenueAt)
	// The factory's beacon and upgrade schedule beside the views that report them: nothing else binds the pending
	// pair, whose executableAt the simulator refuses to quote across (Extra.ScheduledChangeAt).
	layout = layout && common.BytesToAddress(words[9][:]) == s.Implementation &&
		common.BytesToAddress(words[10][12:]) == s.PendingImplementation &&
		eq(words[10], 160, 48, s.ImplementationExecutableAt)
	eqWord := func(w common.Hash, off, width uint, want *uint256.Int) bool {
		f := storageField(w, off, width)
		return f.Eq(want)
	}
	flag := func(v bool) uint64 {
		if v {
			return 1
		}
		return 0
	}
	rw := words[11:]
	layout = layout && eq(rw[0], 0, 8, flag(rt.GlobalPaused)) &&
		common.BytesToAddress(rw[1][12:]) == s.Asset && eqWord(rw[1], 160, 64, &rt.PinLtvWad) &&
		eq(rw[1], 224, 8, uint64(rt.MaxDrawnAssets)) &&
		eqWord(rw[2], 0, 64, &rt.SafetyGapWad) && eqWord(rw[2], 64, 64, &rt.OracleBandWad) &&
		new(uint256.Int).SetBytes(rw[3][:]).Eq(&s.RouterLoanCount) && new(uint256.Int).SetBytes(rw[4][:]).Eq(&s.VenueCount)
	for k, order := range orders {
		layout = layout && eq(rw[5+k], 0, 256, uint64(len(*order)))
	}
	rw = rw[9:]
	for _, order := range orders {
		for j, id := range *order {
			layout = layout && eq(rw[j/16], uint(16*(j%16)), 16, uint64(id))
		}
		rw = rw[(len(*order)+15)/16:]
	}
	rl := &rt.Loans[0]
	layout = layout && common.BytesToAddress(rw[0][12:]) == s.RouterLoanToken && eq(rw[0], 160, 8, uint64(rl.Decimals)) &&
		eq(rw[0], 168, 8, flag(rl.BorrowEnabled)) && eq(rw[0], 176, 8, flag(rl.Retired)) &&
		new(uint256.Int).SetBytes(rw[1][:]).Eq(&rl.LoanScale) && eqWord(rw[2], 0, 128, &rl.DebtCap) &&
		eqWord(rw[2], 128, 128, &rl.SupplyCap)
	for i := range venues {
		w := words[venuesAt+6*i:]
		v := &rt.Venues[i]
		layout = layout && common.BytesToAddress(w[0][:]) == s.VenueAccounts[i] && w[1] == s.VenueIDs[i] &&
			eq(w[2], 0, 8, uint64(v.Kind)) && eq(w[2], 8, 8, uint64(v.LoanIndex)) && eqWord(w[2], 16, 64, &v.LltvWad) &&
			eq(w[2], 80, 8, flag(v.BorrowEnabled)) && eq(w[2], 88, 8, flag(v.SupplyEnabled)) &&
			eq(w[2], 96, 8, flag(v.Retired)) && eqWord(w[2], 104, 128, &v.DebtCap) &&
			eqWord(w[3], 0, 128, &v.SupplyCap) && eqWord(w[3], 128, 64, &v.MaxBorrowRateWad)
		v.ManagedCollateral.SetBytes(w[4][:])
		v.ManagedSupplyShares.SetBytes(w[5][:])
	}
	s.LayoutOk = layout
	for i := range f.Tokens {
		if !f.Tokens[i].Known {
			f.Tokens[i].Round = roundReads{} // PriceFeed never reaches an unregistered token's aggregator
		}
	}
	return s, nil
}

// oracleFeedGetters are MorphoChainlinkOracleV2's four price feed immutables, in field order. An oracle that is
// not one answers none of them.
var oracleFeedGetters = [4]string{"BASE_FEED_1", "BASE_FEED_2", "QUOTE_FEED_1", "QUOTE_FEED_2"}

// readOracleFeeds fills each venue's OracleFeeds from its market oracle, in one aggregate at block. The feeds are
// immutables of an oracle that is itself immutable per market id, so they are read with the venue set rather than
// every refresh; only the aggregator each of them currently proxies to is (readAggregators).
func readOracleFeeds(ctx context.Context, rpc *mcRPC, venues []StaticVenue, block *big.Int) error {
	p := &readPlan{}
	for i := range venues {
		v := &venues[i]
		if v.Oracle == (common.Address{}) {
			continue
		}
		for k, getter := range oracleFeedGetters {
			p.optional(fmt.Sprintf("oracle(%d).%s", i, getter), v.Oracle, &oracleABI, getter, nil,
				func(vals []any) {
					if a, err := valAt[common.Address](vals, 0); err == nil {
						v.OracleFeeds[k] = a
					}
				})
		}
	}
	if len(p.calls) == 0 {
		return nil
	}
	_, err := p.run(ctx, rpc, block)
	return err
}

// readAggregators resolves, at block, the Chainlink aggregator behind every price feed proxy the pool prices
// through: the PriceFeed's two token proxies, the sequencer proxy, and the feeds of each venue's market oracle.
// The proxy is the address the configuration names, but the aggregator is what emits the rounds -- a proxy emits
// nothing, and its `aggregator()` moves on a phase change without a log of its own -- so the set is resolved on
// every refresh and published for the dependency trigger (pool_tracker.go). A proxy that has no `aggregator()`,
// or none the call decodes, resolves to nothing and simply contributes no dependency.
func readAggregators(ctx context.Context, rpc *mcRPC, se *StaticExtra, venues []StaticVenue,
	block *big.Int) ([]common.Address, error) {
	proxies := make([]common.Address, 0, 3+4*len(venues))
	proxies = append(proxies, se.Aggregators[0], se.Aggregators[1], se.SequencerFeed)
	for i := range venues {
		proxies = append(proxies, venues[i].OracleFeeds[:]...)
	}
	impls := make([]common.Address, len(proxies))
	p := &readPlan{}
	for i, proxy := range proxies {
		if proxy == (common.Address{}) {
			continue
		}
		p.optional(fmt.Sprintf("aggregator(%s)", proxy), proxy, &aggregatorABI, "aggregator", nil,
			func(vals []any) {
				if a, err := valAt[common.Address](vals, 0); err == nil {
					impls[i] = a
				}
			})
	}
	if len(p.calls) == 0 {
		return nil, nil
	}
	if _, err := p.run(ctx, rpc, block); err != nil {
		return nil, err
	}
	return impls, nil
}

// readOracleAhead reads what each venue's Morpho market oracle answers at `at`: MorphoBlueAccount.oraclePrice(id)
// at the refresh block with only the block timestamp overridden, so the answer is the one the same state gives a
// later block with no transaction in between. It is the one input of a snapshot that the clock alone moves. The
// BTC/USD feed behind the market oracle is a Chainlink SVR DualAggregator, which withholds each primary round
// for a fixed delay and reveals it once block.timestamp passes it, so price() -- and with it MMRouterLib.bandOk
// and Morpho's own health check -- changes with time and no transaction. Both of those read the price and nothing
// else reads it: bandOk accepts an interval around the PriceFeed cross (MMRouterLib.sol:780-788, `dev <= tol`)
// and health is a bound that only rises with the price (morpho-blue Morpho.sol:527-539, `maxBorrow >= borrowed`
// with maxBorrow monotone in it), so a fill that settles identically at the snapshot answer and at this one
// settles identically at every answer between them, and the reveal makes the answer step from one to the other.
func readOracleAhead(ctx context.Context, rpc *mcRPC, venues []StaticVenue, block *big.Int,
	at uint64) ([]OracleAnswer, error) {
	out := make([]OracleAnswer, len(venues))
	p := &readPlan{}
	for i := range venues {
		v := &venues[i]
		p.add(fmt.Sprintf("account.oraclePrice(%d) at %d", i, at), v.Account, &accountABI, "oraclePrice",
			[]any{v.MarketID}, func(vals []any) (err error) {
				if out[i].Ok, err = valAt[bool](vals, 0); err != nil {
					return err
				}
				out[i].Price, err = wordAt(vals, 1)
				return err
			})
	}
	if len(p.calls) == 0 {
		return nil, nil
	}
	if _, err := p.runWith(ctx, rpc, block, callOpts{time: at}); err != nil {
		return nil, err
	}
	return out, nil
}
