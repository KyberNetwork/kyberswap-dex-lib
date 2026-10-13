package tickerdualpool

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

// AaveReserve contains the reserve-wide state at the quote timestamp. ScaledSupply
// includes stored treasury shares; PendingTreasury accrues on the next mutation.
// It is shared by
// every StataToken holder, not just this TickerVault.
// Rounding follows Aave v3.5 TokenMath and ERC4626StataTokenUpgradeable:
// https://github.com/aave-dao/aave-v3-origin/tree/8305565ae342f1773c42cd2e4593f175fe5968a0/src/contracts
// The tracker reads stored state and computes the pending treasury accrual.
type AaveReserve struct {
	Cash, ScaledSupply, SupplyCap *uint256.Int
	PendingTreasury               *uint256.Int
	Active, Paused, Frozen        bool
}

func (r AaveReserve) Clone() AaveReserve {
	if r.PendingTreasury != nil {
		r.PendingTreasury = r.PendingTreasury.Clone()
	}
	if r.Cash != nil {
		r.Cash = r.Cash.Clone()
	}
	if r.ScaledSupply != nil {
		r.ScaledSupply = r.ScaledSupply.Clone()
	}
	if r.SupplyCap != nil {
		r.SupplyCap = r.SupplyCap.Clone()
	}
	return r
}
func (r AaveReserve) validate(rate *uint256.Int) error {
	if r.Cash == nil || r.ScaledSupply == nil || r.SupplyCap == nil || rate == nil || rate.IsZero() {
		return ErrInvalidState
	}
	return nil
}

// The strategy converts cash to shares, then back to cash. Preserve both floors:
// a reserve with 1,000 available units can report a withdrawal limit of 999.
func (r AaveReserve) withdrawalLimit(held, rate *uint256.Int) (*uint256.Int, error) {
	if err := r.validate(rate); err != nil {
		return nil, err
	}
	if held == nil {
		return nil, ErrInvalidState
	}
	if !r.Active || r.Paused {
		return new(uint256.Int), nil
	}
	shares, err := v3Utils.MulDiv(r.Cash, ray, rate)
	if err != nil {
		return nil, err
	}
	if shares.Cmp(held) > 0 {
		shares.Set(held)
	}
	return v3Utils.MulDiv(shares, rate, ray)
}
func (r AaveReserve) depositLimit(rate *uint256.Int) (*uint256.Int, error) {
	if err := r.validate(rate); err != nil {
		return nil, err
	}
	if !r.Active || r.Paused || r.Frozen {
		return new(uint256.Int), nil
	}
	if r.SupplyCap.IsZero() {
		return new(uint256.Int).SetAllOne(), nil
	}
	supplied, err := v3Utils.MulDivRoundingUp(r.ScaledSupply, rate, ray)
	if err != nil {
		return nil, err
	}
	if supplied.Cmp(r.SupplyCap) >= 0 {
		return new(uint256.Int), nil
	}
	return new(uint256.Int).Sub(r.SupplyCap, supplied), nil
}

// withdraw runs before the swap's redeposit. Withdrawing frees supply-cap room;
// it does not unfreeze deposits. Callers use a cloned reserve during quoting.
func (r *AaveReserve) withdraw(amount, held, rate *uint256.Int) error {
	if amount == nil {
		return ErrInvalidState
	}
	limit, err := r.withdrawalLimit(held, rate)
	if err != nil {
		return err
	}
	if amount.Cmp(limit) > 0 {
		return ErrVaultIlliquid
	}
	burned, err := v3Utils.MulDivRoundingUp(amount, ray, rate)
	if err != nil {
		return err
	}
	if burned.Cmp(r.ScaledSupply) > 0 {
		return ErrInvalidState
	}
	r.Cash = new(uint256.Int).Sub(r.Cash, amount)
	supply := new(uint256.Int).Sub(r.ScaledSupply, burned)
	if r.PendingTreasury != nil {
		var err error
		supply, err = addChecked(supply, r.PendingTreasury)
		if err != nil {
			return err
		}
	}
	r.ScaledSupply = supply
	r.PendingTreasury = new(uint256.Int)
	return nil
}

// deposit returns false when the wrapper deposit would fail. Its caller then
// leaves the money in raw inventory, exactly as DualPool's try/catch does.
func (r *AaveReserve) deposit(amount, rate *uint256.Int) (bool, error) {
	if amount == nil {
		return false, ErrInvalidState
	}
	limit, err := r.depositLimit(rate)
	if err != nil {
		return false, err
	}
	if amount.Cmp(limit) > 0 {
		return false, nil
	}
	minted, err := v3Utils.MulDiv(amount, ray, rate)
	if err != nil {
		return false, err
	}
	if minted.IsZero() {
		return false, nil
	}
	pending := r.PendingTreasury
	if pending == nil {
		pending = new(uint256.Int)
	}
	supply, err := addChecked(r.ScaledSupply, minted, pending)
	if err != nil {
		return false, err
	}
	// Pool's validation runs after treasury accrual, unlike maxDeposit above.
	if !r.SupplyCap.IsZero() {
		total, err := v3Utils.MulDiv(supply, rate, ray)
		if err != nil {
			return false, err
		}
		if total.Cmp(r.SupplyCap) > 0 {
			return false, nil
		}
	}
	cash, err := addChecked(r.Cash, amount)
	if err != nil {
		return false, err
	}
	r.ScaledSupply, r.Cash = supply, cash
	r.PendingTreasury = new(uint256.Int)
	return true, nil
}
