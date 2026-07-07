package shared

import _ "embed"

//go:embed abis/DodoV1Pool.json
var v1PoolABIJson []byte

//go:embed abis/DodoV2Pool.json
var v2PoolABIJson []byte
