package midas

import (
	"github.com/goccy/go-json"
)

func unmarshalVault(staticExtra StaticExtra, poolExtra string,
	mTokenDecimals, tokenDecimals uint8) (IDepositVault, IRedemptionVault, error) {
	if staticExtra.IsDepositVault {
		var dVault VaultState
		if err := json.Unmarshal([]byte(poolExtra), &dVault); err != nil {
			return nil, nil, err
		}
		return NewDepositVault(&dVault, mTokenDecimals, tokenDecimals), nil, nil
	}

	switch staticExtra.VaultType {
	case redemptionVault:
		var dVault VaultState
		if err := json.Unmarshal([]byte(poolExtra), &dVault); err != nil {
			return nil, nil, err
		}
		return nil, NewRedemptionVault(&dVault, mTokenDecimals, tokenDecimals), nil
	case redemptionVaultSwapper:
		var dVault RedemptionVaultWithSwapperState
		if err := json.Unmarshal([]byte(poolExtra), &dVault); err != nil {
			return nil, nil, err
		}
		return nil, NewRedemptionVaultSwapper(&dVault, mTokenDecimals, tokenDecimals), nil
	case redemptionVaultUstb:
		var dVault RedemptionVaultWithUstbState
		if err := json.Unmarshal([]byte(poolExtra), &dVault); err != nil {
			return nil, nil, err
		}
		return nil, NewRedemptionVaultUstb(&dVault, mTokenDecimals, tokenDecimals), nil
	default:
		return nil, nil, ErrNotSupported
	}
}
