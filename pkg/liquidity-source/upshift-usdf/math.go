package upshiftusdf

import (
	"github.com/holiman/uint256"
)

var (
	usdfMultiplier = uint256.NewInt(1_000_000)   // USDF_DECIMAL_MULTIPLIER
	aprScale       = uint256.NewInt(100_000_000) // 1e6 * 100, as in compute()
	u36500         = uint256.NewInt(36500)
	u10000         = uint256.NewInt(10000)
)

// rollDays ports the day count compute() advances at block timestamp ts:
// min(diffDays(startOfYear, ts) - currentPeriod, 30), or 0 when not behind.
func rollDays(startOfYear, currentPeriod, ts uint64) (uint64, error) {
	if ts < startOfYear {
		return 0, ErrClockBeforeStart
	}
	period := (ts - startOfYear) / secondsPerDay
	if period <= currentPeriod {
		return 0, nil
	}
	return min(period-currentPeriod, maxRollDays), nil
}

// rolledPrice sets res = tokenPrice after `days` iterations of compute():
// price += (apr*1e6*100/36500)/10000 each, both divisions floored, all checked.
func rolledPrice(res, apr, tokenPrice *uint256.Int, days uint64) error {
	var step, n uint256.Int
	if _, overflow := step.MulOverflow(apr, aprScale); overflow {
		return ErrOverflow
	}
	step.Div(&step, u36500).Div(&step, u10000)
	if _, overflow := n.MulOverflow(&step, n.SetUint64(days)); overflow {
		return ErrOverflow
	}
	if _, overflow := res.AddOverflow(tokenPrice, &n); overflow {
		return ErrOverflow
	}
	return nil
}

// mintShares ports deposit() after compute(): amount*1e6/price, then canMint and mint checks.
func mintShares(res, amount, price, totalSupply, maxSupply *uint256.Int) error {
	if price.IsZero() {
		return ErrZeroPrice
	}
	if _, overflow := res.MulOverflow(amount, usdfMultiplier); overflow {
		return ErrOverflow
	}
	res.Div(res, price)
	if res.IsZero() {
		return ErrZeroShares
	}
	if !maxSupply.IsZero() {
		var sum uint256.Int
		if _, overflow := sum.AddOverflow(totalSupply, res); overflow || sum.Gt(maxSupply) {
			return ErrSupplyExceeded
		}
	}
	return nil
}
