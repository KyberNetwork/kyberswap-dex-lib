package makerpsm

import _ "embed"

//go:embed dexconfig/ethereum.json
var ethereumDexConfigBytes []byte

var bytesByPath = map[string][]byte{
	"dexconfig/ethereum.json": ethereumDexConfigBytes,
}

//go:embed abis/PSM.json
var makerPSMPSMBytes []byte

//go:embed abis/Vat.json
var makerPSMVatBytes []byte
