package generic_rate

import "math/big"

type Config struct {
	DexID      string   `json:"dexID"`
	PoolPath   string   `json:"poolPath"`
	DefaultGas *big.Int `json:"defaultGas"`
}
