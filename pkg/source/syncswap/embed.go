package syncswap

import _ "embed"

//go:embed abi/Master.json
var masterABIData []byte

//go:embed abi/ClassicPool.json
var classicPoolABIData []byte

//go:embed abi/StablePool.json
var stablePoolABIData []byte
