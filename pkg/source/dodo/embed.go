package dodo

import _ "embed"

//go:embed abi/DodoV1Pool.json
var v1PoolData []byte

//go:embed abi/DodoV2Pool.json
var v2PoolData []byte
