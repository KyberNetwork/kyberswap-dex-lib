package lotflow

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// The LOT/USDG venue swap of NavJitHookV17 (MemeFiLP contracts/src/NavJitHookV17.sol `_quote`,
// `_place`, `_range`, `_reposition`, `_afterSwap`, `_requireInBand`, `_unwind`; P2-FIX.md §5.3).
// Exact-input only. Per swap the hook sizes an inventory, puts a one-sided JIT position of
// WidthTicks ticks at a price edge (cost+spread on a buy, NAV-spread on a sell), lets the user's
// swap fill against it, removes it, checks the band and unwinds through the basket.

// Params is NavJitHookV17.Params.
type Params struct {
	BuySpreadBps  uint64
	SellSpreadBps uint64
	BandBps       uint64
	SizeBufferBps uint64
	WidthTicks    int
}

// Venue is one LOT/USDG pool on the hook. The pool has fee 0, tickSpacing 1 and (in every case the
// simulator supports) no standing position: the JIT position is the only liquidity.
type Venue struct {
	UsdgIs0      bool // USDG is currency0
	UsdgDecimals int
	Params       Params

	// SqrtPriceX96 / Tick are the pool's slot0 before the swap (the reposition starts here).
	SqrtPriceX96 *big.Int
	Tick         int
	// Standing is the hook's standing position on this venue (nil = none); see standing.go.
	Standing *StandingPosition
}

func (v *Venue) toUsd18(usdgRaw *big.Int) *big.Int {
	return new(big.Int).Mul(usdgRaw, pow10(18-v.UsdgDecimals))
}

func (v *Venue) fromUsd18(usd18 *big.Int) *big.Int {
	return new(big.Int).Quo(usd18, pow10(18-v.UsdgDecimals))
}

// ZeroForOne is the venue swap direction of a buy (USDG in) or sell (LOT in).
func (v *Venue) ZeroForOne(buyLot bool) bool { return buyLot == v.UsdgIs0 }

// BuyInventory is `_quote` on an exact-input buy of `amountIn` raw USDG: the LOT to mint,
// floor(floor(a*1e12*1e18/nav)*(1e4+sizeBufferBps)/1e4).
func (v *Venue) BuyInventory(amountIn, nav *big.Int) (*big.Int, error) {
	inv := bpsUp(mulDiv(v.toUsd18(amountIn), bigE18, nav), v.Params.SizeBufferBps)
	if inv.Sign() == 0 {
		return nil, ErrTradeTooSmall
	}
	return inv, nil
}

// BuyPrice18 is the buy edge: the executed constituent cost (raw USDG) per LOT plus buySpreadBps.
func (v *Venue) BuyPrice18(cost, inventory *big.Int) *big.Int {
	return bpsUp(mulDiv(v.toUsd18(cost), bigE18, inventory), v.Params.BuySpreadBps)
}

// SellPrice18 is the sell edge: NAV less sellSpreadBps.
func (v *Venue) SellPrice18(nav *big.Int) *big.Int { return bpsDown(nav, v.Params.SellSpreadBps) }

// SellInventory is `_quote` on an exact-input sell of `amountIn` LOT wei: the USDG (raw) placed,
// floor(floor(a*price18/1e18)/1e12*(1e4+sizeBufferBps)/1e4).
func (v *Venue) SellInventory(amountIn, price18 *big.Int) (*big.Int, error) {
	inv := bpsUp(v.fromUsd18(mulDiv(amountIn, price18, bigE18)), v.Params.SizeBufferBps)
	if inv.Sign() == 0 {
		return nil, ErrTradeTooSmall
	}
	return inv, nil
}

// SqrtPriceForUsd18 is `_sqrtPriceForUsd18`.
func (v *Venue) SqrtPriceForUsd18(p18 *big.Int) *big.Int {
	scale := pow10(36 - v.UsdgDecimals)
	var ratioX192 *big.Int
	if v.UsdgIs0 {
		ratioX192 = mulDiv(scale, bigQ192, p18)
	} else {
		ratioX192 = mulDiv(p18, bigQ192, scale)
	}
	return new(big.Int).Sqrt(ratioX192)
}

