package btr

import _ "embed"

// BTR does not ship a Go pricing model; simulation is done on-chain via
// getSwapQuote (see pool_simulator.go). The Pool ABI is the only embedded
// artifact, kept as a file so `go:generate` can refresh it from the repo pin.
//
//go:embed abis/Pool.json
var poolABIJSONBytes []byte
