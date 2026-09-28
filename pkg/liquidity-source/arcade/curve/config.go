package curve

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config wires the pool factory and tracker to the ArcadeHook deployments of a chain.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Hook is an ArcadeHook address: it emits LaunchCreated for every launch and
	// exposes buy/sell for the bonding-curve phase of PUMP launches.
	Hook string `json:"hook"`
	// Hooks lists further ArcadeHook deployments served by this source. The curve is the
	// same contract code and math on each generation (on Arc: v1
	// 0x695cfF9C7F11fa87ca05c7b0A0fa64C3554B3eCe, v2 0x7706d261f0C370e8f0E273164A603E885C89beC2);
	// a launch trades on the hook that created it, recorded in its StaticExtra. Hook and
	// Hooks are merged, so a config written for a single hook keeps working.
	Hooks []string `json:"hooks"`
}

// isHook reports whether a log emitter is one of the configured ArcadeHook deployments.
func (c *Config) isHook(address common.Address) bool {
	if c.Hook != "" && common.HexToAddress(c.Hook) == address {
		return true
	}
	for _, hook := range c.Hooks {
		if hook != "" && common.HexToAddress(hook) == address {
			return true
		}
	}
	return false
}

// defaultHook is the hook assumed for a pool saved without one in its StaticExtra.
func (c *Config) defaultHook() string {
	if c.Hook != "" || len(c.Hooks) == 0 {
		return c.Hook
	}
	return c.Hooks[0]
}
