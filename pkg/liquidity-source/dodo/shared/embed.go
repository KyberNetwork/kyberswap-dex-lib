package shared

import _ "embed"

//go:embed blacklist/bsc.txt
var bscBlacklistFilePath []byte

var BytesByPath = map[string][]byte{
	"blacklist/bsc.txt": bscBlacklistFilePath,
}

//go:embed abis/DodoV1Pool.json
var v1PoolABIJson []byte

//go:embed abis/DodoV2Pool.json
var v2PoolABIJson []byte
