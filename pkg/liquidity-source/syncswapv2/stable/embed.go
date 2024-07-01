package syncswapv2stable

import _ "embed"

//go:embed abi/Master.json
var masterABIData []byte

//go:embed abi/StablePool.json
var stablePoolABIData []byte
