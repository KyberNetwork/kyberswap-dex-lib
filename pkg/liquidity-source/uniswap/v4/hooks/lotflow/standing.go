package lotflow

import "math/big"

// Standing venue liquidity (NavJitHookV17 `_liftStanding`, `_restoreStanding`, `_addStanding`;
// unchanged from V15). An owner-funded two-sided position, salt STANDING_SALT, kept centred on NAV
// for scanners. Per swap:
//  1. beforeSwap, before `_quote`: the position is removed (amounts rounded down) and the amounts
//     remembered. The user's swap therefore trades only the JIT position; standing capital never
//     fills a user and never enters `_unwind`'s profit check.
//  2. afterSwap, after `_removeJit` and `_requireInBand`, before `_unwind`: the empty pool is
//     repositioned to the centre tick of this swap's NAV and the position re-added on
//     [centre-half, centre+half], half = (upper-lower)/2, with the most liquidity the lifted amounts
//     allow less one wei per side (LiquidityAmounts.getLiquidityForAmounts). Whatever does not fit
//     is taken from the PoolManager to the hook's owner (the shared standing vault on live venues).
//     A re-add that rounds to zero liquidity deletes the standing record.
// A swap that reverts rolls all of this back.

// StandingPosition is NavJitHookV17.Standing for one venue.
type StandingPosition struct {
	Liquidity    *big.Int
	Lower, Upper int
}

// StandingResult is what one swap did to the standing position.
type StandingResult struct {
	Lifted0, Lifted1 *big.Int          // removed before the swap
	After            *StandingPosition // re-added position; nil = deleted (or the swap reverted first)
	Added0, Added1   *big.Int          // owed for the re-add (rounded up)
	// ToOwner0/1: lifted minus re-added, taken from the PoolManager to owner(); nil if not restored
	ToOwner0, ToOwner1 *big.Int
	// slot0 after the restore's reposition
	SqrtPriceX96 *big.Int
	Tick         int
}

// liftStanding is `_liftStanding` at the venue's pre-swap slot0.
func (v *Venue) liftStanding() (*StandingResult, error) {
	st := v.Standing
	if st == nil || st.Liquidity == nil || st.Liquidity.Sign() == 0 {
		return nil, nil
	}
	a0, a1, err := positionAmounts(v.SqrtPriceX96, v.Tick, st.Lower, st.Upper, st.Liquidity, false)
	if err != nil {
		return nil, err
	}
	return &StandingResult{Lifted0: a0, Lifted1: a1}, nil
}

// restoreStanding is `_restoreStanding` from the pool's slot0 after the user's swap.
func (v *Venue) restoreStanding(r *StandingResult, nav, sqrtP *big.Int, tick int) error {
	st := v.Standing
	half := (st.Upper - st.Lower) / 2 // Solidity int24 division truncates toward zero, as Go's does
	center, err := TickAtSqrtPrice(v.SqrtPriceForUsd18(nav))
	if err != nil {
		return err
	}
	target, err := SqrtPriceAtTick(center)
	if err != nil {
		return err
	}
	cur, curTick, err := repositionFrom(sqrtP, tick, target)
	if err != nil {
		return err
	}
	r.SqrtPriceX96, r.Tick = cur, curTick
	lower, upper := center-half, center+half
	sa, err := SqrtPriceAtTick(lower)
	if err != nil {
		return err
	}
	sb, err := SqrtPriceAtTick(upper)
	if err != nil {
		return err
	}
	liq := liquidityForAmounts(cur, sa, sb, holdBack(r.Lifted0), holdBack(r.Lifted1))
	if liq.Cmp(maxU128) > 0 {
		return ErrLiquidityCast
	}
	r.Added0, r.Added1 = new(big.Int), new(big.Int)
	if liq.Sign() != 0 {
		if r.Added0, r.Added1, err = positionAmounts(cur, curTick, lower, upper, liq, true); err != nil {
			return err
		}
		r.After = &StandingPosition{Liquidity: liq, Lower: lower, Upper: upper}
	}
	r.ToOwner0 = new(big.Int).Sub(r.Lifted0, r.Added0)
	r.ToOwner1 = new(big.Int).Sub(r.Lifted1, r.Added1)
	return nil
}

func holdBack(a *big.Int) *big.Int {
	if a.Sign() > 0 {
		return new(big.Int).Sub(a, big.NewInt(1))
	}
	return new(big.Int)
}

// liquidityForAmounts is v4-periphery LiquidityAmounts.getLiquidityForAmounts.
func liquidityForAmounts(cur, sa, sb, a0, a1 *big.Int) *big.Int {
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}
	switch {
	case cur.Cmp(sa) <= 0:
		return liquidityForAmount0(sa, sb, a0)
	case cur.Cmp(sb) < 0:
		l0, l1 := liquidityForAmount0(cur, sb, a0), liquidityForAmount1(sa, cur, a1)
		if l0.Cmp(l1) < 0 {
			return l0
		}
		return l1
	default:
		return liquidityForAmount1(sa, sb, a1)
	}
}

func liquidityForAmount0(sa, sb, a *big.Int) *big.Int {
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}
	return mulDiv(a, mulDiv(sa, sb, bigQ96), new(big.Int).Sub(sb, sa))
}

func liquidityForAmount1(sa, sb, a *big.Int) *big.Int {
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}
	return mulDiv(a, bigQ96, new(big.Int).Sub(sb, sa))
}
