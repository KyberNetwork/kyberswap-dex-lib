package generic_simple_rate

import "math/big"

type Config struct {
	DexID           string   `json:"dexID"`
	PoolPath        string   `json:"poolPath"`
	PausedMethod    string   `json:"pausedMethod"`
	RateMethod      string   `json:"rateMethod"`
	DefaultRate     *big.Int `json:"defaultRate"`
	RateUnit        *big.Int `json:"rateUnit"`
	IsRateInversed  bool     `json:"isRateInversed"`
	IsRateUpdatable bool     `json:"isRateUpdatable"`
	IsBidirectional bool     `json:"isBidirectional"`
	DefaultGas      *big.Int `json:"defaultGas"`

	// If IsRateInversed = true, amountToken0 = amountToken1 * rateUnit / rate
	// If IsRateInversed = false, amountToken0 = amountToken1 * rate / rateUnit
}
