package upshiftvaultv2

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// depositShares ports deposit() for the reference asset (previewDeposit returns shares =
// amount*supply/valuation and the amount unchanged), then _beforeDeposit's checks.
// amount*supply is full-precision (MulDivDown); the whitelist's own mulDiv is unverified.
func (e *Extra) depositShares(res, amount *uint256.Int) error {
	if e.DepositsPaused {
		return ErrDepositsPaused
	}
	if e.SenderGated {
		return ErrSenderGated
	}
	if amount.IsZero() {
		return ErrZeroAmount
	}
	if e.Supply.IsZero() || e.Valuation.IsZero() {
		return ErrNoSupply
	}
	big256.MulDivDown(res, amount, e.Supply, e.Valuation)
	if res.IsZero() {
		return ErrInsufficientShr
	}
	if amount.Gt(e.MaxDeposit) {
		return ErrMaxDeposit
	}
	var total uint256.Int
	if _, overflow := total.AddOverflow(amount, e.TotalAssets); overflow || total.Gt(e.DepositCap) {
		return ErrDepositCap
	}
	return nil
}

// Reserves reports the deposit headroom in the asset (min of maxDeposit and the cap) and its LP value.
func (e *Extra) Reserves() (asset, lp uint256.Int) {
	if e.DepositCap.Gt(e.TotalAssets) {
		asset.Sub(e.DepositCap, e.TotalAssets)
	}
	if asset.Gt(e.MaxDeposit) {
		asset.Set(e.MaxDeposit)
	}
	if !e.Valuation.IsZero() {
		big256.MulDivDown(&lp, &asset, e.Supply, e.Valuation)
	}
	return
}
