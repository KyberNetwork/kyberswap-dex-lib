package valueobject

// SimpleRoute contains minimal data of a route
type SimpleRoute struct {
	// Distributions distribution rate of amountIn in basis point
	Distributions []int64 `json:"distributions"`

	// Paths contains data of a path
	Paths [][]Swap `json:"paths"`
}

// SimpleSwap ...
type SimpleSwap struct {
	PoolID          string `json:"poolId"`
	TokenInAddress  string `json:"tokenInAddress"`
	TokenOutAddress string `json:"tokenOutAddress"`
}
