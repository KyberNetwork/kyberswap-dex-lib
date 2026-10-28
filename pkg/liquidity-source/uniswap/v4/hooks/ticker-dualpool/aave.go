package tickerdualpool

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

// AaveInventory is the per-pool inventory plus the TickerVault and StataToken
// accounting at a single timestamp. This model is specific to those contracts;
// it must not be selected merely because an arbitrary vault is ERC-4626.
type AaveInventory struct {
	Raw, Claims, PoolShares, Supply, Idle, StrategyShares, Rate, PendingYield *uint256.Int
}

var ray = uint256.MustFromDecimal("1000000000000000000000000000")
var virtualShares = uint256.NewInt(1000000)

func (v AaveInventory) Clone() AaveInventory {
	copyInt := func(x *uint256.Int) *uint256.Int {
		if x == nil {
			return nil
		}
		return x.Clone()
	}
	return AaveInventory{copyInt(v.Raw), copyInt(v.Claims), copyInt(v.PoolShares), copyInt(v.Supply), copyInt(v.Idle), copyInt(v.StrategyShares), copyInt(v.Rate), copyInt(v.PendingYield)}
}
func (v AaveInventory) assets() (*uint256.Int, error) {
	if err := v.validate(); err != nil {
		return nil, err
	}
	assets, err := v3Utils.MulDiv(v.StrategyShares, v.Rate, ray)
	if err != nil {
		return nil, err
	}
	if _, overflow := assets.AddOverflow(assets, v.Idle); overflow {
		return nil, ErrInvalidState
	}
	return assets, nil
}
func (v AaveInventory) poolAssets() (*uint256.Int, error) {
	assets, err := v.assets()
	if err != nil {
		return nil, err
	}
	supply, err := addChecked(v.Supply, virtualShares, v.PendingYield)
	if err != nil {
		return nil, err
	}
	assets, err = addChecked(assets, uint256.NewInt(1))
	if err != nil {
		return nil, err
	}
	claim, err := v3Utils.MulDiv(v.PoolShares, assets, supply)
	if err != nil {
		return nil, err
	}
	return addChecked(claim, v.Raw, v.Claims)
}

// AfterJIT follows the actual order: redeem old backed claims, withdraw the
// deployment shortfall, resolve JIT's net delta, then deposit remaining raw cash.
// All old claims must be backed; the caller must enforce the full-liquidity rule.
// New positive deltas remain claims until the router settles, NOT vault deposits.
// This helper assumes successful uncapped deposits; gated/capped paths need a
// separate modeled outcome before the tracker can enable them.
func (v AaveInventory) AfterJIT(minted, returned *uint256.Int) (AaveInventory, error) {
	return v.afterJIT(minted, returned, nil)
}

// AfterJITWithReserve also models Aave's cash, cap and permission changes. The
// original inventory and reserve remain untouched even if the quote fails.
func (v AaveInventory) AfterJITWithReserve(minted, returned *uint256.Int, reserve AaveReserve) (AaveInventory, AaveReserve, error) {
	nextReserve := reserve.Clone()
	next, err := v.afterJIT(minted, returned, &nextReserve)
	if err != nil {
		return AaveInventory{}, AaveReserve{}, err
	}
	return next, nextReserve, nil
}