// PoolUsd18 is `_poolUsd18`: USD (1e18) per LOT at venue sqrtPrice s. A sqrtPrice low enough that
// raw18 floors to zero is FullMath.mulDiv's zero-denominator revert on chain.
func (v *Venue) PoolUsd18(s *big.Int) (*big.Int, error) {
	scale := pow10(36 - v.UsdgDecimals)
	raw18 := mulDiv(s, new(big.Int).Mul(s, bigE18), bigQ192)
	if v.UsdgIs0 {
		if raw18.Sign() == 0 {
			return nil, ErrMulDivZero
		}
		return new(big.Int).Quo(mulDiv(scale, bigE36, raw18), bigE18), nil
	}
	return mulDiv(raw18, scale, bigE36), nil
}

// Position is the per-swap JIT position after `_place`: the pool repositioned to the edge, then
// Liquidity added to [Lower, Upper] for Added0/Added1 (rounded up, v4 modifyLiquidity).
type Position struct {
	Lower, Upper int
	EdgeTick     int
	Liquidity    *big.Int
	// pool slot0 after the reposition (the zero-amount hook Swap the indexer must skip)
	SqrtPriceX96 *big.Int
	Tick         int
	// what adding the position cost the hook, per currency
	Added0, Added1 *big.Int
	inventory      *big.Int
	buyLot         bool
}

// Range is `_range`: the edge tick b, rounded in the hook's favour, and the position
// [b-width, b] if the user's swap moves the price down, else [b, b+width].
func (v *Venue) Range(price18 *big.Int, buyLot bool) (lower, upper, edge int, err error) {
	if price18.Sign() <= 0 {
		return 0, 0, 0, fmt.Errorf("lotflow: zero edge price") // FullMath.mulDiv reverts on chain
	}
	t, err := TickAtSqrtPrice(v.SqrtPriceForUsd18(price18))
	if err != nil {
		return 0, 0, 0, err
	}
	roundUp := v.UsdgIs0
	if buyLot {
		roundUp = !v.UsdgIs0
	}
	b := t
	if roundUp {
		b = t + 1
	}
	w := v.Params.WidthTicks
	if v.ZeroForOne(buyLot) { // priceDown
		return b - w, b, b, nil
	}
	return b, b + w, b, nil
}

// Place is `_range` + `_reposition` + the JIT add of `_place`. The pool's pre-swap slot0 must be set
// on the Venue (SqrtPriceX96, Tick).
func (v *Venue) Place(price18, inventory *big.Int, buyLot bool) (*Position, error) {
	lower, upper, b, err := v.Range(price18, buyLot)
	if err != nil {
		return nil, err
	}
	edge, err := SqrtPriceAtTick(b)
	if err != nil {
		return nil, err
	}
	sqrtP, tick, err := v.reposition(edge)
	if err != nil {
		return nil, err
	}
	sa, err := SqrtPriceAtTick(lower)
	if err != nil {
		return nil, err
	}
	sb, err := SqrtPriceAtTick(upper)
	if err != nil {
		return nil, err
	}
	var liq *big.Int
	if v.ZeroForOne(buyLot) {
		// LiquidityAmounts.getLiquidityForAmount1
		liq = mulDiv(inventory, bigQ96, new(big.Int).Sub(sb, sa))
	} else {
		// LiquidityAmounts.getLiquidityForAmount0
		liq = mulDiv(inventory, mulDiv(sa, sb, bigQ96), new(big.Int).Sub(sb, sa))
	}
	if liq.Cmp(maxU128) > 0 {
		return nil, ErrLiquidityCast
	}
	if liq.Sign() == 0 {
		return nil, ErrTradeTooSmall
	}
	a0, a1, err := positionAmounts(sqrtP, tick, lower, upper, liq, true)
	if err != nil {
		return nil, err
	}
	return &Position{Lower: lower, Upper: upper, EdgeTick: b, Liquidity: liq, SqrtPriceX96: sqrtP, Tick: tick,
		Added0: a0, Added1: a1, inventory: inventory, buyLot: buyLot}, nil
}

// reposition is `_reposition` from the venue's pre-swap slot0.
func (v *Venue) reposition(target *big.Int) (*big.Int, int, error) {
	if v.SqrtPriceX96 == nil {
		return nil, 0, fmt.Errorf("lotflow: venue pre-swap slot0 not set")
	}
	return repositionFrom(v.SqrtPriceX96, v.Tick, target)
}

