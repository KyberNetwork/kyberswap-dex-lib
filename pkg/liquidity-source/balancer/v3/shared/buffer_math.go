package shared

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/holiman/uint256"
)

var (
	WAD                   = uint256.NewInt(1e18) // 10**18
	MINIMUM_WRAP_AMOUNT   = uint256.NewInt(10000)
	ErrInvalidRate        = errors.New("invalid rate")
	ErrWrapAmountTooSmall = errors.New("wrap amount too small")
)

func (b *ExtraBuffer) ConvertToShares(assets *uint256.Int) (*uint256.Int, error) {
	if assets.Lt(MINIMUM_WRAP_AMOUNT) {
		return nil, ErrWrapAmountTooSmall
	}

	return erc4626.GetClosestRate(b.DepositRates, assets)
}

func (b *ExtraBuffer) ConvertToAssets(shares *uint256.Int) (*uint256.Int, error) {
	if shares.Lt(MINIMUM_WRAP_AMOUNT) {
		return nil, ErrWrapAmountTooSmall
	}

	return erc4626.GetClosestRate(b.RedeemRates, shares)
}
