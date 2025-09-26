package midas

import (
	"strings"
)

type Config struct {
	DexId    string `json:"dexId"`
	Executor string `json:"executor"`
	MTokens  map[string]MTokenConfig
}

type MTokenConfig struct {
	DvType VaultType `json:"dvT"`
	Dv     string    `json:"dv"`
	RvType VaultType `json:"rvT"`
	Rv     string    `json:"rv"`
}
type RvConfig struct {
	Address string
	MToken  string
	RvType  VaultType
}

func getRvConfig(mTokensCfg map[string]MTokenConfig) map[string]RvConfig {
	rvConfigs := make(map[string]RvConfig)
	for mToken, cfg := range mTokensCfg {
		address := strings.ToLower(cfg.Rv)
		rvConfigs[address] = RvConfig{
			Address: address,
			MToken:  strings.ToLower(mToken),
			RvType:  cfg.RvType,
		}
	}

	return rvConfigs
}