// repositionFrom is `_reposition`: a 1-wei exact-input swap with limit `target` through the empty
// pool. It returns slot0 after it; the tick follows the v4 swap loop (a stop exactly on a word edge
// moving down leaves tick = edge-1), so it is run through the same swap engine as everything else.
func repositionFrom(sqrtP *big.Int, tick int, target *big.Int) (*big.Int, int, error) {
	if sqrtP.Cmp(target) == 0 {
		return new(big.Int).Set(sqrtP), tick, nil
	}
	p, err := newV3Pool(0, sqrtP, new(big.Int), tick, nil, 1)
	if err != nil {
		return nil, 0, err
	}
	zfo := target.Cmp(sqrtP) < 0
	res, err := p.Swap(zfo, *uint256.NewInt(1), *uint256.MustFromBig(target))
	if err != nil {
		return nil, 0, err
	}
	return res.SqrtRatioX96.ToBig(), res.CurrentTick, nil
}

// VenueSwapResult is the user's swap against the JIT position.
type VenueSwapResult struct {
	AmountIn     *big.Int // input consumed (== amountIn unless the swap ran into its price limit)
	AmountOut    *big.Int
	SqrtPriceX96 *big.Int
	Tick         int
	// what removing the position returned to the hook, per currency (rounded down)
	Removed0, Removed1 *big.Int
	pos                *Position
	usdgIs0            bool
}

// Swap runs the user's exact-input swap of amountIn with sqrtPriceLimitX96 against pos, then the
// position's removal (`_removeJit`).
func (v *Venue) Swap(pos *Position, buyLot bool, amountIn, sqrtPriceLimitX96 *big.Int) (*VenueSwapResult, error) {
	active := new(big.Int)
	if pos.Lower <= pos.Tick && pos.Tick < pos.Upper {
		active.Set(pos.Liquidity)
	}
	ticks := []uniswapv3.TickU256{
		{Index: pos.Lower, LiquidityGross: uint256.MustFromBig(pos.Liquidity),
			LiquidityNet: int256.MustFromBig(pos.Liquidity)},
		{Index: pos.Upper, LiquidityGross: uint256.MustFromBig(pos.Liquidity),
			LiquidityNet: int256.MustFromBig(new(big.Int).Neg(pos.Liquidity))},
	}
	p, err := newV3Pool(0, pos.SqrtPriceX96, active, pos.Tick, ticks, 1)
	if err != nil {
		return nil, err
	}
	amt, overflow := uint256.FromBig(amountIn)
	if overflow || amt.Sign() < 0 {
		return nil, fmt.Errorf("lotflow: amountIn out of range")
	}
	lim, overflow := uint256.FromBig(sqrtPriceLimitX96)
	if overflow {
		return nil, fmt.Errorf("lotflow: sqrtPriceLimitX96 out of range")
	}
	res, err := p.Swap(v.ZeroForOne(buyLot), *amt, *lim)
	if err != nil {
		return nil, err
	}
	consumed := new(big.Int).Sub(amountIn, signed(&res.RemainingAmountIn))
	out := new(big.Int).Neg(signed(&res.AmountCalculated))
	sqrtP := res.SqrtRatioX96.ToBig()
	r0, r1, err := positionAmounts(sqrtP, res.CurrentTick, pos.Lower, pos.Upper, pos.Liquidity, false)
	if err != nil {
		return nil, err
	}
	return &VenueSwapResult{AmountIn: consumed, AmountOut: out, SqrtPriceX96: sqrtP, Tick: res.CurrentTick,
		Removed0: r0, Removed1: r1, pos: pos, usdgIs0: v.UsdgIs0}, nil
}

func (r *VenueSwapResult) usdgLot(a0, a1 *big.Int) (usdg, lot *big.Int) {
	if r.usdgIs0 {
		return a0, a1
	}
	return a1, a0
}

// LotLeftOnHook is the LOT `_unwind` redeems: on a buy, the minted inventory the position did not
// take plus what its removal returned; on a sell, what the removal returned (the user's LOT, which
// the hook takes from the PoolManager). inventory is the buy's minted LOT (ignored on a sell).
func (r *VenueSwapResult) LotLeftOnHook(buyLot bool, inventory *big.Int) *big.Int {
	_, lotRemoved := r.usdgLot(r.Removed0, r.Removed1)
	left := new(big.Int).Set(lotRemoved)
	if buyLot {
		_, lotAdded := r.usdgLot(r.pos.Added0, r.pos.Added1)
		left.Add(left, new(big.Int).Sub(inventory, lotAdded))
	}
	return left
}

