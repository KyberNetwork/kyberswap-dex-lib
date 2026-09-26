package everlongflamm

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// flammReads is everything the state is built from, in the shape of the calls that return it. Every field is
// a view getter's answer except three words no view exposes, read from raw storage at the same block:
//   - Pool.LastLeverSpreadPpm: FLAMMStore.lastLeverSpreadPpm, ERC-7201 slot
//     0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4516 (base + 22), bits 160..191;
//   - Router venue i ManagedCollateral / ManagedSupplyShares: keccak256(keccak256(pool . 1) + 3) + 6i + 4 / + 5.
//
// The views, per contract:
//   - pool: asset, router, priceFeed, paused, switches, hooks, poolAssetPosition, totalSupply, dials (phiWad,
//     ltvWad), limits (roomEpsilonWad, feeFloorWad, feeCapWad), loanCount, loanConfig(i);
//   - the hooks, each by its kind (hook_kinds.go), one field per kind that has state: an EverlongHook's params
//     (aWad, tuning.fee, tuning.invSkewKappaWad, tuning.invSkewBandWad), support, anchorSqrtX96, reservationPriceWad,
//     kappa, xWad, reserveStable, idleStable, reserveVolatile, idleVolatile, rvWad, LOAN_SCALE (Hook), and a
//     LeverageSpreadHook's spread, maxSpreadAge, lastSetTs (Spread). A field is absent when the pool lists no hook of
//     its kind;
//   - price feed: SEQUENCER_FEED, SEQUENCER_GRACE, config(token) for the pool asset and every loan asset, and
//     latestRoundData on each aggregator and on the sequencer feed (a revert is Ok = false);
//   - router: globalPaused, pin(pool), maxDrawnAssets(pool), loan(pool, i), priorities(pool), venue(pool, i);
//   - Morpho: market(id), position(id, account), idToMarketParams(id); the IRM's rateAtTarget(id) and whether
//     borrowRateView(params, market) answers; the account's oraclePrice(id); and whether the market oracle's price()
//     answers zero without reverting (OracleZero).
type flammReads struct {
	Block     uint64           `json:"block"`
	Timestamp uint64           `json:"timestamp"`
	Pool      poolReads        `json:"pool"`
	Hook      *hookReads       `json:"hook,omitempty"`
	Spread    *spreadHookState `json:"spread,omitempty"`
	Feed      feedReads        `json:"feed"`
	Router    routerReads      `json:"router"`
}

type poolReads struct {
	Asset              common.Address    `json:"asset"`
	Router             common.Address    `json:"router"`
	PriceFeed          common.Address    `json:"priceFeed"`
	Paused             bool              `json:"paused"`
	Features           uint256.Int       `json:"features"`
	LevPaused          bool              `json:"levPaused"`
	Hooks              [7]common.Address `json:"hooks"` // invariant, fee, recenter, controller, leverage, spread, loanSwap
	Physical           uint256.Int       `json:"physical"`
	Gross              uint256.Int       `json:"gross"`
	TotalSupply        uint256.Int       `json:"totalSupply"`
	PhiWad             uint256.Int       `json:"phiWad"`
	LtvWad             uint256.Int       `json:"ltvWad"`
	RoomEpsilonWad     uint256.Int       `json:"roomEpsilonWad"`
	FeeFloorWad        uint256.Int       `json:"feeFloorWad"`
	FeeCapWad          uint256.Int       `json:"feeCapWad"`
	Loans              []loanConfigReads `json:"loans"`
	LastLeverSpreadPpm uint256.Int       `json:"lastLeverSpreadPpm"`
}

// loanConfigReads is FLAMM.loanConfig(i).
type loanConfigReads struct {
	Token            common.Address `json:"token"`
	Decimals         uint8          `json:"decimals"`
	SwapPriceBandWad uint256.Int    `json:"swapPriceBandWad"`
	FeeFloorWad      uint256.Int    `json:"feeFloorWad"`
	MaxSwapNotional  uint256.Int    `json:"maxSwapNotional"`
	ReserveTarget    uint256.Int    `json:"reserveTarget"`
	Liquid           uint256.Int    `json:"liquid"`
}

