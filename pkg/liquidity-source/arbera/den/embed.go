package arberaden

import _ "embed"

//go:embed abis/IndexManager.json
var indexManagerABIData []byte

//go:embed abis/WeightedIndex.json
var weightedIndexABIData []byte
