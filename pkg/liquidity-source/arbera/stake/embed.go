package arberastake

import _ "embed"

//go:embed pools/berachain.json
var berachainPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/berachain.json": berachainPoolData,
}
