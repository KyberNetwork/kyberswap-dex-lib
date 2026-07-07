package wbeth

import _ "embed"

//go:embed abis/wBETH.json
var wbethABIData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed pools/bsc.json
var bscPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
	"pools/bsc.json":      bscPoolData,
}
