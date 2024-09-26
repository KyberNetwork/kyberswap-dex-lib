package generic_fixed_rate

import "math/big"

type Config struct {
	DexID       string   `json:"dexID"`
	Pool        string   `json:"pool"`
	RateMethod  string   `json:"rateMethod"`
	RateDefault *big.Int `json:"rateDefault"`
}