// HookUsdgDelta is the hook's USDG PoolManager delta from the venue alone: removed minus added. The
// caller adds sell-leg proceeds and subtracts buy-leg cost to get the `_unwind` delta.
func (r *VenueSwapResult) HookUsdgDelta(_ bool) *big.Int {
	usdgRemoved, _ := r.usdgLot(r.Removed0, r.Removed1)
	usdgAdded, _ := r.usdgLot(r.pos.Added0, r.pos.Added1)
	return new(big.Int).Sub(usdgRemoved, usdgAdded)
}

// RequireInBand is `_requireInBand` on the venue price after the user's swap.
func (v *Venue) RequireInBand(sqrtP, nav *big.Int) error {
	px, err := v.PoolUsd18(sqrtP)
	if err != nil {
		return err
	}
	diff := new(big.Int).Sub(px, nav)
	diff.Abs(diff)
	if new(big.Int).Mul(diff, bigBps).Cmp(new(big.Int).Mul(nav, new(big.Int).SetUint64(v.Params.BandBps))) > 0 {
		return &OutsideBandError{Px: px, Nav: new(big.Int).Set(nav)}
	}
	return nil
}

// RequireFloat is `_unwind`'s pay-after sell check: the LOT the hook takes must be in the PoolManager.
func RequireFloat(floatLot, needed *big.Int) error {
	if floatLot.Cmp(needed) < 0 {
		return &SellExceedsFloatError{Float: new(big.Int).Set(floatLot), Needed: new(big.Int).Set(needed)}
	}
	return nil
}

// RequireProfitable is `_unwind`'s final check on the hook's USDG delta.
func RequireProfitable(usdgDelta *big.Int) error {
	if usdgDelta.Sign() < 0 {
		return &NotProfitableError{Delta: new(big.Int).Set(usdgDelta)}
	}
	return nil
}

// ---------------------------------------------------------------- shared swap engine

// sentinel ticks: initialized with zero net at the ends of the tick range, so a swap that runs out
// of real liquidity keeps walking (exactly, at zero liquidity) to its price limit as on chain
// instead of stopping at the end of the tick list.
func sentinels(spacing int) (lo, hi int) {
	lo = uniswapv3.MinTick / spacing * spacing
	hi = uniswapv3.MaxTick / spacing * spacing
	return lo, hi
}

// newV3Pool builds dex-lib's uniswap v3 swap engine (the one its v3 and v4 pool simulators run on)
// directly from slot0, in-range liquidity and initialized ticks. It bypasses NewPool's checks on
// purpose: a real pool's tick may sit one below its sqrtPrice's tick after a downward stop on a tick
// boundary, and a fixture's tick window need not sum to zero net liquidity.
func newV3Pool(fee uint32, sqrtP, liquidity *big.Int, tick int, ticks []uniswapv3.TickU256,
	spacing int) (*uniswapv3.Pool, error) {
	lo, hi := sentinels(spacing)
	all := make([]uniswapv3.TickU256, 0, len(ticks)+2)
	if len(ticks) == 0 || ticks[0].Index > lo {
		all = append(all, uniswapv3.TickU256{Index: lo, LiquidityGross: uint256.NewInt(1),
			LiquidityNet: int256.NewInt(0)})
	}
	all = append(all, ticks...)
	if len(ticks) == 0 || ticks[len(ticks)-1].Index < hi {
		all = append(all, uniswapv3.TickU256{Index: hi, LiquidityGross: uint256.NewInt(1),
			LiquidityNet: int256.NewInt(0)})
	}
	prices := make([]uint256.Int, len(all))
	for i, t := range all {
		if err := uniswapv3.GetSqrtRatioAtTick(t.Index, &prices[i]); err != nil {
			return nil, err
		}
	}
	return &uniswapv3.Pool{
		Fee:            uniswapv3.FeeAmount(fee),
		TickSpacing:    spacing,
		TickCurrent:    tick,
		SqrtRatioX96:   *uint256.MustFromBig(sqrtP),
		Liquidity:      *uint256.MustFromBig(liquidity),
		Ticks:          all,
		TickSqrtPrices: prices,
	}, nil
}

// signed reads a uint256 holding a two's-complement int256.
func signed(x *uint256.Int) *big.Int {
	if x.Sign() < 0 {
		var n uint256.Int
		n.Neg(x)
		return new(big.Int).Neg(n.ToBig())
	}
	return x.ToBig()
}
