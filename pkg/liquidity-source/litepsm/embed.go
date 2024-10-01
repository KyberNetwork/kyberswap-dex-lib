package litepsm

import _ "embed"

//go:embed dexconfig/ethereum.json
var ethereumDexConfigBytes []byte

var bytesByPath = map[string][]byte{
	"dexconfig/ethereum.json": ethereumDexConfigBytes,
}

//go:embed abis/ERC20.json
var erc20ABIBytes []byte

//go:embed abis/LitePSM.json
var litePSMBytes []byte