type hookReads struct {
	AWad                uint256.Int    `json:"aWad"`
	Fee                 [8]uint256.Int `json:"fee"` // EverlongStrategy.FeeParams in field order
	InvSkewKappaWad     uint256.Int    `json:"invSkewKappaWad"`
	InvSkewBandWad      uint256.Int    `json:"invSkewBandWad"`
	Support             [4]uint256.Int `json:"support"` // aWad, xLo, xHi, yHi
	AnchorSqrtX96       uint256.Int    `json:"anchorSqrtX96"`
	ReservationPriceWad uint256.Int    `json:"reservationPriceWad"`
	Kappa               uint256.Int    `json:"kappa"`
	XWad                uint256.Int    `json:"xWad"`
	ReserveStable       uint256.Int    `json:"reserveStable"`
	IdleStable          uint256.Int    `json:"idleStable"`
	ReserveVolatile     uint256.Int    `json:"reserveVolatile"`
	IdleVolatile        uint256.Int    `json:"idleVolatile"`
	RvWad               uint256.Int    `json:"rvWad"`
	LoanScale           uint256.Int    `json:"loanScale"`
}

type roundReads struct {
	Ok        bool        `json:"ok"`
	RoundId   uint256.Int `json:"roundId"`
	Answer    int256.Int  `json:"answer"`
	StartedAt uint256.Int `json:"startedAt"`
	UpdatedAt uint256.Int `json:"updatedAt"`
}

type feedTokenReads struct {
	Token      common.Address `json:"token"`
	Known      bool           `json:"known"` // config(token) answered (not UnknownToken)
	Heartbeat  uint256.Int    `json:"heartbeat"`
	Scale      uint256.Int    `json:"scale"`
	Unit       uint256.Int    `json:"unit"`
	PegBandWad uint256.Int    `json:"pegBandWad"`
	Round      roundReads     `json:"round"`
}

type feedReads struct {
	SequencerFeed  common.Address   `json:"sequencerFeed"`
	SequencerGrace uint256.Int      `json:"sequencerGrace"`
	Sequencer      roundReads       `json:"sequencer"`
	Tokens         []feedTokenReads `json:"tokens"`
}

type routerLoanReads struct {
	Decimals      uint8       `json:"decimals"`
	LoanScale     uint256.Int `json:"loanScale"`
	DebtCap       uint256.Int `json:"debtCap"`
	SupplyCap     uint256.Int `json:"supplyCap"`
	BorrowEnabled bool        `json:"borrowEnabled"`
	Retired       bool        `json:"retired"`
}

type venueReads struct {
	Kind                uint8          `json:"kind"`
	LoanIndex           uint8          `json:"loanIndex"`
	LltvWad             uint256.Int    `json:"lltvWad"`
	BorrowEnabled       bool           `json:"borrowEnabled"`
	SupplyEnabled       bool           `json:"supplyEnabled"`
	Retired             bool           `json:"retired"`
	DebtCap             uint256.Int    `json:"debtCap"`
	SupplyCap           uint256.Int    `json:"supplyCap"`
	MaxBorrowRateWad    uint256.Int    `json:"maxBorrowRateWad"`
	ManagedCollateral   uint256.Int    `json:"managedCollateral"`
	ManagedSupplyShares uint256.Int    `json:"managedSupplyShares"`
	Market              mmMarket       `json:"market"`
	Position            mmPosition     `json:"position"`
	Irm                 common.Address `json:"irm"`
	MarketLltv          uint256.Int    `json:"marketLltv"`
	RateAtTarget        int256.Int     `json:"rateAtTarget"`
	HasIrm              bool           `json:"hasIrm"`
	IrmReadable         bool           `json:"irmReadable"`
	OracleOk            bool           `json:"oracleOk"`
	OraclePrice         uint256.Int    `json:"oraclePrice"`
	OracleZero          bool           `json:"oracleZero"`
}

