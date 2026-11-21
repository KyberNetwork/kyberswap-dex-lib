package base

import (
	"time"

	"github.com/holiman/uint256"
)

// xp returns the rate-scaled balances, as _xp does.
func (t *PoolSimulator) xp() ([]*uint256.Int, error) {
	return xpMemU256(t.Info.Reserves, t.Rates, Precision)
}

// GetDy returns the output of dx of coin i in coin j, and the fee, both in coin j. It recalculates dCached if nil.
func (t *PoolSimulator) GetDy(i, j int, dx, dCached *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	xp, err := t.xp()
	if err != nil {
		return nil, nil, err
	}
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}
	d := dCached
	if d == nil {
		if d, err = t.getDU256(xp, a); err != nil {
			return nil, nil, err
		}
	}
	return t.getDyU256(i, j, dx, xp, a, d)
}

// GetVirtualPrice returns the virtual price and D.
func (t *PoolSimulator) GetVirtualPrice() (*uint256.Int, *uint256.Int, error) {
	xp, err := t.xp()
	if err != nil {
		return nil, nil, err
	}
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}
	d, err := t.getDU256(xp, a)
	if err != nil {
		return nil, nil, err
	}
	return t.getVirtualPriceU256(d, t.LpSupply)
}

func (t *PoolSimulator) CalculateTokenAmount(amounts []*uint256.Int, deposit bool) (*uint256.Int, error) {
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, err
	}
	return t.calculateTokenAmountU256(amounts, deposit, t.Info.Reserves, t.Rates, a, t.LpSupply)
}

func (t *PoolSimulator) CalculateWithdrawOneCoin(tokenAmount *uint256.Int, i int) (*uint256.Int, *uint256.Int, error) {
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}
	xp, err := t.xp()
	if err != nil {
		return nil, nil, err
	}
	d0, err := t.getDU256(xp, a)
	if err != nil {
		return nil, nil, err
	}
	return t.calculateWithdrawOneCoinU256(tokenAmount, i, xp, a, d0, t.LpSupply)
}

// AddLiquidity mints LP for amounts and commits the new reserves and LP supply.
func (t *PoolSimulator) AddLiquidity(amounts []*uint256.Int) (*uint256.Int, error) {
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, err
	}
	mint, reserves, err := t.addLiquidityU256(amounts, t.Info.Reserves, a, t.LpSupply, t.Info.SwapFee, t.AdminFee)
	if err != nil {
		return nil, err
	}
	t.Info.Reserves = reserves
	t.LpSupply = new(uint256.Int).Add(t.LpSupply, mint)
	return mint, nil
}

// RemoveLiquidityOneCoin burns tokenAmount LP for coin i and commits the new reserves and LP supply.
func (t *PoolSimulator) RemoveLiquidityOneCoin(tokenAmount *uint256.Int, i int) (*uint256.Int, error) {
	a, err := t._AU256(time.Now().Unix())
	if err != nil {
		return nil, err
	}
	xp, err := t.xp()
	if err != nil {
		return nil, err
	}
	d0, err := t.getDU256(xp, a)
	if err != nil {
		return nil, err
	}
	dy, reserve, lpSupply, err := t.removeLiquidityOneCoinU256(tokenAmount, i, xp, t.Info.Reserves, a, d0, t.LpSupply, t.AdminFee)
	if err != nil {
		return nil, err
	}
	t.Info.Reserves[i] = reserve
	t.LpSupply = lpSupply
	return dy, nil
}
