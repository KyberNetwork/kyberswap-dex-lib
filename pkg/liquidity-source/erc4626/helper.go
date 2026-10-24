package erc4626

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// GetClosestRate converts amount using rates sampled at PrefetchAmounts (rates[i] = preview(PrefetchAmounts[i])).
// It interpolates linearly between samples, through (0, 0) below the first one and at the last sample's rate
// beyond the last one. isExactOut: amount is an amountOut, and it returns the min amountIn converting to at least
// amount; CalcAmountIn reverses exact-in, so both directions must share this one curve.
func GetClosestRate(rates []*uint256.Int, amount *uint256.Int, isExactOut bool) (*uint256.Int, error) {
	xLo, yLo := u256.U0, u256.U0
	var sampled bool
	for i, y := range rates {
		if y == nil || i >= len(PrefetchAmounts) {
			continue
		}
		x := PrefetchAmounts[i]
		if isExactOut {
			if yLo.Lt(amount) && !y.Lt(amount) {
				return interpolateAmountIn(xLo, yLo, x, y, amount), nil
			}
		} else if !x.Lt(amount) {
			return interpolateAmountOut(xLo, yLo, x, y, amount), nil
		}
		xLo, yLo, sampled = x, y, true
	}
	if !sampled || yLo.IsZero() {
		return nil, ErrInvalidRate
	}

	// beyond the last sample, at its rate
	var result uint256.Int
	if isExactOut {
		return u256.MulDivUp(&result, amount, xLo, yLo), nil
	}
	return u256.MulDivDown(&result, amount, yLo, xLo), nil
}

// interpolateAmountOut returns floor(y) on the line through (xLo, yLo) and (xHi, yHi) at x in (xLo, xHi].
// If (xLo, yLo) lies on the floored line through (0, 0) and (xHi, yHi), as for a linear vault, it uses that
// line instead: it matches on-chain to the wei, while interpolating would carry yLo's rounding error.
func interpolateAmountOut(xLo, yLo, xHi, yHi, x *uint256.Int) *uint256.Int {
	var result, dx, dy uint256.Int
	if u256.MulDivDown(&result, xLo, yHi, xHi).Eq(yLo) {
		return u256.MulDivDown(&result, x, yHi, xHi)
	}
	dx.Sub(xHi, xLo)
	if !yHi.Lt(yLo) {
		u256.MulDivDown(&result, result.Sub(x, xLo), dy.Sub(yHi, yLo), &dx)
		return result.Add(yLo, &result)
	}
	u256.MulDivUp(&result, result.Sub(x, xLo), dy.Sub(yLo, yHi), &dx)
	return result.Sub(yLo, &result)
}

// interpolateAmountIn returns the min x with interpolateAmountOut(x) >= y, for y in (yLo, yHi].
func interpolateAmountIn(xLo, yLo, xHi, yHi, y *uint256.Int) *uint256.Int {
	var result, dx, dy uint256.Int
	if u256.MulDivDown(&result, xLo, yHi, xHi).Eq(yLo) {
		return u256.MulDivUp(&result, y, xHi, yHi)
	}
	u256.MulDivUp(&result, result.Sub(y, yLo), dx.Sub(xHi, xLo), dy.Sub(yHi, yLo))
	return result.Add(xLo, &result)
}

// NativeCap returns a vault-reported cap, or nil when it is unusable: not read or reverted (nil), zero (the
// vault reports no usable limit for the dummy owner) or above MaxUint128 (unbounded).
func NativeCap(v *big.Int) *big.Int {
	if v == nil || v.Sign() == 0 || v.Cmp(bignumber.MaxUint128) > 0 {
		return nil
	}
	return v
}

// Reserves returns the pool reserves [shares, assets]. Each slot is the most of that token you can receive,
// in its own units: a deposit limit (assets) converts to shares and a redeem limit (shares) to assets with the
// totalSupply/totalAssets ratio. A vault's own bounded maxMint (shares) / maxWithdraw (assets) wins over that
// conversion since the vault accounts for offsets, fees and caps. A nil limit means unbounded (totalSupply /
// totalAssets). A non-nil zero maxRedeem stays "0", and so does the redeem side of a vault with no redeem gas
// (redeem unsupported). Without a usable ratio, a limit is kept as is.
func Reserves(cfg VaultCfg, s *PoolState) entity.PoolReserves {
	var supply, assets, lim, out uint256.Int
	if s.TotalSupply != nil {
		supply.SetFromBig(s.TotalSupply)
	}
	if s.TotalAssets != nil {
		assets.SetFromBig(s.TotalAssets)
	}
	canConvert := s.TotalSupply != nil && !assets.IsZero()
	// convert returns limit * num / denom (512-bit product), saturating at 2^256-1 on overflow.
	convert := func(limit *big.Int, num, denom *uint256.Int) string {
		lim.SetFromBig(limit)
		if _, overflow := out.MulDivOverflow(&lim, num, denom); overflow {
			out.SetAllOne()
		}
		return out.Dec()
	}

	var shares, assetOut string
	switch {
	case s.MaxMint != nil:
		shares = s.MaxMint.String()
	case s.MaxDeposit != nil && canConvert:
		shares = convert(s.MaxDeposit, &supply, &assets)
	case s.MaxDeposit != nil:
		shares = s.MaxDeposit.String()
	case s.TotalSupply != nil:
		shares = supply.Dec()
	default: // totalSupply read failed: fall back to totalAssets
		shares = assets.Dec()
	}
	switch {
	case cfg.Gas.Redeem == 0: // redeem unsupported
		assetOut = "0"
	case s.MaxWithdraw != nil:
		assetOut = s.MaxWithdraw.String()
	case s.MaxRedeem != nil && canConvert && !supply.IsZero():
		assetOut = convert(s.MaxRedeem, &assets, &supply)
	case s.MaxRedeem != nil:
		assetOut = s.MaxRedeem.String()
	case s.TotalSupply != nil:
		assetOut = assets.Dec()
	default:
		assetOut = "0"
	}
	return entity.PoolReserves{shares, assetOut}
}
