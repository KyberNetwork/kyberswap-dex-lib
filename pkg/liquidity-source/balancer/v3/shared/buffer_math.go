package shared

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/holiman/uint256"
)

var (
	WAD                   = uint256.NewInt(1e18) // 10**18
	MinimumWrapAmount     = uint256.NewInt(10000)
	ErrInvalidRate        = errors.New("invalid rate")
	ErrWrapAmountTooSmall = errors.New("wrap amount too small")
)

func (b *ExtraBuffer) ConvertToShares(amount *uint256.Int, isExactOut bool) (*uint256.Int, error) {
	if !isExactOut {
		// exact in, amount is amountIn, must validate with minimum wrap amount and max deposit
		if amount.Lt(MinimumWrapAmount) {
			return nil, ErrWrapAmountTooSmall
		}
		if b.MaxDeposit != nil && amount.Gt(b.MaxDeposit) {
			return nil, ErrMaxDepositExceeded
		}
	}

	result, err := erc4626.GetClosestRate(b.DepositRates, amount, isExactOut)
	if err != nil {
		return nil, err
	}

	if isExactOut {
		// exact out, result is amountIn, must validate with minimum wrap amount and max deposit
		if result.Lt(MinimumWrapAmount) {
			return nil, ErrWrapAmountTooSmall
		}
		if b.MaxDeposit != nil && result.Gt(b.MaxDeposit) {
			return nil, ErrMaxDepositExceeded
		}
	}

	return result, nil
}

func (b *ExtraBuffer) ConvertToAssets(amount *uint256.Int, isExactOut bool) (*uint256.Int, error) {
	if !isExactOut {
		if amount.Lt(MinimumWrapAmount) {
			return nil, ErrWrapAmountTooSmall
		}
		if b.MaxRedeem != nil && amount.Gt(b.MaxRedeem) {
			return nil, ErrMaxRedeemExceeded
		}
	}

	result, err := erc4626.GetClosestRate(b.RedeemRates, amount, isExactOut)
	if err != nil {
		return nil, err
	}

	if isExactOut {
		if result.Lt(MinimumWrapAmount) {
			return nil, ErrWrapAmountTooSmall
		}
		if b.MaxRedeem != nil && result.Gt(b.MaxRedeem) {
			return nil, ErrMaxRedeemExceeded
		}
	}

	return result, nil
}
