package gblin

import _ "embed"

//go:embed abis/GBLIN.json
var vaultABIData []byte

//go:embed abis/GBLINLens.json
var lensABIData []byte

//go:embed abis/AggregatorV3.json
var aggregatorV3ABIData []byte

//go:embed abis/Multicall3.json
var multicall3ABIData []byte
