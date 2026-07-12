package ringswapbacking

import "github.com/ethereum/go-ethereum/common"

type RouterConfig struct {
	Address             string `json:"address"`
	ReplaceOrdinaryPair bool   `json:"replaceOrdinaryPair"`
	NoRecallGasToken0   int64  `json:"noRecallGasToken0"`
	NoRecallGasToken1   int64  `json:"noRecallGasToken1"`
	RecallGasToken0     int64  `json:"recallGasToken0"`
	RecallGasToken1     int64  `json:"recallGasToken1"`
}

type Config struct {
	DexID   string         `json:"dexID"`
	Routers []RouterConfig `json:"routers"`
}

func (c *Config) validate() error {
	if c == nil || c.DexID == "" || len(c.Routers) == 0 {
		return ErrInvalidConfig
	}
	seen := make(map[common.Address]struct{}, len(c.Routers))
	for _, router := range c.Routers {
		address := common.HexToAddress(router.Address)
		if !common.IsHexAddress(router.Address) || address == (common.Address{}) ||
			!router.ReplaceOrdinaryPair || router.NoRecallGasToken0 <= 0 ||
			router.NoRecallGasToken1 <= 0 || router.RecallGasToken0 <= 0 ||
			router.RecallGasToken1 <= 0 || router.RecallGasToken0 < router.NoRecallGasToken0 ||
			router.RecallGasToken1 < router.NoRecallGasToken1 {
			return ErrInvalidConfig
		}
		if _, exists := seen[address]; exists {
			return ErrInvalidConfig
		}
		seen[address] = struct{}{}
	}
	return nil
}
