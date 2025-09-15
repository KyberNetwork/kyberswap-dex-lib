package midas

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/midas"
)

type RedemptionVaultSwapper struct {
	*ManageableVault
}

func NewRedemptionVaultSwapper() *RedemptionVaultSwapper {
	return &RedemptionVaultSwapper{}
}

func (r *RedemptionVaultSwapper) CalcAndValidateRedeem(amountMTokenIn *uint256.Int) (*midas.SwapInfo, error) {
	if r.tokenRemoved {
		return nil, midas.ErrTokenRemoved
	}
}