type routerReads struct {
	GlobalPaused   bool              `json:"globalPaused"`
	PinLtvWad      uint256.Int       `json:"pinLtvWad"`
	SafetyGapWad   uint256.Int       `json:"safetyGapWad"`
	OracleBandWad  uint256.Int       `json:"oracleBandWad"`
	MaxDrawnAssets uint8             `json:"maxDrawnAssets"`
	Loans          []routerLoanReads `json:"loans"`
	BorrowOrder    []uint16          `json:"borrowOrder"`
	SupplyOrder    []uint16          `json:"supplyOrder"`
	WithdrawOrder  []uint16          `json:"withdrawOrder"`
	RepayOrder     []uint16          `json:"repayOrder"`
	Venues         []venueReads      `json:"venues"`
}

// adaptiveCurveIrm is the only rate model the port prices (irm.go): Morpho's AdaptiveCurveIrm on Base.
var adaptiveCurveIrm = common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687")

var (
	errReadsInconsistent = errors.New("everlong-flamm: inconsistent state reads")
	errReadsUnsupported  = errors.New("everlong-flamm: unsupported venue rate model")
)

func roundOf(r *roundReads) feedRound {
	return feedRound{Ok: r.Ok, RoundId: r.RoundId, Answer: uint256.Int(r.Answer), StartedAt: r.StartedAt,
		UpdatedAt: r.UpdatedAt}
}

