package btr

import _ "embed"

// BTR does not ship a Go pricing model; simulation is done on-chain via
// getSwapQuote (see pool_simulator.go). The Pool ABI is the only embedded
// artifact: a pinned copy of the deployed BTR Pool ABI.
//
//go:embed abis/Pool.json
var poolABIJSONBytes []byte
