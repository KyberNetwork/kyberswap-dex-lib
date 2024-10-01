package generic_simple_rate

import "math/big"

type Config struct {
	DexID         string `json:"dexID"`
	Pools         string `json:"pools"`
	ABIJsonString string `json:"abi"`
	PausedMethod  string `json:"pausedMethod"`
	// Ensure amountToken0 * rate = amountToken1 * rateUnit for swapping token0 to token1,
	// and by default, token0 must be swappable to token1
	RateMethod      string   `json:"rateMethod"`
	DefaultRate     *big.Int `json:"defaultRate"`
	RateUnit        *big.Int `json:"rateUnit"`
	IsRateUpdatable bool     `json:"isRateUpdatable"`
	IsBidirectional bool     `json:"isBidirectional"`
	DefaultGas      *big.Int `json:"defaultGas"`
}
