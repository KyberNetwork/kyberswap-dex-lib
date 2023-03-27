package valueobject

import (
	"math/big"
)

// Swap contains data of a swap
type Swap struct {
	// Pool address of the pool for swapping
	Pool string `json:"pool"`

	// TokenIn address of token swap from
	TokenIn string `json:"tokenIn"`

	// TokenOut address of token swap to
	TokenOut string `json:"tokenOut"`

	// LimitReturnAmount
	LimitReturnAmount *big.Int `json:"limitReturnAmount"`

	// SwapAmount amount of TokenIn to swap
	SwapAmount *big.Int `json:"swapAmount"`

	// AmountOut amount of TokenOut received
	AmountOut *big.Int `json:"amountOut"`

	// Exchange name of exchange
	Exchange Exchange `json:"exchange"`

	// PoolLength number of tokens inside the pools
	PoolLength int `json:"poolLength"`

	// PoolType type of the pool
	PoolType string `json:"poolType"`

	// Extra metadata of the pool
	PoolExtra interface{} `json:"poolExtra"`

	// EXtra metadata of swap
	Extra interface{} `json:"extra"`
}
