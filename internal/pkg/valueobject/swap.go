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

	// SwapAmount amount of TokenIn to swap
	SwapAmount *big.Int `json:"swapAmount"`

	// AmountOut amount of TokenOut received
	AmountOut *big.Int `json:"amountOut"`

	// Exchange name of exchange
	Exchange Exchange `json:"exchange"`

	// PoolType type of the pool
	PoolType string `json:"poolType"`

	// Extra metadata of the pool
	PoolExtra any `json:"poolExtra"`

	// Extra metadata of swap
	Extra any `json:"extra"`
}

const RouteIDInExtra = "ri"
