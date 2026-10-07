package uscoreprop

import _ "embed"

//go:embed abi/USCorePool.json
var poolABIJson []byte

//go:embed abi/Multicall3Clock.json
var clockABIJson []byte
