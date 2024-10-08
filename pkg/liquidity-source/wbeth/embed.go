package wbeth

import _ "embed"

//go:embed abis/wBETH.json
var wbethABIData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
