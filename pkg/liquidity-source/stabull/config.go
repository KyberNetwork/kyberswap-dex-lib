package stabull

type Config struct {
	DexID          string   `json:"dexID"`
	FactoryAddress string   `json:"factoryAddress"`
	PoolAddresses  []string `json:"poolAddresses,omitempty"` // Optional: List of known pool addresses for initial implementation
	FromBlock      uint64   `json:"fromBlock"`               // Block to start scanning NewCurve events

	// Chainlink oracle feeds
	// Maps token address -> Chainlink aggregator address
	// e.g., NZDS -> NZD/USD feed, USDC -> USDC/USD feed
	ChainlinkOracles map[string]string `json:"chainlinkOracles,omitempty"`
}
