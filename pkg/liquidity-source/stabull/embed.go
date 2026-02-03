package stabull

import _ "embed"

//go:embed abis/StabullFactory.json
var stabullFactoryABIData []byte

//go:embed abis/StabullPool.json
var stabullPoolABIData []byte

//go:embed abis/Assimilator.json
var assimilatorABIData []byte

//go:embed abis/ChainlinkAggregator.json
var chainlinkAggregatorABIData []byte
