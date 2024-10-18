package syncswapv2classic

import _ "embed"

//go:embed abi/Master.json
var masterABIData []byte

//go:embed abi/ClassicPool.json
var classicPoolABIData []byte
