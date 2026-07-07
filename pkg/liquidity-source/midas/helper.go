package midas

import (
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func newVault(state *VaultState, vaultType VaultType, tokenDecimals map[string]uint8) (IDepositVault, IRedemptionVault, error) {
	if state == nil {
		return nil, nil, nil
	}

	switch vaultType {
	case depositVault:
		return NewDepositVault(state, tokenDecimals), nil, nil
	case redemptionVault:
		return nil, NewRedemptionVault(state, tokenDecimals), nil
	case redemptionVaultUstb:
		return nil, NewRedemptionVaultUstb(state, tokenDecimals), nil
	case redemptionVaultSwapper:
		return nil, NewRedemptionVaultSwapper(state, tokenDecimals), nil
	default:
		logger.Errorf("not supported vault %v", vaultType)
		return nil, nil, ErrNotSupported
	}
}

func toU256Slice(s []*big.Int) []*uint256.Int {
	return lo.Map(s, func(b *big.Int, _ int) *uint256.Int {
		return uint256.MustFromBig(b)
	})
}
