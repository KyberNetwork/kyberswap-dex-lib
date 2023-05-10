package valueobject

// SimpleRoute contains minimal data of a route
type SimpleRoute struct {
	// Distributions distribution rate of amountIn in basis point
	Distributions []uint64 `json:"distributions"`

	// Paths contains data of a path
	Paths [][]SimpleSwap `json:"paths"`
}

// SimpleSwap ...
type SimpleSwap struct {
	PoolAddress     string `json:"poolAddress"`
	TokenInAddress  string `json:"tokenInAddress"`
	TokenOutAddress string `json:"tokenOutAddress"`
}

func (s *SimpleRoute) ExtractPoolAddresses() []string {
	poolAddressSet := make(map[string]struct{})

	for _, path := range s.Paths {
		for _, swap := range path {
			poolAddressSet[swap.PoolAddress] = struct{}{}
		}
	}

	poolAddresses := make([]string, 0, len(poolAddressSet))
	for poolAddress := range poolAddressSet {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	return poolAddresses
}
