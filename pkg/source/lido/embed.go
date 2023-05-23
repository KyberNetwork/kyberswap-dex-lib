package lido

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed abis/WstETH.json
var wstETHABIData []byte

//go:embed abis/ERC20.json
var erc20ABIData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
