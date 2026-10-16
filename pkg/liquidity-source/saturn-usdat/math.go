package saturnusdat

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// convertAmounts ports MultiMint._convertAmounts: scale `amount` from
// fromDecimals to toDecimals, flooring when scaling down and using checked
// multiplication (revert on overflow) when scaling up.
// https://etherscan.io/address/0x496a4a33b6181f4536203488d9a05ac1429e702c#code
func convertAmounts(res *uint256.Int, fromDecimals, toDecimals uint8, amount *uint256.Int) error {
	switch {
	case fromDecimals == toDecimals:
		res.Set(amount)
	case fromDecimals > toDecimals:
		diff := fromDecimals - toDecimals
		if diff > maxTenPow {
			res.Clear() // amount / 10**diff where 10**diff > 2^256 always floors to 0
		} else {
			res.Div(amount, big256.TenPow(diff))
		}
	default:
		diff := toDecimals - fromDecimals
		if diff > maxTenPow {
			return ErrOverflow
		}
		if _, overflow := res.MulOverflow(amount, big256.TenPow(diff)); overflow {
			return ErrOverflow
		}
	}
	return nil
}

// wrapAmounts ports the amount handling of MultiMint._wrapAsset and
// isAllowedToWrap for an asset with `assetDecimals` against an extension with
// `extDecimals`. It returns the extension tokens minted (extAmount) and the
// asset amount actually pulled (effective); effective < amount only when
// scaling down leaves dust, which SwapFacility refunds to the caller.
// The cap check is cap >= balance + effective (checked add), and the new balance must fit uint240.
func wrapAmounts(assetDecimals, extDecimals uint8, amount, assetCap, assetBalance *uint256.Int,
) (extAmount, effective uint256.Int, err error) {
	if amount.IsZero() {
		return extAmount, effective, ErrZeroAmount
	}
	if err = convertAmounts(&extAmount, assetDecimals, extDecimals, amount); err != nil {
		return
	}
	if extAmount.IsZero() {
		return extAmount, effective, ErrZeroAmount
	}
	// Cannot fail: effective <= amount (scale-down floors first, so scaling
	// back up never exceeds the original amount).
	_ = convertAmounts(&effective, extDecimals, assetDecimals, &extAmount)

	var total uint256.Int
	if _, overflow := total.AddOverflow(assetBalance, &effective); overflow {
		return extAmount, effective, ErrOverflow
	}
	if assetCap.Lt(&total) {
		return extAmount, effective, ErrAssetCapReached
	}
	// The cap is uint256 but the balance is stored as uint240 (checked), so a cap above 2^240
	// still reverts once the new balance no longer fits.
	if total.BitLen() > 240 {
		return extAmount, effective, ErrOverflow
	}
	return extAmount, effective, nil
}

// remainingCapacity is cap - balance, saturating at zero.
func remainingCapacity(assetCap, assetBalance *uint256.Int) (res uint256.Int) {
	if assetCap.Gt(assetBalance) {
		res.Sub(assetCap, assetBalance)
	}
	return res
}