// state builds the composed state for hooks, the pool's hook set as the registry resolves it (hook_registry.go):
// each role is built by its kind from that kind's reads. It refuses reads taken for another hook set or missing a
// kind's reads, reads it cannot evaluate without an index panic (loan sets that disagree, venues or priorities
// pointing past the arrays, a loan decimals above 18) and venues on a rate model other than AdaptiveCurveIrm.
func (r *flammReads) state(hooks *poolHookSet) (*flammState, error) {
	p := &r.Pool
	n := len(p.Loans)
	if hooks == nil || p.Hooks != hooks.Addrs || n == 0 || len(r.Router.Loans) != n {
		return nil, errReadsInconsistent
	}
	s := &flammState{
		Block:              r.Block,
		Timestamp:          r.Timestamp,
		PoolAsset:          p.Asset,
		Paused:             p.Paused,
		LevPaused:          p.LevPaused,
		FeeFloorWad:        p.FeeFloorWad,
		FeeCapWad:          p.FeeCapWad,
		ShareSupply:        p.TotalSupply,
		LastLeverSpreadPpm: p.LastLeverSpreadPpm,
		Hooks:              poolHooks{Addrs: p.Hooks},
		Pool: gatePool{Physical: p.Physical, LtvWad: p.LtvWad, PhiWad: p.PhiWad, RoomEpsilonWad: p.RoomEpsilonWad,
			Features: p.Features, Loans: make([]gateLoanCfg, n)},
	}
	for i := range p.Loans {
		l := &p.Loans[i]
		if l.Decimals > 18 {
			return nil, errReadsInconsistent
		}
		scale := *big256.TenPow(18 - int(l.Decimals))
		if !r.Router.Loans[i].LoanScale.Eq(&scale) {
			return nil, errReadsInconsistent
		}
		s.Pool.Loans[i] = gateLoanCfg{Token: l.Token, Scale: scale, SwapPriceBandWad: l.SwapPriceBandWad,
			FeeFloorWad: l.FeeFloorWad, MaxSwapNotional: l.MaxSwapNotional, ReserveTarget: l.ReserveTarget,
			Liquid: l.Liquid}
	}

	for role := roleSwap; role < hookRoles; role++ {
		e := hooks.Entries[role]
		if e == nil {
			if role == roleSwap {
				return nil, errReadsInconsistent
			}
			continue
		}
		spec := hookKindSpecOf(e.Kind)
		if spec == nil || spec.role() != role {
			return nil, errReadsInconsistent
		}
		if err := spec.build(r, s); err != nil {
			return nil, err
		}
	}

	f := &r.Feed
	s.Feed = priceFeedState{HasSequencer: f.SequencerFeed != (common.Address{}), SequencerGrace: f.SequencerGrace,
		Sequencer: roundOf(&f.Sequencer), Loans: make([]feedToken, n)}
	token := func(addr common.Address) feedToken {
		for i := range f.Tokens {
			if t := &f.Tokens[i]; t.Token == addr && t.Known {
				return feedToken{Known: true, Heartbeat: t.Heartbeat, Scale: t.Scale, Unit: t.Unit,
					PegBandWad: t.PegBandWad, Round: roundOf(&t.Round)}
			}
		}
		return feedToken{}
	}
	s.Feed.Asset = token(p.Asset)
	for i := range p.Loans {
		s.Feed.Loans[i] = token(p.Loans[i].Token)
	}

	rr := &r.Router
	s.Router = mmRouter{GlobalPaused: rr.GlobalPaused, PinLtvWad: rr.PinLtvWad, SafetyGapWad: rr.SafetyGapWad,
		OracleBandWad: rr.OracleBandWad, MaxDrawnAssets: rr.MaxDrawnAssets, Loans: make([]mmLoan, n),
		Venues: make([]mmVenue, len(rr.Venues))}
	for i := range rr.Loans {
		l := &rr.Loans[i]
		s.Router.Loans[i] = mmLoan{Decimals: l.Decimals, LoanScale: l.LoanScale, DebtCap: l.DebtCap,
			SupplyCap: l.SupplyCap, BorrowEnabled: l.BorrowEnabled, Retired: l.Retired}
	}
	for i := range rr.Venues {
		v := &rr.Venues[i]
		if int(v.LoanIndex) >= n || v.RateAtTarget.Sign() < 0 {
			return nil, errReadsInconsistent
		}
		if v.HasIrm && v.Irm != adaptiveCurveIrm {
			return nil, errReadsUnsupported
		}
		s.Router.Venues[i] = mmVenue{Kind: v.Kind, LoanIndex: v.LoanIndex, LltvWad: v.LltvWad,
			BorrowEnabled: v.BorrowEnabled, SupplyEnabled: v.SupplyEnabled, Retired: v.Retired, DebtCap: v.DebtCap,
			SupplyCap: v.SupplyCap, MaxBorrowRateWad: v.MaxBorrowRateWad, ManagedCollateral: v.ManagedCollateral,
			ManagedSupplyShares: v.ManagedSupplyShares,
			Morpho: mmVenueMarket{Market: v.Market, Position: v.Position, Lltv: v.MarketLltv, HasIrm: v.HasIrm,
				IrmReadable: v.IrmReadable, RateAtTarget: uint256.Int(v.RateAtTarget), OracleOk: v.OracleOk,
				OraclePrice: v.OraclePrice, OracleZero: v.OracleZero}}
	}
	for _, order := range [][]uint16{rr.BorrowOrder, rr.SupplyOrder, rr.WithdrawOrder, rr.RepayOrder} {
		for _, id := range order {
			if int(id) >= len(rr.Venues) {
				return nil, errReadsInconsistent
			}
		}
	}
	s.Router.BorrowOrder = append([]uint16(nil), rr.BorrowOrder...)
	s.Router.SupplyOrder = append([]uint16(nil), rr.SupplyOrder...)
	s.Router.WithdrawOrder = append([]uint16(nil), rr.WithdrawOrder...)
	s.Router.RepayOrder = append([]uint16(nil), rr.RepayOrder...)
	return s, nil
}
