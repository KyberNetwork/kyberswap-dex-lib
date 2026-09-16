package everlongflamm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// The composed pool state the swap and leverage entries evaluate (c104 @ 80abd43): FLAMMStore's ledger, dials,
// limits and switches (src/core/flamm/FLAMMStore.sol), the Router record with every venue's Morpho market
// (src/core/mm), the pool's hooks as their kinds' states (hook_kinds.go poolHooks: the swap hook's storage,
// src/hooks/everlong/EverlongHook.sol, and the spread hook's post, src/hooks/everlong/lev/LeverageSpreadHook.sol;
// the leverage hook is stateless and reads the swap hook's book) and the PriceFeed's inputs (src/core/PriceFeed.sol).
//
// Every entry takes the block timestamp it runs at. Almost nothing in the state is a price or an accrual already
// evaluated at the snapshot: the feed checks, the Morpho accrual and the spread's age are all recomputed at
// `now`. The one exception is a venue's Morpho market oracle answer (mmVenueMarket.OraclePrice), which is read
// once and frozen although the chain's own answer can move with the clock alone: the BTC/USD feed the c104
// market oracle prices from is a Chainlink SVR DualAggregator, which withholds each primary round for a fixed
// delay and reveals it when block.timestamp passes it, with no transaction in between. Only MMRouterLib.bandOk
// (MMRouterLib.sol:780-788) and Morpho's _isHealthy (morpho-blue Morpho.sol:527-539) read that price, and the
// tracker also reads what it answers at the far end of the snapshot window (Extra.OracleAhead), which the
// simulator requires every fill to settle identically at (pool_simulator.go oracleShifted). Subject to that, a
// state read at block B quotes exactly what the chain would at any later timestamp with no transaction in
// between.

// FLAMMStore feature bits (FLAMMStore.sol:190-200).
const (
	flammFeatureSwapSell      = 1
	flammFeatureSwapBuy       = 2
	flammFeatureSupplyLending = 3
	flammFeatureLeverage      = 5
)

// poolContext is IFLAMMHooks.PoolContext (IFLAMMHooks.sol:10), the one hook frame every route builds through
// gateContext (FLAMMGateLib.context): the poolAsset side in native base units, the loan side aggregated into the
// numeraire (N18), priceWad N18 per poolAsset base unit.
type poolContext struct {
	PhysicalPoolAsset uint256.Int
	PostedPoolAsset   uint256.Int
	LiquidLoanAsset   uint256.Int
	SuppliedLoanAsset uint256.Int
	DebtLoanAsset     uint256.Int
	ShareSupply       uint256.Int
	PriceWad          uint256.Int
	PriceTs           uint64
	LoanCount         uint8
}

// swapContext is IFLAMMHooks.SwapContext (IFLAMMHooks.sol:28) without loanAsset, an address no hook reads.
// The loan leg (amountIn of a buy, maxAmountOut of a sell) is N18.
type swapContext struct {
	Pool         poolContext
	PoolAssetIn  bool
	AmountIn     uint256.Int
	MaxAmountOut uint256.Int
	LoanIndex    uint8
	CrossWad     uint256.Int
	FeeFloorWad  uint256.Int
}

// leverContext is IFLAMMLeverage.LeverContext (IFLAMMLeverage.sol:11): AmountIn is poolAsset base units up and
// L18 down; MaxOut is core's ceiling (the credit-zero room up, the gross poolAsset down), which the hook ignores.
type leverContext struct {
	Pool      poolContext
	Up        bool
	SpreadPpm uint256.Int
	AmountIn  uint256.Int
	MaxOut    uint256.Int
}

// spreadHookState is LeverageSpreadHook's storage core's staticcall reads.
type spreadHookState struct {
	Spread       uint256.Int `json:"spread"`
	MaxSpreadAge uint256.Int `json:"maxSpreadAge"`
	LastSetTs    uint256.Int `json:"lastSetTs"`
}

// spreadPpm is LeverageSpreadHook.spreadPpm (:75): no answer once a non-zero maxSpreadAge has elapsed since the
// last post.
func (h *spreadHookState) spreadPpm(now uint64) (bool, uint256.Int) {
	if !h.MaxSpreadAge.IsZero() {
		var deadline, nowU uint256.Int
		deadline.Add(&h.LastSetTs, &h.MaxSpreadAge) // uint48 + uint32
		if nowU.SetUint64(now).Gt(&deadline) {
			return false, uint256.Int{}
		}
	}
	return true, h.Spread
}

// liveThrough reports whether the post answers through now + margin (a live post is
// `block.timestamp <= lastSetTs + maxSpreadAge`, or maxSpreadAge == 0) with a ppm below PPM.
func (h *spreadHookState) liveThrough(now, margin uint64) bool {
	if !h.Spread.Lt(uPpm) {
		return false
	}
	if !h.MaxSpreadAge.IsZero() {
		var deadline, at uint256.Int
		deadline.Add(&h.LastSetTs, &h.MaxSpreadAge)
		at.SetUint64(now)
		if _, overflow := at.AddOverflow(&at, uint256.NewInt(margin)); overflow || at.Gt(&deadline) {
			return false
		}
	}
	return true
}

// expiry is the deadline a post that answers at now stops answering after: (lastSetTs, maxSpreadAge), none when
// the post does not answer or has no staleness window.
func (h *spreadHookState) expiry(now uint64) (lastSet, maxAge uint256.Int, ok bool) {
	if live, _ := h.spreadPpm(now); !live || h.MaxSpreadAge.IsZero() {
		return lastSet, maxAge, false
	}
	return h.LastSetTs, h.MaxSpreadAge, true
}