func (v AaveInventory) afterJIT(minted, returned *uint256.Int, reserve *AaveReserve) (AaveInventory, error) {
	fields := []*uint256.Int{v.Raw, v.Claims, v.PoolShares, v.Supply, v.Idle, v.StrategyShares, v.Rate, v.PendingYield, minted, returned}
	for _, x := range fields {
		if x == nil {
			return AaveInventory{}, ErrInvalidState
		}
	}
	if err := v.validate(); err != nil {
		return AaveInventory{}, err
	}
	if v.Rate.IsZero() {
		return AaveInventory{}, ErrInvalidState
	}
	v = v.Clone()
	var err error
	v.Raw, err = addChecked(v.Raw, v.Claims)
	if err != nil {
		return AaveInventory{}, err
	}
	v.Claims.Clear()
	if v.Raw.BitLen() > 128 {
		return AaveInventory{}, ErrInvalidState
	}
	if minted.Cmp(v.Raw) > 0 {
		needed := new(uint256.Int).Sub(minted, v.Raw)
		assets, err := v.assets()
		if err != nil {
			return AaveInventory{}, err
		}
		v.Supply, err = addChecked(v.Supply, v.PendingYield)
		if err != nil {
			return AaveInventory{}, err
		}
		v.PendingYield.Clear()
		numerator, err := addChecked(v.Supply, virtualShares)
		if err != nil {
			return AaveInventory{}, err
		}
		denominator, err := addChecked(assets, uint256.NewInt(1))
		if err != nil {
			return AaveInventory{}, err
		}
		shares, err := v3Utils.MulDivRoundingUp(needed, numerator, denominator)
		if err != nil {
			return AaveInventory{}, err
		}
		if shares.Cmp(v.PoolShares) > 0 || shares.Cmp(v.Supply) > 0 {
			return AaveInventory{}, ErrVaultIlliquid
		}
		if needed.Cmp(v.Idle) > 0 {
			shortfall := new(uint256.Int).Sub(needed, v.Idle)
			if reserve != nil {
				if err := reserve.withdraw(shortfall, v.StrategyShares, v.Rate); err != nil {
					return AaveInventory{}, err
				}
			}
			burned, err := v3Utils.MulDivRoundingUp(shortfall, ray, v.Rate)
			if err != nil {
				return AaveInventory{}, err
			}
			if burned.Cmp(v.StrategyShares) > 0 {
				return AaveInventory{}, ErrVaultIlliquid
			}
			v.StrategyShares.Sub(v.StrategyShares, burned)
			v.Idle.Clear()
		} else {
			v.Idle.Sub(v.Idle, needed)
		}
		v.PoolShares.Sub(v.PoolShares, shares)
		v.Supply.Sub(v.Supply, shares)
		v.Raw, err = addChecked(v.Raw, needed)
		if err != nil {
			return AaveInventory{}, err
		}
	}
	if v.Raw.BitLen() > 128 {
		return AaveInventory{}, ErrInvalidState
	}
	if returned.Cmp(minted) >= 0 {
		v.Claims.Sub(returned, minted)
	} else {
		owed := new(uint256.Int).Sub(minted, returned)
		if owed.Cmp(v.Raw) > 0 {
			return AaveInventory{}, ErrInvalidState
		}
		v.Raw.Sub(v.Raw, owed)
	}
	if !v.Raw.IsZero() {
		assets, err := v.assets()
		if err != nil {
			return AaveInventory{}, err
		}
		supply, err := addChecked(v.Supply, v.PendingYield)
		if err != nil {
			return AaveInventory{}, err
		}
		numerator, err := addChecked(supply, virtualShares)
		if err != nil {
			return AaveInventory{}, err
		}
		denominator, err := addChecked(assets, uint256.NewInt(1))
		if err != nil {
			return AaveInventory{}, err
		}
		shares, err := v3Utils.MulDiv(v.Raw, numerator, denominator)
		if err != nil {
			return AaveInventory{}, err
		}
		strategyShares, err := v3Utils.MulDiv(v.Raw, ray, v.Rate)
		if err != nil {
			return AaveInventory{}, err
		}
		// Ticker's catch leaves raw inventory intact if either deposit would mint zero.
		if !shares.IsZero() && !strategyShares.IsZero() {
			if reserve != nil {
				accepted, err := reserve.deposit(v.Raw, v.Rate)
				if err != nil {
					return AaveInventory{}, err
				}
				if !accepted {
					return v, v.validate()
				}
			}
			v.Supply, err = addChecked(supply, shares)
			if err != nil {
				return AaveInventory{}, err
			}
			v.PendingYield.Clear()
			v.PoolShares, err = addChecked(v.PoolShares, shares)
			if err != nil {
				return AaveInventory{}, err
			}
			v.StrategyShares, err = addChecked(v.StrategyShares, strategyShares)
			if err != nil {
				return AaveInventory{}, err
			}
			v.Raw.Clear()
		}
	}
	if err := v.validate(); err != nil {
		return AaveInventory{}, err
	}
	return v, nil
}

func addChecked(values ...*uint256.Int) (*uint256.Int, error) {
	total := new(uint256.Int)
	for _, value := range values {
		if value == nil {
			return nil, ErrInvalidState
		}
		if _, overflow := total.AddOverflow(total, value); overflow {
			return nil, ErrInvalidState
		}
	}
	return total, nil
}
func (v AaveInventory) validate() error {
	for _, x := range []*uint256.Int{v.Raw, v.Claims, v.PoolShares, v.Supply, v.Idle, v.StrategyShares, v.Rate, v.PendingYield} {
		if x == nil {
			return ErrInvalidState
		}
	}
	if v.Rate.IsZero() || v.Raw.BitLen() > 128 || v.Claims.BitLen() > 128 || v.PoolShares.Cmp(v.Supply) > 0 {
		return ErrInvalidState
	}
	return nil
}

// realizableAssets values this pool's shares, then caps them by deliverable cash.
// Raw inventory and backed claims are separate from the vault's share value.
func (v AaveInventory) realizableAssets(reserve AaveReserve) (*uint256.Int, error) {
	total, err := v.poolAssets()
	if err != nil {
		return nil, err
	}
	claim := new(uint256.Int).Sub(total, v.Raw)
	claim.Sub(claim, v.Claims)
	withdrawable, err := reserve.withdrawalLimit(v.StrategyShares, v.Rate)
	if err != nil {
		return nil, err
	}
	cash, err := addChecked(v.Idle, withdrawable)
	if err != nil {
		return nil, err
	}
	if claim.Cmp(cash) > 0 {
		claim.Set(cash)
	}
	return addChecked(claim, v.Raw, v.Claims)
}
