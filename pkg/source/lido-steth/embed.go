package lido_steth

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed abis/stETH.json
var stEthABIData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
