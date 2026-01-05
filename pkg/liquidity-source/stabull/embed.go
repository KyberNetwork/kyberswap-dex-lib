package stabull

import _ "embed"

//go:embed abis/StabullFactory.json
var stabullFactoryABIData []byte

//go:embed abis/StabullPool.json
var stabullPoolABIData []byte

// Optionally, if you need to monitor Chainlink oracles directly:
//
//go:embed abis/ChainlinkAggregator.json
var chainlinkAggregatorABIData []byte
