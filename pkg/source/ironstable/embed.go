package ironstable

import _ "embed"

//go:embed pools/avalanche.json
var avalanchePoolsBytes []byte

//go:embed pools/polygon.json
var polygonPoolsBytes []byte

//go:embed abis/IronSwap.json
var ironSwapBytes []byte

//go:embed abis/ERC20.json
var erc20Bytes []byte

var bytesByPath = map[string][]byte{
	"pools/avalanche.json": avalanchePoolsBytes,
	"pools/polygon.json":   polygonPoolsBytes,
}
