package valueobject

import (
	"math/big"
)

// Route output data of finding route process
type Route struct {
	// InputAmount amount of token to swap
	InputAmount *big.Int `json:"inputAmount"`

	// OutputAmount amount of token received
	OutputAmount *big.Int `json:"outputAmount"`

	// TotalGas amount of gas consumed for swapping
	TotalGas int64 `json:"totalGas"`

	// Route contains detail paths and swaps
	Route [][]Swap `json:"route"`
}