// flammState is one pool's complete swap-path state. Pool.PriceWad / Pool.CrossWad are not state: priced fills
// them from Feed at the timestamp of the call.
type flammState struct {
	// Block / Timestamp identify the snapshot the state was read at; entries run at their own timestamp.
	Block     uint64         `json:"block"`
	Timestamp uint64         `json:"timestamp"`
	PoolAsset common.Address `json:"poolAsset"`
	// Pool is FLAMMStore's physical balance, loan set (with each asset's band, fee floor, notional cap, reserve
	// target and liquid) and the ltv / phi / roomEpsilon dials and feature bits.
	Pool        gatePool    `json:"pool"`
	Paused      bool        `json:"paused"`
	LevPaused   bool        `json:"levPaused"`
	FeeFloorWad uint256.Int `json:"feeFloorWad"`
	FeeCapWad   uint256.Int `json:"feeCapWad"`
	ShareSupply uint256.Int `json:"shareSupply"`
	// LastLeverSpreadPpm is FLAMMStore.lastLeverSpreadPpm, the lever-down degrade value (no view: storage).
	LastLeverSpreadPpm uint256.Int `json:"lastLeverSpreadPpm"`
	// Hooks is hooks() with each role's kind and state; an empty leverage role is LeverageDisabled
	// (FLAMMLeverLib.sol:81), an empty spread role answers no spread.
	Hooks  poolHooks      `json:"hooks"`
	Router mmRouter       `json:"router"`
	Feed   priceFeedState `json:"feed"`
}

// clone deep-copies everything an execution writes: the ledger's loans, the Router's loans and venues, the hooks'
// states (the swap hook's book is committed in place) and the lever-down degrade value (a value).
func (s *flammState) clone() *flammState {
	c := *s
	c.Pool = *s.Pool.clone()
	c.Router = *s.Router.clone()
	c.Hooks = s.Hooks.clone()
	c.Feed = s.Feed.clone()
	return &c
}

// feature is FLAMMStore.requireFeature (:362): FeatureDisabled(bit) when the bit is clear.
func (s *flammState) feature(bit uint) error {
	var w uint256.Int
	if w.Rsh(&s.Pool.Features, bit).Uint64()&1 == 0 {
		return ErrFeatureDisabled
	}
	return nil
}

// price is FLAMMStore.price (:354): the checked loan-asset-0 / poolAsset cross, reverting as PriceFeed.cross does.
func (s *flammState) price(now uint64) (uint256.Int, uint64, error) {
	if len(s.Feed.Loans) == 0 {
		return uint256.Int{}, 0, errPanicIndex
	}
	return s.Feed.cross(&s.Feed.Asset, &s.Feed.Loans[0], now)
}

// pegOk is FLAMMStore.pegOk (:358): PriceFeed.pegOk(loans[idx].token), whose UnknownToken is not caught.
func (s *flammState) pegOk(idx int, now uint64) (bool, error) {
	if idx >= len(s.Feed.Loans) {
		return false, errPanicIndex
	}
	return s.Feed.pegOk(&s.Feed.Loans[idx], now)
}

// priced is FLAMMGateLib.priced (:196): the price-free book (Router positions at `now`), then priceIn (:179) --
// every leg's peekCross, and for loan assets past the first the USD ratio crossWad = mulDiv(usd_i, WAD, usd_0)
// when both USD peeks answer and usd_0 != 0. The frame is also written to pool.PriceWad / CrossWad, where the
// Router settlement legs (reclaim's price vector, the gate re-assertions) read it at the same timestamp.
func (s *flammState) priced(pool *gatePool, now uint64) (gateBook, error) {
	b, err := gateBookOf(pool, &s.Router, now)
	if err != nil {
		return b, err
	}
	n := len(b.Legs)
	if len(s.Feed.Loans) < n {
		return b, errPanicIndex
	}
	priceWad, crossWad := make([]uint256.Int, n), make([]uint256.Int, n)
	if n > 0 {
		if ok, p, _ := s.Feed.peekCross(&s.Feed.Asset, &s.Feed.Loans[0], now); ok {
			priceWad[0] = p
		}
		crossWad[0] = *uWad
	}
	if n > 1 {
		okUsd0, usd0, _ := s.Feed.peekUsd(&s.Feed.Loans[0], now)
		for i := 1; i < n; i++ {
			if ok, p, _ := s.Feed.peekCross(&s.Feed.Asset, &s.Feed.Loans[i], now); ok {
				priceWad[i] = p
			}
			okUsd, usdI, _ := s.Feed.peekUsd(&s.Feed.Loans[i], now)
			if okUsd0 && okUsd && !usd0.IsZero() {
				if crossWad[i], err = mmMulDivOZ(&usdI, uWad, &usd0); err != nil {
					return b, err
				}
			}
		}
	}
	pool.PriceWad, pool.CrossWad = priceWad, crossWad
	for i := range b.Legs {
		b.Legs[i].PriceWad, b.Legs[i].CrossWad = priceWad[i], crossWad[i]
	}
	return b, nil
}

// pair is FLAMMSwapLib.pair (:112): exactly one side is poolAsset and the other a live loan asset.
func (s *flammState) pair(tokenIn, tokenOut common.Address) (uint8, bool, error) {
	var other common.Address
	poolAssetIn := false
	switch {
	case tokenIn == s.PoolAsset:
		poolAssetIn, other = true, tokenOut
	case tokenOut == s.PoolAsset:
		other = tokenIn
	default:
		return 0, false, ErrInvalidPair
	}
	for i := range s.Pool.Loans {
		if s.Pool.Loans[i].Token == other {
			return uint8(i), poolAssetIn, nil
		}
	}
	return 0, false, ErrInvalidPair
}
