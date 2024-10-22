package syncswapv2aqua

import _ "embed"

//go:embed abi/Master.json
var masterABIData []byte

//go:embed abi/AquaPool.json
var aquaPoolABIData []byte

//go:embed abi/FeeManagerV2.json
var feeManagerV2ABIData []byte
