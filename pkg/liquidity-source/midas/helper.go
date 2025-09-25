package midas

import (
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
)

type rvConfig struct {
	Address string
	RvType  VaultType
	MToken  string

	LiquidityProvider     string
	MTbillRedemptionVault string

	UstbRedemption  string
	SuperstateToken string
}

var rvConfigs map[string]map[string]rvConfig

func init() {
	rvConfigs = make(map[string]map[string]rvConfig)
	for path, byteData := range bytesByPath {
		var mTokenConfigs map[string]MTokenConfig
		if err := json.Unmarshal(byteData, &mTokenConfigs); err != nil {
			panic("failed to unmarshal midas config")
		}

		rvConfigs[path] = make(map[string]rvConfig)
		for _, cfg := range mTokenConfigs {
			address := strings.ToLower(cfg.RedemptionVault)
			rvConfigs[path][address] = rvConfig{
				Address: address,
				MToken:  strings.ToLower(cfg.MToken),
				RvType:  cfg.RedemptionVaultType,

				LiquidityProvider:     strings.ToLower(cfg.LiquidityProvider),
				MTbillRedemptionVault: strings.ToLower(cfg.MTbillRedemptionVault),

				UstbRedemption:  cfg.UstbRedemption,
				SuperstateToken: cfg.SuperstateToken,
			}
		}
	}
}

func newVault(vaultState *VaultState, vaultType VaultType, mTokenDecimals, tokenDecimals uint8) (IDepositVault, IRedemptionVault, error) {
	if vaultState == nil {
		return nil, nil, nil
	}

	switch vaultType {
	case depositVault:
		return NewDepositVault(vaultState, mTokenDecimals, tokenDecimals), nil, nil
	case redemptionVault:
		return nil, NewRedemptionVault(vaultState, mTokenDecimals, tokenDecimals), nil
	case redemptionVaultUstb:
		return nil, NewRedemptionVaultUstb(vaultState, mTokenDecimals, tokenDecimals), nil
	case redemptionVaultSwapper:
		return nil, NewRedemptionVaultSwapper(vaultState, mTokenDecimals, tokenDecimals), nil
	default:
		logger.Errorf("not supported vault %v", vaultType)
		return nil, nil, ErrNotSupported
	}
}
